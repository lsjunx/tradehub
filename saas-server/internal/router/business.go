package router

import "github.com/gin-gonic/gin"

// registerBusinessAPI 业务接口：操作员 / 前端用的 HTTPS REST。
// 统一响应 {code,message,data,timestamp}；不承载机箱长连接。
//
// 设备与指令：
//
//	GET  /api/devices
//	GET  /api/devices/:id
//	POST /api/devices/:id/commands  → 异步受理，可带 account_id 走 Policy
//	GET  /api/commands/:cmd_id
//
// 账号与出口（群控配置）：
//
//	POST/GET /api/egress
//	POST/GET /api/accounts
//	POST     /api/accounts/:id/egress  → 绑定并尽量推 egress.apply
func registerBusinessAPI(r *gin.Engine, h handlers) {
	api := r.Group("/api")
	{
		// —— 设备视图 ——
		api.GET("/devices", h.device.ListDevices)
		api.GET("/devices/:id", h.device.GetDevice)

		// —— 异步指令 ——
		api.POST("/devices/:id/commands", h.device.SendCommand)
		api.GET("/commands/:cmd_id", h.device.GetCommand)

		// —— 代理池（SaaS 管控的业务出口 IP）——
		api.POST("/egress", h.egress.Upsert)
		api.GET("/egress", h.egress.ListEgress)

		// —— 聊天账号与出口绑定 ——
		api.POST("/accounts", h.egress.UpsertAccount)
		api.GET("/accounts", h.egress.ListAccounts)
		api.POST("/accounts/:id/egress", h.egress.BindEgress)
	}
}
