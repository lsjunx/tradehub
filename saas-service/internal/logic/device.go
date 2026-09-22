// Package logic SaaS 业务层：设备、异步指令、Gateway 上行、UI 推送。
package logic

import (
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/local/saas-service/internal/common/cloudwire"
	bizerr "github.com/local/saas-service/internal/common/errors"
	"github.com/local/saas-service/internal/common/uievent"
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
		return nil, bizerr.NotFound("device not found")
	}
	return dev, nil
}

func (l *DeviceLogic) GetCommand(cmdID string) (*store.CommandRecord, error) {
	rec, ok := l.Commands.Get(cmdID)
	if !ok {
		return nil, bizerr.NotFound("command not found")
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
		return nil, bizerr.BadRequest("action required")
	}
	dev, ok := l.Store.Get(deviceID)
	if !ok {
		return nil, bizerr.NotFound("device not found")
	}
	if !dev.Online {
		return nil, bizerr.NotFound("device offline")
	}
	if !l.Hub.HasGateway(dev.GatewayID) {
		return nil, bizerr.New(http.StatusConflict, "gateway not connected")
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
	err := l.Hub.SendJSON(dev.GatewayID, cloudwire.TypeCommand, map[string]string{
		"device_id": deviceID,
		"cmd_id":    cmdID,
		"action":    req.Action,
		"args":      req.Args,
	})
	if err != nil {
		l.Hub.CancelPending(cmdID)
		l.Commands.Finish(cmdID, store.CmdFailed, false, err.Error())
		return nil, bizerr.Conflict(err)
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
				l.Hub.Emit(uievent.CommandResult, gatewayhub.CommandResult{
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
			l.Hub.Emit(uievent.CommandResult, gatewayhub.CommandResult{
				DeviceID: deviceID,
				CmdID:    cmdID,
				Ok:       false,
				Message:  "timeout",
			})
		}
	}
}
