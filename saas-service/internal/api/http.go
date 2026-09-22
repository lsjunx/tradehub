package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	"github.com/local/saas-service/internal/gatewayhub"
	"github.com/local/saas-service/internal/store"
)

// Server HTTP/WSS API。
type Server struct {
	Store *store.Store
	Hub   *gatewayhub.Hub

	uiMu sync.Mutex
	uis  map[*websocket.Conn]struct{}
}

func NewServer(st *store.Store, hub *gatewayhub.Hub) *Server {
	s := &Server{
		Store: st,
		Hub:   hub,
		uis:   make(map[*websocket.Conn]struct{}),
	}
	hub.OnEvent = s.broadcastUI
	return s
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/devices", s.handleDevices)
	mux.HandleFunc("/api/devices/", s.handleDeviceCommand)
	mux.HandleFunc("/ws/gateway", s.handleGatewayWS)
	mux.HandleFunc("/ws/ui", s.handleUIWS)
	return mux
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.Store.List())
}

func (s *Server) handleDeviceCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// path: /api/devices/{id}/commands
	path := r.URL.Path
	const prefix = "/api/devices/"
	rest := path[len(prefix):]
	var deviceID string
	if i := len(rest); i > 0 {
		if j := indexByte(rest, '/'); j >= 0 {
			deviceID = rest[:j]
			if rest[j+1:] != "commands" {
				http.NotFound(w, r)
				return
			}
		} else {
			http.NotFound(w, r)
			return
		}
	}
	dev, ok := s.Store.Get(deviceID)
	if !ok {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}
	if !dev.Online {
		http.Error(w, "device offline", http.StatusNotFound)
		return
	}
	if !s.Hub.HasGateway(dev.GatewayID) {
		http.Error(w, "gateway not connected", http.StatusConflict)
		return
	}
	var body struct {
		Action string `json:"action"`
		Args   string `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	cmdID := uuid.NewString()
	wait := s.Hub.RegisterPending(cmdID)
	err := s.Hub.SendJSON(dev.GatewayID, "command", map[string]string{
		"device_id": deviceID,
		"cmd_id":    cmdID,
		"action":    body.Action,
		"args":      body.Args,
	})
	if err != nil {
		s.Hub.CancelPending(cmdID)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	select {
	case res, ok := <-wait:
		if !ok {
			writeJSON(w, map[string]any{"ok": false, "message": "timeout", "cmd_id": cmdID})
			return
		}
		writeJSON(w, res)
	case <-time.After(10 * time.Second):
		s.Hub.CancelPending(cmdID)
		writeJSON(w, map[string]any{"ok": false, "message": "timeout", "cmd_id": cmdID, "device_id": deviceID})
	}
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func (s *Server) handleGatewayWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	ctx := r.Context()
	var gatewayID string
	defer func() {
		if gatewayID != "" {
			s.Hub.RemoveGateway(gatewayID, conn)
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
		switch env.Type {
		case "gateway_hello":
			var p struct {
				GatewayID string `json:"gateway_id"`
			}
			_ = json.Unmarshal(env.Payload, &p)
			gatewayID = p.GatewayID
			s.Hub.SetGateway(gatewayID, conn)
			log.Printf("gateway hello %s", gatewayID)
		case "device_register":
			var p struct {
				DeviceID  string `json:"device_id"`
				IP        string `json:"ip"`
				Port      int32  `json:"port"`
				GatewayID string `json:"gateway_id"`
			}
			_ = json.Unmarshal(env.Payload, &p)
			d := s.Store.UpsertRegister(p.DeviceID, p.IP, p.Port, p.GatewayID)
			s.Hub.Emit("device_updated", d)
		case "device_heartbeat":
			var p struct {
				DeviceID  string `json:"device_id"`
				Status    string `json:"status"`
				GatewayID string `json:"gateway_id"`
			}
			_ = json.Unmarshal(env.Payload, &p)
			d := s.Store.TouchHeartbeat(p.DeviceID, p.Status, p.GatewayID)
			s.Hub.Emit("device_updated", d)
		case "device_offline":
			var p struct {
				DeviceID  string `json:"device_id"`
				GatewayID string `json:"gateway_id"`
			}
			_ = json.Unmarshal(env.Payload, &p)
			if d := s.Store.MarkOffline(p.DeviceID, p.GatewayID); d != nil {
				s.Hub.Emit("device_updated", d)
			}
		case "command_result":
			var p gatewayhub.CommandResult
			_ = json.Unmarshal(env.Payload, &p)
			s.Hub.Complete(p)
		}
	}
}

func (s *Server) handleUIWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	s.uiMu.Lock()
	s.uis[conn] = struct{}{}
	s.uiMu.Unlock()
	defer func() {
		s.uiMu.Lock()
		delete(s.uis, conn)
		s.uiMu.Unlock()
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}()
	ctx := r.Context()
	for {
		if _, _, err := conn.Read(ctx); err != nil {
			return
		}
	}
}

func (s *Server) broadcastUI(eventType string, payload any) {
	msg, err := json.Marshal(map[string]any{"type": eventType, "payload": payload})
	if err != nil {
		return
	}
	s.uiMu.Lock()
	defer s.uiMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for c := range s.uis {
		_ = c.Write(ctx, websocket.MessageText, msg)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
