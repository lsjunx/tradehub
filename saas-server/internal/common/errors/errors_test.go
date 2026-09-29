package errors

import (
	"net/http"
	"testing"
)

func TestAsAndHelpers(t *testing.T) {
	err := NotFound("device not found")
	ae, ok := As(err)
	if !ok || ae.Code != http.StatusNotFound || ae.Message != "device not found" {
		t.Fatalf("%v %v", ae, ok)
	}
	if _, ok := As(nil); ok {
		t.Fatal("nil should not convert")
	}
	c := Conflict(New(http.StatusBadRequest, "x"))
	if c.Code != http.StatusConflict {
		t.Fatalf("%+v", c)
	}
}
