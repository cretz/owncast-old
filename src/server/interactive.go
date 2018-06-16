package server

import (
	"bufio"
	"fmt"
	"os"

	"github.com/golang/protobuf/proto"
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
				input.Printfln(connIndex, "ERROR: %v", err)
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
	for {
		msg, err := conn.ReceiveMessage()
		if err != nil {
			return err
		}
		input.Printfln(connIndex, "READ: %v", proto.MarshalTextString(msg))
	}
}
