package bandwidth

import (
	"fmt"
	"net/http"
)

const errorsPath = "errors"

// Error struct
type Error struct {
	ID       string         `json:"id"`
	Category string         `json:"category"`
	Time     string         `json:"time"`
	Code     string         `json:"code"`
	Details  []*ErrorDetail `json:"details"`
}

// ErrorDetail struct
type ErrorDetail struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// GetErrorsQuery is optional parameters of GetErrors()
type GetErrorsQuery struct {
	Page int
	Size int
}


// GetErrors returns list of errors
// It returns list of Error instances or error
func (api *Client) GetErrors(query ...*GetErrorsQuery) ([]*Error, error) {
	var options *GetErrorsQuery
	if len(query) > 0 {
		options = query[0]
	}
	result, _, err := api.makeRequest(http.MethodGet, api.concatUserPath(errorsPath), &[]*Error{}, options)
	if err != nil {
		return nil, err
	}
	return *(result.(*[]*Error)), nil
}

// GetError returns  error by id
// It return Error instance for found error or error object
func (api *Client) GetError(id string) (*Error, error) {
	result, _, err := api.makeRequest(http.MethodGet, fmt.Sprintf("%s/%s", api.concatUserPath(errorsPath), id), &Error{})
	if err != nil {
		return nil, err
	}
	return result.(*Error), nil
}
