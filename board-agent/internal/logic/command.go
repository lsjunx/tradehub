// Package logic 板端业务：指令执行等（不负责网络连接）。
package logic

import (
	"log"

	"github.com/tradehub/board-agent/internal/egress"
	boardv1 "github.com/tradehub/board-agent/internal/pb/board/v1"
)

// EgressStore holds per-account proxy config applied via egress.apply (MVP in-memory).
var EgressStore = egress.NewStore()

var defaultRouter = NewRouter(EchoPlugin{}, NewEgressPlugin(EgressStore))

func HandleCommand(cmd *boardv1.Command) *boardv1.CommandResult {
	log.Printf("收到指令 cmd_id=%s action=%s args=%q", cmd.GetCmdId(), cmd.GetAction(), cmd.GetArgs())
	return defaultRouter.Dispatch(cmd)
}

func DefaultRouter() *Router { return defaultRouter }
