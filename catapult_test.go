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
