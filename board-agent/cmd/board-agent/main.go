package main

import (
	"log"

	"github.com/local/board-agent/internal/agent"
	"github.com/local/board-agent/internal/config"
)

func main() {
	cfg := config.ParseFlags()
	if err := config.ValidateTLSMode(cfg.Mode, cfg.InsecureSkipVerify); err != nil {
		log.Fatal(err)
	}
	config.WarnIfInsecure(cfg.InsecureSkipVerify)

	a, err := agent.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("board-agent device_id=%s", a.DeviceID)
	log.Fatal(a.Run())
}
