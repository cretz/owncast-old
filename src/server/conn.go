package server

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/cretz/owncast/src/server/cast_channel"
	"github.com/golang/protobuf/proto"
)

type PendingConn struct {
	conn   net.Conn
	server *Server
}

func (s *Server) Accept() (*PendingConn, error) {
	if s.tlsListener == nil {
		return nil, fmt.Errorf("No listener")
	}
	conn, err := s.tlsListener.Accept()
	if err != nil {
		return nil, err
	}
	return &PendingConn{conn: conn, server: s}, nil
}

func (p *PendingConn) Close() error { return p.conn.Close() }

type Conn struct {
	conn   net.Conn
	server *Server
}

func (p *PendingConn) Auth() (*Conn, error) {
	conn := &Conn{p.conn, p.server}
	// Get auth message
	msg, err := conn.ReceiveMessage()
	if err != nil {
		return nil, fmt.Errorf("Failed to read cast message: %v", err)
	}
	if msg.GetProtocolVersion() != cast_channel.CastMessage_CASTV2_1_0 {
		return nil, fmt.Errorf("Invalid version: %v", msg.GetProtocolVersion())
	} else if msg.GetNamespace() != "urn:x-cast:com.google.cast.tp.deviceauth" {
		return nil, fmt.Errorf("Expected auth namespace, got: %v", msg.GetNamespace())
	} else if msg.GetPayloadType() != cast_channel.CastMessage_BINARY {
		return nil, fmt.Errorf("Expected binary payload, got: %v", msg.GetPayloadType())
	}
	var authReq cast_channel.DeviceAuthMessage
	if err = proto.Unmarshal(msg.PayloadBinary, &authReq); err != nil {
		return nil, fmt.Errorf("Unable to get auth message: %v", err)
	} else if authReq.Challenge == nil {
		return nil, fmt.Errorf("Missing challenge")
	}
	// Build auth response
	authResp := &cast_channel.DeviceAuthMessage{
		Response: &cast_channel.AuthResponse{
			ClientAuthCertificate: conn.server.authCert.DERBytes,
			SignatureAlgorithm:    authReq.Challenge.SignatureAlgorithm,
			SenderNonce:           authReq.Challenge.SenderNonce,
			HashAlgorithm:         authReq.Challenge.HashAlgorithm,
		},
	}
	for _, inter := range conn.server.intermediateCACerts {
		authResp.Response.IntermediateCertificate = append(authResp.Response.IntermediateCertificate, inter.DERBytes)
	}
	// Create hash
	var hash crypto.Hash
	switch authReq.Challenge.GetHashAlgorithm() {
	case cast_channel.HashAlgorithm_SHA1:
		hash = crypto.SHA1
	case cast_channel.HashAlgorithm_SHA256:
		hash = crypto.SHA256
	default:
		return nil, fmt.Errorf("Unrecognized hash algorithm: %v", authReq.Challenge.GetHashAlgorithm())
	}
	toSign := make([]byte, 0, len(authReq.Challenge.SenderNonce)+len(conn.server.peerCert.DERBytes))
	toSign = append(toSign, authReq.Challenge.SenderNonce...)
	toSign = append(toSign, conn.server.peerCert.DERBytes...)
	hasher := hash.New()
	if _, err = hasher.Write(toSign); err != nil {
		return nil, fmt.Errorf("Failed hashing: %v", err)
	}
	hashed := hasher.Sum(nil)
	// Do the signature
	switch authReq.Challenge.GetSignatureAlgorithm() {
	case cast_channel.SignatureAlgorithm_RSASSA_PKCS1v15:
		authResp.Response.Signature, err = rsa.SignPKCS1v15(rand.Reader, conn.server.authCert.PrivKey, hash, hashed)
		if err != nil {
			return nil, fmt.Errorf("Failed signing: %v", err)
		}
	case cast_channel.SignatureAlgorithm_RSASSA_PSS:
		authResp.Response.Signature, err = rsa.SignPSS(rand.Reader, conn.server.authCert.PrivKey, hash, hashed, nil)
		if err != nil {
			return nil, fmt.Errorf("Failed signing: %v", err)
		}
	default:
		return nil, fmt.Errorf("Unknown sig algo: %v", authReq.Challenge.GetSignatureAlgorithm())
	}
	// Send off the auth request
	if err = conn.SendProtoMessage(msg.GetNamespace(), authResp); err != nil {
		return nil, fmt.Errorf("Failed sending auth message: %v", err)
	}
	return conn, nil
}

func (c *Conn) ReceiveMessage() (*cast_channel.CastMessage, error) {
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
	return &msg, nil
}

func (c *Conn) SendMessage(msg *cast_channel.CastMessage) error {
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
	destinationID := "sender-0"
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
	destinationID := "sender-0"
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

func (c *Conn) Close() error { return c.conn.Close() }
