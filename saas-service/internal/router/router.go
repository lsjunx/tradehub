// Package router 注册 Gin 路由并组装 handle/logic 依赖。
//
// 路由按职责分成三层（路径稳定，仅组织方式调整）：
//
//  1. 业务 API（/api/*）— 给人/前端：设备、指令、账号、代理池
//  2. 网络控制面（/ws/gateway）— 给 Gateway：设备上云、指令下发通道
//  3. UI 推送（/ws/ui）+ 演示页（/）— 给浏览器订阅事件
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/tradehub/saas-service/internal/gatewayhub"
	"github.com/tradehub/saas-service/internal/handle"
	"github.com/tradehub/saas-service/internal/logic"
	"github.com/tradehub/saas-service/internal/policy"
	"github.com/tradehub/saas-service/internal/store"
)

// Deps 由 main 注入的共享依赖（各 Logic 按需取用）。
type Deps struct {
	Store    *store.Store           // 设备表（注册/心跳/离线）
	Commands *store.CommandStore    // 指令记录与查询
	Hub      *gatewayhub.Hub        // Gateway 连接与下行 SendJSON
	UI       *logic.UIHub           // 浏览器事件广播
	Accounts *store.AccountStore    // 聊天账号画像（设备绑定、tier）
	Egress   *store.EgressStore     // 代理出口池（SaaS 管控的业务出口）
	Caps     *store.CapabilityStore // 设备能力声明（板子上报的 action 列表）
	Policy   *policy.Policy         // 下发指令前的策略校验
}

// handlers 各层路由要用到的 HTTP/WS 入口（由 NewEngine 组装一次）。
type handlers struct {
	device  *handle.DeviceHandle
	egress  *handle.EgressHandle
	web     *handle.WebHandle
	gateway *handle.GatewayWSHandle
	ui      *handle.UIWSHandle
}

// NewEngine 组装 gin 引擎并按层注册路由。
func NewEngine(d Deps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	h := wireHandlers(d)
	registerBusinessAPI(r, h)  // /api/* 群控业务
	registerControlPlane(r, h) // /ws/gateway 机箱控制面
	registerUISurface(r, h)    // / 与 /ws/ui 呈现与推送

	return r
}

func wireHandlers(d Deps) handlers {
	deviceLogic := logic.NewDeviceLogic(d.Store, d.Hub, d.Commands, d.Accounts, d.Egress, d.Policy)
	egressLogic := logic.NewEgressLogic(d.Store, d.Hub, d.Accounts, d.Egress)
	gatewayLogic := logic.NewGatewayLogic(d.Store, d.Hub, d.UI, d.Caps, d.Accounts, d.Egress)

	return handlers{
		device:  handle.NewDeviceHandle(deviceLogic),
		egress:  handle.NewEgressHandle(egressLogic),
		web:     handle.NewWebHandle(),
		gateway: handle.NewGatewayWSHandle(gatewayLogic, d.Hub),
		ui:      handle.NewUIWSHandle(d.UI),
	}
}
