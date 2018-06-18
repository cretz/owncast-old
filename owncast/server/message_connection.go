package server

import (
	"encoding/json"
	"fmt"

	"github.com/cretz/owncast/owncast/log"
	"github.com/cretz/owncast/owncast/server/cast_channel"
)

func ParseConnectionMessage(castMessage *cast_channel.CastMessage) (Message, error) {
	connRet := &ConnectMessage{castMessage: castMessage}
	if err := connRet.UnmarshalPayload(castMessage); err != nil {
		return nil, fmt.Errorf("Unable to get payload: %v", err)
	} else if connRet.Type != "CONNECT" {
		return &UnknownMessage{castMessage}, nil
	} else if err := json.Unmarshal([]byte(connRet.JSON), &connRet.ConnectPayload); err != nil {
		return nil, fmt.Errorf("Unable to parse payload: %v", err)
	}
	return connRet, nil
}

type ConnectMessage struct {
	ConnectPayload
	castMessage *cast_channel.CastMessage
}

func NewConnectMessage(castMessage *cast_channel.CastMessage) (*ConnectMessage, error) {
	ret := &ConnectMessage{castMessage: castMessage}
	if err := ret.UnmarshalPayload(castMessage); err != nil {
		return nil, fmt.Errorf("Unable to get payload: %v", err)
	} else if err := json.Unmarshal([]byte(ret.JSON), &ret.ConnectPayload); err != nil {
		return nil, fmt.Errorf("Unable to parse payload: %v", err)
	}
	return ret, nil
}

func (c *ConnectMessage) CastMessage() *cast_channel.CastMessage { return c.castMessage }

func (c *ConnectMessage) HandleDefault(conn *Conn) error {
	log.Debugf("Client connected, sender info: %v", c.SenderInfo)
	conn.Connected = true
	return nil
}
