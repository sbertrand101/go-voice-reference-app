package main

import (
	"net/http"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNewCatapultApi(t *testing.T) {
	os.Setenv("CATAPULT_USER_ID", "UserID")
	os.Setenv("CATAPULT_API_TOKEN", "Token")
	os.Setenv("CATAPULT_API_SECRET", "Secret")
	api, _ := newCatapultAPI(nil)
	assert.Equal(t, "UserID", api.client.UserID)
	assert.Equal(t, "Token", api.client.APIToken)
	assert.Equal(t, "Secret", api.client.APISecret)
}

func TestGetCatapultApiFail(t *testing.T) {
	os.Unsetenv("CATAPULT_USER_ID")
	os.Unsetenv("CATAPULT_API_TOKEN")
	os.Unsetenv("CATAPULT_API_SECRET")
	_, err := newCatapultAPI(nil)
	assert.Error(t, err)
}

func TestGetApplicationIDWithNewApplication(t *testing.T) {
	applicationID = ""
	server, api := startMockCatapultServer(t, []RequestHandler{
		RequestHandler{
			PathAndQuery:  "/v1/users/userID/applications?size=1000",
			Method:        http.MethodGet,
			ContentToSend: `[]`,
		},
		RequestHandler{
			PathAndQuery:     "/v1/users/userID/applications",
			Method:           http.MethodPost,
			EstimatedContent: `{"name":"GolangVoiceReferenceApp","incomingCallUrl":"http://localhost/callCallback","callbackHttpMethod":"POST","autoAnswer":true}`,
			HeadersToSend:    map[string]string{"Location": "/v1/users/userID/applications/123"},
		},
	})
	defer server.Close()
	id, _ := api.GetApplicationID()
	assert.Equal(t, "123", id)
}

func TestGetApplicationIDWithExistingApplication(t *testing.T) {
	applicationID = ""
	server, api := startMockCatapultServer(t, []RequestHandler{
		RequestHandler{
			PathAndQuery:  "/v1/users/userID/applications?size=1000",
			Method:        http.MethodGet,
			ContentToSend: `[{"name": "GolangVoiceReferenceApp", "id": "0123"}]`,
		},
	})
	defer server.Close()
	id, _ := api.GetApplicationID()
	assert.Equal(t, "0123", id)
}

func TestGetApplicationIDRepeating(t *testing.T) {
	applicationID = ""
	server, api := startMockCatapultServer(t, []RequestHandler{
		RequestHandler{
			PathAndQuery:  "/v1/users/userID/applications?size=1000",
			Method:        http.MethodGet,
			ContentToSend: `[{"name": "GolangVoiceReferenceApp", "id": "1234"}]`,
		},
	})
	id, _ := api.GetApplicationID()
	server.Close()
	assert.Equal(t, "1234", id)
	id, _ = api.GetApplicationID()
	assert.Equal(t, "1234", id)
	id, _ = api.GetApplicationID()
	assert.Equal(t, "1234", id)
}

func TestGetApplicationIDFail(t *testing.T) {
	applicationID = ""
	server, api := startMockCatapultServer(t, []RequestHandler{
		RequestHandler{
			PathAndQuery:     "/v1/users/userID/applications?size=1000",
			Method:           http.MethodGet,
			StatusCodeToSend: http.StatusBadRequest,
		},
	})
	defer server.Close()
	_, err := api.GetApplicationID()
	assert.Error(t, err)
}

func TestGetDomainWithNewDomain(t *testing.T) {
	domainID = ""
	server, api := startMockCatapultServer(t, []RequestHandler{
		RequestHandler{
			PathAndQuery:  "/v1/users/userID/domains",
			Method:        http.MethodGet,
			ContentToSend: `[]`,
		},
		RequestHandler{
			PathAndQuery:     "/v1/users/userID/domains",
			Method:           http.MethodPost,
			EstimatedContent: `{"name":"random","description":"GolangVoiceReferenceApp's domain"}`,
			HeadersToSend:    map[string]string{"Location": "/v1/users/userID/domains/123"},
		},
	})
	useMockRandomString()
	defer restoreRandomString()
	defer server.Close()
	id, name, _ := api.GetDomain()
	assert.Equal(t, "123", id)
	assert.Equal(t, "random", name)
}

func TestGetDomainWithExistingDomain(t *testing.T) {
	domainID = ""
	server, api := startMockCatapultServer(t, []RequestHandler{
		RequestHandler{
			PathAndQuery:  "/v1/users/userID/domains",
			Method:        http.MethodGet,
			ContentToSend: `[{"name": "domain", "id": "0123", "description": "GolangVoiceReferenceApp's domain"}]`,
		},
	})
	defer server.Close()
	id, name, _ := api.GetDomain()
	assert.Equal(t, "0123", id)
	assert.Equal(t, "domain", name)
}

func TestGetDomainRepeating(t *testing.T) {
	domainID = ""
	server, api := startMockCatapultServer(t, []RequestHandler{
		RequestHandler{
			PathAndQuery:  "/v1/users/userID/domains",
			Method:        http.MethodGet,
			ContentToSend: `[{"name": "domain1", "id": "1234", "description": "GolangVoiceReferenceApp's domain"}]`,
		},
	})
	id, _, _ := api.GetDomain()
	server.Close()
	assert.Equal(t, "1234", id)
	id, _, _ = api.GetDomain()
	assert.Equal(t, "1234", id)
	id, name, _ := api.GetDomain()
	assert.Equal(t, "1234", id)
	assert.Equal(t, "domain1", name)
}

func TestGetDomainFail(t *testing.T) {
	domainID = ""
	server, api := startMockCatapultServer(t, []RequestHandler{
		RequestHandler{
			PathAndQuery:     "/v1/users/userID/domains",
			Method:           http.MethodGet,
			StatusCodeToSend: http.StatusBadRequest,
		},
	})
	defer server.Close()
	_, _, err := api.GetDomain()
	assert.Error(t, err)
}

func TestCreatePhoneNumber(t *testing.T) {
	applicationID = "123"
	server, api := startMockCatapultServer(t, []RequestHandler{
		RequestHandler{
			PathAndQuery:  "/v1/availableNumbers/local?areaCode=910&quantity=1",
			Method:        http.MethodPost,
			ContentToSend: `[{"number": "+1234567890", "location": "/v1/users/userID/phoneNumbers/1234"}]`,
		},
		RequestHandler{
			PathAndQuery:     "/v1/users/userID/phoneNumbers/1234",
			Method:           http.MethodPost,
			EstimatedContent: `{"applicationId":"123"}`,
		},
	})
	defer server.Close()
	phoneNumber, _ := api.CreatePhoneNumber("910")
	assert.Equal(t, "+1234567890", phoneNumber)
}

func TestCreatePhoneNumberFail(t *testing.T) {
	applicationID = ""
	server, api := startMockCatapultServer(t, []RequestHandler{
		RequestHandler{
			PathAndQuery:     "/v1/users/userID/applications?size=1000",
			Method:           http.MethodGet,
			StatusCodeToSend: http.StatusBadRequest,
		},
	})
	defer server.Close()
	_, err := api.CreatePhoneNumber("910")
	assert.Error(t, err)
}

func TestCreatePhoneNumberFail2(t *testing.T) {
	applicationID = "123"
	server, api := startMockCatapultServer(t, []RequestHandler{
		RequestHandler{
			PathAndQuery:     "/v1/availableNumbers/local?areaCode=910&quantity=1",
			Method:           http.MethodPost,
			StatusCodeToSend: http.StatusBadRequest,
		},
	})
	defer server.Close()
	_, err := api.CreatePhoneNumber("910")
	assert.Error(t, err)
}

func TestCreatePhoneNumberFail3(t *testing.T) {
	applicationID = "123"
	server, api := startMockCatapultServer(t, []RequestHandler{
		RequestHandler{
			PathAndQuery:  "/v1/availableNumbers/local?areaCode=910&quantity=1",
			Method:        http.MethodPost,
			ContentToSend: `[{"number": "+1234567890", "location": "/v1/users/userID/phoneNumbers/1234"}]`,
		},
		RequestHandler{
			PathAndQuery:     "/v1/users/userID/phoneNumbers/1234",
			Method:           http.MethodPost,
			StatusCodeToSend: http.StatusBadRequest,
		},
	})
	defer server.Close()
	_, err := api.CreatePhoneNumber("910")
	assert.Error(t, err)
}

func TestCreateSIPAccount(t *testing.T) {
	applicationID = "123"
	domainID = "456"
	domainName = "domain1"
	server, api := startMockCatapultServer(t, []RequestHandler{
		RequestHandler{
			PathAndQuery:     "/v1/users/userID/domains/456/endpoints",
			Method:           http.MethodPost,
			EstimatedContent: `{"name":"random","description":"GolangVoiceReferenceApp's SIP Account","domainId":"456","applicationId":"123","credentials":{"password":"random"}}`,
			HeadersToSend:    map[string]string{"Location": "/v1/users/userID/domains/456/endpoints/567"},
		},
	})
	useMockRandomString()
	defer server.Close()
	defer restoreRandomString()
	account, _ := api.CreateSIPAccount()
	assert.EqualValues(t, &sipAccount{
		EndpointID: "567",
		URI:        "sip:random@domain1.bwapp.bwsip.io",
		Password:   "random",
	}, account)
}

func TestCreateSIPAccountFail(t *testing.T) {
	applicationID = "123"
	domainID = "456"
	domainName = "domain2"
	server, api := startMockCatapultServer(t, []RequestHandler{
		RequestHandler{
			PathAndQuery:     "/v1/users/userID/domains/456/endpoints",
			Method:           http.MethodPost,
			StatusCodeToSend: http.StatusBadRequest,
		},
	})
	useMockRandomString()
	defer server.Close()
	defer restoreRandomString()
	_, err := api.CreateSIPAccount()
	assert.Error(t, err)
}

func TestCreateSIPAccountFail2(t *testing.T) {
	applicationID = "123"
	domainID = ""
	domainName = ""
	server, api := startMockCatapultServer(t, []RequestHandler{
		RequestHandler{
			PathAndQuery:     "/v1/users/userID/domains",
			Method:           http.MethodGet,
			StatusCodeToSend: http.StatusBadRequest,
		},
	})
	defer server.Close()
	_, err := api.CreateSIPAccount()
	assert.Error(t, err)
}

func TestCreateSIPAccountFail3(t *testing.T) {
	applicationID = ""
	domainID = ""
	domainName = ""
	server, api := startMockCatapultServer(t, []RequestHandler{
		RequestHandler{
			PathAndQuery:     "/v1/users/userID/applications",
			Method:           http.MethodGet,
			StatusCodeToSend: http.StatusBadRequest,
		},
	})
	defer server.Close()
	_, err := api.CreateSIPAccount()
	assert.Error(t, err)
}

func TestCreateSIPAuthToken(t *testing.T) {
	domainID = "123"
	server, api := startMockCatapultServer(t, []RequestHandler{
		RequestHandler{
			PathAndQuery:  "/v1/users/userID/domains/123/endpoints/456/tokens",
			Method:        http.MethodPost,
			ContentToSend: `{"token": "token"}`,
		},
	})
	defer server.Close()
	token, _ := api.CreateSIPAuthToken("456")
	assert.Equal(t, "token", token.Token)
}

func TestCreateSIPAuthTokenFail(t *testing.T) {
	domainID = "123"
	server, api := startMockCatapultServer(t, []RequestHandler{
		RequestHandler{
			PathAndQuery:     "/v1/users/userID/domains/123/endpoints/456/tokens",
			Method:           http.MethodPost,
			StatusCodeToSend: http.StatusBadRequest,
		},
	})
	defer server.Close()
	_, err := api.CreateSIPAuthToken("456")
	assert.Error(t, err)
}

func TestCreateSIPAuthTokenFail2(t *testing.T) {
	domainID = ""
	server, api := startMockCatapultServer(t, []RequestHandler{
		RequestHandler{
			PathAndQuery:     "/v1/users/userID/domains",
			Method:           http.MethodGet,
			StatusCodeToSend: http.StatusBadRequest,
		},
	})
	defer server.Close()
	_, err := api.CreateSIPAuthToken("456")
	assert.Error(t, err)
}

func TestCatapultMiddleware(t *testing.T) {
	os.Setenv("CATAPULT_USER_ID", "UserID")
	os.Setenv("CATAPULT_API_TOKEN", "Token")
	os.Setenv("CATAPULT_API_SECRET", "Secret")
	context := createFakeGinContext()
	catapultMiddleware(context)
	instance := context.MustGet("catapultAPI")
	assert.NotNil(t, instance)
	assert.NotNil(t, instance.(catapultAPIInterface))
}

func TestCatapultMiddlewareFail(t *testing.T) {
	os.Unsetenv("CATAPULT_USER_ID")
	os.Unsetenv("CATAPULT_API_TOKEN")
	os.Unsetenv("CATAPULT_API_SECRET")
	context := createFakeGinContext()
	gin.SetMode(gin.TestMode)
	defer func() {
		_, ok := context.Get("catapultAPI")
		assert.False(t, ok)
		r := recover()
		assert.NotNil(t, r)
	}()
	catapultMiddleware(context)
}

func TestRandomString(t *testing.T) {
	assert.Equal(t, 10, len(randomString(10)))
	assert.Equal(t, 16, len(randomString(16)))
	assert.NotEqual(t, randomString(32), randomString(32))
}

var originalRandomString = randomString

func useMockRandomString() {
	randomString = func(length int) string {
		return "random"
	}
}

func restoreRandomString() {
	randomString = originalRandomString
}
