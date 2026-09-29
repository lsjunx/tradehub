package handle

import (
	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"

	"saas-server/internal/logic"
)

// UIWSHandle 浏览器事件订阅。
type UIWSHandle struct {
	UI *logic.UIHub
}

func NewUIWSHandle(ui *logic.UIHub) *UIWSHandle {
	return &UIWSHandle{UI: ui}
}

func (h *UIWSHandle) Serve(c *gin.Context) {
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	h.UI.Add(conn)
	defer func() {
		h.UI.Remove(conn)
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}()
	ctx := c.Request.Context()
	for {
		if _, _, err := conn.Read(ctx); err != nil {
			return
		}
	}
}
