package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
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
		panic(fmt.Errorf("error parsing config file: %w", err))
	}

	err = json.Unmarshal(cfgFile, &cfg)
	if err != nil {
		panic(fmt.Errorf("error unmarshal: %w", err))
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", server, port))
	if err != nil {
		fmt.Println("Connection failed:", err)
		return
	}
	defer conn.Close()

	resp, err := conn.Write([]byte("HELLO localhost"))
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(resp)
}
