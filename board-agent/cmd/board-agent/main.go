// Board-Agent：板端代理，连接本地 Gateway。
//
// 分层：
//
//	handle  → TLS 连接 Gateway、收发帧
//	logic   → 指令执行（echo 等）
//	frame   → 二进制帧编解码
//	identity / netinfo / config
package main

import (
	"log"

	"github.com/tradehub/board-agent/internal/config"
	"github.com/tradehub/board-agent/internal/handle"
)

func main() {
	cfg := config.ParseFlags()
	if err := config.ValidateTLSMode(cfg.Mode, cfg.InsecureSkipVerify); err != nil {
		log.Fatal(err)
	}
	config.WarnIfInsecure(cfg.InsecureSkipVerify)

	a, err := handle.NewAgent(cfg)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("board-agent device_id=%s", a.DeviceID)
	log.Fatal(a.Run())
}
