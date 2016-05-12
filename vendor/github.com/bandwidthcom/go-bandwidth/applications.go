package bandwidth

import (
	"fmt"
	"net/http"
)

const applicationsPath = "applications"

// Application struct
type Application struct {
	ID                                string `json:"id,omitempty"`
	Name                              string `json:"name,omitempty"`
	IncomingCallURL                   string `json:"incomingCallUrl,omitempty"`
	IncomingCallURLCallbackTimeout    int    `json:"incomingCallUrlCallbackTimeout,omitempty"`
	IncomingCallFallbackURL           string `json:"incomingCallFallbackUrl,omitempty"`
	IncomingMessageURL                string `json:"incomingMessageUrl,omitempty"`
	IncomingMessageURLCallbackTimeout int    `json:"incomingMessageUrlCallbackTimeout,omitempty"`
	IncomingMessageFallbackURL        string `json:"incomingMessageFallbackUrl,omitempty"`
	CallbackHTTPMethod                string `json:"callbackHttpMethod,omitempty"`
	AutoAnswer                        bool   `json:"autoAnswer,omitempty"`
}

// GetApplicationsQuery is optional parameters of GetApplications()
type GetApplicationsQuery struct {
	Page int
	Size int
}

// GetApplications returns list of user's applications
// It returns list of Application instances or error
func (api *Client) GetApplications(query ...*GetApplicationsQuery) ([]*Application, error) {
	var options *GetApplicationsQuery
	if len(query) > 0 {
		options = query[0]
	}
	result, _, err := api.makeRequest(http.MethodGet, api.concatUserPath(applicationsPath), &[]*Application{}, options)
	if err != nil {
		return nil, err
	}
	return *(result.(*[]*Application)), nil
}

// ApplicationData struct
type ApplicationData struct {
	Name                              string `json:"name,omitempty"`
	IncomingCallURL                   string `json:"incomingCallUrl,omitempty"`
	IncomingCallURLCallbackTimeout    int    `json:"incomingCallUrlCallbackTimeout,omitempty"`
	IncomingCallFallbackURL           string `json:"incomingCallFallbackUrl,omitempty"`
	IncomingMessageURL                string `json:"incomingMessageUrl,omitempty"`
	IncomingMessageURLCallbackTimeout int    `json:"incomingMessageUrlCallbackTimeout,omitempty"`
	IncomingMessageFallbackURL        string `json:"incomingMessageFallbackUrl,omitempty"`
	CallbackHTTPMethod                string `json:"callbackHttpMethod,omitempty"`
	AutoAnswer                        bool   `json:"autoAnswer,omitempty"`
}

// CreateApplication creates an application that can handle calls and messages for one of your phone number. Many phone numbers can share an application.
// It returns ID of created application or error
func (api *Client) CreateApplication(data *ApplicationData) (string, error) {
	_, headers, err := api.makeRequest(http.MethodPost, api.concatUserPath(applicationsPath), nil, data)
	if err != nil {
		return "", err
	}
	return getIDFromLocationHeader(headers), nil
}

// GetApplication returns an user's application
// It returns Application instance or error
func (api *Client) GetApplication(id string) (*Application, error) {
	result, _, err := api.makeRequest(http.MethodGet, fmt.Sprintf("%s/%s", api.concatUserPath(applicationsPath), id), &Application{})
	if err != nil {
		return nil, err
	}
	return result.(*Application), nil
}

// UpdateApplication makes changes to an application
// It returns error object
func (api *Client) UpdateApplication(id string, changedData *ApplicationData) error {
	_, _, err := api.makeRequest(http.MethodPost, fmt.Sprintf("%s/%s", api.concatUserPath(applicationsPath), id), nil, changedData)
	return err
}

// DeleteApplication permanently deletes an application
// It returns error object
func (api *Client) DeleteApplication(id string) error {
	_, _, err := api.makeRequest(http.MethodDelete, fmt.Sprintf("%s/%s", api.concatUserPath(applicationsPath), id))
	return err
}
