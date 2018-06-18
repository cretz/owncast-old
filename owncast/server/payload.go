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

func (p *Payload) UnmarshalPayload(msg *cast_channel.CastMessage) error {
	if msg.PayloadUtf8 == nil {
		return fmt.Errorf("Missing string payload")
	}
	p.JSON = *msg.PayloadUtf8
	if err := json.Unmarshal([]byte(p.JSON), p); err != nil {
		return fmt.Errorf("Failed parsing JSON: %v", err)
	}
	return nil
}

type ConnectPayload struct {
	Payload
	ConnType   *int
	Origin     map[string]interface{}
	SenderInfo map[string]interface{}
	UserAgent  string
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

type AppAvailability string

const (
	AppAvailable   AppAvailability = "APP_AVAILABLE"
	AppUnavailable AppAvailability = "APP_UNAVAILABLE"
)

type GetAppAvailabilityResponsePayload struct {
	Payload
	Availability map[AppID]AppAvailability `json:"availability"`
}

type GetReceiverStatusResponsePayload struct {
	Payload
	Status *ReceiverStatus `json:"status,omitempty"`
}

type ReceiverStatus struct {
	Applications  []*ApplicationSession `json:"applications"`
	IsActiveInput bool                  `json:"isActiveInput,omitempty"`
	Volume        *Volume               `json:"volume,omitempty"`
}

type ApplicationSession struct {
	AppID       string   `json:"appId,omitempty"`
	DisplayName string   `json:"displayName,omitempty"`
	Namespaces  []string `json:"namespaces"`
	SessionID   string   `json:"sessionId,omitempty"`
	StatusText  string   `json:"statusText,omitempty"`
	TransportID string   `json:"transportId,omitempty"`
}

type Volume struct {
	Level float64 `json:"level,omitempty"`
	Muted bool    `json:"muted"`
}

type LaunchPayload struct {
	Payload
	AppID    string
	Language string
}
