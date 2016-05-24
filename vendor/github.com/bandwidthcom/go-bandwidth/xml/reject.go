package xml

import "encoding/xml"

// Reject verb is used to reject incoming calls
type Reject struct {
	XMLName xml.Name `xml:"Reject"`
	Reason  string   `xml:"reason,attr,omitempty"`
}
