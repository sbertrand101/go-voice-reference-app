package xml

import "encoding/xml"

// SendMessage is used to send a text message
type SendMessage struct {
	XMLName           xml.Name    `xml:"SendMessage"`
	From              string      `xml:"from,attr,omitempty"`
	To                string      `xml:"to,attr,omitempty"`
	RequestURL        string      `xml:"requestUrl,attr,omitempty"`
	RequestURLTimeout interface{} `xml:"requestUrlTimeout,attr,omitempty"`
	StatusCallbackURL string      `xml:"statusCallbackUrl,attr,omitempty"`
	Text              string      `xml:",chardata"`
}
