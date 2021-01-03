package server

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"github.com/cretz/owncast/owncast/log"
	"github.com/cretz/owncast/owncast/server/cast_channel"
	"github.com/golang/protobuf/proto"
)

type Conn struct {
	conn          net.Conn
	server        *Server
	Authenticated bool
	Connected     bool
}

func (s *Server) Accept() (*Conn, error) {
	if s.tlsListener == nil {
		return nil, fmt.Errorf("No listener")
	}
	log.Debugf("Waiting for connection")
	conn, err := s.tlsListener.Accept()
	if err != nil {
		return nil, err
	}
	return &Conn{conn: conn, server: s}, nil
}

func (c *Conn) Close() error { return c.conn.Close() }

func (c *Conn) ReceiveMessage() (Message, error) {
	castMsg, err := c.ReceiveCastMessage()
	if err != nil {
		return nil, fmt.Errorf("Failed receiving message: %v", err)
	} else if castMsg.GetProtocolVersion() != cast_channel.CastMessage_CASTV2_1_0 {
		return nil, fmt.Errorf("Unrecognized version: %v", castMsg.GetProtocolVersion())
	}
	return ParseMessage(castMsg)
}

func (c *Conn) ReceiveCastMessage() (*cast_channel.CastMessage, error) {
	// Get msg size
	byts := make([]byte, 4)
	if _, err := io.ReadFull(c.conn, byts); err != nil {
		return nil, fmt.Errorf("Failed reading size: %v", err)
	}
	msgSize := binary.BigEndian.Uint32(byts)
	// Get actual message
	byts = make([]byte, msgSize)
	if _, err := io.ReadFull(c.conn, byts); err != nil {
		return nil, fmt.Errorf("Unable to read msg: %v", err)
	}
	var msg cast_channel.CastMessage
	if err := proto.Unmarshal(byts, &msg); err != nil {
		return nil, fmt.Errorf("Unable to unmarshal msg: %v", err)
	}
	log.Debugf("Received message: %v", &msg)
	return &msg, nil
}

func (c *Conn) SendPayload(namespace string, payload interface{}) error {
	byts, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("Failed marshalling payload: %v", err)
	}
	return c.SendStringMessage(namespace, string(byts))
}

func (c *Conn) SendMessage(msg *cast_channel.CastMessage) error {
	log.Debugf("Sending message: %v", msg)
	byts, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("Unable to marshal cast message: %v", err)
	}
	sizeByts := make([]byte, 4)
	binary.BigEndian.PutUint32(sizeByts, uint32(len(byts)))
	if _, err = c.conn.Write(sizeByts); err != nil {
		return fmt.Errorf("Unable to write size: %v", err)
	}
	if _, err = c.conn.Write(byts); err != nil {
		return fmt.Errorf("Unable to write bytes: %v", err)
	}
	return nil
}

func (c *Conn) SendProtoMessage(namespace string, msg proto.Message) error {
	byts, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("Failed marshalling message: %v", err)
	}
	return c.SendBinaryMessage(namespace, byts)
}

func (c *Conn) SendBinaryMessage(namespace string, msg []byte) error {
	version := cast_channel.CastMessage_CASTV2_1_0
	sourceID := "receiver-0"
	destinationID := "*"
	payloadType := cast_channel.CastMessage_BINARY
	return c.SendMessage(&cast_channel.CastMessage{
		ProtocolVersion: &version,
		SourceId:        &sourceID,
		DestinationId:   &destinationID,
		Namespace:       &namespace,
		PayloadType:     &payloadType,
		PayloadBinary:   msg,
	})
}

func (c *Conn) SendStringMessage(namespace string, msg string) error {
	version := cast_channel.CastMessage_CASTV2_1_0
	sourceID := "receiver-0"
	destinationID := "*"
	payloadType := cast_channel.CastMessage_STRING
	return c.SendMessage(&cast_channel.CastMessage{
		ProtocolVersion: &version,
		SourceId:        &sourceID,
		DestinationId:   &destinationID,
		Namespace:       &namespace,
		PayloadType:     &payloadType,
		PayloadUtf8:     &msg,
	})
}
