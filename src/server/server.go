package server

import (
	"crypto/tls"
	"fmt"
	"net"

	"github.com/cretz/owncast/src/cert"
	"github.com/grandcat/zeroconf"
)

type Server struct {
	intermediateCACerts       []*cert.KeyPair
	peerCert                  *cert.KeyPair
	authCert                  *cert.KeyPair
	tlsListener               net.Listener
	tlsListenerCloseOnClose   bool
	mdnsServer                *zeroconf.Server
	mdnsServerShutdownOnClose bool
}

// Just a random v4 uuid I gen'd and then removed dashes
// fb40bc4b-1ef8-4e97-839f-b4a9cf8e5c10
const DefaultID = "fb40bc4b1ef84e97839fb4a9cf8e5c10"

type ServerConf struct {
	// Used and must be present if IntermediateCACerts is nil/empty
	RootCACert *cert.KeyPair
	// If nil/empty, one is generated. The peer and auth certs are created from the last one if present.
	IntermediateCACerts []*cert.KeyPair
	// If empty, generated from last/created intermediate
	PeerCert *cert.KeyPair
	// If empty, generated from last/created intermediate
	AuthCert *cert.KeyPair

	// If empty, it is "tcp"
	TLSListenNetwork string

	// If empty, it is ":0"
	TLSListenAddr string

	// If empty, it is created with other data. If present, it will not be closed on close.
	TLSListenerOverride net.Listener

	// If empty, is "OwnCast"
	BroadcastInstanceName string
	// If empty, is "OwnCast"
	BroadcastFriendlyName string
	// If any value here is empty, it is considered a delete
	BroadcastTextOverrides map[string]string
	// If empty, uses all
	BroadcastIfaces []net.Interface

	// If empty, it is created with other data above. If present, it will not be shutdown on close.
	BroadcastServerOverride *zeroconf.Server

	// If empty, uses DefaultID
	ID string
}

func Listen(conf *ServerConf) (*Server, error) {
	s := &Server{
		intermediateCACerts: conf.IntermediateCACerts,
		peerCert:            conf.PeerCert,
		authCert:            conf.AuthCert,
		tlsListener:         conf.TLSListenerOverride,
		mdnsServer:          conf.BroadcastServerOverride,
	}
	// Create the intermediate cert if necessary
	if len(s.intermediateCACerts) == 0 {
		if conf.RootCACert == nil {
			return nil, fmt.Errorf("RootCACert is required if IntermediateCACerts is empty")
		}
		interCert, err := cert.GenerateIntermediateCAKeyPair(conf.RootCACert, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("Unable to generate intermediate cert: %v", err)
		}
		s.intermediateCACerts = []*cert.KeyPair{interCert}
	}
	lastInterCert := s.intermediateCACerts[len(s.intermediateCACerts)-1]
	// Generate the peer and auth certs if not present
	var err error
	if s.peerCert == nil {
		if s.peerCert, err = cert.GenerateStandardKeyPair(lastInterCert, nil, nil); err != nil {
			return nil, fmt.Errorf("Unable to create peer cert: %v", err)
		}
	}
	if s.authCert == nil {
		if s.authCert, err = cert.GenerateStandardKeyPair(lastInterCert, nil, nil); err != nil {
			return nil, fmt.Errorf("Unable to create auth cert: %v", err)
		}
	}
	// Create TLS listener if not present
	// NOTE: from here on out, we must close the server on failure, not exit early
	if s.tlsListener == nil {
		s.tlsListenerCloseOnClose = true
		tlsNet := conf.TLSListenNetwork
		if tlsNet == "" {
			tlsNet = "tcp"
		}
		tlsAddr := conf.TLSListenAddr
		if tlsAddr == "" {
			tlsAddr = ":0"
		}
		var tlsCert tls.Certificate
		if tlsCert, err = s.peerCert.CreateTLSCertificate(); err == nil {
			s.tlsListener, err = tls.Listen(tlsNet, tlsAddr, &tls.Config{Certificates: []tls.Certificate{tlsCert}})
		}
	}
	// Obtain the port
	port := -1
	if err == nil {
		if addr, ok := s.tlsListener.Addr().(*net.TCPAddr); ok {
			port = addr.Port
		} else {
			err = fmt.Errorf("TLS listener addr is not TCP")
		}
	}
	// Start mdns
	if err == nil && s.mdnsServer == nil {
		s.mdnsServerShutdownOnClose = true
		// Build mdns text
		id := conf.ID
		if id == "" {
			id = DefaultID
		}
		broadcastTextMap := DefaultBroadcastText(id)
		if conf.BroadcastFriendlyName != "" {
			broadcastTextMap["fn"] = conf.BroadcastFriendlyName
		}
		for k, v := range conf.BroadcastTextOverrides {
			if v == "" {
				delete(broadcastTextMap, k)
			} else {
				broadcastTextMap[k] = v
			}
		}
		broadcastText := []string{}
		for k, v := range broadcastTextMap {
			broadcastText = append(broadcastText, k+"="+v)
		}
		// Start the server
		// return zeroconf.Register("TestCase", "_googlecast._tcp", "local.", port, text, nil)
		instName := conf.BroadcastInstanceName
		if instName == "" {
			instName = "OwnCast"
		}
		s.mdnsServer, err = zeroconf.Register(instName, "_googlecast._tcp", "local.", port,
			broadcastText, conf.BroadcastIfaces)
	}
	// If there is an error, close it all
	if err != nil {
		if closeErr := s.Close(); closeErr != nil {
			err = fmt.Errorf("Error on start: %v (also got error trying to close: %v)", err, closeErr)
		}
		return nil, err
	}
	return s, nil
}

func DefaultBroadcastText(id string) map[string]string {
	return map[string]string{
		"id": id,
		"ve": "02",
		"md": "Chromecast",
		"fn": "Owncast",
		"ca": "5",
		"st": "0",
		"rs": "",
		"ic": "/setup/icon.png",
	}
}

func (s *Server) Close() (err error) {
	if s.mdnsServerShutdownOnClose && s.mdnsServer != nil {
		s.mdnsServer.Shutdown()
		s.mdnsServer = nil
	}
	if s.tlsListenerCloseOnClose && s.tlsListener != nil {
		err = s.tlsListener.Close()
		s.tlsListener = nil
	}
	return
}
