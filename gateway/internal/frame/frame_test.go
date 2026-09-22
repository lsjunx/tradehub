package frame

import (
	"bytes"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	want := []byte("hello-proto")
	if err := WriteFrame(&buf, want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFrame(&buf)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestReadFrame_RejectsZeroLength(t *testing.T) {
	var buf bytes.Buffer
	buf.Write([]byte{0, 0, 0, 0})
	if _, err := ReadFrame(&buf); err == nil {
		t.Fatal("expected error")
	}
}
