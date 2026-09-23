package store

import "testing"

func TestBindRequiresHealthyEgress(t *testing.T) {
	es := NewEgressStore()
	as := NewAccountStore()
	es.Put(Egress{ID: "e1", ProxyURL: "socks5://x", Healthy: false})
	as.Upsert(Account{ID: "a1", App: "tg", DeviceID: "d1", Tier: TierNew})
	if err := as.BindEgress(es, "a1", "e1"); err == nil {
		t.Fatal("expected error")
	}
	es.Put(Egress{ID: "e1", ProxyURL: "socks5://x", Healthy: true})
	if err := as.BindEgress(es, "a1", "e1"); err != nil {
		t.Fatal(err)
	}
	acc, ok := as.Get("a1")
	if !ok || acc.EgressID != "e1" {
		t.Fatalf("%+v %v", acc, ok)
	}
}
