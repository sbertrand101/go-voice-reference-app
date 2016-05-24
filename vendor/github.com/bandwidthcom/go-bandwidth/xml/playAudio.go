package xml

import "encoding/xml"

// The PlayAudio verb is used to play an audio file in the call
type PlayAudio struct {
	XMLName xml.Name `xml:"PlayAudio"`
	Digits  string   `xml:"digits,attr,omitempty"`
	URL     string   `xml:",chardata"`
}
