package pit

import (
	"sync"
	"time"
)

// limiter limits requests
type limiter interface {
	// allow allows a new request if true
	allow() bool
}

// nopeLimiter never limits requests
type nopeLimiter bool

func (nopeLimiter) allow() bool { return true }

type tokenLimiter struct {
	mut   sync.Mutex
	limit float64
	burst int
	token int
	last  time.Time
}

// newTokenLimiter returns a full token limiter
func newTokenLimiter(qps int) *tokenLimiter {
	return &tokenLimiter{
		limit: float64(qps),
		burst: qps,
		token: qps,
		last:  time.Now(),
	}
}

func (t *tokenLimiter) allow() bool {
	t.mut.Lock()
	defer t.mut.Unlock()

	if t.token -= t.revoked(time.Since(t.last)); t.token < 0 {
		t.token = 0
	}

	if t.token < t.burst {
		t.token++
		t.last = time.Now()
		return true
	}

	return false
}

// revoked is a unit conversion function from a time duration to the number of tokens
// which could be accumulated during that duration at a rate of limit tokens per second.
func (t *tokenLimiter) revoked(d time.Duration) int {
	// Split the integer and fractional parts ourself to minimize rounding errors.
	sec := float64(d/time.Second) * t.limit
	nsec := float64(d%time.Second) * t.limit
	return int(sec + nsec/1e9)
}
