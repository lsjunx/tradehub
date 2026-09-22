package main

import (
	"log"
	"net/http"
	"time"

	"github.com/local/saas-service/internal/api"
	"github.com/local/saas-service/internal/config"
	"github.com/local/saas-service/internal/gatewayhub"
	"github.com/local/saas-service/internal/store"
)

func main() {
	cfg := config.ParseFlags()
	// saas 自身是 TLS 服务端，不使用 insecure_skip_verify；仍校验 mode 合法
	if err := config.ValidateTLSMode(cfg.Mode, false); err != nil {
		log.Fatal(err)
	}

	st := store.New()
	hub := gatewayhub.New()
	srv := api.NewServer(st, hub)

	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for range t.C {
			for _, d := range st.MarkStaleOffline(30 * time.Second) {
				hub.Emit("device_updated", d)
			}
		}
	}()

	log.Printf("saas HTTPS on %s", cfg.Addr)
	log.Fatal(http.ListenAndServeTLS(cfg.Addr, cfg.CertFile, cfg.KeyFile, srv.Routes()))
}
