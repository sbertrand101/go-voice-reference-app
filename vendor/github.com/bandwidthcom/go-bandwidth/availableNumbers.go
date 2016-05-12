package bandwidth

import (
	"fmt"
	"net/http"
)

// AvailableNumberType is allowed number types
type AvailableNumberType string

const (
	// AvailableNumberTypeLocal is local number
	AvailableNumberTypeLocal AvailableNumberType = "local"

	// AvailableNumberTypeTollFree is toll free number
	AvailableNumberTypeTollFree AvailableNumberType = "tollFree"
)

const availableNumbersPath = "availableNumbers"

// AvailableNumber struct
type AvailableNumber struct {
	Number         string  `json:"number"`
	NationalNumber string  `json:"nationalNumber"`
	City           string  `json:"city"`
	LATA           string  `json:"lata"`
	RateCenter     string  `json:"rateCenter"`
	State          string  `json:"state"`
	Price          float64 `json:"price,string"`
}

// GetAvailableNumberQuery is  query parameters of GetAvailableNumbers() and GetAndOrderAvailableNumbers()
type GetAvailableNumberQuery struct {
	City               string
	State              string
	Zip                string
	AreaCode           string
	LocalNumber        string
	InLocalCallingArea bool
	Quantity           int
	Pattern            string
}

// GetAvailableNumbers looks for available numbers
func (api *Client) GetAvailableNumbers(numberType AvailableNumberType, query *GetAvailableNumberQuery) ([]*AvailableNumber, error) {
	result, _, err := api.makeRequest(http.MethodGet, fmt.Sprintf("%s/%s", availableNumbersPath, numberType), &[]*AvailableNumber{}, query)
	if err != nil {
		return nil, err
	}
	return *(result.(*[]*AvailableNumber)), nil
}

// OrderedNumber struct
type OrderedNumber struct {
	Number         string  `json:"number"`
	NationalNumber string  `json:"nationalNumber"`
	Price          float64 `json:"price,string"`
	Location       string  `json:"Location"`
	ID             string  `json:"-"`
}

// GetAndOrderAvailableNumbers looks for available numbers and orders them
func (api *Client) GetAndOrderAvailableNumbers(numberType AvailableNumberType, query *GetAvailableNumberQuery) ([]*OrderedNumber, error) {
	path := fmt.Sprintf("%s/%s", availableNumbersPath, numberType)
	result, _, err := api.makeRequest(http.MethodPost, path, &[]*OrderedNumber{}, query, true)
	if err != nil {
		return nil, err
	}
	list := *(result.(*[]*OrderedNumber))
	for _, item := range list {
		if item.Location != "" {
			item.ID = getIDFromLocation(item.Location)
		}
	}
	return list, nil
}
