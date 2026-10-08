package handle

import (
	"encoding/json"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"

	"saas-server/internal/common/gatewayhub"
	"saas-server/internal/logic"
)

// GatewayWSHandle Gateway 长连接入口。
type GatewayWSHandle struct {
	Logic *logic.GatewayLogic
	Hub   *gatewayhub.Hub
}

func NewGatewayWSHandle(l *logic.GatewayLogic, hub *gatewayhub.Hub) *GatewayWSHandle {
	return &GatewayWSHandle{Logic: l, Hub: hub}
}

func (h *GatewayWSHandle) Serve(c *gin.Context) {
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	ctx := c.Request.Context()
	var gatewayID string
	defer func() {
		if gatewayID != "" {
			h.Hub.RemoveGateway(gatewayID, conn)
		}
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var env struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(data, &env); err != nil {
			continue
		}
		id, set := h.Logic.HandleMessage(env.Type, env.Payload)
		if set {
			gatewayID = id
			h.Hub.SetGateway(gatewayID, conn)
		}
	}
}
