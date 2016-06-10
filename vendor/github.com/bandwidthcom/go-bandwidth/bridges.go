package bandwidth

import (
	"fmt"
	"net/http"
)

const bridgesPath = "bridges"

// Bridge struct
type Bridge struct {
	ID            string   `json:"id"`
	State         string   `json:"state"`
	BridgeAudio   bool     `json:"bridgeAudio,string"`
	CallIDs       []string `json:"callIds"`
	CreatedTime   string   `json:"createdTime"`
	ActivatedTime string   `json:"activatedTime"`
	CompletedTime string   `json:"completedTime"`
}

// GetBridges returns list of previous bridges
// It returns list of Bridge instances or error
func (api *Client) GetBridges() ([]*Bridge, error) {
	result, _, err := api.makeRequest(http.MethodGet, api.concatUserPath(bridgesPath), &[]*Bridge{})
	if err != nil {
		return nil, err
	}
	return *(result.(*[]*Bridge)), nil
}

// BridgeData struct
type BridgeData struct {
	BridgeAudio bool     `json:"bridgeAudio,string,omitempty"`
	CallIDs     []string `json:"callIds,omitempty"`
}

// CreateBridge creates a bridge
// It returns ID of created bridge
func (api *Client) CreateBridge(data *BridgeData) (string, error) {
	_, headers, err := api.makeRequest(http.MethodPost, api.concatUserPath(bridgesPath), nil, data)
	if err != nil {
		return "", err
	}
	return getIDFromLocationHeader(headers), nil
}

// GetBridge returns a bridge
// It returns Bridge instance fo found bridge or error
func (api *Client) GetBridge(id string) (*Bridge, error) {
	result, _, err := api.makeRequest(http.MethodGet, fmt.Sprintf("%s/%s", api.concatUserPath(bridgesPath), id), &Bridge{})
	if err != nil {
		return nil, err
	}
	return result.(*Bridge), nil
}

// UpdateBridge adds one or two calls in a bridge and also puts the bridge on hold/unhold
// It returns error object
func (api *Client) UpdateBridge(id string, changedData *BridgeData) error {
	_, _, err := api.makeRequest(http.MethodPost, fmt.Sprintf("%s/%s", api.concatUserPath(bridgesPath), id), nil, changedData)
	return err
}

// PlayAudioData struct
type PlayAudioData struct {
	FileURL     string `json:"fileUrl,omitempty"`
	Sentence    string `json:"sentence,omitempty"`
	Gender      string `json:"gender,omitempty"`
	Locale      string `json:"locale,omitempty"`
	Voice       string `json:"voice,omitempty"`
	LoopEnabled bool   `json:"loopEnabled,omitempty"`
	Tag         string `json:"tag,omitempty"`
}

// PlayAudioToBridge plays an audio or speak a sentence in a bridge
// It returns error object
func (api *Client) PlayAudioToBridge(id string, data *PlayAudioData) error {
	_, _, err := api.makeRequest(http.MethodPost, fmt.Sprintf("%s/%s/%s", api.concatUserPath(bridgesPath), id, "audio"), nil, data)
	return err
}

// GetBridgeCalls returns bridge's calls
// It returns list of Call instances or error
func (api *Client) GetBridgeCalls(id string) ([]*Call, error) {
	result, _, err := api.makeRequest(http.MethodGet, fmt.Sprintf("%s/%s/%s", api.concatUserPath(bridgesPath), id, "calls"), &[]*Call{})
	if err != nil {
		return nil, err
	}
	return *(result.(*[]*Call)), nil
}
