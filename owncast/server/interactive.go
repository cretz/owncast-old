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
func RunConnInteractively(connIndex int, conn *Conn, input UserInput) error {
	defer conn.Close()
	defer func() {
		if conn.Connected {
			conn.SendPayload("urn:x-cast:com.google.cast.tp.connection", closePayload)
		}
	}()
	for {
		msg, err := conn.ReceiveMessage()
		if err != nil {
			return fmt.Errorf("Unable to parse message: %v", err)
		}
		switch msg := msg.(type) {
		case MessageWithHandleDefault:
			if err = msg.HandleDefault(conn); err != nil {
				return fmt.Errorf("Failed handling message: %v", err)
			}
		default:
			// TODO: interactive responses
			log.Debugf("Ignoring unknown message: %v", msg)
		}
	}
}

var closePayload = &Payload{Type: "CLOSE"}
