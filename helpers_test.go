package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"github.com/bandwidthcom/go-bandwidth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"os"
)


func createFakeResponse(body string, statusCode int) *http.Response {
	return &http.Response{StatusCode: statusCode,
		Body: ioutil.NopCloser(bytes.NewReader([]byte(body))),
	}
}

type RequestHandler struct {
	PathAndQuery string
	Method       string

	EstimatedContent string
	EstimatedHeaders map[string]string

	HeadersToSend    map[string]string
	ContentToSend    string
	StatusCodeToSend int
}

func startMockCatapultServer(t *testing.T, handlers []RequestHandler) (*httptest.Server, *catapultAPI) {
	client, _ := bandwidth.New("userID", "token", "secret")
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, handler := range handlers {
			if handler.Method == "" {
				handler.Method = http.MethodGet
			}
			if handler.StatusCodeToSend == 0 {
				handler.StatusCodeToSend = http.StatusOK
			}
			if handler.Method == r.Method && handler.PathAndQuery == r.URL.String() {
				if handler.EstimatedContent != "" {
					assert.Equal(t, readText(t, r.Body), handler.EstimatedContent)
				}
				if handler.EstimatedHeaders != nil {
					for key, value := range handler.EstimatedHeaders {
						assert.Equal(t, r.Header.Get(key), value)
					}
				}
				header := w.Header()
				if handler.HeadersToSend != nil {
					for key, value := range handler.HeadersToSend {
						header.Set(key, value)
					}
				}
				if handler.ContentToSend != "" && header.Get("Content-Type") == "" {
					header.Set("Content-Type", "application/json")
				}
				w.WriteHeader(handler.StatusCodeToSend)
				if handler.ContentToSend != "" {
					fmt.Fprintln(w, handler.ContentToSend)
				}
				return
			}
		}
		t.Logf("Unhandled request %s %s", r.Method, r.URL.String())
		w.WriteHeader(http.StatusNotFound)
	}))
	client.APIEndPoint = mockServer.URL
	return mockServer, &catapultAPI{client: client, context: createFakeGinContext()}
}

func createFakeGinContext() *gin.Context{
	request, _ := http.NewRequest("GET", "/test", bytes.NewReader([]byte{}))
	context := &gin.Context{Request: request}
	context.Request.Header.Set("Host", "localhost")
	return context
}

func readText(t *testing.T, r io.Reader) string {
	text, err := ioutil.ReadAll(r)
	if err != nil {
		t.Error("Error on reading content")
		return ""
	}
	return string(text)
}

func openDBConnection(t *testing.T) *gorm.DB {
	connectionString := os.Getenv("DATABASE_URI")
	if connectionString == "" {
		connectionString = "postgresql://postgres@localhost/golang_voice_reference_app_test?sslmode=disable"
	}
	db, err := gorm.Open("postgres", connectionString)
	require.NoError(t, err)
	db.DropTableIfExists(&User{})
	require.NoError(t, AutoMigrate(db).Error)
	return db
}
