package bandwidth

// TerminateConference terminates a  conference
// example: api.TerminateConference("conferenceId")
func (api *Client) TerminateConference(id string) error{
	return api.UpdateConference(id, &UpdateConferenceData{State: "completed"})
}

// MuteConference mutes/unmutes a  conference
// example: api.MuteConference("conferenceId", false) //unmute it
func (api *Client) MuteConference(id string, mute bool) error{
	return api.UpdateConference(id, &UpdateConferenceData{Mute: mute})
}

// DeleteConferenceMember removes the member from the conference
// example: api.DeleteConferenceMember("conferenceId", "memberId")
func (api *Client) DeleteConferenceMember(id string, memberID string) error{
	return api.UpdateConferenceMember(id, memberID, &UpdateConferenceMemberData{State: "completed"})
}

// MuteConferenceMember mute/unmute the conference member
// example: api.MuteConferenceMember("conferenceId", "memberId", true) //mute member
func (api *Client) MuteConferenceMember(id string, memberID string, mute bool) error{
	return api.UpdateConferenceMember(id, memberID, &UpdateConferenceMemberData{Mute: mute})
}

// HoldConferenceMember hold/unhold the conference member
// example: api.HoldConferenceMember("conferenceId", "memberId", true) //hold member
func (api *Client) HoldConferenceMember(id string, memberID string, hold bool) error{
	return api.UpdateConferenceMember(id, memberID, &UpdateConferenceMemberData{Hold: hold})
}
