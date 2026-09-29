// SaaS：云端控制面（HTTPS REST + WSS）。
//
// 分层：
//
//	router       → Gin 路由注册与依赖组装
//	handle       → HTTP / WS 入口（薄）
//	logic        → 设备、指令、出口绑定、Gateway 上行、UI 推送
//	gatewayhub   → Gateway 长连接表与指令等待 channel
//	store        → 设备 / 指令 / 账号 / 出口池 / 能力声明（内存）
//	policy       → 下发前 Allow/Deny（动作目录 + 账号档 + 出口健康）
//	common/*     → REST/WS 包体、业务错误、线网 type、UI 事件名
package main

import (
	"log"
	"time"

	"github.com/tradehub/saas-service/internal/common/uievent"
	"github.com/tradehub/saas-service/internal/config"
	"github.com/tradehub/saas-service/internal/gatewayhub"
	"github.com/tradehub/saas-service/internal/logic"
	"github.com/tradehub/saas-service/internal/policy"
	"github.com/tradehub/saas-service/internal/router"
	"github.com/tradehub/saas-service/internal/store"
)

func main() {
	cfg := config.ParseFlags()
	if err := config.ValidateTLSMode(cfg.Mode, false); err != nil {
		log.Fatal(err)
	}

	// —— 进程内状态（后续可换成 DB/Redis，接口保持注入方式）——
	st := store.New()                   // 设备视图：在线、网关归属、心跳
	cmds := store.NewCommandStore()     // 异步指令生命周期（accepted → 终态）
	accounts := store.NewAccountStore() // 聊天账号：绑定设备、出口、tier 风控档
	egress := store.NewEgressStore()    // SaaS 代理池：线路配置与健康标记（业务出口 IP）
	caps := store.NewCapabilityStore()  // 板子上报的可执行 action 快照（capability）
	pol := policy.New()                 // 发令前门禁：未知动作 / 死号 / 敏感动作要健康出口等

	hub := gatewayhub.New() // Gateway WSS：按 gateway_id 下发 command、等 command_result
	ui := logic.NewUIHub()  // 浏览器 /ws/ui：推 device_updated / command_result / account_risk

	engine := router.NewEngine(router.Deps{
		Store:    st,
		Commands: cmds,
		Hub:      hub,
		UI:       ui,
		Accounts: accounts,
		Egress:   egress,
		Caps:     caps,
		Policy:   pol,
	})

	// 心跳超时扫描：只改设备 store 并推 UI，不经 Gateway
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
