package logic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	bizerr "github.com/local/saas-service/internal/common/errors"
	"github.com/local/saas-service/internal/gatewayhub"
	"github.com/local/saas-service/internal/policy"
	"github.com/local/saas-service/internal/store"
)

func newTestDeviceLogic(t *testing.T, gatewayID string) (*DeviceLogic, *store.AccountStore) {
	t.Helper()
	st := store.New()
	st.UpsertRegister("dev1", "127.0.0.1", 1, gatewayID)
	hub := gatewayhub.New()
	wireTestGateway(t, hub, gatewayID)
	cmds := store.NewCommandStore()
	accounts := store.NewAccountStore()
	egress := store.NewEgressStore()
	return NewDeviceLogic(st, hub, cmds, accounts, egress, policy.New()), accounts
}

func wireTestGateway(t *testing.T, hub *gatewayhub.Hub, gatewayID string) {
	t.Helper()
	ready := make(chan *websocket.Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		ready <- conn
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	client, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close(websocket.StatusNormalClosure, "") })

	select {
	case gwConn := <-ready:
		hub.SetGateway(gatewayID, gwConn)
	case <-time.After(2 * time.Second):
		t.Fatal("gateway accept timeout")
	}
}

func TestAcceptCommandEchoWithoutAccount(t *testing.T) {
	l, _ := newTestDeviceLogic(t, "gw1")
	res, err := l.AcceptCommand("dev1", CommandRequest{Action: "echo", Args: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	l.Hub.Complete(gatewayhub.CommandResult{CmdID: res.CmdID, DeviceID: "dev1", Ok: true})
}

func TestAcceptCommandRequiresAccount(t *testing.T) {
	l, _ := newTestDeviceLogic(t, "gw1")
	_, err := l.AcceptCommand("dev1", CommandRequest{Action: "ui.tap"})
	if err == nil {
		t.Fatal("expected error")
	}
	ae, ok := bizerr.As(err)
	if !ok || ae.Code != http.StatusConflict || ae.Message != "account_required" {
		t.Fatalf("got %v", err)
	}
}

func TestAcceptCommandPolicyDenyDeadAccount(t *testing.T) {
	l, accounts := newTestDeviceLogic(t, "gw1")
	accounts.Upsert(store.Account{ID: "a1", Tier: store.TierDead})
	_, err := l.AcceptCommand("dev1", CommandRequest{Action: "echo", AccountID: "a1"})
	if err == nil {
		t.Fatal("expected error")
	}
	ae, ok := bizerr.As(err)
	if !ok || ae.Message != "account_blocked" {
		t.Fatalf("got %v", err)
	}
}
