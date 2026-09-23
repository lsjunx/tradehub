// Package handle 是 Board 的 I/O 边界：连接 Gateway、收发 protobuf 帧。
package handle

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"github.com/local/board-agent/internal/config"
	"github.com/local/board-agent/internal/frame"
	"github.com/local/board-agent/internal/identity"
	"github.com/local/board-agent/internal/logic"
	"github.com/local/board-agent/internal/netinfo"
	boardv1 "github.com/local/board-agent/internal/pb/board/v1"
)

// Agent 板端连接与会话循环。
type Agent struct {
	Cfg      config.Config
	DeviceID string
	IP       string
}

func NewAgent(cfg config.Config) (*Agent, error) {
	idPath := filepath.Join(cfg.DataDir, "device_id")
	id, err := identity.LoadOrCreate(idPath)
	if err != nil {
		return nil, err
	}
	return &Agent{Cfg: cfg, DeviceID: id, IP: netinfo.FirstIPv4()}, nil
}

// Run 持续连接 Gateway，断线指数退避重连。
func (a *Agent) Run() error {
	backoff := time.Second
	for {
		err := a.serveSession()
		log.Printf("会话结束: %v；%s 后重连", err, backoff)
		time.Sleep(backoff)
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

// serveSession 【关键】单次 TLS 会话：注册 → 心跳 + 收指令。
func (a *Agent) serveSession() error {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: a.Cfg.InsecureSkipVerify,
		MinVersion:         tls.VersionTLS12,
	}
	conn, err := tls.Dial("tcp", a.Cfg.GatewayAddr, tlsCfg)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Printf("已连接 Gateway %s，device_id=%s", a.Cfg.GatewayAddr, a.DeviceID)

	var writeMu sync.Mutex
	send := func(env *boardv1.Envelope) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return writeEnvelope(conn, env)
	}

	// 上行：注册（不含 gateway_id）
	if err := send(&boardv1.Envelope{
		MsgId: uuid.NewString(),
		Payload: &boardv1.Envelope_Register{Register: &boardv1.Register{
			DeviceId: a.DeviceID,
			Ip:       a.IP,
			Port:     int32(a.Cfg.ReportPort),
		}},
	}); err != nil {
		return err
	}

	entries := logic.DefaultRouter().CapabilityEntries()
	_ = send(&boardv1.Envelope{
		MsgId: uuid.NewString(),
		Payload: &boardv1.Envelope_Capability{Capability: &boardv1.Capability{
			DeviceId: a.DeviceID,
			Entries:  entries,
		}},
	})

	ticker := time.NewTicker(a.Cfg.HeartbeatInterval)
	defer ticker.Stop()

	errCh := make(chan error, 1)
	go func() {
		// 【关键·收指令】读帧，遇 Command 交 logic 执行并回 CommandResult
		for {
			raw, err := frame.ReadFrame(conn)
			if err != nil {
				errCh <- err
				return
			}
			var env boardv1.Envelope
			if err := proto.Unmarshal(raw, &env); err != nil {
				errCh <- err
				return
			}
			if cmd := env.GetCommand(); cmd != nil {
				res := logic.HandleCommand(cmd)
				if err := send(&boardv1.Envelope{
					MsgId:   uuid.NewString(),
					Payload: &boardv1.Envelope_CommandResult{CommandResult: res},
				}); err != nil {
					errCh <- err
					return
				}
				for _, ev := range logic.TakePendingEvents() {
					if err := send(&boardv1.Envelope{
						MsgId:   uuid.NewString(),
						Payload: &boardv1.Envelope_Event{Event: ev},
					}); err != nil {
						errCh <- err
						return
					}
				}
			}
		}
	}()

	for {
		select {
		case err := <-errCh:
			return err
		case <-ticker.C:
			if err := send(&boardv1.Envelope{
				MsgId: uuid.NewString(),
				Payload: &boardv1.Envelope_Heartbeat{Heartbeat: &boardv1.Heartbeat{
					DeviceId: a.DeviceID,
					TsUnixMs: time.Now().UnixMilli(),
					Status:   "ok",
				}},
			}); err != nil {
				return err
			}
		}
	}
}

func writeEnvelope(w io.Writer, env *boardv1.Envelope) error {
	raw, err := proto.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return frame.WriteFrame(w, raw)
}
