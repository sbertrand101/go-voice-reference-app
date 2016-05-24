package xml

import "encoding/xml"

// Pause is a verb to specify the length of seconds to wait before executing the next verb
type Pause struct {
	XMLName  xml.Name `xml:"Pause"`
	Duration int      `xml:"duration,attr"`
}
