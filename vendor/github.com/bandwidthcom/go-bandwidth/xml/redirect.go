package xml

import "encoding/xml"

// Redirect verb is used to redirect the current XML execution to another URL.
type Redirect struct {
	XMLName           xml.Name    `xml:"Redirect"`
	RequestURL        string      `xml:"requestUrl,attr,omitempty"`
	RequestURLTimeout interface{} `xml:"requestUrlTimeout,attr,omitempty"`
}
