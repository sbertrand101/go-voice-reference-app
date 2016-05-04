package main

import (
	"net/http"
	"net/http/httptest"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io"
	"bytes"
)


func performRequest(method, path, authToken string, body ...interface{}) *httptest.ResponseRecorder {
	router := gin.Default()
	getRoutes(router, nil)
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
	return w
}
