package web

import (
	"context"
	"crypto/sha256"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func tok(s string) []byte { h := sha256.Sum256([]byte(s)); return h[:] }

// A reader that answers with one session and counts how often it was asked,
// and can be told to start failing.
type countedReads struct {
	reads   int
	failing bool
	until   time.Time
}

func (c *countedReads) SessionFor(context.Context, []byte, time.Time) (squirrel.Session, bool, error) {
	c.reads++
	if c.failing {
		return squirrel.Session{}, false, errors.New("the database is unwell")
	}
	return squirrel.Session{PersonID: 7, Sub: "sub-7", ExpiresAt: c.until}, true, nil
}

func (c *countedReads) OpenSession(context.Context, int64, string, []byte, time.Time, time.Duration) error {
	return nil
}
func (c *countedReads) EndSession(context.Context, []byte) error { return nil }

// nobodyEver answers that no session exists, however often it is asked.
type nobodyEver struct{ reads int }

func (n *nobodyEver) SessionFor(context.Context, []byte, time.Time) (squirrel.Session, bool, error) {
	n.reads++
	return squirrel.Session{}, false, nil
}

func (n *nobodyEver) OpenSession(context.Context, int64, string, []byte, time.Time, time.Duration) error {
	return nil
}
func (n *nobodyEver) EndSession(context.Context, []byte) error { return nil }

// A repeated press does not repeatedly read the database. The header-based
// design never touched Postgres on the request path at all; this is how much
// of that survives a session table.
func TestASessionIsReadOnceAMinute(t *testing.T) {
	r := &countedReads{until: time.Now().Add(time.Hour)}
	c := newSessions(r, time.Minute, 8)

	for range 5 {
		got, found, err := c.For(context.Background(), tok("a"), time.Now())
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, int64(7), got.PersonID)
	}
	require.Equal(t, 1, r.reads, "it read the database on every request")
}

// And the cache is not forever. A session signed out elsewhere has to stop
// working, and a minute is how long that takes.
func TestTheCacheForgetsAfterAMinute(t *testing.T) {
	now := time.Now()
	r := &countedReads{until: now.Add(time.Hour)}
	c := newSessions(r, time.Minute, 8)

	_, _, _ = c.For(context.Background(), tok("a"), now)
	_, _, _ = c.For(context.Background(), tok("a"), now.Add(61*time.Second))

	require.Equal(t, 2, r.reads)
}

// The point of the cache: a database that is briefly unreachable does not sign
// you out mid-thought, and capture keeps spooling.
func TestAWobbleDoesNotSignYouOut(t *testing.T) {
	now := time.Now()
	r := &countedReads{until: now.Add(time.Hour)}
	c := newSessions(r, time.Minute, 8)

	_, _, _ = c.For(context.Background(), tok("a"), now)
	r.failing = true

	got, found, err := c.For(context.Background(), tok("a"), now.Add(2*time.Minute))
	require.NoError(t, err, "a wobble reached the request")
	require.True(t, found)
	require.Equal(t, int64(7), got.PersonID)
}

// The fallback is bounded by the session's own expiry rather than by the
// cache's. An outage is not a reason to honour a session that has run out.
func TestAnOutageDoesNotOutliveTheSession(t *testing.T) {
	now := time.Now()
	r := &countedReads{until: now.Add(time.Hour)}
	c := newSessions(r, time.Minute, 8)

	_, _, _ = c.For(context.Background(), tok("a"), now)
	r.failing = true

	_, _, err := c.For(context.Background(), tok("a"), now.Add(2*time.Hour))
	require.Error(t, err, "an expired session survived on a failed read")
}

// And a token this process has never seen resolve is a refusal, not a guess.
func TestAnOutageWithNothingRememberedRefuses(t *testing.T) {
	r := &countedReads{failing: true}
	c := newSessions(r, time.Minute, 8)

	_, _, err := c.For(context.Background(), tok("never seen"), time.Now())
	require.Error(t, err)
}

// Signing out is immediate for the person doing it, whatever the cache says.
func TestForgettingIsImmediate(t *testing.T) {
	r := &countedReads{until: time.Now().Add(time.Hour)}
	c := newSessions(r, time.Minute, 8)

	_, _, _ = c.For(context.Background(), tok("a"), time.Now())
	c.Forget(tok("a"))
	_, _, _ = c.For(context.Background(), tok("a"), time.Now())

	require.Equal(t, 2, r.reads, "a signed-out session was still cached")
}

// A map nothing evicts is a leak discovered by a pod being OOM-killed months
// later. The door cache learned this the same week.
func TestTheCacheHasABottom(t *testing.T) {
	r := &countedReads{until: time.Now().Add(time.Hour)}
	c := newSessions(r, time.Minute, 8)

	for i := range 40 {
		_, _, _ = c.For(context.Background(), tok(strconv.Itoa(i)), time.Now())
	}
	require.LessOrEqual(t, c.held(), 8, "it kept everything")
}

// Nobody is cached too. Otherwise a stranger with a made-up cookie is a
// database read on every request they make.
func TestNobodyIsCachedAsNobody(t *testing.T) {
	n := &nobodyEver{}
	c := newSessions(n, time.Minute, 8)

	for range 5 {
		_, found, _ := c.For(context.Background(), tok("made up"), time.Now())
		require.False(t, found)
	}
	require.Equal(t, 1, n.reads, "an unknown cookie was a read every time")
}

// unwellStore cannot answer at all.
type unwellStore struct{}

func (unwellStore) SessionFor(context.Context, []byte, time.Time) (squirrel.Session, bool, error) {
	return squirrel.Session{}, false, errors.New("the database is unwell")
}
func (unwellStore) OpenSession(context.Context, int64, string, []byte, time.Time, time.Duration) error {
	return nil
}
func (unwellStore) EndSession(context.Context, []byte) error { return nil }

// sessionForNobody answers with a row nothing should ever have written.
type sessionForNobody struct{}

func (sessionForNobody) SessionFor(context.Context, []byte, time.Time) (squirrel.Session, bool, error) {
	return squirrel.Session{PersonID: 0, ExpiresAt: time.Now().Add(time.Hour)}, true, nil
}
func (sessionForNobody) OpenSession(context.Context, int64, string, []byte, time.Time, time.Duration) error {
	return nil
}
func (sessionForNobody) EndSession(context.Context, []byte) error { return nil }
