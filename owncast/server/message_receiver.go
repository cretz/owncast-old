package server

import (
	"encoding/json"
	"fmt"

	"github.com/cretz/owncast/owncast/log"
	"github.com/cretz/owncast/owncast/server/cast_channel"
)

func ParseReceiverMessage(castMessage *cast_channel.CastMessage) (Message, error) {
	var payload Payload
	if err := payload.UnmarshalPayload(castMessage); err != nil {
		return nil, fmt.Errorf("Unable to get payload: %v")
	}
	switch payload.Type {
	case "GET_APP_AVAILABILITY":
		return NewGetAppAvailabilityMessage(&payload, castMessage)
	case "GET_STATUS":
		return NewGetStatusRequestMessage(&payload, castMessage)
	case "LAUNCH":
		return NewLaunchMessage(&payload, castMessage)
	default:
		return &UnknownMessage{castMessage}, nil
	}
}

type GetAppAvailabilityMessage struct {
	GetAppAvailabilityRequestPayload
	castMessage *cast_channel.CastMessage
}

func NewGetAppAvailabilityMessage(
	payload *Payload,
	castMessage *cast_channel.CastMessage,
) (*GetAppAvailabilityMessage, error) {
	ret := &GetAppAvailabilityMessage{castMessage: castMessage}
	ret.GetAppAvailabilityRequestPayload.Payload = *payload
	if err := json.Unmarshal([]byte(ret.JSON), &ret.GetAppAvailabilityRequestPayload); err != nil {
		return nil, fmt.Errorf("Unable to parse payload: %v", err)
	}
	return ret, nil
}

func (g *GetAppAvailabilityMessage) CastMessage() *cast_channel.CastMessage {
	return g.castMessage
}

func (g *GetAppAvailabilityMessage) HandleDefault(conn *Conn) error {
	log.Debugf("Got app avail request: %v", g.GetAppAvailabilityRequestPayload)
	resp := &GetAppAvailabilityResponsePayload{
		Payload:      g.Payload,
		Availability: make(map[AppID]AppAvailability, len(g.AppID)),
	}
	// Just say they're all available
	for _, appID := range g.AppID {
		resp.Availability[appID] = AppAvailable
	}
	return conn.SendPayload(g.castMessage.GetNamespace(), resp)
}

type GetStatusRequestMessage struct {
	Payload
	castMessage *cast_channel.CastMessage
}

func NewGetStatusRequestMessage(
	payload *Payload,
	castMessage *cast_channel.CastMessage,
) (*GetStatusRequestMessage, error) {
	return &GetStatusRequestMessage{Payload: *payload, castMessage: castMessage}, nil
}

func (g *GetStatusRequestMessage) CastMessage() *cast_channel.CastMessage { return g.castMessage }

func (g *GetStatusRequestMessage) HandleDefault(conn *Conn) error {
	log.Debugf("Got receiver get-status request: %v", &g.Payload)
	resp := &GetReceiverStatusResponsePayload{
		Payload: Payload{Type: "RECEIVER_STATUS", RequestID: g.RequestID},
		Status:  sampleReceiverStatus("CC1AD845"),
	}
	return conn.SendPayload(g.castMessage.GetNamespace(), resp)
}

type LaunchMessage struct {
	LaunchPayload
	castMessage *cast_channel.CastMessage
}

func NewLaunchMessage(payload *Payload, castMessage *cast_channel.CastMessage) (*LaunchMessage, error) {
	ret := &LaunchMessage{castMessage: castMessage}
	ret.LaunchPayload.Payload = *payload
	if err := json.Unmarshal([]byte(ret.JSON), &ret.LaunchPayload); err != nil {
		return nil, fmt.Errorf("Unable to parse payload: %v", err)
	}
	return ret, nil
}

func (l *LaunchMessage) CastMessage() *cast_channel.CastMessage { return l.castMessage }

func (l *LaunchMessage) HandleDefault(conn *Conn) error {
	log.Debugf("Got launch request: %v", l.LaunchPayload)
	resp := &GetReceiverStatusResponsePayload{
		Payload: l.Payload,
		Status:  sampleReceiverStatus(l.AppID),
	}
	return conn.SendPayload(l.castMessage.GetNamespace(), resp)
}

func sampleReceiverStatus(appID string) *ReceiverStatus {
	// TODO: this came from https://github.com/thibauts/node-castv2#controlling-applications
	return &ReceiverStatus{
		Applications: []*ApplicationSession{
			&ApplicationSession{
				AppID:       appID,
				DisplayName: "Default Media Receiver",
				Namespaces: []string{
					"urn:x-cast:com.google.cast.player.message",
					"urn:x-cast:com.google.cast.media",
				},
				SessionID:   "7E2FF513-CDF6-9A91-2B28-3E3DE7BAC174",
				StatusText:  "Ready To Cast",
				TransportID: "web-5",
			},
		},
		IsActiveInput: true,
		Volume: &Volume{
			Level: 1,
			Muted: false,
		},
	}
}
