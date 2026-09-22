package main

import (
	"context"
	"log"
	"path/filepath"
	"time"

	"github.com/local/gateway/internal/boardserver"
	"github.com/local/gateway/internal/boardsession"
	"github.com/local/gateway/internal/cloudclient"
	"github.com/local/gateway/internal/config"
	"github.com/local/gateway/internal/identity"
)

func main() {
	cfg := config.ParseFlags()
	if err := config.ValidateTLSMode(cfg.Mode, cfg.InsecureSkipVerify); err != nil {
		log.Fatal(err)
	}
	config.WarnIfInsecure(cfg.InsecureSkipVerify)

	gwID, err := identity.LoadOrCreate(filepath.Join(cfg.DataDir, "gateway_id"))
	if err != nil {
		log.Fatal(err)
	}

	reg := boardsession.NewRegistry()
	cloud := &cloudclient.Client{
		URL:                cfg.SaasURL,
		GatewayID:          gwID,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}
	srv := &boardserver.Server{
		Listen:    cfg.Listen,
		CertFile:  cfg.CertFile,
		KeyFile:   cfg.KeyFile,
		GatewayID: gwID,
		Registry:  reg,
		Cloud:     cloud,
	}
	cloud.OnCommand = func(deviceID, cmdID, action, args string) {
		log.Printf("cloud command device=%s cmd_id=%s action=%s args=%q", deviceID, cmdID, action, args)
		if err := srv.SendCommand(deviceID, cmdID, action, args); err != nil {
			log.Printf("forward command to %s: %v", deviceID, err)
			_ = cloud.Send(cloudclient.TypeCommandResult, map[string]any{
				"device_id": deviceID,
				"cmd_id":    cmdID,
				"ok":        false,
				"message":   "device unreachable",
			})
			return
		}
		log.Printf("forwarded command to board %s", deviceID)
	}

	go func() {
		for {
			ctx := context.Background()
			if err := cloud.Connect(ctx); err != nil {
				log.Printf("cloud dial: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}
			err := cloud.RunRead(ctx)
			log.Printf("cloud read ended: %v", err)
			cloud.Close()
			time.Sleep(2 * time.Second)
		}
	}()

	log.Fatal(srv.ListenAndServe())
}
