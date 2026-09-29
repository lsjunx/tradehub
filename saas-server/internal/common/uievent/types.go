// Package uievent 定义浏览器 /ws/ui 事件 type（与 Gateway 线网 type 分开）。
package uievent

import "saas-server/internal/common/cloudwire"

const (
	// DeviceUpdated 设备快照变更（注册/心跳降频/离线）。
	DeviceUpdated = "device_updated"
	// AccountRisk 账号/出口风险（仅白名单 event 名会推，如 challenge、session_dead）。
	AccountRisk = "account_risk"
)

// CommandResult 指令执行结果（与 Gateway 线网 type 同名）。
var CommandResult = cloudwire.TypeCommandResult
