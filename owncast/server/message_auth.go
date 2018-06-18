package server

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"fmt"

	"github.com/cretz/owncast/owncast/log"
	"github.com/cretz/owncast/owncast/server/cast_channel"
	"github.com/golang/protobuf/proto"
)

type DeviceAuthMessage struct {
	cast_channel.DeviceAuthMessage
	castMessage *cast_channel.CastMessage
}

func NewDeviceAuthMessage(castMessage *cast_channel.CastMessage) (*DeviceAuthMessage, error) {
	ret := &DeviceAuthMessage{castMessage: castMessage}
	if err := proto.Unmarshal(castMessage.PayloadBinary, &ret.DeviceAuthMessage); err != nil {
		return nil, fmt.Errorf("Unable to get auth message: %v", err)
	} else if ret.Challenge == nil {
		return nil, fmt.Errorf("Missing challenge")
	}
	log.Debugf("Received auth request: %v", &ret.DeviceAuthMessage)
	return ret, nil
}

func (d *DeviceAuthMessage) CastMessage() *cast_channel.CastMessage { return d.castMessage }

func (d *DeviceAuthMessage) HandleDefault(conn *Conn) error {
	// Build auth response
	authResp := &cast_channel.DeviceAuthMessage{
		Response: &cast_channel.AuthResponse{
			ClientAuthCertificate: conn.server.authCert.DERBytes,
			SignatureAlgorithm:    d.Challenge.SignatureAlgorithm,
			SenderNonce:           d.Challenge.SenderNonce,
			HashAlgorithm:         d.Challenge.HashAlgorithm,
		},
	}
	for _, inter := range conn.server.intermediateCACerts {
		authResp.Response.IntermediateCertificate = append(authResp.Response.IntermediateCertificate, inter.DERBytes)
	}
	// Create hash
	var hash crypto.Hash
	switch d.Challenge.GetHashAlgorithm() {
	case cast_channel.HashAlgorithm_SHA1:
		hash = crypto.SHA1
	case cast_channel.HashAlgorithm_SHA256:
		hash = crypto.SHA256
	default:
		return fmt.Errorf("Unrecognized hash algorithm: %v", d.Challenge.GetHashAlgorithm())
	}
	toSign := make([]byte, 0, len(d.Challenge.SenderNonce)+len(conn.server.peerCert.DERBytes))
	toSign = append(toSign, d.Challenge.SenderNonce...)
	toSign = append(toSign, conn.server.peerCert.DERBytes...)
	hasher := hash.New()
	if _, err := hasher.Write(toSign); err != nil {
		return fmt.Errorf("Failed hashing: %v", err)
	}
	hashed := hasher.Sum(nil)
	// Do the signature
	var err error
	switch d.Challenge.GetSignatureAlgorithm() {
	case cast_channel.SignatureAlgorithm_RSASSA_PKCS1v15:
		authResp.Response.Signature, err = rsa.SignPKCS1v15(rand.Reader, conn.server.authCert.PrivKey, hash, hashed)
		if err != nil {
			return fmt.Errorf("Failed signing: %v", err)
		}
	case cast_channel.SignatureAlgorithm_RSASSA_PSS:
		authResp.Response.Signature, err = rsa.SignPSS(rand.Reader, conn.server.authCert.PrivKey, hash, hashed, nil)
		if err != nil {
			return fmt.Errorf("Failed signing: %v", err)
		}
	default:
		return fmt.Errorf("Unknown sig algo: %v", d.Challenge.GetSignatureAlgorithm())
	}
	// Send off the auth request
	log.Debugf("Sending auth response: %v", authResp)
	if err = conn.SendProtoMessage(d.castMessage.GetNamespace(), authResp); err != nil {
		return fmt.Errorf("Failed sending auth message: %v", err)
	}
	conn.Authenticated = true
	return nil
}
