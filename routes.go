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

// CallbackForm is used for call callbacks
type CallbackForm struct {
	From        string `json:"from"`
	To          string `json:"to"`
	State       string `json:"state"`
	EventType   string `json:"eventType"`
	CallID      string `json:"callId"`
	Tag         string `json:"tag"`
	RecordingID string `json:"recordingId"`
	Digits      string `json:"digits"`
}

const beepURL = "https://s3.amazonaws.com/bwdemos/beep.mp3"

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

	router.POST("/callCallback", func(c *gin.Context) {
		form := &CallbackForm{}
		api := c.MustGet("catapultAPI").(catapultAPIInterface)
		timerAPI := c.MustGet("timerAPI").(timerInterface)
		err := c.Bind(form)
		debugf("Catapult Event: %+v\n", *form)
		if err != nil {
			setError(c, http.StatusBadRequest, err)
			return
		}
		user := &User{}
		if !db.First(user, "sip_uri = ? OR phone_number = ?", form.From, form.To).RecordNotFound() {
			if form.EventType == "answer" {
				if form.To == user.PhoneNumber {
					debugf("Transfering incoming call  to  %q\n", user.SIPURI)
					callerID := form.From
					anotherUser := &User{}
					if strings.Index(callerID, "sip:") == 0 && !db.First(anotherUser, "sip_uri = ?", callerID).RecordNotFound() {
						// try to use phone number for caller id instead of sip uri
						callerID = anotherUser.PhoneNumber
					}
					debugf("Using caller id %q\n", callerID)
					transferedCallID, _ := api.UpdateCall(form.CallID, &bandwidth.UpdateCallData{
						State:            "transferring",
						TransferTo:       user.SIPURI,
						TransferCallerID: callerID,
						CallbackURL:      fmt.Sprintf("http://%s/transferCallback", c.Request.Host), // to handle redirection to voice mail
					})
					go func() {
						debugf("Waiting for answer call %s", transferedCallID)
						timerAPI.Sleep(15 * time.Second)
						call, _ := api.GetCall(transferedCallID)
						if call.State == "started" {
							// move to voice mail
							debugf("Moving call to voice mail")
							api.UpdateCall(transferedCallID, &bandwidth.UpdateCallData{
								State: "active",
								Tag:   strconv.FormatUint(uint64(user.ID), 10),
							})
						}
					}()
					return
				}
				if form.From == user.SIPURI {
					debugf("Transfering outgoing call to  %q\n", form.To)
					api.UpdateCall(form.CallID, &bandwidth.UpdateCallData{
						State:            "transferring",
						TransferTo:       form.To,
						TransferCallerID: user.PhoneNumber,
					})
					return
				}
			}
		}
		c.String(http.StatusOK, "")
	})

	router.POST("/transferCallback", func(c *gin.Context) {
		api := c.MustGet("catapultAPI").(catapultAPIInterface)
		form := &CallbackForm{}
		err := c.Bind(form)
		if err != nil {
			setError(c, http.StatusBadRequest, err)
			return
		}
		debugf("Catapult Event for transfered call: %+v\n", *form)
		if form.Tag != "" {
			handleVoiceMailEvent(form, db, api, newVoiceMessageEvent)
		}
		c.String(http.StatusOK, "")
	})

	router.POST("/recordGreeting", authMiddleware.MiddlewareFunc(), func(c *gin.Context) {
		api := c.MustGet("catapultAPI").(catapultAPIInterface)
		user := c.MustGet("user").(*User)
		api.CreateCall(&bandwidth.CreateCallData{
			From:        user.PhoneNumber,
			To:          user.SIPURI,
			CallbackURL: fmt.Sprintf("http://%s/recordCallback", c.Request.Host),
			Tag:         strconv.FormatUint(uint64(user.ID), 10),
		})
	})

	router.POST("/recordCallback", func(c *gin.Context) {
		api := c.MustGet("catapultAPI").(catapultAPIInterface)
		form := &CallbackForm{}
		err := c.Bind(form)
		if err != nil {
			setError(c, http.StatusBadRequest, err)
			return
		}
		debugf("Catapult Event for greeting record: %+v\n", *form)
		if form.EventType == "gather" && form.State == "completed" && form.Tag == "Record" {
			debugf("Stoping recording of call %s\n", form.CallID)
			api.UpdateCall(form.CallID, &bandwidth.UpdateCallData{
				RecordingEnabled: false,
			})
			c.String(http.StatusOK, "")
			return
		}
		user, err := getUserForCall(form, db, api)
		if err != nil {
			debugf("Error on getting user: %s\n", err.Error())
			return
		}
		mainMenu := func() {
			debugf("Main menu of greating recording\n")
			id, err := api.CreateGather(form.CallID, &bandwidth.CreateGatherData{
				MaxDigits:         1,
				InterDigitTimeout: 30,
				Prompt: &bandwidth.GatherPromptData{
					Gender:   "female",
					Voice:    "julie",
					Sentence: "Press 1 to listen to your current greeting. Press 2 to record new greeting. Press 3 to set greeting to default.",
				},
			})
			debugf("CreateGather result %v", []interface{}{id, err})
		}
		switch form.EventType {
		case "answer":
			mainMenu()
			break
		case "gather":
			{
				if form.State == "completed" {
					switch form.Digits {
					case "1":
						playGreeting(form.CallID, user, api)
						mainMenu()
					case "2":
						api.SpeakSentenceToCall(form.CallID, "Say your greeting after beep. Press any key to complete recording.")
						api.CreateGather(form.CallID, &bandwidth.CreateGatherData{
							MaxDigits:         1,
							InterDigitTimeout: 30,
							Prompt:            &bandwidth.GatherPromptData{FileURL: beepURL},
							Tag:               "Record",
						})
					case "3":
						user.GreetingURL = ""
						err := db.Save(user).Error
						if err != nil {
							debugf("Error on saving user's data %s", err.Error())
							break
						}
						api.SpeakSentenceToCall(form.CallID, "Your greeting has been set to default.")
						mainMenu()
					}

				}
				break
			}
		case "recording":
			if form.State == "complete" {
				recording, err := api.GetRecording(form.RecordingID)
				if err != nil {
					debugf("Error getting recording data: %s", err.Error())
					break
				}
				user.GreetingURL = recording.Media
				err = db.Save(user).Error
				if err != nil {
					debugf("Error on saving user's data %s", err.Error())
					break
				}
				call, err := api.GetCall(form.CallID)
				if err != nil {
					debugf("Error getting call data: %s", err.Error())
					break
				}
				if call.State == "active" {
					api.SpeakSentenceToCall(form.CallID, "Your greeting has been saved.")
					mainMenu()
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
			debugf("Error on validating JWT token: %s\n", err.Error())
			return
		}
		user := &User{}
		userID := token.Claims["id"].(string)
		err = db.First(user, userID).Error
		if err != nil {
			debugf("Error on getting user data: %s\n", err.Error())
			return
		}
		channel := newVoiceMessageEvent.Sub(userID)
		defer newVoiceMessageEvent.Unsub(channel)
		clientGone := c.Writer.CloseNotify()
		c.Stream(func(w io.Writer) bool {
			select {
			case message := <-channel:
				{
					json := message.(*VoiceMailMessage).ToJSONObject()
					debugf("Received new message %+v\n", json)
					c.SSEvent("message", json)
					break
				}
			case <-clientGone:
				{
					return false
				}
			}
			return true
		})
	})

	router.StaticFile("/", "./public/index.html")
	return nil
}

func handleVoiceMailEvent(form *CallbackForm, db *gorm.DB, api catapultAPIInterface, newVoiceMessageEvent *pubsub.PubSub) {
	debugf("Handle voice mail event\n")
	user, err := getUserForCall(form, db, api)
	if err != nil {
		debugf("Error on getting user: %s\n", err.Error())
		return
	}
	switch form.EventType {
	case "answer":
		playGreeting(form.CallID, user, api)
		api.PlayAudioToCall(form.CallID, beepURL)
		api.UpdateCall(form.CallID, &bandwidth.UpdateCallData{RecordingEnabled: true})
		break
	case "recording":
		if form.State == "complete" {
			debugf("Recording %s has been completed.\n", form.RecordingID)
			recording, _ := api.GetRecording(form.RecordingID)
			call, _ := api.GetCall(form.CallID)
			message := &VoiceMailMessage{
				MediaURL:  recording.Media,
				StartTime: parseTime(recording.StartTime),
				EndTime:   parseTime(recording.EndTime),
				UserID:    user.ID,
				From:      call.From,
			}
			err := db.Create(message).Error
			if err != nil {
				debugf("Error on on saving voice mail message: %s\n", err.Error())
				return
			}

			// send notification about new voice mail message
			if newVoiceMessageEvent != nil {
				newVoiceMessageEvent.Pub(message, strconv.FormatUint(uint64(user.ID), 10))
			}
		}
	}
}

func getUserForCall(form *CallbackForm, db *gorm.DB, api catapultAPIInterface) (*User, error) {
	user := &User{}
	if form.Tag == "" {
		call, err := api.GetCall(form.CallID)
		if err != nil {
			return nil, err
		}
		form.Tag = call.Tag
	}
	userID, err := strconv.ParseUint(form.Tag, 10, 32)
	if err != nil {
		return nil, err
	}
	err = db.First(user, uint(userID)).Error
	return user, err
}

func playGreeting(callID string, user *User, api catapultAPIInterface) {
	if user.GreetingURL == "" {
		api.SpeakSentenceToCall(callID, fmt.Sprintf("Hello. You have called to %s. Please leave a message after beep.", user.PhoneNumber))
	} else {
		api.PlayAudioToCall(callID, user.GreetingURL)
	}
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
