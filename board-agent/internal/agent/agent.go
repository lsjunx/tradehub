package agent

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
	"github.com/local/board-agent/internal/netinfo"
	boardv1 "github.com/local/board-agent/internal/pb/board/v1"
)

// Agent 板端主循环：注册、心跳、收指令。
type Agent struct {
	Cfg      config.Config
	DeviceID string
	IP       string
}

func New(cfg config.Config) (*Agent, error) {
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
		err := a.session()
		log.Printf("session ended: %v; reconnect in %s", err, backoff)
		time.Sleep(backoff)
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (a *Agent) session() error {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: a.Cfg.InsecureSkipVerify,
		MinVersion:         tls.VersionTLS12,
	}
	conn, err := tls.Dial("tcp", a.Cfg.GatewayAddr, tlsCfg)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Printf("connected to gateway %s as %s", a.Cfg.GatewayAddr, a.DeviceID)

	var writeMu sync.Mutex
	send := func(env *boardv1.Envelope) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return writeEnvelope(conn, env)
	}

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

	ticker := time.NewTicker(a.Cfg.HeartbeatInterval)
	defer ticker.Stop()

	errCh := make(chan error, 1)
	go func() {
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
				res := HandleCommand(cmd)
				if err := send(&boardv1.Envelope{
					MsgId:   uuid.NewString(),
					Payload: &boardv1.Envelope_CommandResult{CommandResult: res},
				}); err != nil {
					errCh <- err
					return
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
