// SaaS：云端控制面（HTTPS REST + WSS）。
//
// 分层：
//
//	router       → Gin 路由注册
//	handle       → HTTP/WS 入口
//	logic        → 设备/指令/Gateway 上行业务
//	gatewayhub   → Gateway 连接与指令等待
//	store        → 设备与指令内存表
//	common/response · errors · uievent → 公共包体与事件名
package main

import (
	"log"
	"time"

	"github.com/local/saas-service/internal/common/uievent"
	"github.com/local/saas-service/internal/config"
	"github.com/local/saas-service/internal/gatewayhub"
	"github.com/local/saas-service/internal/logic"
	"github.com/local/saas-service/internal/policy"
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
	accounts := store.NewAccountStore()
	egress := store.NewEgressStore()
	caps := store.NewCapabilityStore()
	pol := policy.New()
	hub := gatewayhub.New()
	ui := logic.NewUIHub()
	engine := router.NewEngine(router.Deps{
		Store: st, Commands: cmds, Hub: hub, UI: ui,
		Accounts: accounts, Egress: egress, Caps: caps, Policy: pol,
	})

	// 心跳超时扫描：直接改 store 并推 UI（不经 Gateway）
	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for range t.C {
			for _, d := range st.MarkStaleOffline(30 * time.Second) {
				ui.Broadcast(uievent.DeviceUpdated, d)
			}
		}
	}()

	log.Printf("saas HTTPS on %s", cfg.Addr)
	log.Fatal(engine.RunTLS(cfg.Addr, cfg.CertFile, cfg.KeyFile))
}
