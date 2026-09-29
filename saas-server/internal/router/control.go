package router

import "github.com/gin-gonic/gin"

// registerControlPlane 网络控制面：仅给 Gateway 进程连接，不是给人点的。
//
//	GET /ws/gateway
//
// 职责：维持 gateway_id ↔ 连接；收 hello/register/heartbeat/offline/
// command_result/capability/event；下行 command（含业务指令与 egress.apply）。
// 业务语义在 logic.GatewayLogic，本层只挂入口。
func registerControlPlane(r *gin.Engine, h handlers) {
	ws := r.Group("/ws")
	{
		ws.GET("/gateway", h.gateway.Serve)
	}
}
