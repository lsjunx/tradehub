// Package logic 板端业务：指令执行等（不负责网络连接）。
package logic

import (
	"log"

	boardv1 "github.com/local/board-agent/internal/pb/board/v1"
)

// HandleCommand 【关键】处理 Gateway 下发的指令；第一版仅 echo。
// 新增业务命令时在此增加 case，Gateway/SaaS 一般无需改动。
func HandleCommand(cmd *boardv1.Command) *boardv1.CommandResult {
	log.Printf("收到指令 cmd_id=%s action=%s args=%q", cmd.GetCmdId(), cmd.GetAction(), cmd.GetArgs())
	res := &boardv1.CommandResult{
		DeviceId: cmd.GetDeviceId(),
		CmdId:    cmd.GetCmdId(),
	}
	switch cmd.GetAction() {
	case "echo":
		res.Ok = true
		res.Message = cmd.GetArgs()
		log.Printf("echo: %s", cmd.GetArgs())
	default:
		res.Ok = false
		res.Message = "unsupported action: " + cmd.GetAction()
	}
	return res
}
