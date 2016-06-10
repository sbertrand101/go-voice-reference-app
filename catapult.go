package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bandwidthcom/go-bandwidth"
	"github.com/gin-gonic/gin"
)

const applicationName = "GolangVoiceReferenceApp"

type catapultAPI struct {
	client  *bandwidth.Client
	context *gin.Context
}

var applicationIDs map[string]string
var domainID string
var domainName string

type catapultAPIInterface interface {
	GetApplicationID() (string, error)
	GetDomain() (string, string, error)
	CreatePhoneNumber(areaCode string) (string, error)
	CreateSIPAccount() (*sipAccount, error)
	CreateSIPAuthToken(endpointID string) (*bandwidth.DomainEndpointToken, error)
	UpdateCall(callID string, data *bandwidth.UpdateCallData) (string, error)
	GetCall(callID string) (*bandwidth.Call, error)
	PlayAudioToCall(callID string, url string, loop bool, tag string) error
	SpeakSentenceToCall(callID string, text string, tag string) error
	CreateGather(callID string, data *bandwidth.CreateGatherData) (string, error)
	GetRecording(recordingID string) (*bandwidth.Recording, error)
	CreateCall(data *bandwidth.CreateCallData) (string, error)
	DownloadMediaFile(name string) (io.ReadCloser, string, error)
	GetCallRecordings(callID string) ([]*bandwidth.Recording, error)
	CreateBridge(data *bandwidth.BridgeData) (string, error)
	Hangup(callID string) error
}

func newCatapultAPI(context *gin.Context) (*catapultAPI, error) {
	client, err := bandwidth.New(os.Getenv("CATAPULT_USER_ID"), os.Getenv("CATAPULT_API_TOKEN"), os.Getenv("CATAPULT_API_SECRET"))
	return &catapultAPI{client: client, context: context}, err
}

func (api *catapultAPI) GetApplicationID() (string, error) {
	host := api.context.Request.Host
	appName := fmt.Sprintf("%s on %s", applicationName, host)
	if applicationIDs == nil {
		applicationIDs = make(map[string]string, 0)
	}
	applicationID := applicationIDs[host]
	if applicationID != "" {
		return applicationID, nil
	}
	applications, err := api.client.GetApplications(&bandwidth.GetApplicationsQuery{Size: 1000})
	if err != nil {
		return "", err
	}
	var application *bandwidth.Application
	for _, application = range applications {
		if application.Name == appName {
			applicationID = application.ID
			applicationIDs[host] = applicationID
			return applicationID, nil
		}
	}
	applicationID, err = api.client.CreateApplication(&bandwidth.ApplicationData{
		Name:               appName,
		AutoAnswer:         true,
		CallbackHTTPMethod: "GET",
		IncomingCallURL:    fmt.Sprintf("http://%s/callCallback", host),
	})
	if applicationID != "" {
		applicationIDs[host] = applicationID
	}
	return applicationID, err
}

func (api *catapultAPI) GetDomain() (string, string, error) {
	if domainID != "" {
		return domainID, domainName, nil
	}
	domains, err := api.client.GetDomains(&bandwidth.GetDomainsQuery{Size: 100})
	if err != nil {
		return "", "", err
	}
	var domain *bandwidth.Domain
	const description = applicationName + "'s domain"
	for _, domain = range domains {
		if domain.Description == description {
			domainID = domain.ID
			domainName = domain.Name
			return domainID, domainName, nil
		}
	}
	domainName = randomString(15)
	domainID, err = api.client.CreateDomain(&bandwidth.CreateDomainData{
		Name:        domainName,
		Description: description,
	})
	return domainID, domainName, err
}

func (api *catapultAPI) CreatePhoneNumber(areaCode string) (string, error) {
	applicationID, err := api.GetApplicationID()
	if err != nil {
		return "", err
	}
	numbers, err := api.client.GetAndOrderAvailableNumbers(bandwidth.AvailableNumberTypeLocal,
		&bandwidth.GetAvailableNumberQuery{AreaCode: areaCode, Quantity: 1})
	if err != nil {
		return "", err
	}
	err = api.client.UpdatePhoneNumber(numbers[0].ID, &bandwidth.UpdatePhoneNumberData{ApplicationID: applicationID})
	if err != nil {
		return "", err
	}
	return numbers[0].Number, nil
}

type sipAccount struct {
	EndpointID string
	URI        string
	Password   string
}

func (api *catapultAPI) CreateSIPAccount() (*sipAccount, error) {
	applicationID, err := api.GetApplicationID()
	if err != nil {
		return nil, err
	}
	domainID, domainName, err := api.GetDomain()
	if err != nil {
		return nil, err
	}
	sipUserName := randomString(16)
	sipPassword := randomString(10)
	id, err := api.client.CreateDomainEndpoint(domainID, &bandwidth.DomainEndpointData{
		ApplicationID: applicationID,
		DomainID:      domainID,
		Name:          sipUserName,
		Description:   applicationName + "'s SIP Account",
		Credentials:   &bandwidth.DomainEndpointCredentials{Password: sipPassword},
	})
	if err != nil {
		return nil, err
	}
	sipURI := fmt.Sprintf("sip:%s@%s.bwapp.bwsip.io", sipUserName, domainName)
	return &sipAccount{id, sipURI, sipPassword}, nil
}

func (api *catapultAPI) CreateSIPAuthToken(endpointID string) (*bandwidth.DomainEndpointToken, error) {
	domainID, _, err := api.GetDomain()
	if err != nil {
		return nil, err
	}
	return api.client.CreateDomainEndpointToken(domainID, endpointID)
}

func (api *catapultAPI) UpdateCall(callID string, data *bandwidth.UpdateCallData) (string, error) {
	return api.client.UpdateCall(callID, data)
}

func (api *catapultAPI) GetCall(callID string) (*bandwidth.Call, error) {
	return api.client.GetCall(callID)
}

func (api *catapultAPI) PlayAudioToCall(callID string, url string, loop bool, tag string) error {
	return api.client.PlayAudioToCall(callID, &bandwidth.PlayAudioData{
		FileURL:     url,
		LoopEnabled: loop,
		Tag:         tag,
	})
}

func (api *catapultAPI) SpeakSentenceToCall(callID string, text string, tag string) error {
	return api.client.PlayAudioToCall(callID, &bandwidth.PlayAudioData{
		Gender:   "female",
		Locale:   "en_US",
		Voice:    "julie",
		Sentence: text,
		Tag:      tag,
	})
}

func (api *catapultAPI) CreateGather(callID string, data *bandwidth.CreateGatherData) (string, error) {
	return api.client.CreateGather(callID, data)
}

func (api *catapultAPI) GetRecording(recordingID string) (*bandwidth.Recording, error) {
	return api.client.GetRecording(recordingID)
}

func (api *catapultAPI) GetCallRecordings(callID string) ([]*bandwidth.Recording, error) {
	return api.client.GetCallRecordings(callID)
}

func (api *catapultAPI) CreateCall(data *bandwidth.CreateCallData) (string, error) {
	return api.client.CreateCall(data)
}

func (api *catapultAPI) CreateBridge(data *bandwidth.BridgeData) (string, error) {
	return api.client.CreateBridge(data)
}

func (api *catapultAPI) DownloadMediaFile(name string) (io.ReadCloser, string, error) {
	return api.client.DownloadMediaFile(name)
}

func (api *catapultAPI) Hangup(callID string) error {
	return api.client.HangUpCall(callID)
}

func buildBXML(items ...string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response>
	%s
</Response>`, strings.Join(items, "\n\t"))
}

func hangupBXML() string {
	return "<Hangup/>"
}

func playAudioBXML(url string) string {
	return fmt.Sprintf(`<PlayAudio>%s</PlayAudio>`, url)
}

func transferBXML(transferTo string, transferCallerID string, timeout int, requestURL string, tag string) string {
	attrs := ""
	if timeout > 0 {
		attrs = fmt.Sprintf(`%s callTimeout="%d"`, attrs, timeout)
	}
	if requestURL != "" {
		attrs = fmt.Sprintf(`%s requestUrl="%s"`, attrs, requestURL)
	}
	if tag != "" {
		attrs = fmt.Sprintf(`%s tag="%s"`, attrs, tag)
	}
	return fmt.Sprintf(`<Transfer transferTo="%s" transferCallerId="%s"%s/>`, transferTo, transferCallerID, attrs)
}

func speakSentenceBXML(sentence string) string {
	return fmt.Sprintf(`<SpeakSentence locale="en_US" gender="female" voice="julie">%s</SpeakSentence>`, sentence)
}

func recordBXML(requestURL string, terminatingDigits ...string) string {
	digits := "#"
	if len(terminatingDigits) > 0 {
		digits = terminatingDigits[0]
	}
	return fmt.Sprintf(`<Record requestUrl="%s" terminatingDigits="%s" maxDuration="3600"/>`, requestURL, digits)
}

func gatherBXML(requestURL string, children ...string) string {
	return fmt.Sprintf(`<Gather requestUrl="%s" maxDigits="1" interDigitTimeout="30">%s</Gather>`, requestURL, strings.Join(children, ""))
}

func catapultMiddleware(c *gin.Context) {
	api, err := newCatapultAPI(c)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Set("catapultAPI", api)
	c.Next()
}

var randomString = func(strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano() + rand.Int63())
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		l := len(chars)
		if i == 0 {
			l -= 10 // first symbol should be letter
		}
		result[i] = chars[rand.Intn(l)]
	}
	return string(result)
}
