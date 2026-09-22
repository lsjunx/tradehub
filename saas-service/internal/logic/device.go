package logic

import (
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/local/saas-service/internal/gatewayhub"
	"github.com/local/saas-service/internal/store"
)

const commandWaitTimeout = 10 * time.Second

// DeviceLogic 设备查询与指令下发。
type DeviceLogic struct {
	Store    *store.Store
	Hub      *gatewayhub.Hub
	Commands *store.CommandStore
}

func NewDeviceLogic(st *store.Store, hub *gatewayhub.Hub, cmds *store.CommandStore) *DeviceLogic {
	return &DeviceLogic{Store: st, Hub: hub, Commands: cmds}
}

func (l *DeviceLogic) ListDevices() []store.Device {
	return l.Store.List()
}

func (l *DeviceLogic) GetDevice(deviceID string) (*store.Device, error) {
	dev, ok := l.Store.Get(deviceID)
	if !ok {
		return nil, NewAppError(http.StatusNotFound, "device not found")
	}
	return dev, nil
}

func (l *DeviceLogic) GetCommand(cmdID string) (*store.CommandRecord, error) {
	rec, ok := l.Commands.Get(cmdID)
	if !ok {
		return nil, NewAppError(http.StatusNotFound, "command not found")
	}
	return rec, nil
}

// CommandRequest 下发指令入参。
type CommandRequest struct {
	Action string `json:"action"`
	Args   string `json:"args"`
}

// CommandAccept 异步受理响应。
type CommandAccept struct {
	CmdID    string `json:"cmd_id"`
	DeviceID string `json:"device_id"`
	Status   string `json:"status"`
}

// AcceptCommand 立即返回 accepted，结果经 /ws/ui 的 command_result 推送，也可查 GET /api/commands/:id。
func (l *DeviceLogic) AcceptCommand(deviceID string, req CommandRequest) (*CommandAccept, error) {
	if req.Action == "" {
		return nil, NewAppError(http.StatusBadRequest, "action required")
	}
	dev, ok := l.Store.Get(deviceID)
	if !ok {
		return nil, NewAppError(http.StatusNotFound, "device not found")
	}
	if !dev.Online {
		return nil, NewAppError(http.StatusNotFound, "device offline")
	}
	if !l.Hub.HasGateway(dev.GatewayID) {
		return nil, NewAppError(http.StatusConflict, "gateway not connected")
	}

	cmdID := uuid.NewString()
	rec := &store.CommandRecord{
		CmdID:     cmdID,
		DeviceID:  deviceID,
		Action:    req.Action,
		Args:      req.Args,
		Status:    store.CmdAccepted,
		CreatedAt: time.Now(),
	}
	l.Commands.Put(rec)

	log.Printf("accept command device=%s cmd_id=%s action=%s args=%q via gateway=%s",
		deviceID, cmdID, req.Action, req.Args, dev.GatewayID)

	wait := l.Hub.RegisterPending(cmdID)
	err := l.Hub.SendJSON(dev.GatewayID, "command", map[string]string{
		"device_id": deviceID,
		"cmd_id":    cmdID,
		"action":    req.Action,
		"args":      req.Args,
	})
	if err != nil {
		l.Hub.CancelPending(cmdID)
		l.Commands.Finish(cmdID, store.CmdFailed, false, err.Error())
		return nil, WrapConflict(err)
	}

	go l.waitCommandResult(cmdID, deviceID, wait)

	return &CommandAccept{
		CmdID:    cmdID,
		DeviceID: deviceID,
		Status:   store.CmdAccepted,
	}, nil
}

func (l *DeviceLogic) waitCommandResult(cmdID, deviceID string, wait <-chan gatewayhub.CommandResult) {
	select {
	case res, ok := <-wait:
		if !ok {
			if finished := l.Commands.FinishIfAccepted(cmdID, store.CmdTimeout, false, "timeout"); finished != nil {
				log.Printf("command timeout cmd_id=%s device=%s", cmdID, deviceID)
				l.Hub.Emit("command_result", gatewayhub.CommandResult{
					DeviceID: deviceID,
					CmdID:    cmdID,
					Ok:       false,
					Message:  "timeout",
				})
			}
			return
		}
		status := store.CmdSucceeded
		if !res.Ok {
			status = store.CmdFailed
		}
		l.Commands.FinishIfAccepted(cmdID, status, res.Ok, res.Message)
		log.Printf("command result cmd_id=%s ok=%v message=%q", res.CmdID, res.Ok, res.Message)
		// Hub.Complete 已推送 command_result
	case <-time.After(commandWaitTimeout):
		if !l.Hub.CancelPending(cmdID) {
			return // 结果已先到达
		}
		if finished := l.Commands.FinishIfAccepted(cmdID, store.CmdTimeout, false, "timeout"); finished != nil {
			log.Printf("command timeout cmd_id=%s device=%s", cmdID, deviceID)
			l.Hub.Emit("command_result", gatewayhub.CommandResult{
				DeviceID: deviceID,
				CmdID:    cmdID,
				Ok:       false,
				Message:  "timeout",
			})
		}
	}
}
