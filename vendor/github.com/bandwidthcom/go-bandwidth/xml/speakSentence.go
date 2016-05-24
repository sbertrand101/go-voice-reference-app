package xml

import "encoding/xml"

// The SpeakSentence verb is used to convert any text into speak for the caller
type SpeakSentence struct {
	XMLName  xml.Name    `xml:"SpeakSentence"`
	Gender   interface{} `xml:"gender,attr,omitempty"`
	Locale   interface{} `xml:"locale,attr,omitempty"`
	Voice    string      `xml:"voice,attr,omitempty"`
	Sentence string      `xml:",chardata"`
}
