package bandwidth

import (
	"fmt"
	"net/http"
)

const callsPath = "calls"

// Call struct
type Call struct {
	ID                   string            `json:"id"`
	ActiveTime           string            `json:"activeTime"`
	StartTime            string            `json:"startTime"`
	EndTime              string            `json:"endTime"`
	ChargeableDuration   int               `json:"chargeableDuration"`
	Direction            string            `json:"direction"`
	From                 string            `json:"from"`
	RecordingFileFormat  string            `json:"recordingFileFormat"`
	RecordingEnabled     bool              `json:"recordingEnabled"`
	RecordingMaxDuration int               `json:"recordingMaxDuration"`
	State                string            `json:"state"`
	To                   string            `json:"to"`
	TranscriptionEnabled bool              `json:"transcriptionEnabled"`
	SipHeaders           map[string]string `json:"sipHeaders"`
	ConferenceID         string            `json:"conferenceId"`
	BridgeID             string            `json:"bridgeId"`
	TransferCallerID     string            `json:"transferCallerId"`
	TransferTo           string            `json:"transferTo"`
	Tag                  string            `json:"tag"`
	CallbackURL          string            `json:"callbackUrl"`
	CallbackHTTPMethod   string            `json:"callbackHttpMethod"`
	FallbackURL          string            `json:"fallbackUrl"`
	CallbackTimeout      int               `json:"callbackTimeout"`
}

// GetCallsQuery is optional parameters of GetCalls()
type GetCallsQuery struct {
	Page         int
	Size         int
	BridgeID     string
	ConferenceID string
	From         string
	To           string
	SortOrder    string
}

// GetCalls returns list of previous calls that were made or received
// It returns list of Call instances or error
func (api *Client) GetCalls(query ...*GetCallsQuery) ([]*Call, error) {
	var options *GetCallsQuery
	if len(query) > 0 {
		options = query[0]
	}
	result, _, err := api.makeRequest(http.MethodGet, api.concatUserPath(callsPath), &[]*Call{}, options)
	if err != nil {
		return nil, err
	}
	return *(result.(*[]*Call)), nil
}

// CreateCallData struct
type CreateCallData struct {
	From                 string            `json:"from,omitempty"`
	RecordingFileFormat  string            `json:"recordingFileFormat,omitempty"`
	RecordingEnabled     bool              `json:"recordingEnabled,omitempty"`
	RecordingMaxDuration int               `json:"recordingMaxDuration,omitempty"`
	State                string            `json:"state,omitempty"`
	To                   string            `json:"to,omitempty"`
	TranscriptionEnabled bool              `json:"transcriptionEnabled,omitempty"`
	SipHeaders           map[string]string `json:"sipHeaders,omitempty"`
	ConferenceID         string            `json:"conferenceId,omitempty"`
	BridgeID             string            `json:"bridgeId,omitempty"`
	Tag                  string            `json:"tag,omitempty"`
	CallbackURL          string            `json:"callbackUrl,omitempty"`
	CallbackHTTPMethod   string            `json:"callbackHttpMethod,omitempty"`
	FallbackURL          string            `json:"fallbackUrl,omitempty"`
	CallbackTimeout      int               `json:"callbackTimeout,omitempty"`
	CallTimeout          int               `json:"callTimeout,omitempty"`
}

// CreateCall creates an outbound phone call
// It returns ID of created call
func (api *Client) CreateCall(data *CreateCallData) (string, error) {
	_, headers, err := api.makeRequest(http.MethodPost, api.concatUserPath(callsPath), nil, data)
	if err != nil {
		return "", err
	}
	return getIDFromLocationHeader(headers), nil
}

// GetCall returns information about a call that was made or received
// It return Call instance for found call or error
func (api *Client) GetCall(id string) (*Call, error) {
	result, _, err := api.makeRequest(http.MethodGet, fmt.Sprintf("%s/%s", api.concatUserPath(callsPath), id), &Call{})
	if err != nil {
		return nil, err
	}
	return result.(*Call), nil
}

// UpdateCallData struct
type UpdateCallData struct {
	TransferCallerID     string         `json:"transferCallerId,omitempty"`
	TransferTo           string         `json:"transferTo,omitempty"`
	RecordingEnabled     bool           `json:"recordingEnabled,string,omitempty"`
	RecordingFileFormat  string         `json:"recordingFileFormat,omitempty"`
	State                string         `json:"state,omitempty"`
	TranscriptionEnabled bool           `json:"transcriptionEnabled,string,omitempty"`
	CallbackURL          string         `json:"callbackUrl,omitempty"`
	WhisperAudio         *PlayAudioData `json:"whisperAudio,omitempty"`
	Tag                  string         `json:"tag,omitempty"`
}

// UpdateCall manage an active phone call. E.g. Answer an incoming call, reject an incoming call, turn on / off recording, transfer, hang up
// It returns error object
func (api *Client) UpdateCall(id string, changedData *UpdateCallData) (string, error) {
	_, headers, err := api.makeRequest(http.MethodPost, fmt.Sprintf("%s/%s", api.concatUserPath(callsPath), id), nil, changedData)
	return getIDFromLocationHeader(headers), err
}

// PlayAudioToCall plays an audio or speak a sentence in a call
// It returns error object
func (api *Client) PlayAudioToCall(id string, data *PlayAudioData) error {
	_, _, err := api.makeRequest(http.MethodPost, fmt.Sprintf("%s/%s/%s", api.concatUserPath(callsPath), id, "audio"), nil, data)
	return err
}

// PlayAudioToCallWithMap plays an audio or speak a sentence in a call
// It returns error object
func (api *Client) PlayAudioToCallWithMap(id string, data map[string]interface{}) error {
	_, _, err := api.makeRequest(http.MethodPost, fmt.Sprintf("%s/%s/%s", api.concatUserPath(callsPath), id, "audio"), nil, data)
	return err
}

// SendDTMFToCallData struct
type SendDTMFToCallData struct {
	DTMFOut string `json:"dtmfOut,omitempty"`
}

// SendDTMFToCall plays an audio or speak a sentence in a call
// It returns error object
func (api *Client) SendDTMFToCall(id string, data *SendDTMFToCallData) error {
	_, _, err := api.makeRequest(http.MethodPost, fmt.Sprintf("%s/%s/%s", api.concatUserPath(callsPath), id, "dtmf"), nil, data)
	return err
}

// CallEvent struct
type CallEvent struct {
	ID   string `json:"id"`
	Time string `json:"time"`
	Name string `json:"name"`
}

// GetCallEvents returns  the list of call events for a call
// It returns list of CallEvent instances or error
func (api *Client) GetCallEvents(id string) ([]*CallEvent, error) {
	result, _, err := api.makeRequest(http.MethodGet, fmt.Sprintf("%s/%s/%s", api.concatUserPath(callsPath), id, "events"), &[]*CallEvent{})
	if err != nil {
		return nil, err
	}
	return *(result.(*[]*CallEvent)), nil
}

// GetCallEvent returns information about one call event
// It returns CallEvent instance for found event or error
func (api *Client) GetCallEvent(id string, eventID string) (*CallEvent, error) {
	result, _, err := api.makeRequest(http.MethodGet, fmt.Sprintf("%s/%s/%s/%s", api.concatUserPath(callsPath), id, "events", eventID), &CallEvent{})
	if err != nil {
		return nil, err
	}
	return result.(*CallEvent), nil
}

// GetCallRecordings returns  all recordings related to the call
// It return list of Recording instances or error
func (api *Client) GetCallRecordings(id string) ([]*Recording, error) {
	result, _, err := api.makeRequest(http.MethodGet, fmt.Sprintf("%s/%s/%s", api.concatUserPath(callsPath), id, "recordings"), &[]*Recording{})
	if err != nil {
		return nil, err
	}
	return *(result.(*[]*Recording)), nil
}

// GetCallTranscriptions returns  all transcriptions  related to the call
// It return list of Transcription instances or error
func (api *Client) GetCallTranscriptions(id string) ([]*Transcription, error) {
	result, _, err := api.makeRequest(http.MethodGet, fmt.Sprintf("%s/%s/%s", api.concatUserPath(callsPath), id, "transcriptions"), &[]*Transcription{})
	if err != nil {
		return nil, err
	}
	return *(result.(*[]*Transcription)), nil
}

// CreateGatherData struct
type CreateGatherData struct {
	MaxDigits         int               `json:"maxDigits,string,omitempty"`
	InterDigitTimeout int               `json:"interDigitTimeout,string,omitempty"`
	TerminatingDigits string            `json:"terminatingDigits,omitempty"`
	Tag               string            `json:"tag,omitempty"`
	Prompt            *GatherPromptData `json:"prompt,omitempty"`
}

// GatherPromptData struct
type GatherPromptData struct {
	FileURL     string `json:"fileUrl,omitempty"`
	Sentence    string `json:"sentence,omitempty"`
	Gender      string `json:"gender,omitempty"`
	Locale      string `json:"locale,omitempty"`
	Voice       string `json:"voice,omitempty"`
	LoopEnabled bool   `json:"loopEnabled,omitempty"`
	Bargeable   bool   `json:"bargeable, string"`
}

// CreateGather gathers the DTMF digits pressed in a call
// It returns ID of created gather or error
func (api *Client) CreateGather(id string, data *CreateGatherData) (string, error) {
	_, headers, err := api.makeRequest(http.MethodPost, fmt.Sprintf("%s/%s/%s", api.concatUserPath(callsPath), id, "gather"), nil, data)
	if err != nil {
		return "", err
	}
	return getIDFromLocationHeader(headers), nil
}

// Gather struct
type Gather struct {
	ID            string `json:"id"`
	State         string `json:"state"`
	Reason        string `json:"reason"`
	CreatedTime   string `json:"createdTime"`
	CompletedTime string `json:"completedTime"`
	Digits        string `json:"digits"`
}

// GetGather returns the gather DTMF parameters and results of the call
// It returns Gather instance or error
func (api *Client) GetGather(id string, gatherID string) (*Gather, error) {
	result, _, err := api.makeRequest(http.MethodGet, fmt.Sprintf("%s/%s/%s/%s", api.concatUserPath(callsPath), id, "gather", gatherID), &Gather{})
	if err != nil {
		return nil, err
	}
	return result.(*Gather), nil
}

// UpdateGatherData struct
type UpdateGatherData struct {
	State string `json:"state,omitempty"`
}

// UpdateGather updates call's gather data
// It returns error object
func (api *Client) UpdateGather(id string, gatherID string, data *UpdateGatherData) error {
	_, _, err := api.makeRequest(http.MethodPost, fmt.Sprintf("%s/%s/%s/%s", api.concatUserPath(callsPath), id, "gather", gatherID), nil, data)
	return err
}
