package api

import "encoding/json"

// Device is one entry from the Miele REST/SSE device list. ID is the JSON
// object key (the Miele device serial); Data is the raw JSON payload so it can
// be republished without modification on the .../full topic.
type Device struct {
	ID   string
	Data json.RawMessage
}
