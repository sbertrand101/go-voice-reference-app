package xml

import "encoding/xml"

// Transfer verb is used to transfer the call to another number.
type Transfer struct {
	XMLName           xml.Name       `xml:"Transfer"`
	TransferTo        string         `xml:"transferTo,attr,omitempty"`
	TransferCallerID  string         `xml:"transferCallerId,attr,omitempty"`
	RequestURL        string         `xml:"requestUrl,attr,omitempty"`
	RequestURLTimeout interface{}    `xml:"requestUrlTimeout,attr,omitempty"`
	Tag               string         `xml:"tag,attr,omitempty"`
	CallTimeout       interface{}    `xml:"callTimeout,attr,omitempty"`
	PhoneNumbers      []string       `xml:"PhoneNumber"`
	SpeakSentence     *SpeakSentence `xml:",omitempty"`
	PlayAudio         *PlayAudio     `xml:",omitempty"`
	Record            *Record        `xml:",omitempty"`
}
