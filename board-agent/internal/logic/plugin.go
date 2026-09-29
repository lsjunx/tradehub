package logic

import boardv1 "board-agent/internal/pb/board/v1"

type Plugin interface {
	Actions() []string
	Handle(cmd *boardv1.Command) *boardv1.CommandResult
}
