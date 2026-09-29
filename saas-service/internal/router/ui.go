package router

import "github.com/gin-gonic/gin"

// registerUISurface 呈现与推送：浏览器演示页 + 事件订阅。
//
//	GET /       → 简易演示 HTML（非正式群控面板）
//	GET /ws/ui  → 订阅 device_updated / command_result / account_risk
//
// 前端业务写操作仍走 /api/*；此处只读推送（/ws/ui 读循环保活）。
func registerUISurface(r *gin.Engine, h handlers) {
	r.GET("/", h.web.Index)

	ws := r.Group("/ws")
	{
		ws.GET("/ui", h.ui.Serve)
	}
}
