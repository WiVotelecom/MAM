// Package speed implements a self-contained throughput test. Because NetInsight
// must work in air-gapped networks, the server both generates the download
// payload and receives the upload payload — no external speed-test server is
// required.
package speed

import (
	"io"
	"math/rand"
	"time"
)

// MaxPayloadBytes caps a single speed-test transfer to avoid unbounded memory
// or bandwidth use.
const MaxPayloadBytes = 256 << 20 // 256 MiB

// Mbps computes throughput in megabits per second from a byte count and a
// duration. It returns 0 for non-positive durations to avoid division by zero.
func Mbps(bytes int64, d time.Duration) float64 {
	if d <= 0 {
		return 0
	}
	bits := float64(bytes) * 8
	return bits / d.Seconds() / 1e6
}

// ClampSize bounds a requested payload size to the range [1, MaxPayloadBytes].
func ClampSize(n int64) int64 {
	if n <= 0 {
		return 1 << 20 // default 1 MiB
	}
	if n > MaxPayloadBytes {
		return MaxPayloadBytes
	}
	return n
}

// randReader streams pseudo-random bytes without allocating the whole payload.
type randReader struct {
	remaining int64
	rng       *rand.Rand
}

func (r *randReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, io.EOF
	}
	n := len(p)
	if int64(n) > r.remaining {
		n = int(r.remaining)
	}
	for i := 0; i < n; i++ {
		p[i] = byte(r.rng.Intn(256))
	}
	r.remaining -= int64(n)
	return n, nil
}

// PayloadReader returns an io.Reader that yields exactly size pseudo-random
// bytes, suitable for streaming a download payload to a client.
func PayloadReader(size int64) io.Reader {
	return &randReader{
		remaining: ClampSize(size),
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}
