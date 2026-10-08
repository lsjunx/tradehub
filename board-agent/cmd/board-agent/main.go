// Board-Agent：板端代理，连接本地 Gateway。
//
// 分层：
//
//	handle       → TLS 连接 Gateway、收发帧（I/O 边界）
//	logic        → 指令执行 / Plugin Router（echo、egress.*）
//	common/frame · identity · netinfo · egress → 帧编解码、设备 ID、网卡信息、本机出口缓存
//	config       → 启动参数与 TLS 模式
package main

import (
	"log"

	"board-agent/internal/config"
	"board-agent/internal/handle"
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
