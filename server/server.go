package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/nikochiko/smtp-go/common"
	"github.com/nikochiko/smtp-go/smtpstatus"
)

const version = "0.0.1"

var (
	welcomeMessage        = fmt.Sprintf("%d %%s nikochiko/smtp-go server %s welcomes you 🙏", smtpstatus.WelcomeOK, version)
	helloOKMessage        = fmt.Sprintf("%d %%s Hello %%s, this is nikochiko/smtp-go. Pleased to meet you!", smtpstatus.ReplyOK) // taking inspiration from Sendmail
	statusOKMessage	= fmt.Sprintf("%d OK", smtpstatus.ReplyOK)
	badSequenceMessage    = fmt.Sprintf("%d bad sequence of commands", smtpstatus.BadSequence)
	unknownCommandMessage = fmt.Sprintf(`%d unknown command: "%%s"`, smtpstatus.UnknownCommand)
	closingMessage        = "Closing connection. Thank you for interacting with nikochiko/smtp-go server. Come back often :)"
)

type Server struct {
	Port   int
	Domain string
}

func (s *Server) ServeSMTP() error {
	service := fmt.Sprintf("%s:%d", s.Domain, s.Port)

	laddr, err := net.ResolveTCPAddr("tcp4", service)
	common.CheckError(err)

	listener, err := net.ListenTCP("tcp", laddr)
	common.CheckError(err)

	fmt.Printf("Serving SMTP on %s\n", laddr)

	// listen for eternity
	for {
		conn, err := listener.Accept()
		if err != nil {
			// we don't want to stop the server now, so just log and continue
			log.Printf("Error while accepting connection: %s", err.Error())

		}

		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) (err error) {
	defer conn.Close()

	welcomeMessage := s.getWelcomeMessage()
	writeStringWithCRLF(conn, welcomeMessage)

	smtpConn := NewSMTPConn(conn)

	for {
		// get some command with a timeout of 30 seconds
		timeout := 30 * time.Second
		input, err := smtpConn.ReadLineWithTimeout(timeout)
		if err != nil {
			err = fmt.Errorf("error while reading input from client: %s", err.Error())
			break
		}

		command := getCommand(input)
		text := getText(input)

		switch command {
		case "QUIT":
			err = smtpConn.HandleQUIT(command, text)
		case "HELO":
			err = smtpConn.HandleHELO(command, text)
		default:
			err = smtpConn.HandleUnknownCommand(command, text)
		}

		if err != nil {
			err = fmt.Errorf(`error while processing command "%s": %s`, command, err.Error())
			break
		}
	}

	if err != nil {
		log.Fatal(err)
	}

	return
}

func (s *Server) getWelcomeMessage() string {
	return fmt.Sprintf(welcomeMessage, s.Domain)
}

func writeStringWithCRLF(w io.Writer, s string) error {
	b := []byte(s + "\r\n")
	_, err := w.Write(b)
	if err != nil {
		return err
	}

	return nil
}

type StateTable struct {
	From string
	To   []string
	Data string
}

func (st *StateTable) Clear() error {
	st.From = ""
	st.To = []string{}
	st.Data = ""

	return nil
}

type SMTPConn struct {
	conn net.Conn

	ClientDomain string
	stateTable   StateTable
}

func NewSMTPConn(conn net.Conn) SMTPConn {
	return SMTPConn{conn: conn, stateTable: StateTable{}}
}

func (sconn *SMTPConn) ClearState() error {
	err := sconn.stateTable.Clear()
	if err != nil {
		return err
	}

	return nil
}

func (sconn *SMTPConn) ReadLineWithTimeout(timeout time.Duration) (string, error) {
	conn := sconn.conn

	buffer := make([]byte, 1000)

	conn.SetReadDeadline(time.Now().Add(timeout))
	length, err := conn.Read(buffer)

	if err != nil {
		return "", err
	}

	input := string(buffer[:length-2])

	return input, nil
}

func (sconn *SMTPConn) HandleHELO(_ string, text string) (err error) {
	if sconn.stateTable.From != "" {
		// write 503 bad sequence response
		err = writeStringWithCRLF(sconn.conn, badSequenceMessage)
		return
	}

	words := strings.Split(text, " ")

	// TODO: validate domain
	sconn.ClientDomain = words[0]

	err = writeStringWithCRLF(sconn.conn, statusOKMessage)

	return
}

func (sconn *SMTPConn) HandleUnknownCommand(command string, _ string) (err error) {
	messageText := fmt.Sprintf(unknownCommandMessage, command)

	err = writeStringWithCRLF(sconn.conn, messageText)

	return
}

func (sconn *SMTPConn) HandleQUIT(_ string, _ string) error {
	err := writeStringWithCRLF(sconn.conn, closingMessage)
	if err != nil {
		return err
	}

	err = errors.New("client issued QUIT command")
	return err
}

func normalizeCommand(c string) string {
	return strings.ToUpper(c)
}

func getCommand(input string) string {
	parts := strings.SplitN(input, " ", 2)
	if len(parts) >= 1 {
		return parts[0]
	}

	return ""
}

func getText(input string) string {
	parts := strings.SplitN(input, " ", 2)

	if len(parts) >= 2 {
		return parts[1]
	}

	return ""
}
