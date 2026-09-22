package main

import (
	"log"
	"time"

	"github.com/local/saas-service/internal/config"
	"github.com/local/saas-service/internal/gatewayhub"
	"github.com/local/saas-service/internal/logic"
	"github.com/local/saas-service/internal/router"
	"github.com/local/saas-service/internal/store"
)

func main() {
	cfg := config.ParseFlags()
	if err := config.ValidateTLSMode(cfg.Mode, false); err != nil {
		log.Fatal(err)
	}

	st := store.New()
	cmds := store.NewCommandStore()
	hub := gatewayhub.New()
	ui := logic.NewUIHub()
	engine := router.NewEngine(router.Deps{Store: st, Commands: cmds, Hub: hub, UI: ui})

	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for range t.C {
			for _, d := range st.MarkStaleOffline(30 * time.Second) {
				ui.Broadcast("device_updated", d)
			}
		}
	}()

	log.Printf("saas HTTPS on %s", cfg.Addr)
	log.Fatal(engine.RunTLS(cfg.Addr, cfg.CertFile, cfg.KeyFile))
}
