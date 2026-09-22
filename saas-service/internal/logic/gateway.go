package logic

import (
	"encoding/json"
	"log"
	"time"

	"github.com/local/saas-service/internal/gatewayhub"
	"github.com/local/saas-service/internal/store"
)

// GatewayLogic 处理 Gateway 上行消息。
type GatewayLogic struct {
	Store *store.Store
	Hub   *gatewayhub.Hub
	UI    *UIHub
}

func NewGatewayLogic(st *store.Store, hub *gatewayhub.Hub, ui *UIHub) *GatewayLogic {
	l := &GatewayLogic{Store: st, Hub: hub, UI: ui}
	hub.OnEvent = func(eventType string, payload any) {
		ui.Broadcast(eventType, payload)
	}
	return l
}

// HandleMessage 解析并处理一条 Gateway WSS 消息。
func (l *GatewayLogic) HandleMessage(typ string, payload json.RawMessage) (gatewayID string, setGateway bool) {
	switch typ {
	case "gateway_hello":
		var p struct {
			GatewayID string `json:"gateway_id"`
		}
		_ = json.Unmarshal(payload, &p)
		log.Printf("gateway hello %s", p.GatewayID)
		return p.GatewayID, true

	case "device_register":
		var p struct {
			DeviceID  string `json:"device_id"`
			IP        string `json:"ip"`
			Port      int32  `json:"port"`
			GatewayID string `json:"gateway_id"`
		}
		_ = json.Unmarshal(payload, &p)
		d := l.Store.UpsertRegister(p.DeviceID, p.IP, p.Port, p.GatewayID)
		// 注册立即推送
		l.UI.BroadcastDevice(p.DeviceID, "device_updated", d, 0)

	case "device_heartbeat":
		var p struct {
			DeviceID  string `json:"device_id"`
			Status    string `json:"status"`
			GatewayID string `json:"gateway_id"`
		}
		_ = json.Unmarshal(payload, &p)
		d := l.Store.TouchHeartbeat(p.DeviceID, p.Status, p.GatewayID)
		// 心跳 UI 最多每 5s 推一次，避免刷爆浏览器连接
		l.UI.BroadcastDevice(p.DeviceID, "device_updated", d, 5*time.Second)

	case "device_offline":
		var p struct {
			DeviceID  string `json:"device_id"`
			GatewayID string `json:"gateway_id"`
		}
		_ = json.Unmarshal(payload, &p)
		if d := l.Store.MarkOffline(p.DeviceID, p.GatewayID); d != nil {
			l.UI.BroadcastDevice(p.DeviceID, "device_updated", d, 0)
		}

	case "command_result":
		var p gatewayhub.CommandResult
		_ = json.Unmarshal(payload, &p)
		l.Hub.Complete(p)
	}
	return "", false
}
