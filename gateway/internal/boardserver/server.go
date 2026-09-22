package boardserver

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"github.com/local/gateway/internal/boardsession"
	"github.com/local/gateway/internal/cloudclient"
	"github.com/local/gateway/internal/frame"
	boardv1 "github.com/local/gateway/internal/pb/board/v1"
)

// Server 接受 Board TLS 连接并转发到云端。
type Server struct {
	Listen    string
	CertFile  string
	KeyFile   string
	GatewayID string
	Registry  *boardsession.Registry
	Cloud     *cloudclient.Client
}

func (s *Server) ListenAndServe() error {
	cert, err := tls.LoadX509KeyPair(s.CertFile, s.KeyFile)
	if err != nil {
		return err
	}
	ln, err := tls.Listen("tcp", s.Listen, &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	})
	if err != nil {
		return err
	}
	log.Printf("board TLS listen %s", s.Listen)
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	sessionID := uuid.NewString()
	var deviceID string
	defer func() {
		_ = conn.Close()
		if deviceID != "" && s.Registry.RemoveIfSame(deviceID, sessionID) {
			_ = s.Cloud.Send(cloudclient.TypeDeviceOffline, map[string]string{
				"device_id":  deviceID,
				"gateway_id": s.GatewayID,
			})
			log.Printf("device offline %s session=%s", deviceID, sessionID)
		}
	}()

	for {
		raw, err := frame.ReadFrame(conn)
		if err != nil {
			log.Printf("board read: %v", err)
			return
		}
		var env boardv1.Envelope
		if err := proto.Unmarshal(raw, &env); err != nil {
			log.Printf("board unmarshal: %v", err)
			return
		}
		switch p := env.Payload.(type) {
		case *boardv1.Envelope_Register:
			reg := p.Register
			deviceID = reg.GetDeviceId()
			s.Registry.Put(&boardsession.BoardConnection{
				DeviceID:  deviceID,
				SessionID: sessionID,
				Conn:      conn,
			})
			_ = s.Cloud.Send(cloudclient.TypeDeviceRegister, map[string]any{
				"device_id":  deviceID,
				"ip":         reg.GetIp(),
				"port":       reg.GetPort(),
				"gateway_id": s.GatewayID,
			})
			log.Printf("device register %s ip=%s", deviceID, reg.GetIp())
		case *boardv1.Envelope_Heartbeat:
			hb := p.Heartbeat
			_ = s.Cloud.Send(cloudclient.TypeDeviceHeartbeat, map[string]any{
				"device_id":  hb.GetDeviceId(),
				"ts_unix_ms": hb.GetTsUnixMs(),
				"status":     hb.GetStatus(),
				"gateway_id": s.GatewayID,
			})
		case *boardv1.Envelope_CommandResult:
			cr := p.CommandResult
			log.Printf("board result device=%s cmd_id=%s ok=%v message=%q",
				cr.GetDeviceId(), cr.GetCmdId(), cr.GetOk(), cr.GetMessage())
			_ = s.Cloud.Send(cloudclient.TypeCommandResult, map[string]any{
				"device_id": cr.GetDeviceId(),
				"cmd_id":    cr.GetCmdId(),
				"ok":        cr.GetOk(),
				"message":   cr.GetMessage(),
			})
		}
	}
}

// SendCommand 向指定设备下发指令。
func (s *Server) SendCommand(deviceID, cmdID, action, args string) error {
	bc, ok := s.Registry.Get(deviceID)
	if !ok || bc.Conn == nil {
		return fmt.Errorf("device %s not connected", deviceID)
	}
	env := &boardv1.Envelope{
		MsgId: uuid.NewString(),
		Payload: &boardv1.Envelope_Command{Command: &boardv1.Command{
			DeviceId: deviceID,
			CmdId:    cmdID,
			Action:   action,
			Args:     args,
		}},
	}
	raw, err := proto.Marshal(env)
	if err != nil {
		return err
	}
	return frame.WriteFrame(bc.Conn, raw)
}
