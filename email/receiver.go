package email

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
)

type POP3Client struct {
	conn net.Conn
}

func DialPOP3(cfg ClientCfg) (*POP3Client, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", cfg.Address, cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	tlsConn := tls.Client(conn, &tls.Config{
		InsecureSkipVerify: true,
	})

	if err := sendCommand(tlsConn, "USER "+cfg.UserEmail); err != nil {
		return nil, err
	}

	if err := sendCommand(tlsConn, "PASS "+cfg.UserPrivateKey); err != nil {
		return nil, err
	}

	return &POP3Client{tlsConn}, nil
}

func (c *POP3Client) FetchLastFiveEmails() error {
	if err := sendIMAPCommand(c.conn, "LIST"); err != nil {
		return err
	}

	var emailIDs []string
	for {
		response, err := readResponse(c.conn)
		if err != nil {
			return err
		}
		if response == ".\r\n" {
			break
		}
		if strings.Contains(response, " ") {
			parts := strings.Split(response, " ")
			if len(parts) > 1 {
				emailIDs = append(emailIDs, parts[0])
			}
		}
	}
	
	start := len(emailIDs) - 5
	if start < 0 {
		start = 0
	}
	lastFiveIDs := emailIDs[start:]

	// Fetch the last 5 emails
	for _, id := range lastFiveIDs {
		if err := sendIMAPCommand(c.conn, "RETR "+id); err != nil {
			return err
		}
	}

	return nil
}

func (c *POP3Client) Close() error {
	return sendCommand(c.conn, "QUIT")
}

func readResponse(conn net.Conn) (string, error) {
	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return "", err
	}
	return response, nil
}

func sendIMAPCommand(conn net.Conn, command string) error {
	_, err := conn.Write([]byte(command + "\r\n"))
	return err
}
