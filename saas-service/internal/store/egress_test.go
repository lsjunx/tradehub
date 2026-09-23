package store

import "testing"

func TestEgressPutAndGet(t *testing.T) {
	es := NewEgressStore()
	es.Put(Egress{ID: "e1", ProxyURL: "socks5://x", Region: "us", Healthy: true})
	got, ok := es.Get("e1")
	if !ok || got.ProxyURL != "socks5://x" || got.Region != "us" || !got.Healthy {
		t.Fatalf("%+v %v", got, ok)
	}
}
