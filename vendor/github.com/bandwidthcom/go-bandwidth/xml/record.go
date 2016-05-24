package xml

import "encoding/xml"

// Record verb allow call recording
type Record struct {
	XMLName               xml.Name    `xml:"Record"`
	RequestURL            string      `xml:"requestUrl,attr,omitempty"`
	RequestURLTimeout     interface{} `xml:"requestUrlTimeout,attr,omitempty"`
	TerminatingDigits     interface{} `xml:"terminatingDigits,attr,omitempty"`
	MaxDuration           interface{} `xml:"maxDuration,attr,omitempty"`
	Transcribe            interface{} `xml:"transcribe,attr,omitempty"`
	TranscribeCallbackURL string      `xml:"transcribeCallbackUrl,attr,omitempty"`
}
