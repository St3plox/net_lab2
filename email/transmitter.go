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
	"path/filepath"
	"strings"
	"time"
)

type SMTPClient struct {
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
	To             string
	Subject        string
	Body           string
	AttachFilePath string // Path to the JPEG file to attach
}

func Dial(cfg ClientCfg) (*SMTPClient, error) {
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

	return &SMTPClient{conn: conn, fromEmail: cfg.UserEmail}, nil
}

func (c *SMTPClient) Close() error {
	if err := sendCommand(c.conn, "QUIT"); err != nil {
		return err
	}
	return c.conn.Close()
}

func (c *SMTPClient) SendEmail(email Email) error {

	if err := sendCommand(c.conn, "MAIL FROM:<"+c.fromEmail+">"); err != nil {
		return err
	}

	if err := sendCommand(c.conn, "RCPT TO:<"+email.To+">"); err != nil {
		return err
	}

	if err := sendCommand(c.conn, "DATA"); err != nil {
		return err
	}

	message, err := c.buildMIMEMessage(email)
	if err != nil {
		return err
	}

	// Send MIME message
	if err := sendCommand(c.conn, message); err != nil {
		return err
	}

	return nil
}

func (c *SMTPClient) buildMIMEMessage(email Email) (string, error) {
	boundary := "boundary"

	var message strings.Builder

	// Headers
	message.WriteString("From: " + c.fromEmail + "\r\n")
	message.WriteString("To: " + email.To + "\r\n")
	message.WriteString("Subject: " + email.Subject + "\r\n")
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString("Content-Type: multipart/mixed; boundary=" + boundary + "\r\n")

	// Prepare the body first to calculate its length
	body := fmt.Sprintf("--%s\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n", boundary)
	body += "Content-Disposition: inline\r\n\r\n"
	body += email.Body + "\r\n\r\n"
	message.WriteString(body)

	message.WriteString(fmt.Sprintf("--%s\r\nContent-Type: image/jpeg\r\n", boundary))
	message.WriteString("Content-Transfer-Encoding: base64\r\n")
	message.WriteString("Content-Disposition: attachment; filename=\"" + filepath.Base(email.AttachFilePath) + "\"\r\n\r\n")

	// Read and encode the JPEG image as base64 with proper line wrapping
	err := attachJPEG(&message, email.AttachFilePath)
	if err != nil {
		return "", fmt.Errorf("error attaching jpeg: %w", err)
	}

	message.WriteString("\n\r\n\r")
	// End boundary
	message.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	message.WriteString("\r\n" + ".")

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

	if err := sendCommand(conn, ""); err != nil {
		return nil, err
	}

	if err := sendCommand(conn, "EHLO gsmtp"); err != nil {
		return nil, err
	}

	if err := sendCommand(conn, "STARTTLS"); err != nil {
		return nil, err
	}

	tlsConn := tls.Client(conn, &tls.Config{
		InsecureSkipVerify: true,
	})

	if err := sendCommand(tlsConn, "AUTH LOGIN"); err != nil {
		return nil, err
	}

	encodedUsername := base64.StdEncoding.EncodeToString([]byte(cfg.UserEmail))
	if err := sendCommand(tlsConn, encodedUsername); err != nil {
		return nil, err
	}

	encodedPassword := base64.StdEncoding.EncodeToString([]byte(cfg.UserPrivateKey))
	if err := sendCommand(tlsConn, encodedPassword); err != nil {
		return nil, err
	}

	return tlsConn, nil
}

func sendCommand(conn net.Conn, command string) error {
	log.Println("Executing: ", command+"\r\n")

	_, err := conn.Write([]byte(command + "\r\n"))
	if err != nil {
		return err
	}

	log.Println("sent command")

	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return err
	}
	log.Println("Server response:", response)
	return nil
}
