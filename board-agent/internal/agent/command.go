package agent

import (
	"log"

	boardv1 "github.com/local/board-agent/internal/pb/board/v1"
)

// HandleCommand 处理云端下发指令；第一版仅支持 echo。
func HandleCommand(cmd *boardv1.Command) *boardv1.CommandResult {
	log.Printf("recv command cmd_id=%s action=%s args=%q", cmd.GetCmdId(), cmd.GetAction(), cmd.GetArgs())
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
