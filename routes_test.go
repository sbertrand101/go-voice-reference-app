package main

import (
	"net/http"
	"net/http/httptest"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"io"
	"bytes"
	"testing"
	"os"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"errors"
)

func TestRouteRegister(t *testing.T) {
	data := gin.H{
		"userName": "user1",
		"areaCode": "910",
		"password": "123456",
		"repeatPassword": "123456",
	}
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()

	api.On("CreatePhoneNumber", "910").Return("+1234567890", nil)
	api.On("CreateSIPAccount").Return(&sipAccount{
		EndpointID: "endpointId",
		URI: "test@test.net",
		Password: "12345678",
	}, nil)
	w := performRequest(t, api, db, http.MethodPost, "/register", "", data)
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
		"userName": "user1",
		"areaCode": "910",
		"password": "123456",
		"repeatPassword": "000",
	}
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()

	w := performRequest(t, api, db, http.MethodPost, "/register", "", data)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	api.AssertNotCalled(t, "CreatePhoneNumber")
	api.AssertNotCalled(t, "CreateSIPAccount")
	user := &User{}
	assert.True(t, db.First(user, "user_name = ?", "user1").RecordNotFound())
}

func TestRouteRegisterFailWithShortPassword(t *testing.T) {
	data := gin.H{
		"userName": "user1",
		"areaCode": "910",
		"password": "123",
		"repeatPassword": "123",
	}
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()

	w := performRequest(t, api, db, http.MethodPost, "/register", "", data)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	api.AssertNotCalled(t, "CreatePhoneNumber")
	api.AssertNotCalled(t, "CreateSIPAccount")
	user := &User{}
	assert.True(t, db.First(user, "user_name = ?", "user1").RecordNotFound())
}

func TestRouteRegisterFailWithMissingFields(t *testing.T) {
	data := gin.H{
		"password": "123",
		"repeatPassword": "123",
	}
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()

	w := performRequest(t, api, db, http.MethodPost, "/register", "", data)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	api.AssertNotCalled(t, "CreatePhoneNumber")
	api.AssertNotCalled(t, "CreateSIPAccount")
	user := &User{}
	assert.True(t, db.First(user, "user_name = ?", "user1").RecordNotFound())
}


func TestRouteRegisterFailWithSameUser(t *testing.T) {
	data := gin.H{
		"userName": "user1",
		"areaCode": "910",
		"password": "123456",
		"repeatPassword": "123456",
	}
	api := &fakeCatapultAPI{}
	db := openDBConnection(t)
	defer db.Close()
	db.Create(&User{UserName: "user1", AreaCode: "910"})
	w := performRequest(t, api, db, http.MethodPost, "/register", "", data)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	api.AssertNotCalled(t, "CreatePhoneNumber")
	api.AssertNotCalled(t, "CreateSIPAccount")
}

func TestRouteRegisterFailWithFailedCreatePhoneNumber(t *testing.T) {
	data := gin.H{
		"userName": "user1",
		"areaCode": "910",
		"password": "123456",
		"repeatPassword": "123456",
	}
	api := &fakeCatapultAPI{}
	api.On("CreatePhoneNumber", "910").Return("", errors.New("Error"))
	db := openDBConnection(t)
	defer db.Close()
	w := performRequest(t, api, db, http.MethodPost, "/register", "", data)
	assert.Equal(t, http.StatusBadGateway, w.Code)
	api.AssertNotCalled(t, "CreateSIPAccount")
	user := &User{}
	assert.True(t, db.First(user, "user_name = ?", "user1").RecordNotFound())
}

func TestRouteRegisterFailWithFailedCreateSIPAccount(t *testing.T) {
	data := gin.H{
		"userName": "user1",
		"areaCode": "910",
		"password": "123456",
		"repeatPassword": "123456",
	}
	api := &fakeCatapultAPI{}
	api.On("CreatePhoneNumber", "910").Return("+1234567890", nil)
	api.On("CreateSIPAccount").Return((*sipAccount)(nil), errors.New("Error"))
	db := openDBConnection(t)
	defer db.Close()
	w := performRequest(t, api, db, http.MethodPost, "/register", "", data)
	assert.Equal(t, http.StatusBadGateway, w.Code)
	user := &User{}
	assert.True(t, db.First(user, "user_name = ?", "user1").RecordNotFound())
}


func performRequest(t *testing.T, api catapultAPIInterface, db *gorm.DB, method, path, authToken string, body ...interface{}) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	os.Setenv("CATAPULT_USER_ID", "userID")
	os.Setenv("CATAPULT_API_TOKEN", "token")
	os.Setenv("CATAPULT_API_SECRET", "secret")
	router := gin.New()
	router.Use(func (c *gin.Context) {
		c.Set("catapultAPI", api)
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
		req.Header.Set("Authorization", "Bearer " + authToken)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Header().Get("Content-Type") == "application/json" && len(body) > 1 && body[1] != nil {
		json.Unmarshal(w.Body.Bytes(), &body[1])
	}
	return w
}
