package response

import "time"

// WSEvent 浏览器 /ws/ui 推送结构（在 REST Body 上增加 type）。
type WSEvent struct {
	Type      string `json:"type"`
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	Timestamp int64  `json:"timestamp"`
}

// NewWSEvent 构造成功推送事件，code=0。
func NewWSEvent(eventType string, data any) WSEvent {
	return WSEvent{
		Type:      eventType,
		Code:      0,
		Message:   "ok",
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}
}
