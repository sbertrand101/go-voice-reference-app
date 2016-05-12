package bandwidth

import (
	"fmt"
	"net/http"
	"net/url"
)

const phoneNumbersPath = "phoneNumbers"

// PhoneNumber struct
type PhoneNumber struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Number         string  `json:"number"`
	NationalNumber string  `json:"nationalNumber"`
	City           string  `json:"city"`
	State          string  `json:"state"`
	ApplicationID  string  `json:"applicationId"`
	FallbackNumber string  `json:"fallbackNumber"`
	CreatedTime    string  `json:"createdTime"`
	NumberState    string  `json:"numberState"`
	Price          float64 `json:"price,string"`
}

// CreatePhoneNumberData struct
type CreatePhoneNumberData struct {
	Number         string               `json:"number,omitempty"`
	Name           string               `json:"name,omitempty"`
	ApplicationID  string               `json:"applicationId,omitempty"`
	FallbackNumber string               `json:"fallbackNumber,omitempty"`
	Provider       *PhoneNumberProvider `json:"provider,omitempty"`
}

// PhoneNumberProvider struct
type PhoneNumberProvider struct {
	Name       string                 `json:"providerName,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// UpdatePhoneNumberData struct
type UpdatePhoneNumberData struct {
	Name           string `json:"name,omitempty"`
	ApplicationID  string `json:"applicationId,omitempty"`
	FallbackNumber string `json:"fallbackNumber,omitempty"`
}

// GetPhoneNumbersQuery is optional parameters of GetPhoneNumbers()
type GetPhoneNumbersQuery struct {
	Page          int
	Size          int
	ApplicationID string
	State         string
	Name          string
	City          string
	NumberState   string
}

// GetPhoneNumbers returns a list of your numbers
// It returns list of PhoneNumber instances or error
func (api *Client) GetPhoneNumbers(query ...*GetPhoneNumbersQuery) ([]*PhoneNumber, error) {
	var options *GetPhoneNumbersQuery
	if len(query) > 0 {
		options = query[0]
	}
	result, _, err := api.makeRequest(http.MethodGet, api.concatUserPath(phoneNumbersPath), &[]*PhoneNumber{}, options)
	if err != nil {
		return nil, err
	}
	return *(result.(*[]*PhoneNumber)), nil
}

// CreatePhoneNumber creates a new phone number
// It returns ID of created phone number or error
func (api *Client) CreatePhoneNumber(data *CreatePhoneNumberData) (string, error) {
	_, headers, err := api.makeRequest(http.MethodPost, api.concatUserPath(phoneNumbersPath), nil, data)
	if err != nil {
		return "", err
	}
	return getIDFromLocationHeader(headers), nil
}

// GetPhoneNumber returns information for phone number by id or number
// It returns instance of PhoneNumber or error
func (api *Client) GetPhoneNumber(idOrNumber string) (*PhoneNumber, error) {
	result, _, err := api.makeRequest(http.MethodGet, fmt.Sprintf("%s/%s", api.concatUserPath(phoneNumbersPath), url.QueryEscape(idOrNumber)), &PhoneNumber{})
	if err != nil {
		return nil, err
	}
	return result.(*PhoneNumber), nil
}

// UpdatePhoneNumber makes changes to your number
// It returns error object
func (api *Client) UpdatePhoneNumber(idOrNumber string, data *UpdatePhoneNumberData) error {
	_, _, err := api.makeRequest(http.MethodPost, fmt.Sprintf("%s/%s", api.concatUserPath(phoneNumbersPath), url.QueryEscape(idOrNumber)), nil, data)
	return err
}

// DeletePhoneNumber removes a phone number
// It returns error object
func (api *Client) DeletePhoneNumber(id string) error {
	_, _, err := api.makeRequest(http.MethodDelete, fmt.Sprintf("%s/%s", api.concatUserPath(phoneNumbersPath), id))
	return err
}
