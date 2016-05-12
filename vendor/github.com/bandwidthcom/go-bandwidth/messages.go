package bandwidth

import (
	"fmt"
	"net/http"
)

const messagesPath = "messages"

// Message struct
type Message struct {
	ID                  string   `json:"id"`
	From                string   `json:"from"`
	To                  string   `json:"to"`
	Direction           string   `json:"direction"`
	Text                string   `json:"text"`
	Media               []string `json:"media"`
	State               string   `json:"state"`
	Time                string   `json:"time"`
	CallbackURL         string   `json:"callbackUrl"`
	CallbackHTTPMethod  string   `json:"callbackHttpMethod,omitempty"`
	FallbackURL         string   `json:"fallbackUrl,omitempty"`
	CallbackTimeout     int      `json:"callbackTimeout,omitempty"`
	ReceiptRequested    string   `json:"receiptRequested"`
	DeliveryState       string   `json:"deliveryState"`
	DeliveryCode        string   `json:"deliveryCode"`
	DeliveryDescription string   `json:"deliveryDescription"`
	Tag                 string   `json:"tag"`
}

// CreateMessageData struct
type CreateMessageData struct {
	From               string   `json:"from,omitempty"`
	To                 string   `json:"to,omitempty"`
	Text               string   `json:"text,omitempty"`
	Media              []string `json:"media,omitempty"`
	CallbackURL        string   `json:"callbackUrl,omitempty"`
	CallbackHTTPMethod string   `json:"callbackHttpMethod,omitempty"`
	FallbackURL        string   `json:"fallbackUrl,omitempty"`
	CallbackTimeout    int      `json:"callbackTimeout,omitempty"`
	ReceiptRequested   string   `json:"receiptRequested,omitempty"`
	Tag                string   `json:"tag,omitempty"`
}

// GetMessagesQuery is optional parameters of GetMessages()
type GetMessagesQuery struct {
	Page          int
	Size          int
	From          string
	To            string
	FromDateTime  string
	ToDateTime    string
	Direction     string
	State         string
	DeliveryState string
	SortOrder     string
}

// CreateMessageResult stores status of sent message (in batch mode)
type CreateMessageResult struct {
	Result   string `json:"result,omitempty"`
	Location string `json:"location,omitempty"`
	ID       string `json:"-"`
}

// GetMessages returns list of all messages
// It returns list of Message instances or error
func (api *Client) GetMessages(query ...*GetMessagesQuery) ([]*Message, error) {
	var options *GetMessagesQuery
	if len(query) > 0 {
		options = query[0]
	}
	result, _, err := api.makeRequest(http.MethodGet, api.concatUserPath(messagesPath), &[]*Message{}, options)
	if err != nil {
		return nil, err
	}
	return *(result.(*[]*Message)), nil
}

// CreateMessage sends a message (SMS/MMS)
// It returns ID of created message or error
func (api *Client) CreateMessage(data *CreateMessageData) (string, error) {
	_, headers, err := api.makeRequest(http.MethodPost, api.concatUserPath(messagesPath), nil, data)
	if err != nil {
		return "", err
	}
	return getIDFromLocationHeader(headers), nil
}

// CreateMessages sends some messages (SMS/MMS)
// It statuses of created messages or error
func (api *Client) CreateMessages(data ...*CreateMessageData) ([]*CreateMessageResult, error) {
	result, _, err := api.makeRequest(http.MethodPost, api.concatUserPath(messagesPath), &[]*CreateMessageResult{}, data)
	if err != nil {
		return nil, err
	}
	list := *(result.(*[]*CreateMessageResult))
	for _, r := range list {
		r.ID = getIDFromLocation(r.Location)
	}
	return list, nil
}

// GetMessage returns a single message
// It returns Message instance or error
func (api *Client) GetMessage(id string) (*Message, error) {
	result, _, err := api.makeRequest(http.MethodGet, fmt.Sprintf("%s/%s", api.concatUserPath(messagesPath), id), &Message{})
	if err != nil {
		return nil, err
	}
	return result.(*Message), nil
}
