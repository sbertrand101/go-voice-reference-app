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
)

/*func TestRouteRegister(t *testing.T) {
	data := gin.H{
		"userName": "user1",
		"areaCode": "910",
		"password": "123456",
		"repeatPassword": "123456",
	}
	performRequest(t, http.MethodPost, "/register", "", data)
}*/

func performRequest(t *testing.T, method, path, authToken string, body ...interface{}) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	os.Setenv("CATAPULT_USER_ID", "userID")
	os.Setenv("CATAPULT_API_TOKEN", "token")
	os.Setenv("CATAPULT_API_SECRET", "secret")
	router := gin.Default()
	require.NoError(t, getRoutes(router, openDBConnection(t)))
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
	if w.Header().Get("Content-Type") == "application/json" && len(body) > 1 {
		json.Unmarshal(w.Body.Bytes(), &body[1])
	}
	require.False(t, w.Code >= 400)
	return w
}
