package speed

import (
	"io"
	"testing"
	"time"
)

func TestMbps(t *testing.T) {
	// 1,000,000 bytes in 1 second = 8 Mbps
	if got := Mbps(1_000_000, time.Second); got != 8 {
		t.Errorf("Mbps = %v, want 8", got)
	}
	if got := Mbps(1000, 0); got != 0 {
		t.Errorf("Mbps with zero duration = %v, want 0", got)
	}
}

func TestClampSize(t *testing.T) {
	if got := ClampSize(0); got != 1<<20 {
		t.Errorf("default size = %d, want %d", got, 1<<20)
	}
	if got := ClampSize(-5); got != 1<<20 {
		t.Errorf("negative size = %d, want default", got)
	}
	if got := ClampSize(MaxPayloadBytes + 1); got != MaxPayloadBytes {
		t.Errorf("oversize = %d, want %d", got, MaxPayloadBytes)
	}
	if got := ClampSize(4096); got != 4096 {
		t.Errorf("in-range size = %d, want 4096", got)
	}
}

func TestPayloadReaderLength(t *testing.T) {
	r := PayloadReader(5000)
	n, err := io.Copy(io.Discard, r)
	if err != nil {
		t.Fatalf("copy: %v", err)
	}
	if n != 5000 {
		t.Errorf("read %d bytes, want 5000", n)
	}
}
