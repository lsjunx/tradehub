package logic

import (
	"encoding/json"
	"log"
	"time"

	cloudv1 "github.com/local/saas-service/internal/pb/cloud/v1"

	"github.com/local/saas-service/internal/common/cloudwire"
	"github.com/local/saas-service/internal/common/uievent"
	"github.com/local/saas-service/internal/gatewayhub"
	"github.com/local/saas-service/internal/store"
)

// GatewayLogic 处理 Gateway → SaaS 的上行 WSS 消息。
type GatewayLogic struct {
	Store    *store.Store
	Hub      *gatewayhub.Hub
	UI       *UIHub
	Caps     *store.CapabilityStore
	Accounts *store.AccountStore
	Egress   *store.EgressStore
}

func NewGatewayLogic(st *store.Store, hub *gatewayhub.Hub, ui *UIHub, caps *store.CapabilityStore, accounts *store.AccountStore, egress *store.EgressStore) *GatewayLogic {
	l := &GatewayLogic{Store: st, Hub: hub, UI: ui, Caps: caps, Accounts: accounts, Egress: egress}
	hub.OnEvent = func(eventType string, payload any) {
		ui.Broadcast(eventType, payload)
	}
	return l
}

// HandleMessage 【关键】按 proto/cloud MsgType 线网字符串分发 Gateway 上行消息。
// 返回 (gatewayID, true) 仅当 gateway_hello，供 handle 绑定连接。
func (l *GatewayLogic) HandleMessage(typ string, payload json.RawMessage) (gatewayID string, setGateway bool) {
	switch cloudwire.ParseType(typ) {
	case cloudv1.MsgType_MSG_TYPE_GATEWAY_HELLO:
		var p struct {
			GatewayID string `json:"gateway_id"`
		}
		_ = json.Unmarshal(payload, &p)
		log.Printf("gateway hello %s", p.GatewayID)
		return p.GatewayID, true

	case cloudv1.MsgType_MSG_TYPE_DEVICE_REGISTER:
		var p struct {
			DeviceID  string `json:"device_id"`
			IP        string `json:"ip"`
			Port      int32  `json:"port"`
			GatewayID string `json:"gateway_id"`
		}
		_ = json.Unmarshal(payload, &p)
		d := l.Store.UpsertRegister(p.DeviceID, p.IP, p.Port, p.GatewayID)
		l.UI.BroadcastDevice(p.DeviceID, uievent.DeviceUpdated, d, 0)

	case cloudv1.MsgType_MSG_TYPE_DEVICE_HEARTBEAT:
		var p struct {
			DeviceID  string `json:"device_id"`
			Status    string `json:"status"`
			GatewayID string `json:"gateway_id"`
		}
		_ = json.Unmarshal(payload, &p)
		d := l.Store.TouchHeartbeat(p.DeviceID, p.Status, p.GatewayID)
		l.UI.BroadcastDevice(p.DeviceID, uievent.DeviceUpdated, d, 5*time.Second)

	case cloudv1.MsgType_MSG_TYPE_DEVICE_OFFLINE:
		var p struct {
			DeviceID  string `json:"device_id"`
			GatewayID string `json:"gateway_id"`
		}
		_ = json.Unmarshal(payload, &p)
		if d := l.Store.MarkOffline(p.DeviceID, p.GatewayID); d != nil {
			l.UI.BroadcastDevice(p.DeviceID, uievent.DeviceUpdated, d, 0)
		}

	case cloudv1.MsgType_MSG_TYPE_COMMAND_RESULT:
		var p gatewayhub.CommandResult
		_ = json.Unmarshal(payload, &p)
		l.Hub.Complete(p)

	case cloudv1.MsgType_MSG_TYPE_CAPABILITY:
		var p struct {
			DeviceID string `json:"device_id"`
			Entries  []store.CapabilityEntry `json:"entries"`
		}
		if err := json.Unmarshal(payload, &p); err != nil || p.DeviceID == "" {
			break
		}
		l.Caps.Put(p.DeviceID, p.Entries)

	case cloudv1.MsgType_MSG_TYPE_EVENT:
		var p struct {
			EventID     string `json:"event_id"`
			DeviceID    string `json:"device_id"`
			Name        string `json:"name"`
			PayloadJSON string `json:"payload_json"`
			TsUnixMs    int64  `json:"ts_unix_ms"`
			GatewayID   string `json:"gateway_id"`
		}
		if err := json.Unmarshal(payload, &p); err != nil || p.Name == "" {
			break
		}
		l.applyRiskEvent(p.Name, p.DeviceID, p.PayloadJSON)
		l.UI.Broadcast(uievent.AccountRisk, p)

	default:
		log.Printf("未知 Gateway 消息 type=%q", typ)
	}
	return "", false
}

func (l *GatewayLogic) applyRiskEvent(name, deviceID, payloadJSON string) {
	accountID, egressID := parseRiskPayload(payloadJSON)
	switch name {
	case "egress.unhealthy":
		if egressID != "" {
			_ = l.Egress.SetHealthy(egressID, false)
			return
		}
		for _, acc := range l.Accounts.ListByDevice(deviceID) {
			if acc.EgressID != "" {
				_ = l.Egress.SetHealthy(acc.EgressID, false)
			}
		}
	case "account.session_dead":
		if accountID != "" {
			_ = l.Accounts.SetTier(accountID, store.TierDead)
		}
	case "account.challenge", "account.rate_limited":
		if accountID != "" {
			_ = l.Accounts.SetTier(accountID, store.TierRestricted)
		}
	}
}

func parseRiskPayload(payloadJSON string) (accountID, egressID string) {
	if payloadJSON == "" {
		return "", ""
	}
	var m struct {
		AccountID string `json:"account_id"`
		EgressID  string `json:"egress_id"`
	}
	if err := json.Unmarshal([]byte(payloadJSON), &m); err != nil {
		return "", ""
	}
	return m.AccountID, m.EgressID
}
