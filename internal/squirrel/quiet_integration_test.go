//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// setAside parks a note and backdates when it happened, so a test can be about
// three weeks ago without waiting three weeks.
func setAside(t *testing.T, store *squirrel.Store, p int64, text string,
	state squirrel.ItemState, ago time.Duration) int64 {
	t.Helper()
	ctx := context.Background()
	sub := "sub-seed"
	id, err := store.InsertItemReturningID(ctx, squirrel.Item{
		Transport: squirrel.ScreenTransport, SenderID: &sub, PersonID: &p,
		RawText: text, ReceivedAt: time.Now(), Payload: []byte(`{}`),
	})
	require.NoError(t, err)
	_, err = store.HoldItem(ctx, p, id, state, "the surgery", time.Now())
	require.NoError(t, err)
	_, err = store.Pool().Exec(ctx,
		`update items set state_at = $2 where id = $1`, id, time.Now().Add(-ago))
	require.NoError(t, err)
	return id
}

func TestSomethingWaitingLongEnoughComesBack(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	setAside(t, store, p, "the referral", squirrel.ItemWaiting, 22*24*time.Hour)

	got, found, err := store.GoneQuiet(context.Background(), p, time.Now())
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "the referral", got.Text)
	require.Equal(t, "the surgery", got.Because)
	require.InDelta(t, (22 * 24 * time.Hour).Hours(), got.Since.Hours(), 2)
}

func TestSomethingParkedRecentlyStaysQuiet(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	setAside(t, store, p, "the referral", squirrel.ItemWaiting, 5*24*time.Hour)

	_, found, err := store.GoneQuiet(context.Background(), p, time.Now())
	require.NoError(t, err)
	require.False(t, found, "it spoke up about something parked five days ago")
}

// Blocked is shorter than waiting: a thing you are blocked on is usually
// something you can unblock, where waiting is somebody else's move.
func TestBlockedSpeaksUpSoonerThanWaiting(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	ago := 16 * 24 * time.Hour

	setAside(t, store, p, "the blocked one", squirrel.ItemBlocked, ago)
	got, found, err := store.GoneQuiet(context.Background(), p, time.Now())
	require.NoError(t, err)
	require.True(t, found, "blocked did not speak up after sixteen days")
	require.Equal(t, "the blocked one", got.Text)

	// The same age, waiting, is still inside its own window.
	store2 := withStore(t)
	p2 := owner(t, store2)
	setAside(t, store2, p2, "the waiting one", squirrel.ItemWaiting, ago)
	_, found, err = store2.GoneQuiet(context.Background(), p2, time.Now())
	require.NoError(t, err)
	require.False(t, found, "waiting spoke up on blocked's schedule")
}

// **someday never speaks up.** It is the state that means "not now, and do not
// ask me"; a product that came back to it in three weeks would have taken the
// one place you can put something down and made it a delayed nag.
func TestSomedayNeverSpeaksUp(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	setAside(t, store, p, "learn the piano", squirrel.ItemSomeday, 200*24*time.Hour)

	_, found, err := store.GoneQuiet(context.Background(), p, time.Now())
	require.NoError(t, err)
	require.False(t, found, "someday was brought back up")
}

// The oldest one, not all of them. This is a sentence, not a second pile.
func TestTheOldestOneComesBack(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	setAside(t, store, p, "the newer one", squirrel.ItemWaiting, 22*24*time.Hour)
	setAside(t, store, p, "the older one", squirrel.ItemWaiting, 60*24*time.Hour)

	got, found, err := store.GoneQuiet(context.Background(), p, time.Now())
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "the older one", got.Text)
}

func TestStillWaitingMovesTheClockAndNothingElse(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	id := setAside(t, store, p, "the referral", squirrel.ItemWaiting, 22*24*time.Hour)

	ok, err := store.StillHolding(ctx, p, id, time.Now())
	require.NoError(t, err)
	require.True(t, ok)

	_, found, err := store.GoneQuiet(ctx, p, time.Now())
	require.NoError(t, err)
	require.False(t, found, "it spoke up again straight after being told still")

	// The note has not moved: same state, same reason, still set aside.
	held, _, err := store.HeldItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, held, 1)
	require.Equal(t, squirrel.ItemWaiting, held[0].State)
	require.Equal(t, "the surgery", held[0].Because)
}

// Somebody else's is never yours.
func TestWhatHasGoneQuietBelongsToOnePerson(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	mine := owner(t, store)
	theirs, err := store.PersonForLogin(ctx, "sub-theirs", "theirs")
	require.NoError(t, err)
	id := setAside(t, store, theirs, "their referral", squirrel.ItemWaiting, 60*24*time.Hour)

	_, found, err := store.GoneQuiet(ctx, mine, time.Now())
	require.NoError(t, err)
	require.False(t, found, "somebody else's parked note was brought to me")

	moved, err := store.StillHolding(ctx, mine, id, time.Now())
	require.NoError(t, err)
	require.False(t, moved, "somebody else reset the clock on their note")
}
