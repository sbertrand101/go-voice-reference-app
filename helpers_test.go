package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/bandwidthcom/go-bandwidth"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func createFakeGinContext() *gin.Context {
	request, _ := http.NewRequest("GET", "/test", bytes.NewReader([]byte{}))
	context := &gin.Context{Request: request}
	context.Request.Header.Set("Host", "localhost")
	context.Request.Host = "localhost"
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
	connectionString := os.Getenv("TEST_DATABASE_URL")
	if connectionString == "" {
		// to use with Docker's links
		host := os.Getenv("DB_PORT_5432_TCP_ADDR")
		port := os.Getenv("DB_PORT_5432_TCP_PORT")
		if host != "" && port != "" {
			connectionString = fmt.Sprintf("postgresql://postgres@%s:%s/postgres?sslmode=disable", host, port)
		}
	}
	if connectionString == "" {
		connectionString = "postgresql://postgres@localhost/golang_voice_reference_app_test?sslmode=disable"
	}
	db, err := gorm.Open("postgres", connectionString)
	require.NoError(t, err)
	db.DropTableIfExists(&User{})
	require.NoError(t, AutoMigrate(db).Error)
	return db
}

type fakeCatapultAPI struct {
	mock.Mock
}

func (m *fakeCatapultAPI) GetApplicationID() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *fakeCatapultAPI) GetDomain() (string, string, error) {
	args := m.Called()
	return args.String(0), args.String(1), args.Error(2)
}

func (m *fakeCatapultAPI) CreatePhoneNumber(areaCode string) (string, error) {
	args := m.Called(areaCode)
	return args.String(0), args.Error(1)
}

func (m *fakeCatapultAPI) CreateSIPAccount() (*sipAccount, error) {
	args := m.Called()
	return args.Get(0).(*sipAccount), args.Error(1)
}

func (m *fakeCatapultAPI) CreateSIPAuthToken(endpointID string) (*bandwidth.DomainEndpointToken, error) {
	args := m.Called(endpointID)
	return args.Get(0).(*bandwidth.DomainEndpointToken), args.Error(1)
}

func (m *fakeCatapultAPI) UpdateCall(callID string, data *bandwidth.UpdateCallData) (string, error) {
	args := m.Called(callID, data)
	return args.String(0), args.Error(1)
}

func (m *fakeCatapultAPI) GetCall(callID string) (*bandwidth.Call, error) {
	args := m.Called(callID)
	return args.Get(0).(*bandwidth.Call), args.Error(1)
}

func (m *fakeCatapultAPI) PlayAudioToCall(callID string, url string) error {
	args := m.Called(callID, url)
	return args.Error(0)
}

func (m *fakeCatapultAPI) SpeakSentenceToCall(callID string, text string) error {
	args := m.Called(callID, text)
	return args.Error(0)
}

func (m *fakeCatapultAPI) CreateGather(callID string, data *bandwidth.CreateGatherData) (string, error) {
	args := m.Called(callID, data)
	return args.String(0), args.Error(1)
}

func (m *fakeCatapultAPI) GetRecording(recordingID string) (*bandwidth.Recording, error) {
	args := m.Called(recordingID)
	return args.Get(0).(*bandwidth.Recording), args.Error(1)
}

func (m *fakeCatapultAPI) CreateCall(data *bandwidth.CreateCallData) (string, error) {
	args := m.Called(data)
	return args.String(0), args.Error(1)
}

func (m *fakeCatapultAPI) DownloadMediaFile(name string) (io.ReadCloser, string, error) {
	args := m.Called(name)
	return args.Get(0).(io.ReadCloser), args.String(1), args.Error(2)
}

type fakeTimerAPI struct {
	mock.Mock
}

func (m *fakeTimerAPI) Sleep(d time.Duration) {
	m.Called(d)
}

type fakeSSEEmiter struct {
	mock.Mock
}

func (m *fakeSSEEmiter) SSEvent(name string, message interface{}) {
	m.Called(name, message)
}
