package email

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

type Client struct {
	fromEmail string
	conn      net.Conn
}

type ClientCfg struct {
	Address        string
	Port           int
	UserEmail      string
	UserPrivateKey string
}

type Email struct {
	To      string
	Subject string
	Body    string
	AttachFilePath string // Path to the JPEG file to attach
}

func Dial(cfg ClientCfg) (*Client, error) {
	// Connect to SMTP server
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", cfg.Address, cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("error establishing connection: %w", err)
	}

	// Say hello and authenticate
	conn, err = sayHelloAndAuth(conn, cfg)
	if err != nil {
		return nil, fmt.Errorf("error during handshake and auth: %w", err)
	}

	return &Client{conn: conn, fromEmail: cfg.UserEmail}, nil
}

func (c *Client) Close() error {
	if err := sendCommand(c.conn, "QUIT"); err != nil {
		return err
	}
	return c.conn.Close()
}

func (c *Client) SendEmail(email Email) error {
	// Send MAIL FROM command with the correct sender email
	if err := sendCommand(c.conn, "MAIL FROM:<"+c.fromEmail+">"); err != nil {
		return err
	}

	// Send RCPT TO command with the recipient email
	if err := sendCommand(c.conn, "RCPT TO:<"+email.To+">"); err != nil {
		return err
	}

	log.Println("Sending DATA")
	if err := sendCommand(c.conn, "DATA"); err != nil {
		return err
	}

	// Prepare MIME message with the text and JPEG attachment
	message, err := c.buildMIMEMessage(email)
	if err != nil {
		return err
	}

	// Send MIME message
	if err := sendCommand(c.conn, message); err != nil {
		return err
	}

	// End DATA command
	if err := sendCommand(c.conn, "."); err != nil {
		return err
	}


	return nil
}

func (c *Client) buildMIMEMessage(email Email) (string, error) {
	// Boundary for separating the parts in the email
	boundary := "my-boundary-123"

	// Build the MIME message
	var message strings.Builder

	// Headers
	message.WriteString("From: " + c.fromEmail + "\r\n")
	message.WriteString("To: " + email.To + "\r\n")
	message.WriteString("Subject: " + email.Subject + "\r\n")
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString("Content-Type: multipart/mixed; boundary=" + boundary + "\r\n")
	message.WriteString("\r\n")

	// Body (text part)
	message.WriteString("--" + boundary + "\r\n")
	message.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	message.WriteString("\r\n")
	message.WriteString(email.Body + "\r\n")
	message.WriteString("\r\n")

	// JPEG attachment part
	message.WriteString("--" + boundary + "\r\n")
	message.WriteString("Content-Type: image/jpeg\r\n")
	message.WriteString("Content-Transfer-Encoding: base64\r\n")
	message.WriteString("Content-Disposition: attachment; filename=\"image.jpg\"\r\n")
	message.WriteString("\r\n")

	// Read and encode the JPEG image as base64
	err := attachJPEG(&message, email.AttachFilePath)
	if err != nil {
		return "", fmt.Errorf("error attaching jpeg: %w", err)
	}

	// End boundary
	message.WriteString("\r\n--" + boundary + "--\r\n")

	return message.String(), nil
}

func attachJPEG(builder *strings.Builder, filePath string) error {
	// Open the JPEG file
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a base64 encoder and attach the image content
	base64Encoder := base64.NewEncoder(base64.StdEncoding, builder)
	defer base64Encoder.Close()

	_, err = io.Copy(base64Encoder, file)
	if err != nil {
		return err
	}

	return nil
}

func sayHelloAndAuth(conn net.Conn, cfg ClientCfg) (net.Conn, error) {
	// Read server greeting
	if err := sendCommand(conn, ""); err != nil {
		return nil, err
	}

	// Send HELO command
	if err := sendCommand(conn, "HELO localhost"); err != nil {
		return nil, err
	}

	// Start TLS
	if err := sendCommand(conn, "STARTTLS"); err != nil {
		return nil, err
	}

	// Upgrade the connection to TLS
	tlsConn := tls.Client(conn, &tls.Config{
		InsecureSkipVerify: true, // This skips certificate verification (for dev purposes only)
	})

	// Send AUTH LOGIN
	if err := sendCommand(tlsConn, "AUTH LOGIN"); err != nil {
		return nil, err
	}

	// Send base64-encoded username
	encodedUsername := base64.StdEncoding.EncodeToString([]byte(cfg.UserEmail))
	if err := sendCommand(tlsConn, encodedUsername); err != nil {
		return nil, err
	}

	// Send base64-encoded password
	encodedPassword := base64.StdEncoding.EncodeToString([]byte(cfg.UserPrivateKey))
	if err := sendCommand(tlsConn, encodedPassword); err != nil {
		return nil, err
	}

	return tlsConn, nil // Return upgraded TLS connection
}

func sendCommand(conn net.Conn, command string) error {
	log.Println("Executing:", command)
	_, err := fmt.Fprintf(conn, command+"\r\n")
	if err != nil {
		return err
	}
	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return err
	}
	log.Println("Server response:", response)
	return nil
}
