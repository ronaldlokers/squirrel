package squirrel

import (
	"crypto/subtle"
	"net/http"
	"sync"
	"time"
)

// PresenceOptions configures the arrival webhook. Delay is how long to wait
// after an arrival before nudging — you have a coat on — and Debounce is how
// long to ignore further arrivals.
type PresenceOptions struct {
	Secret   string
	Debounce time.Duration
	Delay    time.Duration
	OnArrive func()
	Now      func() time.Time
}

// MountPresence adds the arrival route.
//
// This is the one inbound event in this system that is deliberately NOT
// spooled. Everything else is written to disk and fsynced before it is
// acknowledged, because losing it means losing a thought. A presence ping is
// not a thought: losing one costs a nudge, and the evening message catches the
// same day. Spooling it would also give it an items row, which would put "you
// came home" in the capture list — the same mistake phase 3 spent a fix
// removing when button taps ended up there.
func MountPresence(s *Server, path string, o PresenceOptions) {
	if o.Now == nil {
		o.Now = time.Now
	}
	var (
		mu   sync.Mutex
		last time.Time
	)

	s.Post(path, func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-Squirrel-Token")
		if subtle.ConstantTimeCompare([]byte(got), []byte(o.Secret)) != 1 {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		now := o.Now()
		mu.Lock()
		within := !last.IsZero() && now.Sub(last) < o.Debounce
		if !within {
			last = now
		}
		mu.Unlock()

		w.WriteHeader(http.StatusNoContent)
		if within || o.OnArrive == nil {
			return
		}

		if o.Delay <= 0 {
			o.OnArrive()
			return
		}
		go func() {
			time.Sleep(o.Delay)
			o.OnArrive()
		}()
	})
}
