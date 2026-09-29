// Package handle 是 Gateway 的 I/O 边界：TLS 接受 Board 连接并读帧。
package handle

import (
	"crypto/tls"
	"log"
	"net"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"github.com/tradehub/gateway/internal/frame"
	"github.com/tradehub/gateway/internal/logic"
	boardv1 "github.com/tradehub/gateway/internal/pb/board/v1"
)

// BoardServer Board 侧 TLS 服务端。
type BoardServer struct {
	Listen   string
	CertFile string
	KeyFile  string
	Bridge   *logic.Bridge
}

// ListenAndServe 【关键】监听 Board TLS，每条连接一个 goroutine。
func (s *BoardServer) ListenAndServe() error {
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
	log.Printf("Board TLS 监听 %s", s.Listen)
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.serveConn(conn)
	}
}

// serveConn 【关键·收 Board】读 length-prefix + protobuf，交给 logic 上行处理。
func (s *BoardServer) serveConn(conn net.Conn) {
	sessionID := uuid.NewString()
	var deviceID string
	defer func() {
		_ = conn.Close()
		s.Bridge.NotifyBoardOffline(deviceID, sessionID)
	}()

	for {
		raw, err := frame.ReadFrame(conn)
		if err != nil {
			log.Printf("Board 读帧结束: %v", err)
			return
		}
		var env boardv1.Envelope
		if err := proto.Unmarshal(raw, &env); err != nil {
			log.Printf("Board protobuf 解析失败: %v", err)
			return
		}
		if err := s.Bridge.HandleBoardEnvelope(conn, sessionID, &deviceID, &env); err != nil {
			log.Printf("上行处理失败: %v", err)
			return
		}
	}
}
