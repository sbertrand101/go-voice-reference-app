Catapult API in Go [![GoDoc](https://godoc.org/bandwidthcom/go-bandwidth?status.svg)](https://godoc.org/github.com/bandwidthcom/go-bandwidth) [![Build Status](https://travis-ci.org/bandwidthcom/go-bandwidth.svg)](https://travis-ci.org/bandwidthcom/go-bandwidth)
===============


Bandwidth [Bandwidth's App Platform](http://ap.bandwidth.com/?utm_medium=social&utm_source=github&utm_campaign=dtolb&utm_content=) Go SDK

With go-bandwidth  you have access to the entire set of API methods including:
* **Account** - get user's account data and transactions,
* **Application** - manage user's applications,
* **AvailableNumber** - search free local or toll-free phone numbers,
* **Bridge** - control bridges between calls,
* **Call** - get access to user's calls,
* **Conference** - manage user's conferences,
* **ConferenceMember** - make actions with conference members,
* **Domain** - get access to user's domains,
* **EntryPoint** - control of endpoints of domains,
* **Error** - list of errors,
* **Media** - list, upload and download files to Bandwidth API server,
* **Message** - send SMS/MMS, list messages,
* **NumberInfo** - receive CNUM info by phone number,
* **PhoneNumber** - get access to user's phone numbers,
* **Recording** - mamange user's recordings.

Also you can work with Bandwidth XML using special types. 

## Install

```
     go get github.com/bandwidthcom/go-bandwidth
```


## Getting Started

* Install `go-bandwidth`,
* **Get user id, api token and secret** - to use the Catapult API you need these data.  You can get them [here](https://catapult.inetwork.com/pages/catapult.jsf) on the tab "Account",
* **Set user id, api token and secret**

```golang
	import "github.com/bandwidthcom/go-bandwidth"
	
	api := bandwidth.New("userId", "apiToken", "apiSecret")
```

Read [Catapult Api documentation](http://ap.bandwidth.com/) for more details

## Examples

*All examples assume you have already setup your auth data!*

List all calls from special number

```go
  list, _ := api.GetCalls(&bandwidth.GetCallsQuery{From: "+19195551212"})
```

List all received messages

```go
  list, _ := api.GetMessages(&bandwidth.GetMessagesQuery{State: "received"})
```

Send SMS

```go
  messageId, _ := api.CreateMessage(&bandwidth.CreateMessageData{From: "+19195551212", To: "+191955512142", Text:"Test"})
```


Send some SMSes

```go
  statuses, error := api.CreateMessages(
	  &bandwidth.CreateMessageData{From: "+19195551212", To: "+191955512141", Text:"Test1"}, 
	  &bandwidth.CreateMessageData{From: "+19195551212", To: "+191955512142", Text:"Test2"})
```

Upload file

```go
  api.UploadMediaFile("avatar.png", "/local/path/to/file.png", "image/png")
```

Make a call

```go
  api.CreateCall(&bandwidth.CreateCallData{From: "+19195551212",  To: "+191955512142"})
```

Reject incoming call

```go
  api.RejectIncomingCall(callId)
```

Create a gather
```go
  api.CreateGather(callId, &bandwidth.CreateGatherData{MaxDigits: 3, InterDigitTimeout: 5, Prompt: &bandwidth.GatherPromptData{Sentence: "Please enter 3 digits"}})
```

Start a conference
```go
  api.CreateConference(&bandwidth.CreateConferenceData{From: "+19195551212"})
```

Add a member to the conference

```go
  api.CreateConferenceMember(conferenceId, &bandwidth.CreateConferenceMemberData{CallId: "id_of_call_to_add_to_this_conference", JoinTone: true, LeavingTone: true})
```


Connect 2 calls to a bridge

```go
  api.CreateBridge(&bandwidth.BridgeData{CallIDs: []string{callId1, callId2}})
```

Search available local numbers to buy

```go
  list, _ := api.GetAvailableNumbers(bandwidth.AvailableNumberTypeLocal, &bandwidth.GetAvailableNumberQuery{City: "Cary", State: "NC", Quantity: 3})
```
Get CNAM info for a number

```go
  info, _ := api.GetNumberInfo("+19195551212")
```

Buy a phone number

```go
  api.CreatePhoneNumber(&bandwidth.CreatePhoneNumberData{Number: "+19195551212"})
```

List recordings

```go
  list, _ := api.GetRecordings()
```

Generate Bandwidth XML
```go
   import (
	   "github.com/bandwidthcom/go-bandwidth/xml"
	   "fmt"
   )
   
   response := &xml.Response{}
   speakSentence := xml.SpeakSentence{Sentence: "Transferring your call, please wait.", Voice: "paul", Gender: "male", Locale: "en_US"}
   transfer := xml.Transfer{
        TransferTo: "+13032218749",
        TransferCallerId: "private",
        SpeakSentence: &SpeakSentence{
            Sentence: "Inner speak sentence.",
            Voice: "paul",
            Gender: "male",
            Locale: "en_US"}}
    hangup := xml.Hangup{}

    append(response.Verbs, speakSentence)
	append(response.Verbs, transfer)
	append(response.Verbs, hangup)


    //as alternative we can pass list of verbs as
    //response = &xml.Response{Verbs = []interface{}{speakSentence, transfer, hangup}}

    fmt.Println(response.ToXML())
```

See directory `examples` for more demos.

# Bugs/Issues
Please open an issue in this repository and we'll handle it directly. If you have any questions please contact us at openapi@bandwidth.com.

