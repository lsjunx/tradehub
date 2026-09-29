package identity

import (
	"path/filepath"
	"testing"
)

func TestLoadOrCreate_CreatesAndReuses(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "device_id")
	id1, err := LoadOrCreate(path)
	if err != nil || id1 == "" {
		t.Fatalf("first: %v %q", err, id1)
	}
	id2, err := LoadOrCreate(path)
	if err != nil || id2 != id1 {
		t.Fatalf("reuse: got %q want %q err=%v", id2, id1, err)
	}
}
