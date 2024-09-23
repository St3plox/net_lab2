package email

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

type POP3Client struct {
	conn   net.Conn
	reader *bufio.Reader
}

// and auth
func DialPOP3(cfg ClientCfg) (*POP3Client, error) {

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", cfg.Address, cfg.Port), 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	tlsConn := tls.Client(conn, &tls.Config{
		InsecureSkipVerify: true,
	})

	tlsConn.SetDeadline(time.Now().Add(10 * time.Second))
	err = tlsConn.Handshake()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("TLS handshake failed: %w", err)
	}

	tlsConn.SetDeadline(time.Time{})


	reader := bufio.NewReader(tlsConn)

	client := &POP3Client{
		conn:   tlsConn,
		reader: reader,
	}

	greeting, err := client.readSingleLineResponse()
	if err != nil {
		tlsConn.Close()
		return nil, fmt.Errorf("failed to read greeting: %w", err)
	}
	if !strings.HasPrefix(greeting, "+OK") {
		tlsConn.Close()
		return nil, fmt.Errorf("server returned error: %s", greeting)
	}
	log.Println("Server greeting:", greeting)


	if err := client.authenticate(cfg.UserEmail, cfg.UserPrivateKey); err != nil {
		tlsConn.Close()
		return nil, err
	}

	return client, nil
}

func (c *POP3Client) authenticate(user, pass string) error {
	if err := c.sendCommand("USER " + user); err != nil {
		return err
	}

	userResp, err := c.readSingleLineResponse()
	if err != nil {
		return fmt.Errorf("failed to read USER response: %w", err)
	}
	log.Println("USER response:", userResp)
	if !strings.HasPrefix(userResp, "+OK") {
		return fmt.Errorf("USER command failed: %s", userResp)
	}

	if err := c.sendCommand("PASS " + pass); err != nil {
		return err
	}

	passResp, err := c.readSingleLineResponse()
	if err != nil {
		return fmt.Errorf("failed to read PASS response: %w", err)
	}
	log.Println("PASS response:", passResp)
	if !strings.HasPrefix(passResp, "+OK") {
		return fmt.Errorf("PASS command failed: %s", passResp)
	}

	return nil
}

func (c *POP3Client) FetchEmails(n int) error {


	statResp, err := c.sendAndReadSingleLine("STAT")
	if err != nil {
		return fmt.Errorf("failed to execute STAT command: %w", err)
	}
	if !strings.HasPrefix(statResp, "+OK") {
		return fmt.Errorf("STAT command failed: %s", statResp)
	}

	parts := strings.Split(statResp, " ")
	if len(parts) < 2 {
		return fmt.Errorf("unexpected STAT response: %s", statResp)
	}

	totalMessages := 0
	_, err = fmt.Sscanf(parts[1], "%d", &totalMessages)
	if err != nil {
		return fmt.Errorf("failed to parse number of messages: %w", err)
	}

	if totalMessages == 0 {
		fmt.Println("No emails to retrieve.")
		return nil
	}

	start := totalMessages - n + 1
	if start < 1 {
		start = 1
	}

	// Fetch each of the last 'n' emails.
	for id := start; id <= totalMessages; id++ {
		if err := c.fetchEmail(id); err != nil {
			fmt.Printf("Error fetching email %d: %v\n", id, err)
		}
	}

	return nil
}

// fetchEmail retrieves and prints the content of a specific email by its ID.
func (c *POP3Client) fetchEmail(id int) error {
	cmd := fmt.Sprintf("RETR %d", id)
	resp, err := c.sendAndReadSingleLine(cmd)
	if err != nil {
		return fmt.Errorf("failed to execute %s command: %w", cmd, err)
	}
	log.Printf("%s response: %s", cmd, resp)
	if !strings.HasPrefix(resp, "+OK") {
		return fmt.Errorf("%s command failed: %s", cmd, resp)
	}

	// Read multi-line email content.
	emailContent, err := c.readMultiLineResponse()
	if err != nil {
		return fmt.Errorf("error reading %s response: %w", cmd, err)
	}

	// Print the email content.
	log.Printf("Email %d content:\n%s\n", id, emailContent)

	return nil
}

// Close gracefully closes the POP3 connection by sending the QUIT command.
func (c *POP3Client) Close() error {
	resp, err := c.sendAndReadSingleLine("QUIT")
	if err != nil {
		return fmt.Errorf("failed to execute QUIT command: %w", err)
	}

	log.Println("QUIT response:", resp)
	return c.conn.Close()
}

// sendCommand sends a command to the POP3 server.
func (c *POP3Client) sendCommand(command string) error {
	log.Printf("Executing: %s", command)
	_, err := c.conn.Write([]byte(command + "\r\n"))
	if err != nil {
		return fmt.Errorf("failed to send command '%s': %w", command, err)
	}
	log.Println("Sent command")
	return nil
}

// sendAndReadSingleLine sends a command and reads a single-line response.
func (c *POP3Client) sendAndReadSingleLine(command string) (string, error) {
	if err := c.sendCommand(command); err != nil {
		return "", err
	}
	return c.readSingleLineResponse()
}

// readSingleLineResponse reads a single line response from the server.
func (c *POP3Client) readSingleLineResponse() (string, error) {
	response, err := c.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	return strings.TrimRight(response, "\r\n"), nil
}

// readMultiLineResponse reads a multi-line response from the server, terminated by a single dot.
func (c *POP3Client) readMultiLineResponse() (string, error) {
	var lines []string
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read multi-line response: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "." {
			break
		}
		// Handle dot-stuffing by removing an extra dot at the beginning.
		if strings.HasPrefix(line, "..") {
			line = line[1:]
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n"), nil
}
