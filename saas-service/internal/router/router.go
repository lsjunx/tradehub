// Package router 注册 Gin 路由并组装 handle/logic 依赖。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/local/saas-service/internal/gatewayhub"
	"github.com/local/saas-service/internal/handle"
	"github.com/local/saas-service/internal/logic"
	"github.com/local/saas-service/internal/policy"
	"github.com/local/saas-service/internal/store"
)

// Deps 路由层依赖。
type Deps struct {
	Store    *store.Store
	Commands *store.CommandStore
	Hub      *gatewayhub.Hub
	UI       *logic.UIHub
	Accounts *store.AccountStore
	Egress   *store.EgressStore
	Caps     *store.CapabilityStore
	Policy   *policy.Policy
}

// NewEngine 组装 gin 引擎。
//
// REST：统一 {code,message,data,timestamp}
//
//	GET  /api/devices
//	GET  /api/devices/:id
//	POST /api/devices/:id/commands  → data={cmd_id,status:accepted}
//	GET  /api/commands/:cmd_id
//	POST /api/egress
//	GET  /api/egress
//	POST /api/accounts
//	GET  /api/accounts
//	POST /api/accounts/:id/egress
//
// WS：
//
//	/ws/gateway ← Gateway（type 见 proto/cloud MsgType）
//	/ws/ui      ← 浏览器（uievent.DeviceUpdated | CommandResult）
func NewEngine(d Deps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	deviceLogic := logic.NewDeviceLogic(d.Store, d.Hub, d.Commands, d.Accounts, d.Egress, d.Policy)
	egressLogic := logic.NewEgressLogic(d.Store, d.Hub, d.Accounts, d.Egress)
	gatewayLogic := logic.NewGatewayLogic(d.Store, d.Hub, d.UI, d.Caps, d.Accounts, d.Egress)

	deviceH := handle.NewDeviceHandle(deviceLogic)
	egressH := handle.NewEgressHandle(egressLogic)
	webH := handle.NewWebHandle()
	gwWSH := handle.NewGatewayWSHandle(gatewayLogic, d.Hub)
	uiWSH := handle.NewUIWSHandle(d.UI)

	r.GET("/", webH.Index)
	r.GET("/api/devices", deviceH.ListDevices)
	r.GET("/api/devices/:id", deviceH.GetDevice)
	r.POST("/api/devices/:id/commands", deviceH.SendCommand)
	r.GET("/api/commands/:cmd_id", deviceH.GetCommand)
	r.POST("/api/egress", egressH.Upsert)
	r.GET("/api/egress", egressH.ListEgress)
	r.POST("/api/accounts", egressH.UpsertAccount)
	r.GET("/api/accounts", egressH.ListAccounts)
	r.POST("/api/accounts/:id/egress", egressH.BindEgress)
	r.GET("/ws/gateway", gwWSH.Serve)
	r.GET("/ws/ui", uiWSH.Serve)

	return r
}
