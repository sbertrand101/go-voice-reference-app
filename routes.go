package main

import (
	"fmt"
	"strconv"
	"time"

	"errors"
	"net/http"

	"github.com/appleboy/gin-jwt"
	"github.com/bandwidthcom/go-bandwidth"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
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
	From      string `json:"from"`
	To        string `json:"to"`
	EventType string `json:"eventType"`
	CallID    string `json:"callId"`
	Tag       string `json:"tag"`
}

var bridges map[string]string

func getRoutes(router *gin.Engine, db *gorm.DB) error {

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
		phoneNumber, err := api.CreatePhoneNumber(user.AreaCode)
		if err != nil {
			setError(c, http.StatusBadGateway, err, "Error on creating phone number: "+err.Error())
			return
		}
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
			"token":       token.Token,
			"expire":      time.Now().Add(time.Duration(token.Expires) * time.Second),
		})
	})

	router.POST("/callCallback", func(c *gin.Context) {
		form := &CallbackForm{}
		api := c.MustGet("catapultAPI").(catapultAPIInterface)
		err := c.Bind(form)
		fmt.Printf("Got: %s from %s to %s", form.EventType, form.From, form.To)
		if bridges == nil {
			bridges = make(map[string]string, 0)
		}
		if err != nil {
			setError(c, http.StatusBadRequest, err)
			return
		}
		user := &User{}
		if !db.First(user, "sip_uri = ? OR phone_number = ? OR phone_number = ?", form.From, form.From, form.To).RecordNotFound() {
			switch form.EventType {
			case "answer":
				handleAnswer(form, c, user, api)
			case "hangup":
				handleHangup(form, c, user, api)
			}
		}
		c.String(http.StatusOK, "")
	})

	router.StaticFile("/", "./public/index.html")

	return nil
}

func handleAnswer(form *CallbackForm, c *gin.Context, user *User, api catapultAPIInterface) {
	if form.Tag != "" {
		return
	}
	fmt.Printf("Answered %s -> %s\n", form.From, form.To)
	if form.To == user.PhoneNumber {
		fmt.Printf("Transfering call to  %s\n", user.SIPURI)
		api.UpdateCall(form.CallID, &bandwidth.UpdateCallData{
			State:            "transferring",
			TransferTo:       user.SIPURI,
			TransferCallerID: form.From,
		})
		return
	}
	fmt.Println("Play wait sound")
	api.PlayAudioToCall(form.CallID, &bandwidth.PlayAudioData{
		FileURL:     fmt.Sprintf("http://%s/audio/ring.mp3", c.Request.Host),
		LoopEnabled: true,
	})
	fmt.Println("Creating bridge")
	bridgeID, _ := api.CreateBridge(&bandwidth.BridgeData{
		CallIDs:     []string{form.CallID},
		BridgeAudio: true,
	})
	bridges[form.CallID] = bridgeID
	fmt.Printf("Making bridged call to another leg %s\n", form.To)
	callID, _ := api.MakeCall(&bandwidth.CreateCallData{
		From:        user.PhoneNumber,
		To:          form.To,
		BridgeID:    bridgeID,
		Tag:         form.CallID,
		CallbackURL: fmt.Sprintf("http://%s/callCallback", c.Request.Host),
	})
	bridges[callID] = bridgeID
}

func handleHangup(form *CallbackForm, c *gin.Context, user *User, api catapultAPIInterface) {
	bridgeID := bridges[form.CallID]
	if bridgeID == "" {
		return
	}
	calls, _ := api.GetBridgeCalls(bridgeID)
	for _, call := range calls {
		delete(bridges, call.ID)
		if call.State == "active" {
			fmt.Println("Hang up bridged call")
			api.Hangup(call.ID)
		}
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
