package squirrel

import (
	"crypto/subtle"
	"log/slog"
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
	if o.Secret == "" {
		// subtle.ConstantTimeCompare("", "") returns 1, so an unset secret
		// would authenticate every request — including one with no token
		// header at all. Refuse to mount rather than serve an effectively
		// open arrival hook; putting that check here means the safety
		// doesn't depend on every caller remembering to set a secret.
		slog.Error("presence: refusing to mount with an empty secret", "path", path)
		return
	}
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

		// Always through a goroutine, even with no Delay: Go doesn't flush a
		// bodyless response until the handler returns, so a synchronous call
		// here would hold the response open despite reading as
		// non-blocking. Recovering here matters too — OnArrive will touch
		// the database for nudge scheduling, http.go's recover only guards
		// the goroutine serving the mux, and an unrecovered panic in any
		// goroutine takes the whole process down. That's not hypothetical:
		// a byte-slicing bug in the chore-name parser did exactly that in
		// phase 3.
		go func() {
			defer func() {
				if rec := recover(); rec != nil {
					slog.Error("presence OnArrive panicked", "panic", rec)
				}
			}()
			if o.Delay > 0 {
				time.Sleep(o.Delay)
			}
			o.OnArrive()
		}()
	})
}
