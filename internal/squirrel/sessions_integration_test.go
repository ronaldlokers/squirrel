//go:build integration

package squirrel_test

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func hashed(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// A session opens, resolves to its person, and carries the sub the capture
// path needs.
func TestASessionResolvesToItsPerson(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	now := time.Now()

	require.NoError(t, store.OpenSession(ctx, p, "sub-123", hashed("a"), now, 30*24*time.Hour))

	got, found, err := store.SessionFor(ctx, hashed("a"), now)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, p, got.PersonID)
	require.Equal(t, "sub-123", got.Sub)
}

// A token nobody opened resolves to nobody. Not an error — an unknown cookie
// is the ordinary state of a stranger.
func TestAnUnknownTokenIsNobody(t *testing.T) {
	store := withStore(t)
	_, found, err := store.SessionFor(context.Background(), hashed("never opened"), time.Now())
	require.NoError(t, err)
	require.False(t, found)
}

// Expiry is enforced by the query rather than by the caller, so there is no
// path that forgets to check it.
func TestAnExpiredSessionIsNobody(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	long := time.Now().Add(-48 * time.Hour)

	require.NoError(t, store.OpenSession(ctx, p, "sub-123", hashed("a"), long, time.Hour))

	_, found, err := store.SessionFor(ctx, hashed("a"), time.Now())
	require.NoError(t, err)
	require.False(t, found, "an expired session still resolved")
}

// Using a session pushes its expiry out, so a session in daily use never
// expires under you and one left alone for a month is gone.
func TestUsingASessionKeepsItAlive(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	opened := time.Now().Add(-20 * 24 * time.Hour)

	require.NoError(t, store.OpenSession(ctx, p, "sub-123", hashed("a"), opened, 30*24*time.Hour))
	_, found, err := store.SessionFor(ctx, hashed("a"), time.Now())
	require.NoError(t, err)
	require.True(t, found)

	// Twenty-five days after it was opened, and still alive because it was
	// touched five days ago.
	_, found, err = store.SessionFor(ctx, hashed("a"), time.Now().Add(25*24*time.Hour))
	require.NoError(t, err)
	require.True(t, found, "a session in use expired from when it was opened")
}

// Signing out ends it immediately, whatever the expiry says.
func TestEndingASessionEndsIt(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	require.NoError(t, store.OpenSession(ctx, p, "sub-123", hashed("a"), time.Now(), time.Hour))

	require.NoError(t, store.EndSession(ctx, hashed("a")))

	_, found, err := store.SessionFor(ctx, hashed("a"), time.Now())
	require.NoError(t, err)
	require.False(t, found)
}

// Ending one nobody opened is not an error. Signing out twice is a thing
// somebody does, and the second one must not be a failure page.
func TestEndingAnUnknownSessionIsFine(t *testing.T) {
	store := withStore(t)
	require.NoError(t, store.EndSession(context.Background(), hashed("never opened")))
}

// The table does not grow forever.
func TestReapingRemovesTheExpired(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	old := time.Now().Add(-48 * time.Hour)
	require.NoError(t, store.OpenSession(ctx, p, "s", hashed("old"), old, time.Hour))
	require.NoError(t, store.OpenSession(ctx, p, "s", hashed("live"), time.Now(), time.Hour))

	gone, err := store.ReapSessions(ctx, time.Now())
	require.NoError(t, err)
	require.Equal(t, int64(1), gone)

	_, found, err := store.SessionFor(ctx, hashed("live"), time.Now())
	require.NoError(t, err)
	require.True(t, found, "reaping took a live session")
}

// The token itself is never stored. Read access to this table must not be
// read access to the product.
func TestTheTokenItselfIsNeverStored(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	require.NoError(t, store.OpenSession(ctx, p, "s", hashed("the-secret-token"), time.Now(), time.Hour))

	var rows int
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select count(*) from sessions where encode(token_sha256, 'escape') like '%the-secret-token%'`).
		Scan(&rows))
	require.Zero(t, rows, "the raw token is in the table")
}
