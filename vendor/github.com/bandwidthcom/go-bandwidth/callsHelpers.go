package bandwidth


func mergeMaps(src, dst map[string]interface{}){
	if dst == nil {
		dst = map[string]interface{}{}
	}
	for k, v := range dst {
		src[k] = v
	}
}

// AnswerIncomingCall  answers an incoming call
// It returns error object
// example: api.CalAnswerIncomingCall("callId")
func (api *Client) AnswerIncomingCall(id string) error{
	return api.UpdateCall(id,  &UpdateCallData{State: "active"})
}

// RejectIncomingCall  answers an incoming call
// It returns error object
// example: api.RejectIncomingCall("callId")
func (api *Client) RejectIncomingCall(id string) error{
	return api.UpdateCall(id,  &UpdateCallData{State:  "rejected"})
}

// HangUpCall  hangs up the call
// It returns error object
// example: api.HangUpCall("callId")
func (api *Client) HangUpCall(id string) error{
	return api.UpdateCall(id,  &UpdateCallData{State:  "completed"})
}

// SetCallRecodingEnabled  hangs up the call
// It returns error object
// example: api.SetCallRecodingEnabled("callId", true) // enable recording
func (api *Client) SetCallRecodingEnabled(id string, enabled bool) error{
	return api.UpdateCall(id,  &UpdateCallData{RecordingEnabled: enabled})
}


// StopGather stops call's gather
// It returns error object
// example: api.StopGather("callId")
func (api *Client) StopGather(id string, gatherID string) error{
	return api.UpdateGather(id, gatherID, &UpdateGatherData{State: "completed"})
}

// SendDTMFCharactersToCall sends some dtmf characters to call
// It returns error object
// example: api.SendDTMFCharactersToCall("callId", "1")
func (api *Client) SendDTMFCharactersToCall(id string, dtmfOut string) error{
	return api.SendDTMFToCall(id,  &SendDTMFToCallData{DTMFOut: dtmfOut})
}
