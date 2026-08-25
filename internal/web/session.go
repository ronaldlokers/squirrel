package web

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A session, remembered for a minute.
//
// The header-based design never touched Postgres on the request path, and the
// comment on ResolvePerson says why: the request path not touching Postgres is
// what makes an outage survivable. A session table gives that up. This gives
// most of it back.
//
// What it costs is stated so it can be found again: signing out elsewhere
// takes up to a minute to bite here. That is the right trade for what
// revocation is actually for — "I am done with that demo account" rather than
// an incident. If it ever needs to be immediate, delete this file and call the
// store directly.
const (
	// cacheFor is how long an answer is trusted without asking again.
	cacheFor = time.Minute
	// cacheMost is how many are held. A map nothing evicts is a leak found by
	// a pod being OOM-killed months later, which the door cache learned in the
	// same week this was written.
	cacheMost = 512
)

type remembered struct {
	session squirrel.Session
	known   bool
	until   time.Time
}

type sessions struct {
	read  func(context.Context, []byte, time.Time) (squirrel.Session, bool, error)
	life  time.Duration
	most  int
	mu    sync.Mutex
	known map[string]remembered
	order []string
}

func newSessions(read func(context.Context, []byte, time.Time) (squirrel.Session, bool, error), life time.Duration, most int) *sessions {
	return &sessions{read: read, life: life, most: most, known: map[string]remembered{}}
}

// For resolves a token, from memory when it can.
//
// A read that fails falls back on what was remembered, which is the whole
// point: a Postgres blip must not sign you out mid-thought. The fallback is
// bounded by the session's own expiry rather than by the cache's, so an outage
// cannot keep a session alive past the moment it was going to end anyway.
// Nothing remembered and a read that fails is a refusal: the cache softens an
// outage, it does not invent an answer during one.
func (c *sessions) For(ctx context.Context, token []byte, at time.Time) (squirrel.Session, bool, error) {
	key := hex.EncodeToString(token)

	c.mu.Lock()
	had, seen := c.known[key]
	c.mu.Unlock()
	if seen && at.Before(had.until) {
		return had.session, had.known, nil
	}

	session, known, err := c.read(ctx, token, at)
	if err != nil {
		switch {
		case seen && !had.known:
			// Remembered as nobody. Repeating that is a refusal, which is the
			// safe answer anyway.
			return squirrel.Session{}, false, nil
		case seen && at.Before(had.session.ExpiresAt):
			return had.session, true, nil
		}
		return squirrel.Session{}, false, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if _, already := c.known[key]; !already {
		c.order = append(c.order, key)
	}
	// Nobody is remembered too, or a stranger with a made-up cookie is a
	// database read on every request they make.
	c.known[key] = remembered{session: session, known: known, until: at.Add(c.life)}
	for len(c.order) > c.most {
		delete(c.known, c.order[0])
		c.order = c.order[1:]
	}
	return session, known, nil
}

// Forget drops one, for the person signing out. Immediate whatever the cache
// would otherwise have said.
func (c *sessions) Forget(token []byte) {
	key := hex.EncodeToString(token)
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.known, key)
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
}

// held is how many are remembered, for the test that proves there is a bottom.
func (c *sessions) held() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.known)
}
