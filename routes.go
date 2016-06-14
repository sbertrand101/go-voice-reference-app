package main

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"errors"
	"net/http"

	"strings"

	"github.com/appleboy/gin-jwt"
	"github.com/bandwidthcom/go-bandwidth"
	j "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/tuxychandru/pubsub"
)

// RegisterForm is used on used registering
type RegisterForm struct {
	UserName       string `form:"userName",json:"userName",binding:"required"`
	Password       string `form:"password",json:"password",binding:"required"`
	RepeatPassword string `form:"repeatPassword",json:"repeatPassword",binding:"required"`
	AreaCode       string `form:"areaCode",json:"areaCode",binding:"required"`
}

const beepURL = "https://s3.amazonaws.com/bwdemos/beep.mp3"
const tonesURL = "https://s3.amazonaws.com/bwdemos/media/ring.mp3"

func getRoutes(router *gin.Engine, db *gorm.DB, newVoiceMessageEvent *pubsub.PubSub) error {
	if newVoiceMessageEvent == nil {
		newVoiceMessageEvent = pubsub.New(1)
	}

	authMiddleware := &jwt.GinJWTMiddleware{
		Realm:      "Bandwidth",
		Key:        []byte("9SbPxeIyvoT3HkIQ19wN9p_e_b6Xb7iJ"),
		Timeout:    time.Hour * 24,
		MaxRefresh: time.Hour * 24 * 7,
		Authenticator: func(userId string, password string, c *gin.Context) (string, bool) {
			user := &User{}
			if db.First(user, "user_name = ?", userId).RecordNotFound() {
				return "", false
			}
			return strconv.FormatUint(uint64(user.ID), 10), user.ComparePasswords(password)
		},
		Authorizator: func(userId string, c *gin.Context) bool {
			user := &User{}
			id, err := strconv.ParseUint(userId, 10, 32)
			if err != nil {
				return false
			}
			if db.First(user, id).RecordNotFound() {
				return false
			}
			c.Set("user", user)
			return true
		},
		Unauthorized: setErrorMessage,
	}

	router.POST("/login", authMiddleware.LoginHandler)
	router.GET("/refreshToken", authMiddleware.MiddlewareFunc(), authMiddleware.RefreshHandler)

	router.POST("/register", func(c *gin.Context) {
		api := c.MustGet("catapultAPI").(catapultAPIInterface)
		form := &RegisterForm{}
		err := c.Bind(form)
		if err != nil {
			setError(c, http.StatusBadRequest, err)
			return
		}
		if form.UserName == "" || form.Password == "" || form.AreaCode == "" {
			setError(c, http.StatusBadRequest, errors.New("Missing some required fields"))
			return
		}
		if form.Password != form.RepeatPassword {
			setError(c, http.StatusBadRequest, errors.New("Passwords are mismatched"))
			return
		}
		user := &User{
			UserName: form.UserName,
			AreaCode: form.AreaCode,
		}
		if err = user.SetPassword(form.Password); err != nil {
			setError(c, http.StatusBadRequest, err)
			return
		}
		if !db.First(&User{}, "user_name = ?", form.UserName).RecordNotFound() {
			setError(c, http.StatusBadRequest, errors.New("User with such name is registered already"))
			return
		}
		debugf("Reserving phone number for area code %s\n", form.AreaCode)
		phoneNumber, err := api.CreatePhoneNumber(user.AreaCode)
		if err != nil {
			setError(c, http.StatusBadGateway, err, "Error on creating phone number: "+err.Error())
			return
		}
		debugf("Creating SIP account\n")
		sipAccount, err := api.CreateSIPAccount()
		if err != nil {
			setError(c, http.StatusBadGateway, err, "Error on creating SIP Account: "+err.Error())
			return
		}
		user.PhoneNumber = phoneNumber
		user.SIPURI = sipAccount.URI
		user.SIPPassword = sipAccount.Password
		user.EndpointID = sipAccount.EndpointID
		if err = db.Create(user).Error; err != nil {
			setError(c, http.StatusBadGateway, err, "Error on saving user's data")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"id": user.ID,
		})
	})

	router.GET("/sipData", authMiddleware.MiddlewareFunc(), func(c *gin.Context) {
		user := c.MustGet("user").(*User)
		api := c.MustGet("catapultAPI").(catapultAPIInterface)
		token, err := api.CreateSIPAuthToken(user.EndpointID)
		if err != nil {
			setError(c, http.StatusBadGateway, err, "Error on getting auth token for SIP account")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"phoneNumber": user.PhoneNumber,
			"sipUri":      user.SIPURI,
			"sipPassword": user.SIPPassword, // only to show to user. it is not used by webrtc auth
			"token":       token.Token,
			"expire":      time.Now().Add(time.Duration(token.Expires) * time.Second),
		})
	})

	router.GET("/callCallback", func(c *gin.Context) {
		debugf("Catapult Event: %v\n", c.Request.URL.RawQuery)
		api := c.MustGet("catapultAPI").(catapultAPIInterface)
		timerAPI := c.MustGet("timerAPI").(timerInterface)
		form := c.Request.URL.Query()
		user, _ := getUserForCall(form.Get("callId"), db)
		switch form.Get("eventType") {
		case "answer":
			user = &User{}
			from := form.Get("from")
			to := form.Get("to")
			callID := form.Get("callId")
			tag := form.Get("tag")
			values := strings.Split(tag, ":")
			if len(values) == 3 && values[0] == "AnotherLeg" {
				debugf("Another leg has answered\n")
				api.StopPlayAudioToCall(values[1]) // stop tones
				err := db.Create(&ActiveCall{
					CallID:   callID,
					BridgeID: values[2],
					UserID:   0, // user is not required for bridged calls
					From:     from,
					To:       to,
				}).Error
				debugf("Error on adding active call %v\n", err)
				c.String(http.StatusOK, "")
				return
			}

			if db.First(user, "sip_uri = ? OR phone_number = ?", from, to).RecordNotFound() {
				c.String(http.StatusOK, "")
				return
			}
			if to == user.PhoneNumber {
				debugf("Bridging incoming call with %q\n", user.SIPURI)
				callerID := getCallerID(db, from)
				api.PlayAudioToCall(callID, tonesURL, true, "")

				debugf("Using caller id %q\n", callerID)
				bridgeID, err := api.CreateBridge(&bandwidth.BridgeData{
					CallIDs:     []string{callID},
					BridgeAudio: true,
				})

				if err != nil {
					debugf("Error on creating a bridge: %s\n", err.Error())
					c.String(http.StatusOK, "")
					return
				}

				db.Create(&ActiveCall{
					CallID:   callID,
					BridgeID: bridgeID,
					UserID:   user.ID,
					From:     from,
					To:       to,
				})

				debugf("Calling to another leg %s\n", user.SIPURI)
				anotherCallID, err := api.CreateCall(&bandwidth.CreateCallData{
					BridgeID:           bridgeID,
					From:               callerID,
					To:                 user.SIPURI,
					Tag:                fmt.Sprintf("AnotherLeg:%v:%v", callID, bridgeID),
					CallbackHTTPMethod: "GET",
					CallbackURL:        fmt.Sprintf("http://%s/callCallback", c.Request.Host),
				})

				if err != nil {
					debugf("Error on creating a another leg call: %s\n", err.Error())
					c.String(http.StatusOK, "")
					return
				}

				go func() {
					timerAPI.Sleep(15 * time.Second)
					call, _ := api.GetCall(anotherCallID)
					debugf("Another call state %s\n", call.State)
					if call.State == "started" {
						api.StopPlayAudioToCall(callID)
						api.Hangup(anotherCallID)
						// redirect to voice mail after some seconds of waiting
						debugf("Moving to voice mail\n")
						if user.GreetingURL == "" {
							debugf("Play default greeting\n")
							// err = api.PlayAudioToCall(callID, beepURL, false, "Greeting")
							err = api.SpeakSentenceToCall(callID, "Hello. Please leave a message after beep.", "Greeting")
						} else {
							debugf("Play user's greeting\n")
							err = api.PlayAudioToCall(callID, user.GreetingURL, false, "Greeting")
						}
						debugf("Error on play greeting %v\n", err)
					}
				}()
				c.String(http.StatusOK, "")
				return
			}
			if from == user.SIPURI {
				debugf("Transfering outgoing call to  %q\n", to)
				c.Header("Content-Type", "text/xml")
				c.String(http.StatusOK, buildBXML(transferBXML(to, user.PhoneNumber, 0, "", "")))
				return
			}
			break
		case "playback", "speak":
			if form.Get("status") == "done" {
				switch form.Get("tag") {
				case "Greeting":
					debugf("Play beep\n")
					api.PlayAudioToCall(form.Get("callId"), beepURL, false, "Beep")
					break
				case "Beep":
					debugf("Starting call recording\n")
					api.UpdateCall(form.Get("callId"), &bandwidth.UpdateCallData{RecordingEnabled: true})
					break
				}

			}
			break
		case "hangup":
			callID := form.Get("callId")
			call, err := api.GetCall(callID)
			if err != nil {
				debugf("Error on getting call: %s\n", err.Error())
				break
			}
			recordings, err := api.GetCallRecordings(callID)
			if err != nil {
				debugf("Error on Getting call recordings: %s\n", err.Error())
				break
			}
			if user != nil && recordings != nil && len(recordings) > 0 {
				debugf("Saving recorded voice message to db\n")
				recording := recordings[0]
				message := &VoiceMailMessage{
					MediaURL:  recording.Media,
					StartTime: parseTime(recording.StartTime),
					EndTime:   parseTime(recording.EndTime),
					UserID:    user.ID,
					From:      getCallerID(db, call.From),
				}
				err = db.Create(message).Error
				if err != nil {
					debugf("Error on on saving voice mail message: %s\n", err.Error())
					break
				}

				// send notification about new voice mail message
				if newVoiceMessageEvent != nil {
					newVoiceMessageEvent.Pub(message, strconv.FormatUint(uint64(user.ID), 10))
				}
			}
			activeCalls := []ActiveCall{}
			activeCall := ActiveCall{}
			debugf("Hangup %+v\n", *call)
			if db.First(&activeCall, "call_id = ?", callID).RecordNotFound() {
				break
			}
			err = db.Debug().Find(&activeCalls, "bridge_id = ? AND call_id <> ?", activeCall.BridgeID, callID).Error
			if err != nil {
				debugf("Error on getting bridged calls: %s\n", err.Error())
				break
			}
			debugf("Hangup other %d calls\n", len(activeCalls))
			for _, call := range activeCalls {
				api.Hangup(call.CallID)
			}
			break
		}

		c.String(http.StatusOK, "")
	})

	router.POST("/recordGreeting", authMiddleware.MiddlewareFunc(), func(c *gin.Context) {
		api := c.MustGet("catapultAPI").(catapultAPIInterface)
		user := c.MustGet("user").(*User)
		callID, err := api.CreateCall(&bandwidth.CreateCallData{
			From:               user.PhoneNumber,
			To:                 user.SIPURI,
			CallbackHTTPMethod: "GET",
			CallbackURL:        fmt.Sprintf("http://%s/recordCallback", c.Request.Host),
		})
		if err != nil {
			setError(c, http.StatusBadGateway, err)
			return
		}
		db.Create(&ActiveCall{
			UserID: user.ID,
			CallID: callID,
			From:   user.PhoneNumber,
			To:     user.SIPURI,
		})
	})

	router.GET("/recordCallback", func(c *gin.Context) {
		api := c.MustGet("catapultAPI").(catapultAPIInterface)
		form := c.Request.URL.Query()
		debugf("Catapult Event for greeting record: %s\n", c.Request.URL.RawQuery)
		user, _ := getUserForCall(form.Get("callId"), db)
		mainMenu := func() string {
			return gatherBXML("/recordCallback",
				speakSentenceBXML("Press 1 to listen to your current greeting. Press 2 to record new greeting. Press 3 to set greeting to default."))
		}
		switch form.Get("eventType") {
		case "answer":
			debugf("Play voice menu\n")
			c.Header("Content-Type", "text/xml")
			c.String(http.StatusOK, buildBXML(mainMenu()))
			return
		case "gather":
			if form.Get("state") == "completed" {
				switch form.Get("digits") {
				case "1":
					debugf("Play greeting\n")
					c.Header("Content-Type", "text/xml")
					c.String(http.StatusOK, buildBXML(playGreeting(user), mainMenu()))
					return
				case "2":
					debugf("Record greeting\n")
					c.Header("Content-Type", "text/xml")
					c.String(http.StatusOK, buildBXML(speakSentenceBXML("Say your greeting after beep. Press 0 to complete recording."),
						playBeep(), recordBXML("/recordCallback", "0")))
					return
				case "3":
					debugf("Reset greeting\n")
					user.GreetingURL = ""
					err := db.Save(user).Error
					if err != nil {
						debugf("Error on saving user's data %s\n", err.Error())
						break
					}
					c.Header("Content-Type", "text/xml")
					c.String(http.StatusOK, buildBXML(speakSentenceBXML("Your greeting has been set to default."),
						mainMenu()))
					return
				}

			}
			break
		case "recording":
			if form.Get("state") == "complete" {
				callID := form.Get("callId")
				recordingID := form.Get("recordingId")
				recording, err := api.GetRecording(recordingID)
				if err != nil {
					debugf("Error getting recording data: %s\n", err.Error())
					break
				}
				user.GreetingURL = recording.Media
				err = db.Save(user).Error
				if err != nil {
					debugf("Error on saving user's data %s\n", err.Error())
					break
				}
				call, err := api.GetCall(callID)
				if err != nil {
					debugf("Error getting call data: %s\n", err.Error())
					break
				}
				if call.State == "active" {
					c.Header("Content-Type", "text/xml")
					c.String(http.StatusOK, buildBXML(speakSentenceBXML("Your greeting has been saved."),
						mainMenu()))
					return
				}
			}
			break
		}

		c.String(http.StatusOK, "")
	})

	router.GET("/voiceMessages", authMiddleware.MiddlewareFunc(), func(c *gin.Context) {
		user := c.MustGet("user").(*User)
		list := []VoiceMailMessage{}
		err := db.Order("start_time desc").Model(user).Related(&list).Error
		if err != nil {
			setError(c, http.StatusBadGateway, err, "Error on getting voice messages")
			return
		}
		result := make([]interface{}, len(list))
		for i, m := range list {
			result[i] = m.ToJSONObject()
		}
		c.JSON(http.StatusOK, result)
	})

	router.GET("/voiceMessages/:id/media", authMiddleware.MiddlewareFunc(), func(c *gin.Context) {
		api := c.MustGet("catapultAPI").(catapultAPIInterface)
		user := c.MustGet("user").(*User)
		message := &VoiceMailMessage{}
		err := db.Where("user_id = ? and id = ?", user.ID, c.Param("id")).First(message).Error
		if err != nil {
			setError(c, http.StatusBadGateway, err, "Error on getting voice message data")
			return
		}
		parts := strings.Split(message.MediaURL, "/")
		reader, contentType, err := api.DownloadMediaFile(parts[len(parts)-1])
		if err != nil {
			setError(c, http.StatusBadGateway, err, "Error on downloading media file")
			return
		}
		defer reader.Close()
		c.Header("Content-Type", contentType)
		length, _ := io.Copy(c.Writer, reader)
		c.Header("Content-Length", strconv.FormatInt(length, 10))
	})

	router.DELETE("/voiceMessages/:id", authMiddleware.MiddlewareFunc(), func(c *gin.Context) {
		user := c.MustGet("user").(*User)
		err := db.Where("user_id = ? and id = ?", user.ID, c.Param("id")).Delete(VoiceMailMessage{}).Error
		if err != nil {
			setError(c, http.StatusBadGateway, err, "Error on removing a voice message")
			return
		}
		c.Status(http.StatusOK)
	})

	router.GET("/voiceMessagesStream", func(c *gin.Context) {

		tokenString := c.Query("token")
		token, err := j.Parse(tokenString, func(token *j.Token) (interface{}, error) {
			if j.GetSigningMethod("HS256") != token.Method {
				return nil, errors.New("Invalid signing algorithm")
			}
			return authMiddleware.Key, nil
		})
		if err != nil {
			setError(c, http.StatusBadRequest, err, "Error on validating JWT token")
			return
		}
		user := &User{}
		userID := token.Claims["id"].(string)
		err = db.First(user, userID).Error
		if err != nil {
			setError(c, http.StatusBadGateway, err, "Error on getting user's data")
			return
		}
		channel := newVoiceMessageEvent.Sub(userID)
		defer newVoiceMessageEvent.Unsub(channel)
		debugf("Started streaming of new voice messages\n")
		c.Stream(func(w io.Writer) bool {
			return streamNewVoceMailMessage(c, channel)
		})
	})

	router.StaticFile("/", "./public/index.html")
	return nil
}

type sseEmiter interface {
	SSEvent(name string, message interface{})
}

func streamNewVoceMailMessage(c sseEmiter, channel chan interface{}) bool {
	message := <-channel
	json := message.(*VoiceMailMessage).ToJSONObject()
	debugf("Received new message %+v\n", json)
	c.SSEvent("message", json)
	return true
}

func getUserForCall(callID string, db *gorm.DB) (*User, error) {
	call := &ActiveCall{}
	user := &User{}
	if callID == "" {
		return nil, errors.New("callId is empty")
	}
	err := db.First(&call, "call_id=?", callID).Error
	if err != nil {
		return nil, err
	}
	err = db.First(user, call.UserID).Error
	return user, err
}

func playGreeting(user *User) string {
	if user.GreetingURL == "" {
		return speakSentenceBXML("Hello. Please leave a message after beep.")
	}
	return playAudioBXML(user.GreetingURL)
}

func playBeep() string {
	return playAudioBXML(beepURL)
}

func getCallerID(db *gorm.DB, number string) string {
	user := &User{}
	if strings.Index(number, "sip:") == 0 && !db.First(user, "sip_uri = ?", number).RecordNotFound() {
		// try to use phone number for caller id instead of sip uri
		return user.PhoneNumber
	}
	return number
}

func setErrorMessage(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{
		"code":    code,
		"message": message,
	})
}

func setError(c *gin.Context, code int, err error, message ...string) {
	c.Error(err)
	var errorMessage string
	if len(message) > 0 {
		errorMessage = message[0]
	} else {
		errorMessage = err.Error()
	}
	setErrorMessage(c, code, errorMessage)
}

func parseTime(isoTime string) time.Time {
	time, _ := time.Parse(time.RFC3339Nano, isoTime)
	return time
}

func debugf(format string, a ...interface{}) {
	if gin.IsDebugging() {
		format = "[routes] " + format
		if len(a) > 0 {
			fmt.Printf(format, a)
		} else {
			fmt.Print(format)
		}
	}
}
