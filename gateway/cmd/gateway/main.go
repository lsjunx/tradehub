// Gateway：本地边车，透传 Board↔SaaS。
//
// 分层：
//
//	handle       → Board TLS 接入与读帧（I/O 边界）
//	logic        → 上行 Board→SaaS、下行 SaaS→Board 转发
//	common/cloud · session · frame · identity → SaaS WSS 客户端、Board 会话表、帧编解码、gateway_id
//	config       → 启动参数与 TLS 模式
package main

import (
	"context"
	"gateway/internal/common/cloud"
	"gateway/internal/common/identity"
	"gateway/internal/common/session"
	"log"
	"path/filepath"
	"time"

	"gateway/internal/config"
	"gateway/internal/handle"
	"gateway/internal/logic"
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

	reg := session.NewRegistry()
	cloudCli := &cloud.Client{
		URL:                cfg.SaasURL,
		GatewayID:          gwID,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}
	bridge := &logic.Bridge{
		GatewayID: gwID,
		Registry:  reg,
		Cloud:     cloudCli,
	}
	// 【关键】SaaS 下行 command → logic 转发 Board
	cloudCli.OnCommand = bridge.HandleCloudCommand

	boardSrv := &handle.BoardServer{
		Listen:   cfg.Listen,
		CertFile: cfg.CertFile,
		KeyFile:  cfg.KeyFile,
		Bridge:   bridge,
	}

	// 后台维持与 SaaS 的长连接（断线重连）
	go func() {
		for {
			ctx := context.Background()
			if err := cloudCli.Connect(ctx); err != nil {
				log.Printf("连接 SaaS 失败: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}
			// 【关键】阻塞读 SaaS；仅处理 command
			err := cloudCli.RunRead(ctx)
			log.Printf("SaaS 读循环结束: %v", err)
			cloudCli.Close()
			time.Sleep(2 * time.Second)
		}
	}()

	// 【关键】阻塞服务 Board TLS
	log.Fatal(boardSrv.ListenAndServe())
}
