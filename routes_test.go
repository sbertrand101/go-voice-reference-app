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
	"strings"
	"testing"
	"time"

	"github.com/bandwidthcom/go-bandwidth"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tuxychandru/pubsub"
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
	w := makeRequest(t, api, db, http.MethodPost, "/register", "", data)
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

	w := makeRequest(t, api, db, http.MethodPost, "/register", "", data)
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

	w := makeRequest(t, api, db, http.MethodPost, "/register", "", data)
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

	w := makeRequest(t, api, db, http.MethodPost, "/register", "", data)
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
	w := makeRequest(t, api, db, http.MethodPost, "/register", "", data)
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
	w := makeRequest(t, api, db, http.MethodPost, "/register", "", data)
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
	w := makeRequest(t, api, db, http.MethodPost, "/register", "", data)
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
	w := makeRequest(t, nil, db, http.MethodPost, "/login", "", data, &result)
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
	w := makeRequest(t, nil, db, http.MethodPost, "/login", "", data)
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
	w := makeRequest(t, nil, db, http.MethodPost, "/login", "", data)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRouteRefreshToken(t *testing.T) {
	db := openDBConnection(t)
	defer db.Close()
	token := createUserAndLogin(t, db)
	result := map[string]string{}
	w := makeRequest(t, nil, db, http.MethodGet, "/refreshToken", token, nil, &result)
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
	w := makeRequest(t, api, db, http.MethodGet, "/sipData", token, nil, &result)
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
	w := makeRequest(t, api, db, http.MethodGet, "/sipData", token)
	assert.Equal(t, http.StatusBadGateway, w.Code)
	api.AssertExpectations(t)
}

func TestRouteSIPDataFailUnauthorized(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	w := makeRequest(t, api, db, http.MethodGet, "/sipData", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRouteIndex(t *testing.T) {
	w := makeRequest(t, nil, nil, http.MethodGet, "/", "")
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
	w := makeRequest(t, api, db, http.MethodGet, "/callCallback?callId=callID&eventType=answer&from=sip:otest@test.com&to=%2B1472583690", "")
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteCallCallbackIncomingCallSimple(t *testing.T) {
	api := &fakeCatapultAPI{}
	api.On("PlayAudioToCall", "callID", tonesURL, true, "").Return(nil)
	api.On("CreateBridge", &bandwidth.BridgeData{
		CallIDs:     []string{"callID"},
		BridgeAudio: true,
	}).Return("bridgeId", nil)
	api.On("CreateCall", &bandwidth.CreateCallData{From: "+1472583688", RecordingFileFormat: "", RecordingEnabled: false, RecordingMaxDuration: 0, State: "", To: "sip:itest@test.com", TranscriptionEnabled: false, SipHeaders: map[string]string(nil), ConferenceID: "", BridgeID: "bridgeId", Tag: "AnotherLeg:callID", CallbackURL: "http:///callCallback", CallbackHTTPMethod: "GET", FallbackURL: "", CallbackTimeout: 0, CallTimeout: 10}).Return("callId1", nil)
	db := openDBConnection(t)
	defer db.Close()
	db.Delete(&ActiveCall{})
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:itest@test.com",
		PhoneNumber: "+1234567892",
		UserName:    "iuser",
	}
	user.SetPassword("123456")
	db.Save(user)
	w := makeRequest(t, api, db, http.MethodGet, "/callCallback?callId=callID&eventType=answer&from=%2B1472583688&to=%2B1234567892", "")
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
	count := 0
	db.Model(&ActiveCall{}).Count(&count)
	assert.Equal(t, 2, count)
}

func TestRouteCallCallbackIncomingCallSipToSip(t *testing.T) {
	api := &fakeCatapultAPI{}
	api.On("PlayAudioToCall", "callID", "https://s3.amazonaws.com/bwdemos/media/ring.mp3", true, "").Return(nil)
	api.On("CreateBridge", &bandwidth.BridgeData{BridgeAudio: true, CallIDs: []string{"callID"}}).Return("456", nil)
	api.On("CreateCall", &bandwidth.CreateCallData{From: "+1234567802", RecordingFileFormat: "", RecordingEnabled: false, RecordingMaxDuration: 0, State: "", To: "sip:i1test@test.com", TranscriptionEnabled: false, SipHeaders: map[string]string(nil), ConferenceID: "", BridgeID: "456", Tag: "AnotherLeg:callID", CallbackURL: "http:///callCallback", CallbackHTTPMethod: "GET", FallbackURL: "", CallbackTimeout: 0, CallTimeout: 10}).Return("callID", nil)
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
	w := makeRequest(t, api, db, http.MethodGet, "/callCallback?callId=callID&eventType=answer&from=sip:i2test@test.com&to=%2B1234567801", "")
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteCallCallbackIncomingCallFail1(t *testing.T) {
	api := &fakeCatapultAPI{}
	api.On("PlayAudioToCall", "callID", tonesURL, true, "").Return(nil)
	api.On("CreateBridge", &bandwidth.BridgeData{
		CallIDs:     []string{"callID"},
		BridgeAudio: true,
	}).Return("bridgeId", nil)
	api.On("CreateCall", &bandwidth.CreateCallData{From: "+1472583688", RecordingFileFormat: "", RecordingEnabled: false, RecordingMaxDuration: 0, State: "", To: "sip:itest@test.com", TranscriptionEnabled: false, SipHeaders: map[string]string(nil), ConferenceID: "", BridgeID: "bridgeId", Tag: "AnotherLeg:callID", CallbackURL: "http:///callCallback", CallbackHTTPMethod: "GET", FallbackURL: "", CallbackTimeout: 0, CallTimeout: 10}).Return("", errors.New("error"))
	db := openDBConnection(t)
	defer db.Close()
	db.Delete(&ActiveCall{})
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:itest@test.com",
		PhoneNumber: "+1234567892",
		UserName:    "iuser100",
	}
	user.SetPassword("123456")
	db.Save(user)
	w := makeRequest(t, api, db, http.MethodGet, "/callCallback?callId=callID&eventType=answer&from=%2B1472583688&to=%2B1234567892", "")
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
	count := 0
	db.Model(&ActiveCall{}).Count(&count)
	assert.Equal(t, 1, count)
}

func TestRouteCallCallbackIncomingCallFail2(t *testing.T) {
	api := &fakeCatapultAPI{}
	api.On("PlayAudioToCall", "callID", tonesURL, true, "").Return(nil)
	api.On("CreateBridge", &bandwidth.BridgeData{
		CallIDs:     []string{"callID"},
		BridgeAudio: true,
	}).Return("", errors.New("error"))
	db := openDBConnection(t)
	defer db.Close()
	db.Delete(&ActiveCall{})
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:itest@test.com",
		PhoneNumber: "+1234567892",
		UserName:    "iuser101",
	}
	user.SetPassword("123456")
	db.Save(user)
	w := makeRequest(t, api, db, http.MethodGet, "/callCallback?callId=callID&eventType=answer&from=%2B1472583688&to=%2B1234567892", "")
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
	count := 0
	db.Model(&ActiveCall{}).Count(&count)
	assert.Equal(t, 0, count)
}

func TestRouteCallCallbackAnswerAnotherLeg(t *testing.T) {
	api := &fakeCatapultAPI{}
	api.On("StopPlayAudioToCall", "callId").Return(nil)
	db := openDBConnection(t)
	defer db.Close()
	w := makeRequest(t, api, db, http.MethodGet, "/callCallback?callId=callID1&eventType=answer&from=%2B1472583688&to=%2B1234567892&&tag=AnotherLeg%3AcallId", "")
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteCallCallbackHangupWithBridgedCalls(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:i5test@test.com",
		PhoneNumber: "+1234567803",
		UserName:    "i5user",
	}
	user.SetPassword("123456")
	db.Save(user)
	db.Delete(&ActiveCall{}, "bridge_id = ? or call_id = ?", "bridgeID", "callId")
	activeCall := &ActiveCall{
		UserID:   user.ID,
		CallID:   "callId",
		BridgeID: "bridgeID",
	}
	db.Save(activeCall)
	activeCall = &ActiveCall{
		UserID:   user.ID,
		CallID:   "callId1",
		BridgeID: "bridgeID",
	}
	db.Save(activeCall)
	api.On("Hangup", "callId1").Return(nil)
	w := makeRequest(t, api, db, http.MethodGet, "/callCallback?callId=callId&eventType=hangup", "")
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteCallCallbackHangupDoNothingForMissingBridges(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	db.Delete(&ActiveCall{}, "call_id = ?", "callId")
	w := makeRequest(t, api, db, http.MethodGet, "/callCallback?callId=callId&eventType=hangup", "")
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteCallCallbackRecordingDoNothingForNonCompletedRecording(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	w := makeRequest(t, api, db, http.MethodGet, "/callCallback?callId=callId&eventType=recording&state=start", "")
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteCallCallbackRecordingForCompletedRecording(t *testing.T) {
	api := &fakeCatapultAPI{}
	api.On("GetRecording", "recordingId").Return(&bandwidth.Recording{
		Media: "url",
	}, nil)
	api.On("GetCall", "callId").Return(&bandwidth.Call{
		From: "+1472583690",
	}, nil)
	db := openDBConnection(t)
	defer db.Close()
	db.Delete(&ActiveCall{}, "call_id = ?", "callId")
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:r1test@test.com",
		PhoneNumber: "+1234567803",
		UserName:    "r1user",
	}
	user.SetPassword("123456")
	db.Save(user)
	activeCall := &ActiveCall{
		UserID: user.ID,
		CallID: "callId",
	}
	db.Save(activeCall)
	w := makeRequest(t, api, db, http.MethodGet, "/callCallback?callId=callId&eventType=recording&state=complete&recordingId=recordingId", "")
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
	voiceMessage := &VoiceMailMessage{}
	assert.NoError(t, db.First(voiceMessage, "user_id = ?", user.ID).Error)
	assert.Equal(t, "+1472583690", voiceMessage.From)
	assert.Equal(t, "url", voiceMessage.MediaURL)
}

func TestRouteCallCallbackRecordingForCompletedRecordingFail1(t *testing.T) {
	api := &fakeCatapultAPI{}
	api.On("GetRecording", "recordingId").Return(&bandwidth.Recording{}, errors.New("error"))
	db := openDBConnection(t)
	defer db.Close()
	db.Delete(&ActiveCall{}, "call_id = ?", "callId")
	db.Delete(&VoiceMailMessage{})
	w := makeRequest(t, api, db, http.MethodGet, "/callCallback?callId=callId&eventType=recording&state=complete&recordingId=recordingId", "")
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
	voiceMessage := &VoiceMailMessage{}
	assert.True(t, db.First(voiceMessage).RecordNotFound())
}

func TestRouteCallCallbackRecordingForCompletedRecordingFail2(t *testing.T) {
	api := &fakeCatapultAPI{}
	api.On("GetRecording", "recordingId").Return(&bandwidth.Recording{
		Media: "url",
	}, nil)
	api.On("GetCall", "callId").Return(&bandwidth.Call{}, errors.New("error"))
	db := openDBConnection(t)
	defer db.Close()
	db.Delete(&ActiveCall{}, "call_id = ?", "callId")
	db.Delete(&VoiceMailMessage{})
	w := makeRequest(t, api, db, http.MethodGet, "/callCallback?callId=callId&eventType=recording&state=complete&recordingId=recordingId", "")
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
	voiceMessage := &VoiceMailMessage{}
	assert.True(t, db.First(voiceMessage).RecordNotFound())
}

func TestRouteCallCallbackTimeoutWithDefaultGreeting(t *testing.T) {
	api := &fakeCatapultAPI{}
	api.On("StopPlayAudioToCall", "callId").Return(nil)
	api.On("SpeakSentenceToCall", "callId", "Hello. Please leave a message after beep.", "Greeting").Return(nil)
	db := openDBConnection(t)
	defer db.Close()
	db.Delete(&ActiveCall{}, "call_id = ?", "callId")
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:t1test@test.com",
		PhoneNumber: "+1234567803",
		UserName:    "t1user",
	}
	user.SetPassword("123456")
	db.Save(user)
	activeCall := &ActiveCall{
		UserID: user.ID,
		CallID: "callId",
	}
	db.Save(activeCall)
	w := makeRequest(t, api, db, http.MethodGet, "/callCallback?callId=callId1&eventType=timeout&tag=AnotherLeg%3AcallId", "")
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteCallCallbackTimeoutWithUserGreeting(t *testing.T) {
	api := &fakeCatapultAPI{}
	api.On("StopPlayAudioToCall", "callId").Return(nil)
	api.On("PlayAudioToCall", "callId", "url", false, "Greeting").Return(nil)
	db := openDBConnection(t)
	defer db.Close()
	db.Delete(&ActiveCall{}, "call_id = ?", "callId")
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:t2test@test.com",
		PhoneNumber: "+1234567805",
		UserName:    "t2user",
		GreetingURL: "url",
	}
	user.SetPassword("123456")
	db.Save(user)
	activeCall := &ActiveCall{
		UserID: user.ID,
		CallID: "callId",
	}
	db.Save(activeCall)
	w := makeRequest(t, api, db, http.MethodGet, "/callCallback?callId=callId1&eventType=timeout&tag=AnotherLeg%3AcallId", "")
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteCallCallbackPlaybackGreeting(t *testing.T) {
	api := &fakeCatapultAPI{}
	api.On("PlayAudioToCall", "callId", beepURL, false, "Beep").Return(nil)
	db := openDBConnection(t)
	defer db.Close()
	w := makeRequest(t, api, db, http.MethodGet, "/callCallback?callId=callId&eventType=playback&status=done&tag=Greeting", "")
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteCallCallbackPlaybackBeep(t *testing.T) {
	api := &fakeCatapultAPI{}
	api.On("UpdateCall", "callId", &bandwidth.UpdateCallData{RecordingEnabled: true}).Return("", nil)
	db := openDBConnection(t)
	defer db.Close()
	w := makeRequest(t, api, db, http.MethodGet, "/callCallback?callId=callId&eventType=playback&status=done&tag=Beep", "")
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteCallCallbackPlaybackDoNothingForNonCompletedStatus(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	w := makeRequest(t, api, db, http.MethodGet, "/callCallback?callId=callId&eventType=playback&status=started&tag=Beep", "")
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteCallCallbackPlaybackDoNothingForInvalidTag(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	w := makeRequest(t, api, db, http.MethodGet, "/callCallback?callId=callId&eventType=playback&status=done&tag=Unknown", "")
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteCallCallbackWithUnknownNumber(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	db.Delete(&ActiveCall{})
	w := makeRequest(t, api, db, http.MethodGet, "/callCallback?callId=newCallID&eventType=answer&from=+1472583688&to=+1456567890", "")
	assert.Equal(t, http.StatusOK, w.Code)
	count := 0
	db.Find(&ActiveCall{}).Count(&count)
	assert.Equal(t, 0, count)
}

func TestRouteRecordGreeting(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	token := createUserAndLogin(t, db)
	user := &User{}
	db.First(user, "user_name = ?", "user1")
	api.On("CreateCall", &bandwidth.CreateCallData{
		From:               user.PhoneNumber,
		To:                 user.SIPURI,
		CallbackHTTPMethod: "GET",
		CallbackURL:        "http:///recordCallback",
	}).Return("callID", nil)
	w := makeRequest(t, api, db, http.MethodPost, "/recordGreeting", token)
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteRecordCallbackWithoutUser(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	db.Delete(&ActiveCall{})
	w := makeRequest(t, api, db, http.MethodGet, "/recordCallback?callId=callId&eventType=answer", "")
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
	w := makeRequest(t, api, db, http.MethodGet, "/recordCallback?callId=callID&eventType=answer", "")
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
	activeCall := &ActiveCall{
		UserID: user.ID,
		CallID: "callId",
	}
	db.Save(activeCall)
	w := makeRequest(t, api, db, http.MethodGet, "/recordCallback?callId=callId&eventType=gather&state=completed&digits=1", "")
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}

func TestRouteRecordCallbackGather1WithDefautGreeting(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	user := &User{
		AreaCode:    "910",
		SIPURI:      "sip:rtest10@test.com",
		PhoneNumber: "+1334567810",
		UserName:    "ruser10",
	}
	user.SetPassword("123456")
	db.Save(user)
	activeCall := &ActiveCall{
		UserID: user.ID,
		CallID: "callId",
	}
	db.Save(activeCall)
	w := makeRequest(t, api, db, http.MethodGet, "/recordCallback?callId=callId&eventType=gather&state=completed&digits=1", "")
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
	activeCall := &ActiveCall{
		UserID: user.ID,
		CallID: "callId",
	}
	db.Save(activeCall)
	w := makeRequest(t, api, db, http.MethodGet, "/recordCallback?callId=callId&eventType=gather&state=completed&digits=2", "")
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
	activeCall := &ActiveCall{
		UserID: user.ID,
		CallID: "callId",
	}
	db.Save(activeCall)
	w := makeRequest(t, api, db, http.MethodGet, "/recordCallback?callId=callId&eventType=gather&state=completed&digits=3", "")
	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
	assert.NoError(t, db.First(user, user.ID).Error)
	assert.Empty(t, user.GreetingURL)

}

func TestRouteRecordCallbackGatherCompleteRecord(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	w := makeRequest(t, api, db, http.MethodGet, "/recordCallback?callId=callID&eventType=gather&state=completed&digits=0", "")
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
	activeCall := &ActiveCall{
		UserID: user.ID,
		CallID: "callId",
	}
	db.Save(activeCall)
	api.On("GetRecording", "recordingID").Return(&bandwidth.Recording{
		Media: "url",
	}, nil)
	api.On("GetCall", "callId").Return(&bandwidth.Call{
		State: "active",
	}, nil)
	w := makeRequest(t, api, db, http.MethodGet, "/recordCallback?callId=callId&eventType=recording&state=complete&recordingId=recordingID", "")
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
	activeCall := &ActiveCall{
		UserID: user.ID,
		CallID: "callID",
	}
	db.Save(activeCall)
	api.On("GetRecording", "recordingID").Return(&bandwidth.Recording{
		Media: "url",
	}, nil)
	api.On("GetCall", "callID").Return(&bandwidth.Call{
		State: "completed",
	}, nil)
	w := makeRequest(t, api, db, http.MethodGet, "/recordCallback?callId=callID&eventType=recording&state=complete&recordingId=recordingID", "")
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
	activeCall := &ActiveCall{
		UserID: user.ID,
		CallID: "callID",
	}
	db.Save(activeCall)
	api.On("GetRecording", "recordingID").Return(&bandwidth.Recording{
		Media: "url",
	}, nil)
	api.On("GetCall", "callID").Return(&bandwidth.Call{}, errors.New("Error"))
	w := makeRequest(t, api, db, http.MethodGet, "/recordCallback?callId=callID&eventType=recording&state=complete&recordingId=recordingID", "")
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
	activeCall := &ActiveCall{
		UserID: user.ID,
		CallID: "callID",
	}
	db.Save(activeCall)
	api.On("GetRecording", "recordingID").Return(&bandwidth.Recording{}, errors.New("Error"))
	w := makeRequest(t, api, db, http.MethodGet, "/recordCallback?callId=callID&eventType=recording&state=complete&recordingId=recordingID", "")
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
	w := makeRequest(t, api, db, http.MethodGet, "/voiceMessages", token, nil, &result)
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
	w := makeRequest(t, api, db, http.MethodGet, fmt.Sprintf("/voiceMessages/%v/media", message.ID), token)
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
	w := makeRequest(t, api, db, http.MethodGet, fmt.Sprintf("/voiceMessages/%v/media", message.ID), token)
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
	w := makeRequest(t, api, db, http.MethodDelete, fmt.Sprintf("/voiceMessages/%v", message.ID), token)
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
	w := makeRequest(t, api, db, http.MethodDelete, "/voiceMessages/unknown", token)
	assert.Equal(t, http.StatusBadGateway, w.Code)
}

func TestStreamNewVoceMailMessage(t *testing.T) {
	msg := &VoiceMailMessage{
		StartTime: parseTime("2016-05-31T10:00:00Z"),
		EndTime:   parseTime("2016-05-31T10:01:00Z"),
		From:      "+1234567980",
	}
	msg.ID = 1
	context := &fakeSSEEmiter{}
	context.On("SSEvent", "message", msg.ToJSONObject()).Return()
	channel := make(chan interface{})
	defer close(channel)
	go func() {
		time.Sleep(10 * time.Millisecond)
		channel <- msg
	}()
	assert.True(t, streamNewVoceMailMessage(context, channel))
}

func TestRouteGetVoiceMessageStream(t *testing.T) {
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	token := createUserAndLogin(t, db)
	user := &User{}
	db.First(user, "user_name = ?", "user1")
	go func() {
		makeRequest(t, api, db, http.MethodGet, fmt.Sprintf("/voiceMessagesStream?token=%s", token), "")
	}()
	time.Sleep(100 * time.Millisecond)
}

var newVoiceMailMessage *pubsub.PubSub

func makeRequest(t *testing.T, api catapultAPIInterface, db *gorm.DB, method, path, authToken string, body ...interface{}) *responseRecorder {
	os.Setenv("CATAPULT_USER_ID", "userID")
	os.Setenv("CATAPULT_API_TOKEN", "token")
	os.Setenv("CATAPULT_API_SECRET", "secret")
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("catapultAPI", api)
		c.Next()
	})
	require.NoError(t, getRoutes(router, db, newVoiceMailMessage))
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
	return make(chan bool)
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
	w := makeRequest(t, nil, db, http.MethodPost, "/login", "", data, &result)
	assert.Equal(t, http.StatusOK, w.Code)
	return result["token"]
}
