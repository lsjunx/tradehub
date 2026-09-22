package router

import (
	"github.com/gin-gonic/gin"

	"github.com/local/saas-service/internal/gatewayhub"
	"github.com/local/saas-service/internal/handle"
	"github.com/local/saas-service/internal/logic"
	"github.com/local/saas-service/internal/store"
)

// Deps 路由层依赖。
type Deps struct {
	Store    *store.Store
	Commands *store.CommandStore
	Hub      *gatewayhub.Hub
	UI       *logic.UIHub
}

// NewEngine 组装 gin 引擎与分层依赖。
//
// REST（真前端契约）：
//
//	GET  /api/devices
//	GET  /api/devices/:id
//	POST /api/devices/:id/commands  → 202 {cmd_id,status:accepted}
//	GET  /api/commands/:cmd_id
//
// WS：
//
//	/ws/ui      ← device_updated | command_result
//	/ws/gateway ← 仅 Gateway
func NewEngine(d Deps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	deviceLogic := logic.NewDeviceLogic(d.Store, d.Hub, d.Commands)
	gatewayLogic := logic.NewGatewayLogic(d.Store, d.Hub, d.UI)

	deviceH := handle.NewDeviceHandle(deviceLogic)
	webH := handle.NewWebHandle()
	gwWSH := handle.NewGatewayWSHandle(gatewayLogic, d.Hub)
	uiWSH := handle.NewUIWSHandle(d.UI)

	r.GET("/", webH.Index)
	r.GET("/api/devices", deviceH.ListDevices)
	r.GET("/api/devices/:id", deviceH.GetDevice)
	r.POST("/api/devices/:id/commands", deviceH.SendCommand)
	r.GET("/api/commands/:cmd_id", deviceH.GetCommand)
	r.GET("/ws/gateway", gwWSH.Serve)
	r.GET("/ws/ui", uiWSH.Serve)

	return r
}
