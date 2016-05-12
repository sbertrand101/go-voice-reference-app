package bandwidth

import (
	"fmt"
	"net/http"
)

const endpointsPath = "endpoints"

// DomainEndpoint struct
type DomainEndpoint struct {
	ID            string                     `json:"id"`
	Name          string                     `json:"name"`
	Description   string                     `json:"description"`
	DomainID      string                     `json:"domainId"`
	ApplicationID string                     `json:"applicationId"`
	Enabled       bool                       `json:"enabled,string"`
	SipURI        string                     `json:"sipUri"`
	Credentials   *DomainEndpointCredentials `json:"credentials"`
}

// DomainEndpointData struct
type DomainEndpointData struct {
	Name          string                     `json:"name,omitempty"`
	Description   string                     `json:"description,omitempty"`
	DomainID      string                     `json:"domainId,omitempty"`
	ApplicationID string                     `json:"applicationId,omitempty"`
	Enabled       bool                       `json:"enabled,string,omitempty"`
	SipURI        string                     `json:"sipUri,omitempty"`
	Credentials   *DomainEndpointCredentials `json:"credentials,omitempty"`
}

// DomainEndpointCredentials struct
type DomainEndpointCredentials struct {
	Password string `json:"password,omitempty"`
	UserName string `json:"username,omitempty"`
	Realm    string `json:"realm,omitempty"`
}

// DomainEndpointToken struct
type DomainEndpointToken struct {
	Token   string `json:"token"`
	Expires int    `json:"expires"`
}

// GetDomainEndpoints returns list of all endpoints for a domain
// It returns list of DomainEndpoint instances or error
func (api *Client) GetDomainEndpoints(id string) ([]*DomainEndpoint, error) {
	result, _, err := api.makeRequest(http.MethodGet, fmt.Sprintf("%s/%s/%s", api.concatUserPath(domainsPath), id, endpointsPath), &[]*DomainEndpoint{})
	if err != nil {
		return nil, err
	}
	return *(result.(*[]*DomainEndpoint)), nil
}

// CreateDomainEndpoint creates a new endpoint for a domain
// It returns ID of created endpoint or error
func (api *Client) CreateDomainEndpoint(id string, data *DomainEndpointData) (string, error) {
	_, headers, err := api.makeRequest(http.MethodPost, fmt.Sprintf("%s/%s/%s", api.concatUserPath(domainsPath), id, endpointsPath), nil, data)
	if err != nil {
		return "", err
	}
	return getIDFromLocationHeader(headers), nil
}

// GetDomainEndpoint returns   single enpoint for a domain
// It returns DomainEndpoint instance or error
func (api *Client) GetDomainEndpoint(id string, endpointID string) (*DomainEndpoint, error) {
	result, _, err := api.makeRequest(http.MethodGet, fmt.Sprintf("%s/%s/%s/%s", api.concatUserPath(domainsPath), id, endpointsPath, endpointID), &DomainEndpoint{})
	if err != nil {
		return nil, err
	}
	return result.(*DomainEndpoint), nil
}

// DeleteDomainEndpoint removes a endpoint from domain
// It returns error object
func (api *Client) DeleteDomainEndpoint(id string, endpointID string) error {
	_, _, err := api.makeRequest(http.MethodDelete, fmt.Sprintf("%s/%s/%s/%s", api.concatUserPath(domainsPath), id, endpointsPath, endpointID))
	return err
}

// UpdateDomainEndpoint removes a endpoint from domain
// It returns error object
func (api *Client) UpdateDomainEndpoint(id string, endpointID string, changedData *DomainEndpointData) error {
	_, _, err := api.makeRequest(http.MethodPost, fmt.Sprintf("%s/%s/%s/%s", api.concatUserPath(domainsPath), id, endpointsPath, endpointID), nil, changedData)
	return err
}

// CreateDomainEndpointToken creates a new auth token for a domain's enpoint
// It returns token or error
func (api *Client) CreateDomainEndpointToken(id, endpointID string) (*DomainEndpointToken, error) {
	result, _, err := api.makeRequest(http.MethodPost, fmt.Sprintf("%s/%s/%s/%s/tokens", api.concatUserPath(domainsPath), id, endpointsPath, endpointID), &DomainEndpointToken{}, nil)
	if err != nil {
		return nil, err
	}
	return result.(*DomainEndpointToken), nil
}
