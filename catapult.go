package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/bandwidthcom/go-bandwidth"
	"github.com/gin-gonic/gin"
)

const applicationName = "GolangVoiceReferenceApp"

func getCatapultAPI() (*bandwidth.Client, error) {
	return bandwidth.New(os.Getenv("CATAPULT_USER_ID"), os.Getenv("CATAPULT_API_TOKEN"), os.Getenv("CATAPULT_API_SECRET"))
}

var applicationID string
var domainID string
var domainName string

func getApplicationID(context *gin.Context, api *bandwidth.Client) (string, error) {
	if applicationID != "" {
		return applicationID, nil
	}
	applications, err := api.GetApplications(&bandwidth.GetApplicationsQuery{Size: 1000})
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
	applicationID, err = api.CreateApplication(&bandwidth.ApplicationData{
		Name:               applicationName,
		AutoAnswer:         true,
		CallbackHTTPMethod: "POST",
		IncomingCallURL:    fmt.Sprintf("http://%s/callCallback", context.Request.Header.Get("Host")),
	})
	return applicationID, err
}

func getDomain(api *bandwidth.Client) (string, string, error) {
	if domainID != "" {
		return domainID, domainName, nil
	}
	domains, err := api.GetDomains()
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
	domainID, err = api.CreateDomain(&bandwidth.CreateDomainData{
		Name:        domainName,
		Description: description,
	})
	return domainID, domainName, err
}

func createPhoneNumber(context *gin.Context, api *bandwidth.Client, areaCode string) (string, error) {
	applicationID, err := getApplicationID(context, api)
	if err != nil {
		return "", err
	}
	numbers, err := api.GetAndOrderAvailableNumbers(bandwidth.AvailableNumberTypeLocal,
		&bandwidth.GetAvailableNumberQuery{AreaCode: areaCode, Quantity: 1})
	if err != nil {
		return "", err
	}
	err = api.UpdatePhoneNumber(numbers[0].ID, &bandwidth.UpdatePhoneNumberData{ApplicationID: applicationID})
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

type phoneData struct {
	SipAccount  *sipAccount
	PhoneNumber string
}

func createSIPAccount(context *gin.Context, api *bandwidth.Client) (*sipAccount, error) {
	applicationID, err := getApplicationID(context, api)
	if err != nil {
		return nil, err
	}
	domainID, domainName, err := getDomain(api)
	if err != nil {
		return nil, err
	}
	sipUserName := randomString(16)
	sipPassword := randomString(10)
	id, err := api.CreateDomainEndpoint(domainID, &bandwidth.DomainEndpointData{
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

func createPhoneData(context *gin.Context, api *bandwidth.Client, areaCode string) (*phoneData, error) {
	phoneNumber, err := createPhoneNumber(context, api, areaCode)
	if err != nil {
		return nil, err
	}
	account, err := createSIPAccount(context, api)
	if err != nil {
		return nil, err
	}
	return &phoneData{
		PhoneNumber: phoneNumber,
		SipAccount:  account,
	}, nil
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
