package transport

import (
	"sync"
	"time"
)

const campfireRateBurst = 20

const campfireRateRefillPerSecond = 5.0

type rateLimiter struct {
	mu     sync.Mutex
	tokens float64
	max    float64
	rate   float64
	last   time.Time
	now    func() time.Time
}

func newRateLimiter(max, ratePerSecond float64) *rateLimiter {
	return &rateLimiter{tokens: max, max: max, rate: ratePerSecond, now: time.Now}
}

func (r *rateLimiter) allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	n := r.now()
	if r.last.IsZero() {
		r.last = n
	}
	if elapsed := n.Sub(r.last).Seconds(); elapsed > 0 {
		r.tokens = min(r.max, r.tokens+elapsed*r.rate)
		r.last = n
	}
	if r.tokens < 1 {
		return false
	}
	r.tokens--
	return true
}
