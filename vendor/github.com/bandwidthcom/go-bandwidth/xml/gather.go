package xml

import (
	"encoding/xml"
)

//Gather verb is used to collect digits for some period of time.
type Gather struct {
	XMLName           xml.Name    `xml:"Gather"`
	RequestURL        string      `xml:"requestUrl,attr,omitempty"`
	RequestURLTimeout interface{} `xml:"requestUrlTimeout,attr,omitempty"`
	TerminatingDigits interface{} `xml:"terminatingDigits,attr,omitempty"`
	MaxDigits         interface{} `xml:"maxDigits,attr,omitempty"`
	InterDigitTimeout interface{} `xml:"interDigitTimeout,attr,omitempty"`
	Bargeable         interface{} `xml:"bargeable,attr,omitempty"`
}
