package cloud

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// CommandHandler 云端下发指令时的回调（由 logic 层注入）。
type CommandHandler func(deviceID, cmdID, action, args string)

// Client 主动连接 SaaS /ws/gateway，上报设备事件并接收 command。
type Client struct {
	URL                string
	GatewayID          string
	InsecureSkipVerify bool
	OnCommand          CommandHandler

	mu   sync.Mutex
	conn *websocket.Conn
}

// Connect 拨号 WSS，并发送 gateway_hello。
func (c *Client) Connect(ctx context.Context) error {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: c.InsecureSkipVerify,
		MinVersion:         tls.VersionTLS12,
	}
	opts := &websocket.DialOptions{
		HTTPClient: &http.Client{
			Transport: &http.Transport{TLSClientConfig: tlsCfg},
		},
	}
	conn, _, err := websocket.Dial(ctx, c.URL, opts)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	hello, err := MarshalEnvelope(TypeGatewayHello, map[string]string{"gateway_id": c.GatewayID})
	if err != nil {
		return err
	}
	if err := c.write(ctx, hello); err != nil {
		return err
	}
	log.Printf("已连接 SaaS，gateway_id=%s", c.GatewayID)
	return nil
}

// RunRead 【关键】从 SaaS 读下行消息；当前仅处理 type=command。
func (c *Client) RunRead(ctx context.Context) error {
	for {
		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()
		if conn == nil {
			return fmt.Errorf("not connected")
		}
		_, data, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		env, err := UnmarshalEnvelope(data)
		if err != nil {
			log.Printf("云端消息解析失败: %v", err)
			continue
		}
		// 云端下发指令入口
		if env.Type == TypeCommand && c.OnCommand != nil {
			var p struct {
				DeviceID string `json:"device_id"`
				CmdID    string `json:"cmd_id"`
				Action   string `json:"action"`
				Args     string `json:"args"`
			}
			if err := json.Unmarshal(env.Payload, &p); err != nil {
				continue
			}
			c.OnCommand(p.DeviceID, p.CmdID, p.Action, p.Args)
		}
	}
}

// Send 向 SaaS 上报一条上行事件（register / heartbeat / offline / command_result）。
func (c *Client) Send(typ string, payload any) error {
	raw, err := MarshalEnvelope(typ, payload)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return c.write(ctx, raw)
}

func (c *Client) write(ctx context.Context, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}
	return c.conn.Write(ctx, websocket.MessageText, data)
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		_ = c.conn.Close(websocket.StatusNormalClosure, "")
		c.conn = nil
	}
}
