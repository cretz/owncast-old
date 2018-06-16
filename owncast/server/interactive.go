package server

import (
	"bufio"
	"fmt"
	"os"

	"github.com/cretz/owncast/owncast/log"
)

type UserInput interface {
	Printfln(connIndex int, format string, v ...interface{})
	Askfln(connIndex int, format string, v ...interface{}) (string, error)
}

type stdioUserInput struct{ stdin *bufio.Reader }

func (*stdioUserInput) Printfln(connIndex int, format string, v ...interface{}) {
	fmt.Printf("[conn-%v] %v\n", connIndex, fmt.Sprintf(format, v...))
}

func (s *stdioUserInput) Askfln(connIndex int, format string, v ...interface{}) (string, error) {
	fmt.Printf("[conn-%v] %v", connIndex, fmt.Sprintf(format, v...))
	return s.stdin.ReadString('\n')
}

var StdioUserInput UserInput = &stdioUserInput{bufio.NewReader(os.Stdin)}

// Closes server when done
func RunServerInteractively(s *Server, input UserInput) error {
	defer s.Close()
	connIndexCounter := 0
	for {
		pconn, err := s.Accept()
		if err != nil {
			return err
		}
		connIndexCounter++
		go func(connIndex int) {
			if err := RunConnInteractively(connIndex, pconn, input); err != nil {
				input.Printfln(connIndex, "Closed connection due to error: %v", err)
			}
		}(connIndexCounter)
	}
}

// Closes conn when done
func RunConnInteractively(connIndex int, pconn *PendingConn, input UserInput) error {
	defer pconn.Close()
	conn, err := pconn.Auth()
	if err != nil {
		return fmt.Errorf("Failed auth: %v", err)
	}
	connected := false
	for {
		msg, payload, err := conn.ReceivePayload()
		if err != nil {
			return err
		}
		switch ns := msg.GetNamespace(); ns {
		case "urn:x-cast:com.google.cast.tp.heartbeat":
			if payload.Type != "PING" {
				return fmt.Errorf("Expected ping, got %v", payload)
			}
			conn.SendPayload(ns, pongPayload)
		case "urn:x-cast:com.google.cast.tp.connection":
			if payload.Type != "CONNECT" {
				log.Debugf("Ignoring unknown connection payload: %v", payload)
			} else if connected {
				return fmt.Errorf("Attempted connect while already connected")
			} else {
				connect, err := payload.ParseConnect()
				if err != nil {
					return fmt.Errorf("Unable to parse connect message: %v", err)
				}
				connected = true
				input.Printfln(connIndex, "Client connected, sender info: %v", connect.SenderInfo)
				defer conn.SendPayload(ns, closePayload)
			}
		case "urn:x-cast:com.google.cast.receiver":
			switch payload.Type {
			case "GET_APP_AVAILABILITY":
				// Just say they're all available
				req, err := payload.ParseGetAppAvailabilityRequest()
				if err != nil {
					return fmt.Errorf("Unable to parse app avail request: %v", err)
				}
				log.Debugf("Got app avail request: %v", req)
				resp := &GetAppAvailabilityResponsePayload{
					Payload:      req.Payload,
					Availability: make(map[AppID]AppAvailability, len(req.AppID)),
				}
				for _, appID := range req.AppID {
					resp.Availability[appID] = AppAvailable
				}
				conn.SendPayload(ns, resp)
			default:
				log.Debugf("Ignoring unknown receiver command: %v", payload)
			}
		default:
			log.Debugf("Ignoring unknown namespace %v", ns)
		}
	}
}

var pongPayload = &Payload{Type: "PONG"}
var closePayload = &Payload{Type: "CLOSE"}
