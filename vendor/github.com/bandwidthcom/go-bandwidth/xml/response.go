package xml

import (
    "encoding/xml"
)


// Response is response element of BXML
type Response struct {
	Verbs []interface{} `xml:"."`
}

// ToXML builds BXML as string
func (r *Response) ToXML() string{
	bytes, _ := xml.Marshal(r)
	return string(bytes)
}
