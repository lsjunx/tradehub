// Package logic 负责 Board↔Cloud 的转发业务（不直接管监听 socket）。
package logic

import (
	"fmt"
	"gateway/internal/common/cloud"
	"gateway/internal/common/frame"
	"gateway/internal/common/session"
	"log"
	"net"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	boardv1 "gateway/internal/pb/board/v1"
)

// CloudSender 向 SaaS 上报上行消息（便于测试注入 fake）。
type CloudSender interface {
	Send(typ string, payload any) error
}

// Bridge 网关转发核心：上行（Board→SaaS）、下行（SaaS→Board）。
type Bridge struct {
	GatewayID string
	Registry  *session.Registry
	Cloud     CloudSender
}

// HandleBoardEnvelope 【关键·上行】处理 Board 发来的一条 protobuf Envelope。
// conn 用于 Register 时写入会话表；deviceID 由 Register 回填，供断连时使用。
func (b *Bridge) HandleBoardEnvelope(conn net.Conn, sessionID string, deviceID *string, env *boardv1.Envelope) error {
	switch p := env.Payload.(type) {
	case *boardv1.Envelope_Register:
		// Board 只上报 device_id/ip/port；gateway_id 由本层注入后再发给 SaaS
		reg := p.Register
		*deviceID = reg.GetDeviceId()
		b.Registry.Put(&session.BoardConn{
			DeviceID:  *deviceID,
			SessionID: sessionID,
			Conn:      conn,
		})
		_ = b.Cloud.Send(cloud.TypeDeviceRegister, map[string]any{
			"device_id":  *deviceID,
			"ip":         reg.GetIp(),
			"port":       reg.GetPort(),
			"gateway_id": b.GatewayID,
		})
		log.Printf("设备注册 device=%s ip=%s", *deviceID, reg.GetIp())

	case *boardv1.Envelope_Heartbeat:
		hb := p.Heartbeat
		_ = b.Cloud.Send(cloud.TypeDeviceHeartbeat, map[string]any{
			"device_id":  hb.GetDeviceId(),
			"ts_unix_ms": hb.GetTsUnixMs(),
			"status":     hb.GetStatus(),
			"gateway_id": b.GatewayID,
		})

	case *boardv1.Envelope_CommandResult:
		cr := p.CommandResult
		log.Printf("设备回执 device=%s cmd_id=%s ok=%v msg=%q",
			cr.GetDeviceId(), cr.GetCmdId(), cr.GetOk(), cr.GetMessage())
		_ = b.Cloud.Send(cloud.TypeCommandResult, map[string]any{
			"device_id": cr.GetDeviceId(),
			"cmd_id":    cr.GetCmdId(),
			"ok":        cr.GetOk(),
			"message":   cr.GetMessage(),
		})

	case *boardv1.Envelope_Event:
		ev := p.Event
		_ = b.Cloud.Send(cloud.TypeEvent, map[string]any{
			"event_id":     ev.GetEventId(),
			"device_id":    ev.GetDeviceId(),
			"name":         ev.GetName(),
			"payload_json": ev.GetPayloadJson(),
			"ts_unix_ms":   ev.GetTsUnixMs(),
			"gateway_id":   b.GatewayID,
		})

	case *boardv1.Envelope_Capability:
		cap := p.Capability
		entries := make([]map[string]string, 0, len(cap.GetEntries()))
		for _, e := range cap.GetEntries() {
			entries = append(entries, map[string]string{
				"action":    e.GetAction(),
				"risk_hint": e.GetRiskHint(),
			})
		}
		_ = b.Cloud.Send(cloud.TypeCapability, map[string]any{
			"device_id":  cap.GetDeviceId(),
			"gateway_id": b.GatewayID,
			"entries":    entries,
		})
	}
	return nil
}

// NotifyBoardOffline 【关键·上行】仅当 SessionID 仍匹配时向云端报离线。
func (b *Bridge) NotifyBoardOffline(deviceID, sessionID string) {
	if deviceID == "" {
		return
	}
	if !b.Registry.RemoveIfSame(deviceID, sessionID) {
		return // 已是新连接，忽略旧断开
	}
	_ = b.Cloud.Send(cloud.TypeDeviceOffline, map[string]string{
		"device_id":  deviceID,
		"gateway_id": b.GatewayID,
	})
	log.Printf("设备离线 device=%s session=%s", deviceID, sessionID)
}

// HandleCloudCommand 【关键·下行】将 SaaS 指令转发到对应 Board。
func (b *Bridge) HandleCloudCommand(deviceID, cmdID, action, args string) {
	log.Printf("云端指令 device=%s cmd_id=%s action=%s args=%q", deviceID, cmdID, action, args)
	if err := b.sendCommandToBoard(deviceID, cmdID, action, args); err != nil {
		log.Printf("转发 Board 失败 device=%s: %v", deviceID, err)
		_ = b.Cloud.Send(cloud.TypeCommandResult, map[string]any{
			"device_id": deviceID,
			"cmd_id":    cmdID,
			"ok":        false,
			"message":   "device unreachable",
		})
		return
	}
	log.Printf("已转发指令到 Board device=%s", deviceID)
}

func (b *Bridge) sendCommandToBoard(deviceID, cmdID, action, args string) error {
	bc, ok := b.Registry.Get(deviceID)
	if !ok || bc.Conn == nil {
		return fmt.Errorf("device %s not connected", deviceID)
	}
	env := &boardv1.Envelope{
		MsgId: uuid.NewString(),
		Payload: &boardv1.Envelope_Command{Command: &boardv1.Command{
			DeviceId: deviceID,
			CmdId:    cmdID,
			Action:   action,
			Args:     args,
		}},
	}
	raw, err := proto.Marshal(env)
	if err != nil {
		return err
	}
	return frame.WriteFrame(bc.Conn, raw)
}
