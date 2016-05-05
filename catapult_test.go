package main

import (
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCatapultApi(t *testing.T) {
	os.Setenv("CATAPULT_USER_ID", "UserID")
	os.Setenv("CATAPULT_API_TOKEN", "Token")
	os.Setenv("CATAPULT_API_SECRET", "Secret")
	api, _ := getCatapultAPI()
	assert.Equal(t, "UserID", api.UserID)
	assert.Equal(t, "Token", api.APIToken)
	assert.Equal(t, "Secret", api.APISecret)
	os.Unsetenv("CATAPULT_USER_ID")
	os.Unsetenv("CATAPULT_API_TOKEN")
	os.Unsetenv("CATAPULT_API_SECRET")
}

func TestGetCatapultApiFail(t *testing.T) {
	os.Unsetenv("CATAPULT_USER_ID")
	os.Unsetenv("CATAPULT_API_TOKEN")
	os.Unsetenv("CATAPULT_API_SECRET")
	_, err := getCatapultAPI()
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
	id, _ := getApplicationID(createFakeGinContext(), api)
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
	id, _ := getApplicationID(createFakeGinContext(), api)
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
	id, _ := getApplicationID(createFakeGinContext(), api)
	server.Close()
	assert.Equal(t, "1234", id)
	id, _ = getApplicationID(createFakeGinContext(), api)
	assert.Equal(t, "1234", id)
	id, _ = getApplicationID(createFakeGinContext(), api)
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
	_, err := getApplicationID(createFakeGinContext(), api)
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
	id, name, _ := getDomain(api)
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
	id, name, _ := getDomain(api)
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
	id, _, _ := getDomain(api)
	server.Close()
	assert.Equal(t, "1234", id)
	id, _, _ = getDomain(api)
	assert.Equal(t, "1234", id)
	id, name, _ := getDomain(api)
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
	_, _,  err := getDomain(api)
	assert.Error(t, err)
}

var originalRandomString = randomString

func useMockRandomString(){
	randomString = func(length int) string {
		return "random"
	}
}

func restoreRandomString(){
	randomString = originalRandomString
}
