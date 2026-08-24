//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Deleting an appointment must never delete the owner's words. The note
// returns to the pile instead, which is what happens when it is over anyway.
func TestDeletingAFixedPointKeepsTheNotes(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	m, err := store.CreateMoment(ctx, p, squirrel.Moment{
		Label: "dentist", Starts: time.Now().Add(3 * time.Hour),
		Travel: 15 * time.Minute, Ready: 10 * time.Minute,
	})
	require.NoError(t, err)
	id := taskOf(t, store, p, "the referral letter")

	_, err = store.Pool().Exec(ctx, `update items set moment_id = $1 where id = $2`, m.ID, id)
	require.NoError(t, err)
	_, err = store.Pool().Exec(ctx, `delete from moments where id = $1`, m.ID)
	require.NoError(t, err)

	var moment *int64
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select moment_id from items where id = $1`, id).Scan(&moment))
	require.Nil(t, moment, "the words survive the appointment")
}

// aFixedPoint keeps the fixtures honest: a bare Moment{} has zero travel and
// ready, so WarnAt equals the start time and every window assertion is wrong.
func aFixedPoint(t *testing.T, store *squirrel.Store, personID int64, label string, in time.Duration) squirrel.Moment {
	t.Helper()
	m, err := store.CreateMoment(context.Background(), personID, squirrel.Moment{
		Label: label, Starts: time.Now().Add(in),
		Travel: 15 * time.Minute, Ready: 10 * time.Minute,
	})
	require.NoError(t, err)
	return m
}

// noteOf is taskOf without the promotion: a thought in the pile, which is what
// gets pointed at an appointment.
func noteOf(t *testing.T, store *squirrel.Store, personID int64, text string) int64 {
	t.Helper()
	id, err := store.InsertItemReturningID(context.Background(), squirrel.Item{
		Transport: "screen", PersonID: &personID, RawText: text,
		Payload: []byte(squirrel.ScreenCapture), ReceivedAt: time.Now(),
	})
	require.NoError(t, err)
	return id
}

func someoneElse(t *testing.T, store *squirrel.Store) int64 {
	t.Helper()
	id, err := store.SeedOwner(context.Background(), "someone-else", nil)
	require.NoError(t, err)
	return id
}

func TestANoteCanBePointedAtAFixedPointAndBack(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	m := aFixedPoint(t, store, p, "dentist", 3*time.Hour)
	id := noteOf(t, store, p, "the referral letter")

	ok, err := store.AttachNote(ctx, p, id, m.ID)
	require.NoError(t, err)
	require.True(t, ok)

	notes, err := store.NotesFor(ctx, p, m.ID)
	require.NoError(t, err)
	require.Len(t, notes, 1)
	require.Equal(t, "the referral letter", notes[0].RawText)

	ok, err = store.DetachNote(ctx, p, id)
	require.NoError(t, err)
	require.True(t, ok)

	notes, err = store.NotesFor(ctx, p, m.ID)
	require.NoError(t, err)
	require.Empty(t, notes, "detaching is the reversal, and every transition here reverses")
}

// Somebody else's row is not yours to move, and saying so with a boolean
// rather than an error is the shape HoldItem already uses.
func TestPointingAtAFixedPointIsOnlyEverYourOwn(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	stranger := someoneElse(t, store)

	m := aFixedPoint(t, store, p, "dentist", 3*time.Hour)
	id := noteOf(t, store, p, "the referral letter")

	ok, err := store.AttachNote(ctx, stranger, id, m.ID)
	require.NoError(t, err)
	require.False(t, ok)
}

// A note with somewhere to be is not waiting to be decided about.
func TestThePileHidesANoteThatIsOnAFixedPoint(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	m := aFixedPoint(t, store, p, "dentist", 3*time.Hour)
	id := noteOf(t, store, p, "the referral letter")

	before, _, err := store.OpenItems(ctx, p, 20)
	require.NoError(t, err)
	require.Len(t, before, 1)

	_, err = store.AttachNote(ctx, p, id, m.ID)
	require.NoError(t, err)

	after, _, err := store.OpenItems(ctx, p, 20)
	require.NoError(t, err)
	require.Empty(t, after, "it has somewhere to be")
}

// And it is waiting again the moment the appointment is over, with no write and
// nothing running on a schedule. This is the half that is a read rule, so it is
// the half most worth a test.
func TestThePileShowsItAgainOnceTheFixedPointIsPast(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	past := aFixedPoint(t, store, p, "dentist", -2*time.Hour)
	id := noteOf(t, store, p, "the referral letter")
	_, err := store.AttachNote(ctx, p, id, past.ID)
	require.NoError(t, err)

	items, _, err := store.OpenItems(ctx, p, 20)
	require.NoError(t, err)
	require.Len(t, items, 1, "the occasion happened; it is a thought again")
}

// The same rule on the pile's own paging query. Two queries that disagree about
// what the pile holds is the bug this package's comments warn about most.
func TestPagingThroughThePileObeysTheSameRule(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	m := aFixedPoint(t, store, p, "dentist", 3*time.Hour)
	// The attached one is the OLDER of the two, because this query returns what
	// is older than the cursor. With it the other way round the query returns
	// nothing whatever the clause says, and the test passes having exercised
	// none of it — which is what the first draft of this did.
	attached := noteOf(t, store, p, "the referral letter")
	cursor := noteOf(t, store, p, "buy milk")

	reached, _, err := store.OpenItemsAfter(ctx, p, cursor, 20)
	require.NoError(t, err)
	require.Len(t, reached, 1, "the fixture only means something if the cursor reaches it")

	_, err = store.AttachNote(ctx, p, attached, m.ID)
	require.NoError(t, err)

	items, _, err := store.OpenItemsAfter(ctx, p, cursor, 20)
	require.NoError(t, err)
	require.Empty(t, items)
}
