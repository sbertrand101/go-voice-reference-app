package bandwidth

import (
	"fmt"
	"net/http"
	"net/url"
)

const numberInfoPath = "phoneNumbers/numberInfo"

// NumberInfo struct
type NumberInfo struct {
	Created string `json:"created"`
	Name    string `json:"name"`
	Number  string `json:"number"`
	Updated string `json:"updated"`
}

// GetNumberInfo returns information fo given number
// It returns NumberInfo instance or error
func (api *Client) GetNumberInfo(number string) (*NumberInfo, error) {
	result, _, err := api.makeRequest(http.MethodGet, fmt.Sprintf("%s/%s", numberInfoPath, url.QueryEscape(number)), &NumberInfo{})
	if err != nil {
		return nil, err
	}
	return result.(*NumberInfo), nil
}
