package config

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestValidateTLSMode(t *testing.T) {
	if err := ValidateTLSMode("development", true); err != nil {
		t.Fatal(err)
	}
	if err := ValidateTLSMode("production", true); err == nil {
		t.Fatal("expected error")
	}
	if err := ValidateTLSMode("production", false); err != nil {
		t.Fatal(err)
	}
}

func TestWarnIfInsecure(t *testing.T) {
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	WarnIfInsecure(true)
	_ = w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	got := buf.String()
	if !strings.HasPrefix(strings.TrimSpace(got), "WARNING: TLS certificate verification is disabled") {
		t.Fatalf("unexpected warning: %q", got)
	}
}
