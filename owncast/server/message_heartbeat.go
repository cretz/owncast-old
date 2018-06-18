package server

import (
	"fmt"

	"github.com/cretz/owncast/owncast/server/cast_channel"
)

type PingMessage struct {
	Payload
	castMessage *cast_channel.CastMessage
}

func NewPingMessage(castMessage *cast_channel.CastMessage) (*PingMessage, error) {
	ret := &PingMessage{castMessage: castMessage}
	if err := ret.UnmarshalPayload(castMessage); err != nil {
		return nil, fmt.Errorf("Unable to get payload: %v", err)
	} else if ret.Type != "PING" {
		return nil, fmt.Errorf("Expected ping, got %v", ret.Type)
	}
	return ret, nil
}

func (p *PingMessage) CastMessage() *cast_channel.CastMessage { return p.castMessage }

var pongPayload = &Payload{Type: "PONG"}

func (p *PingMessage) HandleDefault(conn *Conn) error {
	return conn.SendPayload(p.castMessage.GetNamespace(), pongPayload)
}
