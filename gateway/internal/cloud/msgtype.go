package cloud

import (
	"strings"

	cloudv1 "github.com/local/gateway/internal/pb/cloud/v1"
)

// 线网 type 由 MsgType 枚举名推导：MSG_TYPE_GATEWAY_HELLO → gateway_hello。
// 增删枚举只需改 proto，再 gen-proto，无需维护单独映射表。
var (
	TypeGatewayHello    = TypeName(cloudv1.MsgType_MSG_TYPE_GATEWAY_HELLO)
	TypeDeviceRegister  = TypeName(cloudv1.MsgType_MSG_TYPE_DEVICE_REGISTER)
	TypeDeviceHeartbeat = TypeName(cloudv1.MsgType_MSG_TYPE_DEVICE_HEARTBEAT)
	TypeDeviceOffline   = TypeName(cloudv1.MsgType_MSG_TYPE_DEVICE_OFFLINE)
	TypeCommand         = TypeName(cloudv1.MsgType_MSG_TYPE_COMMAND)
	TypeCommandResult   = TypeName(cloudv1.MsgType_MSG_TYPE_COMMAND_RESULT)
	TypeEvent           = TypeName(cloudv1.MsgType_MSG_TYPE_EVENT)
	TypeCapability      = TypeName(cloudv1.MsgType_MSG_TYPE_CAPABILITY)
)

// TypeName 用生成代码的 MsgType_name 转为线网字符串。
func TypeName(t cloudv1.MsgType) string {
	name, ok := cloudv1.MsgType_name[int32(t)]
	if !ok || t == cloudv1.MsgType_MSG_TYPE_UNSPECIFIED {
		return ""
	}
	const prefix = "MSG_TYPE_"
	if !strings.HasPrefix(name, prefix) {
		return ""
	}
	return strings.ToLower(strings.TrimPrefix(name, prefix))
}

// ParseType 用生成代码的 MsgType_value 解析线网字符串。
func ParseType(s string) cloudv1.MsgType {
	key := "MSG_TYPE_" + strings.ToUpper(s)
	if v, ok := cloudv1.MsgType_value[key]; ok {
		return cloudv1.MsgType(v)
	}
	return cloudv1.MsgType_MSG_TYPE_UNSPECIFIED
}
