package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/bandwidthcom/go-bandwidth"
	"github.com/gin-gonic/gin"
)

const applicationName = "GolangVoiceReferenceApp"

type catapultAPI struct {
	client  *bandwidth.Client
	context *gin.Context
}

var applicationID string
var domainID string
var domainName string

type catapultAPIInterface interface {
	GetApplicationID() (string, error)
	GetDomain() (string, string, error)
	CreatePhoneNumber(areaCode string) (string, error)
	CreateSIPAccount() (*sipAccount, error)
	CreateSIPAuthToken(endpointID string) (*bandwidth.DomainEndpointToken, error)
}

func newCatapultAPI(context *gin.Context) (*catapultAPI, error) {
	client, err := bandwidth.New(os.Getenv("CATAPULT_USER_ID"), os.Getenv("CATAPULT_API_TOKEN"), os.Getenv("CATAPULT_API_SECRET"))
	return &catapultAPI{client: client, context: context}, err
}

func (api *catapultAPI) GetApplicationID() (string, error) {
	if applicationID != "" {
		return applicationID, nil
	}
	applications, err := api.client.GetApplications(&bandwidth.GetApplicationsQuery{Size: 1000})
	if err != nil {
		return "", err
	}
	var application *bandwidth.Application
	for _, application = range applications {
		if application.Name == applicationName {
			break
		}
	}
	if application != nil {
		applicationID = application.ID
		return applicationID, nil
	}
	applicationID, err = api.client.CreateApplication(&bandwidth.ApplicationData{
		Name:               applicationName,
		AutoAnswer:         true,
		CallbackHTTPMethod: "POST",
		IncomingCallURL:    fmt.Sprintf("http://%s/callCallback", api.context.Request.Header.Get("Host")),
	})
	return applicationID, err
}

func (api *catapultAPI) GetDomain() (string, string, error) {
	if domainID != "" {
		return domainID, domainName, nil
	}
	domains, err := api.client.GetDomains()
	if err != nil {
		return "", "", err
	}
	var domain *bandwidth.Domain
	const description = applicationName + "'s domain"
	for _, domain = range domains {
		if domain.Description == description {
			break
		}
	}
	if domain != nil {
		domainID = domain.ID
		domainName = domain.Name
		return domainID, domainName, nil
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