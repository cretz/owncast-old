package server

import (
	"encoding/json"
	"fmt"

	"github.com/cretz/owncast/owncast/server/cast_channel"
)

type Payload struct {
	Type      string `json:"type"`
	RequestID *int   `json:"requestId,omitempty"`
	JSON      string `json:"-"`
}

func UnmarshalPayload(msg *cast_channel.CastMessage) (*Payload, error) {
	if msg.PayloadUtf8 == nil {
		return nil, fmt.Errorf("Missing string payload")
	}
	ret := &Payload{JSON: *msg.PayloadUtf8}
	if err := json.Unmarshal([]byte(ret.JSON), ret); err != nil {
		return nil, fmt.Errorf("Failed parsing JSON: %v", err)
	}
	return ret, nil
}

type ConnectPayload struct {
	Payload
	ConnType   *int
	Origin     map[string]interface{}
	SenderInfo map[string]interface{}
	UserAgent  string
}

func (p *Payload) ParseConnect() (*ConnectPayload, error) {
	var ret ConnectPayload
	return &ret, json.Unmarshal([]byte(p.JSON), &ret)
}

type AppID string

const (
	MirroringAppID      AppID = "0F5096E8"
	AudioMirroringAppID AppID = "85CDB22F"
)

type GetAppAvailabilityRequestPayload struct {
	Payload
	AppID []AppID
}

func (p *Payload) ParseGetAppAvailabilityRequest() (*GetAppAvailabilityRequestPayload, error) {
	var ret GetAppAvailabilityRequestPayload
	return &ret, json.Unmarshal([]byte(p.JSON), &ret)
}

type AppAvailability string

const (
	AppAvailable   AppAvailability = "APP_AVAILABLE"
	AppUnavailable AppAvailability = "APP_UNAVAILABLE"
)

type GetAppAvailabilityResponsePayload struct {
	Payload
	Availability map[AppID]AppAvailability `json:"availability"`
}
