package transport

import (
	"testing"
	"time"
)

func TestRateLimiterAllowsUpToTheBurstThenRefuses(t *testing.T) {
	r := newRateLimiter(3, 1)
	at := time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC)
	r.now = func() time.Time { return at }

	for i := range 3 {
		if !r.allow() {
			t.Fatalf("request %d was refused within the burst", i)
		}
	}
	if r.allow() {
		t.Fatal("a request past the burst was allowed")
	}
}

func TestRateLimiterRefillsOverTime(t *testing.T) {
	r := newRateLimiter(1, 1)
	at := time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC)
	r.now = func() time.Time { return at }

	if !r.allow() {
		t.Fatal("the first request in an empty test was refused")
	}
	if r.allow() {
		t.Fatal("a second request was allowed with no time having passed")
	}

	at = at.Add(time.Second)
	if !r.allow() {
		t.Fatal("a request a full second later was still refused")
	}
}

func TestRateLimiterNeverExceedsItsMax(t *testing.T) {
	r := newRateLimiter(2, 100)
	at := time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC)
	r.now = func() time.Time { return at }

	at = at.Add(time.Hour)
	for i := range 2 {
		if !r.allow() {
			t.Fatalf("request %d was refused after a long idle period", i)
		}
	}
	if r.allow() {
		t.Fatal("a long idle period let the bucket hold more than its max")
	}
}
