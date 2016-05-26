package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bandwidthcom/go-bandwidth"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouteRegister(t *testing.T) {
	data := gin.H{
		"userName":       "user1",
		"areaCode":       "910",
		"password":       "123456",
		"repeatPassword": "123456",
	}
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()

	api.On("CreatePhoneNumber", "910").Return("+1234567890", nil)
	api.On("CreateSIPAccount").Return(&sipAccount{
		EndpointID: "endpointId",
		URI:        "test@test.net",
		Password:   "12345678",
	}, nil)
	w := makeRequest(t, api, nil, db, http.MethodPost, "/register", "", data)
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
	user := &User{}
	assert.False(t, db.First(user, "user_name = ?", "user1").RecordNotFound())
	assert.Equal(t, "910", user.AreaCode)
	assert.Equal(t, "+1234567890", user.PhoneNumber)
	assert.Equal(t, "endpointId", user.EndpointID)
	assert.Equal(t, "test@test.net", user.SIPURI)
	assert.Equal(t, "12345678", user.SIPPassword)
	assert.True(t, user.ComparePasswords("123456"))
}

func TestRouteRegisterFailWithMismatchedPaswords(t *testing.T) {
	data := gin.H{
		"userName":       "user1",
		"areaCode":       "910",
		"password":       "123456",
		"repeatPassword": "000",
	}
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()

	w := makeRequest(t, api, nil, db, http.MethodPost, "/register", "", data)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	api.AssertNotCalled(t, "CreatePhoneNumber")
	api.AssertNotCalled(t, "CreateSIPAccount")
	user := &User{}
	assert.True(t, db.First(user, "user_name = ?", "user1").RecordNotFound())
}

func TestRouteRegisterFailWithShortPassword(t *testing.T) {
	data := gin.H{
		"userName":       "user1",
		"areaCode":       "910",
		"password":       "123",
		"repeatPassword": "123",
	}
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()

	w := makeRequest(t, api, nil, db, http.MethodPost, "/register", "", data)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	api.AssertNotCalled(t, "CreatePhoneNumber")
	api.AssertNotCalled(t, "CreateSIPAccount")
	user := &User{}
	assert.True(t, db.First(user, "user_name = ?", "user1").RecordNotFound())
}

func TestRouteRegisterFailWithMissingFields(t *testing.T) {
	data := gin.H{
		"password":       "123",
		"repeatPassword": "123",
	}
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()

	w := makeRequest(t, api, nil, db, http.MethodPost, "/register", "", data)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	api.AssertNotCalled(t, "CreatePhoneNumber")
	api.AssertNotCalled(t, "CreateSIPAccount")
	user := &User{}
	assert.True(t, db.First(user, "user_name = ?", "user1").RecordNotFound())
}

func TestRouteRegisterFailWithSameUser(t *testing.T) {
	data := gin.H{
		"userName":       "user1",
		"areaCode":       "910",
		"password":       "123456",
		"repeatPassword": "123456",
	}
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	db.Create(&User{UserName: "user1", AreaCode: "910"})
	w := makeRequest(t, api, nil, db, http.MethodPost, "/register", "", data)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	api.AssertNotCalled(t, "CreatePhoneNumber")
	api.AssertNotCalled(t, "CreateSIPAccount")
}

func TestRouteRegisterFailWithFailedCreatePhoneNumber(t *testing.T) {
	data := gin.H{
		"userName":       "user1",
		"areaCode":       "910",
		"password":       "123456",
		"repeatPassword": "123456",
	}
	api := &fakeCatapultAPI{}
	api.On("CreatePhoneNumber", "910").Return("", errors.New("Error"))
	db := openDBConnection(t)
	defer db.Close()
	w := makeRequest(t, api, nil, db, http.MethodPost, "/register", "", data)
	assert.Equal(t, http.StatusBadGateway, w.Code)
	api.AssertNotCalled(t, "CreateSIPAccount")
	user := &User{}
	assert.True(t, db.First(user, "user_name = ?", "user1").RecordNotFound())
}

func TestRouteRegisterFailWithFailedCreateSIPAccount(t *testing.T) {
	data := gin.H{
		"userName":       "user1",
		"areaCode":       "910",
		"password":       "123456",
		"repeatPassword": "123456",
	}
	api := &fakeCatapultAPI{}
	api.On("CreatePhoneNumber", "910").Return("+1234567890", nil)
	api.On("CreateSIPAccount").Return((*sipAccount)(nil), errors.New("Error"))
	db := openDBConnection(t)
	defer db.Close()
	w := makeRequest(t, api, nil, db, http.MethodPost, "/register", "", data)
	assert.Equal(t, http.StatusBadGateway, w.Code)
	user := &User{}
	assert.True(t, db.First(user, "user_name = ?", "user1").RecordNotFound())
}

func TestRouteLogin(t *testing.T) {
	data := gin.H{
		"userName": "user1",
		"password": "123456",
	}
	db := openDBConnection(t)
	defer db.Close()
	user := &User{UserName: "user1", AreaCode: "999"}
	user.SetPassword("123456")
	assert.NoError(t, db.Create(user).Error)
	result := map[string]string{}
	w := makeRequest(t, nil, nil, db, http.MethodPost, "/login", "", data, &result)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, result["token"])
	assert.NotEmpty(t, result["expire"])
}

func TestRouteLoginFailWithWrongPassword(t *testing.T) {
	data := gin.H{
		"userName": "user1",
		"password": "1234567",
	}
	db := openDBConnection(t)
	defer db.Close()
	user := &User{UserName: "user1", AreaCode: "999"}
	user.SetPassword("123456")
	assert.NoError(t, db.Create(user).Error)
	w := makeRequest(t, nil, nil, db, http.MethodPost, "/login", "", data)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRouteLoginFailWithWrongUserName(t *testing.T) {
	data := gin.H{
		"userName": "user2",
		"password": "123456",
	}
	db := openDBConnection(t)
	defer db.Close()
	user := &User{UserName: "user1", AreaCode: "999"}
	user.SetPassword("123456")
	assert.NoError(t, db.Create(user).Error)
	w := makeRequest(t, nil, nil, db, http.MethodPost, "/login", "", data)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRouteRefreshToken(t *testing.T) {
	db := openDBConnection(t)
	defer db.Close()
	token := createUserAndLogin(t, db)
	result := map[string]string{}
	w := makeRequest(t, nil, nil, db, http.MethodGet, "/refreshToken", token, nil, &result)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, result["token"])
	assert.NotEmpty(t, result["expire"])
}

func TestRouteSIPData(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	api.On("CreateSIPAuthToken", "789").Return(&bandwidth.DomainEndpointToken{
		Expires: 10,
		Token:   "123",
	}, nil)
	token := createUserAndLogin(t, db)
	result := map[string]string{}
	w := makeRequest(t, api, nil, db, http.MethodGet, "/sipData", token, nil, &result)
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
	assert.Equal(t, "test@test.net", result["sipUri"])
	assert.Equal(t, "123", result["token"])
	assert.NotEmpty(t, result["expire"])
}

func TestRouteSIPDataFail(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	api.On("CreateSIPAuthToken", "789").Return((*bandwidth.DomainEndpointToken)(nil), errors.New("Error"))
	token := createUserAndLogin(t, db)
	w := makeRequest(t, api, nil, db, http.MethodGet, "/sipData", token)
	assert.Equal(t, http.StatusBadGateway, w.Code)
	api.AssertExpectations(t)
}

func TestRouteSIPDataFailUnauthorized(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	w := makeRequest(t, api, nil, db, http.MethodGet, "/sipData", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRouteIndex(t *testing.T) {
	w := makeRequest(t, nil, nil, nil, http.MethodGet, "/", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRouteCallCallbackOutgoingCall(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:otest@test.com",
		PhoneNumber: "+1234567891",
		UserName:    "ouser",
	}
	user.SetPassword("123456")
	db.Save(user)
	api.On("UpdateCall", "callID", &bandwidth.UpdateCallData{
		State:            "transferring",
		TransferTo:       "+1472583690",
		TransferCallerID: "+1234567891",
	}).Return("", nil)
	w := makeRequest(t, api, nil, db, http.MethodPost, "/callCallback", "", &CallbackForm{
		CallID:    "callID",
		EventType: "answer",
		From:      "sip:otest@test.com",
		To:        "+1472583690",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteCallCallbackIncomingCall(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:itest@test.com",
		PhoneNumber: "+1234567892",
		UserName:    "iuser",
	}
	user.SetPassword("123456")
	db.Save(user)
	api.On("UpdateCall", "callID", &bandwidth.UpdateCallData{
		State:            "transferring",
		TransferTo:       "sip:itest@test.com",
		TransferCallerID: "+1472583688",
		CallbackURL:      "http:///transferCallback",
	}).Return("", nil)
	w := makeRequest(t, api, nil, db, http.MethodPost, "/callCallback", "", &CallbackForm{
		CallID:    "callID",
		EventType: "answer",
		From:      "+1472583688",
		To:        "+1234567892",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteCallCallbackIncomingCallSipToSip(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	user1 := &User{
		AreaCode:    "910",
		SIPURI:      "sip:i1test@test.com",
		PhoneNumber: "+1234567801",
		UserName:    "i1user",
	}
	user1.SetPassword("123456")
	db.Save(user1)
	user2 := &User{
		AreaCode:    "910",
		SIPURI:      "sip:i2test@test.com",
		PhoneNumber: "+1234567802",
		UserName:    "i2user",
	}
	user2.SetPassword("123456")
	db.Save(user2)
	api.On("UpdateCall", "callID", &bandwidth.UpdateCallData{
		State:            "transferring",
		TransferTo:       "sip:i1test@test.com",
		TransferCallerID: "+1234567802",
		CallbackURL:      "http:///transferCallback",
	}).Return("", nil)
	w := makeRequest(t, api, nil, db, http.MethodPost, "/callCallback", "", &CallbackForm{
		CallID:    "callID",
		EventType: "answer",
		From:      "sip:i2test@test.com",
		To:        "+1234567801",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteCallCallbackWithUnknownNumber(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	w := makeRequest(t, api, nil, db, http.MethodPost, "/callCallback", "", &CallbackForm{
		CallID:    "newCallID",
		EventType: "answer",
		From:      "+1472583688",
		To:        "+1456567890",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertNotCalled(t, "UpdateCall")
}

func TestRouteCallCallbackIncomingCallRedirectToVoiceMail(t *testing.T) {
	api := &fakeCatapultAPI{}
	timerAPI := &fakeTimerAPI{}
	db := openDBConnection(t)
	defer db.Close()
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:vmtest@test.com",
		PhoneNumber: "+1234567800",
		UserName:    "vmiuser",
	}
	user.SetPassword("123456")
	db.Save(user)
	api.On("UpdateCall", "callID", &bandwidth.UpdateCallData{
		State:            "transferring",
		TransferTo:       "sip:vmtest@test.com",
		TransferCallerID: "+1472583688",
		CallbackURL:      "http:///transferCallback",
	}).Return("111", nil)
	api.On("GetCall", "111").Return(&bandwidth.Call{
		State: "started",
	}, nil)
	api.On("UpdateCall", "111", &bandwidth.UpdateCallData{
		State: "active",
		Tag:   strconv.FormatUint(uint64(user.ID), 10),
	}).Return("111", nil)
	timerAPI.On("Sleep", 15*time.Second).Return()
	w := makeRequest(t, api, timerAPI, db, http.MethodPost, "/callCallback", "", &CallbackForm{
		CallID:    "callID",
		EventType: "answer",
		From:      "+1472583688",
		To:        "+1234567800",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	time.Sleep(5 * time.Millisecond)
	timerAPI.AssertExpectations(t)
	api.AssertExpectations(t)
}

func TestRouteCallCallbackIncomingCallDoNothingForAnsweredAndCompletedCalls(t *testing.T) {
	api := &fakeCatapultAPI{}
	timerAPI := &fakeTimerAPI{}
	db := openDBConnection(t)
	defer db.Close()
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:vmtest1@test.com",
		PhoneNumber: "+1234567800",
		UserName:    "vmi1user",
	}
	user.SetPassword("123456")
	db.Save(user)
	api.On("UpdateCall", "callID", &bandwidth.UpdateCallData{
		State:            "transferring",
		TransferTo:       "sip:vmtest1@test.com",
		TransferCallerID: "+1472583688",
		CallbackURL:      "http:///transferCallback",
	}).Return("111", nil)
	api.On("GetCall", "111").Return(&bandwidth.Call{
		State: "active",
	}, nil)
	timerAPI.On("Sleep", 15*time.Second).Return()
	w := makeRequest(t, api, timerAPI, db, http.MethodPost, "/callCallback", "", &CallbackForm{
		CallID:    "callID",
		EventType: "answer",
		From:      "+1472583688",
		To:        "+1234567800",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	time.Sleep(5 * time.Millisecond)
	timerAPI.AssertExpectations(t)
	api.AssertExpectations(t)
}

func TestRouteTransferCallbackDoNothingForMissingTag(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	w := makeRequest(t, api, nil, db, http.MethodPost, "/transferCallback", "", &CallbackForm{
		CallID:    "callID",
		EventType: "answer",
		From:      "+1472583688",
		To:        "+1234567800",
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRouteTransferCallbackDoNothingForMissingUser(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	w := makeRequest(t, api, nil, db, http.MethodPost, "/transferCallback", "", &CallbackForm{
		CallID:    "callID",
		EventType: "answer",
		From:      "+1472583688",
		To:        "+1234567800",
		Tag:       "0",
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRouteTransferCallbackDoNothingForWrongUserID(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	w := makeRequest(t, api, nil, db, http.MethodPost, "/transferCallback", "", &CallbackForm{
		CallID:    "callID",
		EventType: "answer",
		From:      "+1472583688",
		To:        "+1234567800",
		Tag:       "userID",
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRouteTransferCallbackAnswerCall(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:atest@test.com",
		PhoneNumber: "+1234567801",
		UserName:    "avmuser",
		GreetingURL: "greetingURL",
	}
	user.SetPassword("123456")
	db.Save(user)
	api.On("PlayAudioToCall", "callID", "greetingURL").Return(nil)
	api.On("PlayAudioToCall", "callID", beepURL).Return(nil)
	api.On("UpdateCall", "callID", &bandwidth.UpdateCallData{RecordingEnabled: true}).Return("", nil)
	w := makeRequest(t, api, nil, db, http.MethodPost, "/transferCallback", "", &CallbackForm{
		CallID:    "callID",
		EventType: "answer",
		From:      "+1472583688",
		To:        "+1234567801",
		Tag:       strconv.FormatUint(uint64(user.ID), 10),
	})
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteTransferCallbackAnswerCallWithDefaultGreeting(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:atest@test.com",
		PhoneNumber: "+1234567802",
		UserName:    "avm1user",
	}
	user.SetPassword("123456")
	db.Save(user)
	api.On("SpeakSentenceToCall", "callID", fmt.Sprintf("Hello. You have called to %s. Please leave a message after beep.", user.PhoneNumber)).Return(nil)
	api.On("PlayAudioToCall", "callID", beepURL).Return(nil)
	api.On("UpdateCall", "callID", &bandwidth.UpdateCallData{RecordingEnabled: true}).Return("", nil)
	w := makeRequest(t, api, nil, db, http.MethodPost, "/transferCallback", "", &CallbackForm{
		CallID:    "callID",
		EventType: "answer",
		From:      "+1472583688",
		To:        "+1234567802",
		Tag:       strconv.FormatUint(uint64(user.ID), 10),
	})
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteTransferCallbackRecordCall(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:atest@test.com",
		PhoneNumber: "+1234567803",
		UserName:    "avm2user",
	}
	user.SetPassword("123456")
	db.Save(user)
	db.Delete(&VoiceMailMessage{}, "user_id = ?", user.ID)
	api.On("GetRecording", "recordingID").Return(&bandwidth.Recording{
		Media:     "url",
		StartTime: "2016-05-26T10:00:00Z",
		EndTime:   "2016-05-26T10:01:00Z",
	}, nil)
	w := makeRequest(t, api, nil, db, http.MethodPost, "/transferCallback", "", &CallbackForm{
		CallID:      "callID",
		EventType:   "recording",
		State:       "complete",
		From:        "+1472583688",
		To:          "+1234567803",
		Tag:         strconv.FormatUint(uint64(user.ID), 10),
		RecordingID: "recordingID",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
	message := &VoiceMailMessage{}
	assert.NoError(t, db.First(message, "user_id = ?", user.ID).Error)
	assert.Equal(t, "url", message.MediaURL)
}

func TestRouteRecordGreeting(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	token := createUserAndLogin(t, db)
	user := &User{}
	db.First(user, "user_name = ?", "user1")
	api.On("CreateCall", &bandwidth.CreateCallData{
		From:        user.PhoneNumber,
		To:          user.SIPURI,
		CallbackURL: "http:///recordCallback",
		Tag:         strconv.FormatUint(uint64(user.ID), 10),
	}).Return("callId", nil)
	w := makeRequest(t, api, nil, db, http.MethodPost, "/recordGreeting", token)
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteRecordCallbackWithoutUser(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	w := makeRequest(t, api, nil, db, http.MethodPost, "/recordCallback", "", &CallbackForm{
		CallID:    "callID",
		EventType: "answer",
		Tag:       "0",
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRouteRecordCallbackAnswer(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:rtest@test.com",
		PhoneNumber: "+1334567800",
		UserName:    "ruser",
	}
	user.SetPassword("123456")
	db.Save(user)
	api.On("CreateGather", "callID", &bandwidth.CreateGatherData{
		MaxDigits:         1,
		InterDigitTimeout: 60,
		Prompt: &bandwidth.GatherPromptData{
			Gender:   "female",
			Voice:    "julie",
			Sentence: "Press 1 to listen to your current greeting. Press 2 to record new greeting. Press 3 to set greeting to default.",
		},
	}).Return("", nil)
	w := makeRequest(t, api, nil, db, http.MethodPost, "/recordCallback", "", &CallbackForm{
		CallID:    "callID",
		EventType: "answer",
		Tag:       strconv.FormatUint(uint64(user.ID), 10),
	})
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteRecordCallbackGather1(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:rtest1@test.com",
		PhoneNumber: "+1334567801",
		UserName:    "ruser1",
		GreetingURL: "greetingURL",
	}
	user.SetPassword("123456")
	db.Save(user)
	api.On("PlayAudioToCall", "callID", "greetingURL").Return(nil)
	api.On("CreateGather", "callID", &bandwidth.CreateGatherData{
		MaxDigits:         1,
		InterDigitTimeout: 60,
		Prompt: &bandwidth.GatherPromptData{
			Gender:   "female",
			Voice:    "julie",
			Sentence: "Press 1 to listen to your current greeting. Press 2 to record new greeting. Press 3 to set greeting to default.",
		},
	}).Return("", nil)
	w := makeRequest(t, api, nil, db, http.MethodPost, "/recordCallback", "", &CallbackForm{
		CallID:    "callID",
		EventType: "gather",
		State:     "completed",
		Tag:       strconv.FormatUint(uint64(user.ID), 10),
		Digits:    "1",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteRecordCallbackGather2(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:rtest2@test.com",
		PhoneNumber: "+1334567802",
		UserName:    "ruser2",
		GreetingURL: "greetingURL",
	}
	user.SetPassword("123456")
	db.Save(user)
	api.On("SpeakSentenceToCall", "callID", "Say your greeting after beep. Press any key to complete recording.").Return(nil)
	api.On("CreateGather", "callID", &bandwidth.CreateGatherData{
		MaxDigits:         1,
		InterDigitTimeout: 60,
		Prompt:            &bandwidth.GatherPromptData{FileURL: beepURL},
		Tag:               "Record",
	}).Return("", nil)
	w := makeRequest(t, api, nil, db, http.MethodPost, "/recordCallback", "", &CallbackForm{
		CallID:    "callID",
		EventType: "gather",
		State:     "completed",
		Tag:       strconv.FormatUint(uint64(user.ID), 10),
		Digits:    "2",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteRecordCallbackGather3(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:rtest2@test.com",
		PhoneNumber: "+1334567802",
		UserName:    "ruser3",
		GreetingURL: "greetingURL",
	}
	user.SetPassword("123456")
	db.Save(user)
	api.On("SpeakSentenceToCall", "callID", "Your greeting has been set to default.").Return(nil)
	api.On("CreateGather", "callID", &bandwidth.CreateGatherData{
		MaxDigits:         1,
		InterDigitTimeout: 60,
		Prompt: &bandwidth.GatherPromptData{
			Gender:   "female",
			Voice:    "julie",
			Sentence: "Press 1 to listen to your current greeting. Press 2 to record new greeting. Press 3 to set greeting to default.",
		},
	}).Return("", nil)
	w := makeRequest(t, api, nil, db, http.MethodPost, "/recordCallback", "", &CallbackForm{
		CallID:    "callID",
		EventType: "gather",
		State:     "completed",
		Tag:       strconv.FormatUint(uint64(user.ID), 10),
		Digits:    "3",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
	assert.NoError(t, db.First(user, user.ID).Error)
	assert.Empty(t, user.GreetingURL)

}

func TestRouteRecordCallbackGatherCompleteRecord(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	api.On("UpdateCall", "callID", &bandwidth.UpdateCallData{
		RecordingEnabled: false,
	}).Return("", nil)
	w := makeRequest(t, api, nil, db, http.MethodPost, "/recordCallback", "", &CallbackForm{
		CallID:    "callID",
		EventType: "gather",
		State:     "completed",
		Tag:       "Record",
		Digits:    "0",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteRecordCallbackSaveRecording(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:rtest4@test.com",
		PhoneNumber: "+1334567804",
		UserName:    "ruser4",
		GreetingURL: "greetingURL",
	}
	user.SetPassword("123456")
	db.Save(user)
	api.On("GetRecording", "recordingID").Return(&bandwidth.Recording{
		Media: "url",
	}, nil)
	api.On("GetCall", "callID").Return(&bandwidth.Call{
		State: "active",
	}, nil)
	api.On("SpeakSentenceToCall", "callID", "Your greeting has been saved.").Return(nil)
	api.On("CreateGather", "callID", &bandwidth.CreateGatherData{
		MaxDigits:         1,
		InterDigitTimeout: 60,
		Prompt: &bandwidth.GatherPromptData{
			Gender:   "female",
			Voice:    "julie",
			Sentence: "Press 1 to listen to your current greeting. Press 2 to record new greeting. Press 3 to set greeting to default.",
		},
	}).Return("", nil)
	w := makeRequest(t, api, nil, db, http.MethodPost, "/recordCallback", "", &CallbackForm{
		CallID:      "callID",
		EventType:   "recording",
		State:       "complete",
		Tag:         strconv.FormatUint(uint64(user.ID), 10),
		RecordingID: "recordingID",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
	assert.NoError(t, db.First(user, user.ID).Error)
	assert.Equal(t, "url", user.GreetingURL)

}

func TestRouteRecordCallbackSaveRecordingWithoutMenu(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:rtest5@test.com",
		PhoneNumber: "+1334567805",
		UserName:    "ruser5",
		GreetingURL: "greetingURL",
	}
	user.SetPassword("123456")
	db.Save(user)
	api.On("GetRecording", "recordingID").Return(&bandwidth.Recording{
		Media: "url",
	}, nil)
	api.On("GetCall", "callID").Return(&bandwidth.Call{
		State: "completed",
	}, nil)
	w := makeRequest(t, api, nil, db, http.MethodPost, "/recordCallback", "", &CallbackForm{
		CallID:      "callID",
		EventType:   "recording",
		State:       "complete",
		Tag:         strconv.FormatUint(uint64(user.ID), 10),
		RecordingID: "recordingID",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
	assert.NoError(t, db.First(user, user.ID).Error)
	assert.Equal(t, "url", user.GreetingURL)
}

func TestRouteRecordCallbackSaveRecordingWithoutMenu2(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:rtest6@test.com",
		PhoneNumber: "+1334567806",
		UserName:    "ruser6",
		GreetingURL: "greetingURL",
	}
	user.SetPassword("123456")
	db.Save(user)
	api.On("GetRecording", "recordingID").Return(&bandwidth.Recording{
		Media: "url",
	}, nil)
	api.On("GetCall", "callID").Return(&bandwidth.Call{}, errors.New("Error"))
	w := makeRequest(t, api, nil, db, http.MethodPost, "/recordCallback", "", &CallbackForm{
		CallID:      "callID",
		EventType:   "recording",
		State:       "complete",
		Tag:         strconv.FormatUint(uint64(user.ID), 10),
		RecordingID: "recordingID",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
	assert.NoError(t, db.First(user, user.ID).Error)
	assert.Equal(t, "url", user.GreetingURL)
}

func TestRouteRecordCallbackDoNothingForWrongRecordingID(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:rtest7@test.com",
		PhoneNumber: "+1334567807",
		UserName:    "ruser7",
		GreetingURL: "greetingURL",
	}
	user.SetPassword("123456")
	db.Save(user)
	api.On("GetRecording", "recordingID").Return(&bandwidth.Recording{}, errors.New("Error"))
	w := makeRequest(t, api, nil, db, http.MethodPost, "/recordCallback", "", &CallbackForm{
		CallID:      "callID",
		EventType:   "recording",
		State:       "complete",
		Tag:         strconv.FormatUint(uint64(user.ID), 10),
		RecordingID: "recordingID",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
	assert.NoError(t, db.First(user, user.ID).Error)
	assert.Equal(t, "greetingURL", user.GreetingURL)
}

func TestRouteGetVoiceMessages(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	token := createUserAndLogin(t, db)
	user := &User{}
	db.First(user, "user_name = ?", "user1")
	db.Delete(&VoiceMailMessage{}, "user_id = ?", user.ID)
	db.Create(&VoiceMailMessage{
		MediaURL:  "url1",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		UserID:    user.ID,
	})
	db.Create(&VoiceMailMessage{
		MediaURL:  "url2",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		UserID:    user.ID,
	})
	result := []map[string]interface{}{}
	w := makeRequest(t, api, nil, db, http.MethodGet, "/voiceMessages", token, nil, &result)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 2, len(result))
	assert.NotEmpty(t, result[0]["id"])
	assert.NotEmpty(t, result[1]["id"])
}

func TestRouteDownloadVoiceMessage(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	token := createUserAndLogin(t, db)
	user := &User{}
	db.First(user, "user_name = ?", "user1")
	db.Delete(&VoiceMailMessage{}, "user_id = ?", user.ID)
	message := &VoiceMailMessage{
		MediaURL:  "http://some-host/name1",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		UserID:    user.ID,
	}
	db.Create(message)
	api.On("DownloadMediaFile", "name1").Return(ioutil.NopCloser(strings.NewReader("1234")), "text/plain", nil)
	w := makeRequest(t, api, nil, db, http.MethodGet, fmt.Sprintf("/voiceMessages/%v/media", message.ID), token)
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
	assert.Equal(t, "text/plain", w.HeaderMap.Get("Content-Type"))
	assert.Equal(t, "4", w.HeaderMap.Get("Content-Length"))
	assert.Equal(t, "1234", w.Body.String())
}

func TestRouteDownloadVoiceMessageFail(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	token := createUserAndLogin(t, db)
	user := &User{}
	db.First(user, "user_name = ?", "user1")
	db.Delete(&VoiceMailMessage{}, "user_id = ?", user.ID)
	message := &VoiceMailMessage{
		MediaURL:  "http://some-host/name1",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		UserID:    user.ID,
	}
	db.Create(message)
	api.On("DownloadMediaFile", "name1").Return(ioutil.NopCloser(nil), "", errors.New("error"))
	w := makeRequest(t, api, nil, db, http.MethodGet, fmt.Sprintf("/voiceMessages/%v/media", message.ID), token)
	assert.Equal(t, http.StatusBadGateway, w.Code)
	api.AssertExpectations(t)
}

func TestRouteDeleteVoiceMessage(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	token := createUserAndLogin(t, db)
	user := &User{}
	db.First(user, "user_name = ?", "user1")
	db.Delete(&VoiceMailMessage{}, "user_id = ?", user.ID)
	message := &VoiceMailMessage{
		MediaURL:  "http://some-host/name1",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		UserID:    user.ID,
	}
	db.Create(message)
	w := makeRequest(t, api, nil, db, http.MethodDelete, fmt.Sprintf("/voiceMessages/%v", message.ID), token)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, db.First(message, message.ID).RecordNotFound())
}

func TestRouteDeleteVoiceMessageFail(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	token := createUserAndLogin(t, db)
	user := &User{}
	db.First(user, "user_name = ?", "user1")
	db.Delete(&VoiceMailMessage{}, "user_id = ?", user.ID)
	w := makeRequest(t, api, nil, db, http.MethodDelete, "/voiceMessages/unknown", token)
	assert.Equal(t, http.StatusBadGateway, w.Code)
}

func TestRouteGetVoiceMessageStream(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	token := createUserAndLogin(t, db)
	user := &User{}
	db.First(user, "user_name = ?", "user1")
	message := &VoiceMailMessage{
		MediaURL:  "http://some-host/name1",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		UserID:    user.ID,
	}
	db.Create(message)

	go func() {
		makeRequest(t, api, nil, db, http.MethodGet, "/voiceMessagesStream", token)
	}()
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, len(newVoiceMailChannels[user.ID]))
	newVoiceMailChannels[user.ID][0] <- message
	closer <- true
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 0, len(newVoiceMailChannels[user.ID]))
}

var closer chan bool

func init() {
	closer = make(chan bool)
}

func makeRequest(t *testing.T, api catapultAPIInterface, timerAPI timerInterface, db *gorm.DB, method, path, authToken string, body ...interface{}) *responseRecorder {
	os.Setenv("CATAPULT_USER_ID", "userID")
	os.Setenv("CATAPULT_API_TOKEN", "token")
	os.Setenv("CATAPULT_API_SECRET", "secret")
	router := gin.New()
	if timerAPI == nil {
		timerAPI = &timer{}
	}
	router.Use(func(c *gin.Context) {
		c.Set("catapultAPI", api)
		c.Set("timerAPI", timerAPI)
		c.Next()
	})
	require.NoError(t, getRoutes(router, db))
	var bodyIo io.Reader
	if len(body) > 0 && body[0] != nil {
		rawJSON, _ := json.Marshal(body[0])
		bodyIo = bytes.NewReader(rawJSON)
	}
	req, _ := http.NewRequest(method, path, bodyIo)
	if bodyIo != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	w := &responseRecorder{httptest.NewRecorder()}
	router.ServeHTTP(w, req)
	if strings.Contains(w.Header().Get("Content-Type"), "application/json") && len(body) > 1 && body[1] != nil {
		json.Unmarshal(w.Body.Bytes(), &body[1])
	}
	return w
}

type responseRecorder struct {
	*httptest.ResponseRecorder
}

func (r *responseRecorder) CloseNotify() <-chan bool {
	return closer
}

func createUserAndLogin(t *testing.T, db *gorm.DB) string {
	data := gin.H{
		"userName": "user1",
		"password": "123456",
	}
	db.Delete(&User{}, "user_name = ?", "user1")
	user := &User{
		UserName:    "user1",
		AreaCode:    "999",
		PhoneNumber: "+1234567890",
		SIPURI:      "test@test.net",
		SIPPassword: "654321",
		EndpointID:  "789",
	}
	user.SetPassword("123456")
	assert.NoError(t, db.Create(user).Error)
	result := map[string]string{}
	w := makeRequest(t, nil, nil, db, http.MethodPost, "/login", "", data, &result)
	assert.Equal(t, http.StatusOK, w.Code)
	return result["token"]
}
