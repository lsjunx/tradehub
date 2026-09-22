// Package uievent 定义浏览器 /ws/ui 事件 type（与 Gateway 线网 type 分开）。
package uievent

import "github.com/local/saas-service/internal/common/cloudwire"

const (
	// DeviceUpdated 设备快照变更（注册/心跳降频/离线）。
	DeviceUpdated = "device_updated"
)

// CommandResult 指令执行结果（与 Gateway 线网 type 同名）。
var CommandResult = cloudwire.TypeCommandResult
