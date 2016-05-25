package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

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

func makeRequest(t *testing.T, api catapultAPIInterface, timerAPI timerInterface, db *gorm.DB, method, path, authToken string, body ...interface{}) *httptest.ResponseRecorder {
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
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if strings.Contains(w.Header().Get("Content-Type"), "application/json") && len(body) > 1 && body[1] != nil {
		json.Unmarshal(w.Body.Bytes(), &body[1])
	}
	return w
}

func createUserAndLogin(t *testing.T, db *gorm.DB) string {
	data := gin.H{
		"userName": "user1",
		"password": "123456",
	}
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
