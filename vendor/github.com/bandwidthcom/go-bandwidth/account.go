package bandwidth

import (
	"fmt"
	"net/http"
)

const accountPath = "account"

// Account struct
type Account struct {
	Balance     float64 `json:"balance,string"`
	AccountType string  `json:"accountType"`
}

// GetAccount returns account information (balance, etc)
// It returns Account instance or error
func (api *Client) GetAccount() (*Account, error) {
	result, _, err := api.makeRequest(http.MethodGet, api.concatUserPath(accountPath), &Account{})
	if err != nil {
		return nil, err
	}
	return result.(*Account), nil
}

// AccountTransaction struct
type AccountTransaction struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Time        string  `json:"time"`
	Amount      float64 `json:"amount,string"`
	Units       string  `json:"units"`
	ProductType string  `json:"productType"`
	Number      string  `json:"number"`
}

// GetAccountTransactions returns transactions from the user's account
// It returns list of AccountTransaction instances or error
func (api *Client) GetAccountTransactions() ([]*AccountTransaction, error) {
	result, _, err := api.makeRequest(http.MethodGet, fmt.Sprintf("%s/%s", api.concatUserPath(accountPath), "transactions"), &[]*AccountTransaction{})
	if err != nil {
		return nil, err
	}
	return *(result.(*[]*AccountTransaction)), nil
}
