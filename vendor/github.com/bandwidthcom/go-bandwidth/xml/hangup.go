package xml

import "encoding/xml"

// The Hangup verb is used to hangup current call
type Hangup struct {
	XMLName xml.Name `xml:"Hangup"`
}
