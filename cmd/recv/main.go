package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/St3plox/net_lab2/email"
)

const (
	userCfgPath = "cfg/user.json"
	server      = "pop.gmail.com"
	port        = 995
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

	client, err := email.DialPOP3(mailCfg)
	if err != nil {
		fmt.Println("Error during connection setup:", err)
		return
	}
	defer client.Close()

	err = client.FetchLastFiveEmails()
	if err != nil {
		fmt.Println("Error fetching emails:", err)
	}
}
