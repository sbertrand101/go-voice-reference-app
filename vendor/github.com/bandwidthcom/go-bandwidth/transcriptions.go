package bandwidth

import (
	"fmt"
	"net/http"
)

const transcriptionsPath = "transcriptions"

// Transcription struct
type Transcription struct {
	ID                 string `json:"id"`
	ChargeableDuration int    `json:"chargeableDuration"`
	Text               string `json:"text"`
	TextSize           int    `json:"textSize"`
	TextURL            string `json:"textUrl"`
	Time               string `json:"time"`
}

// GetRecordingTranscriptions returns list of all transcriptions for a recording
// It returns list of Transcription instances or error
func (api *Client) GetRecordingTranscriptions(id string) ([]*Transcription, error) {
	result, _, err := api.makeRequest(http.MethodGet, fmt.Sprintf("%s/%s/%s", api.concatUserPath(recordingsPath), id, transcriptionsPath), &[]*Transcription{})
	if err != nil {
		return nil, err
	}
	return *(result.(*[]*Transcription)), nil
}

// CreateRecordingTranscription creates a new transcription for a recording
// It returns ID of created transcription or error
func (api *Client) CreateRecordingTranscription(id string) (string, error) {
	_, headers, err := api.makeRequest(http.MethodPost, fmt.Sprintf("%s/%s/%s", api.concatUserPath(recordingsPath), id, transcriptionsPath))
	if err != nil {
		return "", err
	}
	return getIDFromLocationHeader(headers), nil
}

// GetRecordingTranscription returns   single enpoint for a recording
// It returns Transcription instance or error
func (api *Client) GetRecordingTranscription(recordingID string, transcriptionID string) (*Transcription, error) {
	result, _, err := api.makeRequest(http.MethodGet, fmt.Sprintf("%s/%s/%s/%s", api.concatUserPath(recordingsPath), recordingID, transcriptionsPath, transcriptionID), &Transcription{})
	if err != nil {
		return nil, err
	}
	return result.(*Transcription), nil
}
