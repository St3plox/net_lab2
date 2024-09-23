package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/St3plox/net_lab2/email"
)

const (
	userCfgPath = "cfg/user.json"
	server      = "smtp.gmail.com"
	port        = 587
)

func main() {

	subject := flag.String("subject", "Test Email with JPEG", "Email subject")
	body := flag.String("body", "This is a test email with a JPEG attachment.", "Email body")
	filePath := flag.String("filepath", "assets/image.jpeg", "Path to the attachment file")
	to := flag.String("to", "sukhae83@gmail.com", "Recipient email address")

	// Parse the command-line flags
	flag.Parse()

	cfg := struct {
		UserEmail string `json:"user_email"`
		UserKey   string `json:"user_key"`
	}{}

	cfgFile, err := os.ReadFile(userCfgPath)
	if err != nil {
		panic(fmt.Errorf("error reading config file: %w", err))
	}

	err = json.Unmarshal(cfgFile, &cfg)
	if err != nil {
		panic(fmt.Errorf("error unmarshalling config file: %w", err))
	}

	mailCfg := email.ClientCfg{
		UserEmail:      cfg.UserEmail,
		UserPrivateKey: cfg.UserKey,
		Address:        server,
		Port:           port,
	}

	client, err := email.Dial(mailCfg)
	if err != nil {
		fmt.Println("Error during connection setup:", err)
		return
	}
	defer client.Close()

	email := email.Email{
		To:             *to,
		Subject:        *subject,
		Body:           *body,
		AttachFilePath: *filePath,
	}

	err = client.SendEmail(email)
	if err != nil {
		fmt.Println("Error sending email:", err)
	}
}
