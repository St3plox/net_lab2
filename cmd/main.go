package main

import (
	"encoding/json"
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
		To:             "stepanenkoegorengo@gmail.com",
		Subject:        "Test Email with JPEG",
		Body:           "This is a test email with a JPEG attachment.",
		AttachFilePath: "assets/images.jpeg",
	}

	err = client.SendEmail(email)
	if err != nil {
		fmt.Println("Error sending email:", err)
	}
}
