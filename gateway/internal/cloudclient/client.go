package cloudclient

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

// Client 维持到 saas 的 WSS，并转发上下行消息。
type Client struct {
	URL                string
	GatewayID          string
	InsecureSkipVerify bool
	OnCommand          func(deviceID, cmdID, action, args string)

	mu   sync.Mutex
	conn *websocket.Conn
}

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
	log.Printf("cloud connected, gateway_id=%s", c.GatewayID)
	return nil
}

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
			log.Printf("bad cloud msg: %v", err)
			continue
		}
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
