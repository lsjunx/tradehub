package logic

import (
	"encoding/json"
	"net/http"
	"testing"

	bizerr "github.com/local/saas-service/internal/common/errors"
	"github.com/local/saas-service/internal/store"
)

type fakeEgressHub struct {
	gateways map[string]bool
	sends    []map[string]string
}

func (f *fakeEgressHub) HasGateway(id string) bool { return f.gateways[id] }

func (f *fakeEgressHub) SendJSON(_ string, typ string, payload any) error {
	m, _ := payload.(map[string]string)
	cp := make(map[string]string, len(m))
	for k, v := range m {
		cp[k] = v
	}
	f.sends = append(f.sends, cp)
	return nil
}

func TestBindAndPush(t *testing.T) {
	es := store.NewEgressStore()
	es.Put(store.Egress{ID: "e1", ProxyURL: "socks5://pool", Healthy: true})
	as := store.NewAccountStore()
	as.Upsert(store.Account{ID: "a1", DeviceID: "d1"})

	t.Run("bind_without_push_when_offline", func(t *testing.T) {
		st := store.New()
		st.UpsertRegister("d1", "127.0.0.1", 1, "gw1")
		hub := &fakeEgressHub{gateways: map[string]bool{}}
		l := NewEgressLogic(st, hub, as, es)
		if err := l.BindAndPush("a1", "e1"); err != nil {
			t.Fatal(err)
		}
		if len(hub.sends) != 0 {
			t.Fatalf("expected no push, got %d", len(hub.sends))
		}
	})

	t.Run("push_egress_apply_when_online", func(t *testing.T) {
		st := store.New()
		st.UpsertRegister("d1", "127.0.0.1", 1, "gw1")
		hub := &fakeEgressHub{gateways: map[string]bool{"gw1": true}}
		l := NewEgressLogic(st, hub, as, es)
		if err := l.BindAndPush("a1", "e1"); err != nil {
			t.Fatal(err)
		}
		if len(hub.sends) != 1 {
			t.Fatalf("expected 1 send, got %d", len(hub.sends))
		}
		cmd := hub.sends[0]
		if cmd["action"] != "egress.apply" || cmd["device_id"] != "d1" || cmd["cmd_id"] == "" {
			t.Fatalf("cmd %+v", cmd)
		}
		var args map[string]string
		if err := json.Unmarshal([]byte(cmd["args"]), &args); err != nil {
			t.Fatal(err)
		}
		if args["account_id"] != "a1" || args["egress_id"] != "e1" || args["proxy_url"] != "socks5://pool" {
			t.Fatalf("args %+v", args)
		}
	})

	t.Run("bind_rejects_unhealthy", func(t *testing.T) {
		bad := store.NewEgressStore()
		bad.Put(store.Egress{ID: "bad", ProxyURL: "x", Healthy: false})
		l := NewEgressLogic(store.New(), &fakeEgressHub{}, as, bad)
		err := l.BindAndPush("a1", "bad")
		ae, ok := bizerr.As(err)
		if !ok || ae.Code != http.StatusBadRequest {
			t.Fatalf("got %v", err)
		}
	})
}
