// Package logic 板端业务：指令执行等（不负责网络连接）。
package logic

import (
	"log"

	boardv1 "github.com/local/board-agent/internal/pb/board/v1"
)

var defaultRouter = NewRouter(EchoPlugin{})

func HandleCommand(cmd *boardv1.Command) *boardv1.CommandResult {
	log.Printf("收到指令 cmd_id=%s action=%s args=%q", cmd.GetCmdId(), cmd.GetAction(), cmd.GetArgs())
	return defaultRouter.Dispatch(cmd)
}

func DefaultRouter() *Router { return defaultRouter }
