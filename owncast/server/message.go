package server

import (
	"github.com/cretz/owncast/owncast/log"
	"github.com/cretz/owncast/owncast/server/cast_channel"
)

type Message interface {
	CastMessage() *cast_channel.CastMessage
}

type MessageWithHandleDefault interface {
	Message
	HandleDefault(*Conn) error
}

type UnknownMessage struct {
	castMessage *cast_channel.CastMessage
}

func (u *UnknownMessage) CastMessage() *cast_channel.CastMessage { return u.castMessage }
func (u *UnknownMessage) HandleDefault(conn *Conn) error {
	log.Debugf("Ignoring unrecognized message: %v", u.castMessage)
	return nil
}

func ParseMessage(msg *cast_channel.CastMessage) (Message, error) {
	switch ns := msg.GetNamespace(); ns {
	case "urn:x-cast:com.google.cast.receiver":
		return ParseReceiverMessage(msg)
	case "urn:x-cast:com.google.cast.tp.connection":
		return ParseConnectionMessage(msg)
	case "urn:x-cast:com.google.cast.tp.deviceauth":
		return NewDeviceAuthMessage(msg)
	case "urn:x-cast:com.google.cast.tp.heartbeat":
		return NewPingMessage(msg)
	default:
		return &UnknownMessage{msg}, nil
	}
}
