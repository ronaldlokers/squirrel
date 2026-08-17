//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestCompletionResetsTheClock(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	overdue := time.Now().Add(twoWeeks + time.Hour)
	due, err := store.DueChores(ctx, p, overdue)
	require.NoError(t, err)
	require.Len(t, due, 1)

	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack", overdue))

	due, err = store.DueChores(ctx, p, overdue.Add(time.Hour))
	require.NoError(t, err)
	require.Empty(t, due, "completing it makes it not due")
}

// The claim the whole derived-baseline design rests on: a completion from a
// sensor resets the clock with no chore-specific code involved.
func TestSensorEventResetsTheClockToo(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	overdue := time.Now().Add(twoWeeks + time.Hour)

	_, err = store.Pool().Exec(ctx,
		`insert into events (chore_id, person_id, source, occurred_at, payload)
		 values ($1, $2, 'sensor', $3, '{"device":"roborock"}')`, c.ID, p, overdue)
	require.NoError(t, err)

	due, err := store.DueChores(ctx, p, overdue.Add(time.Hour))
	require.NoError(t, err)
	require.Empty(t, due)
}

func TestSinceDaysCountsFromTheLastCompletion(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	done := time.Now().Add(-19 * 24 * time.Hour)
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack", done))

	due, err := store.DueChores(ctx, p, time.Now())
	require.NoError(t, err)
	require.Len(t, due, 1)
	require.Equal(t, 19, due[0].SinceDays)
	require.Equal(t, 14, due[0].EveryDays)
}

func TestRetractionRestoresTheClock(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	backdateChore(t, store, c.ID, 15*24*time.Hour)

	due, err := store.DueChores(ctx, p, time.Now())
	require.NoError(t, err)
	require.Len(t, due, 1, "overdue before the completion")

	promptID, err := store.RecordPrompt(ctx, p, "9", "digest", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)

	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack", time.Now()))
	due, err = store.DueChores(ctx, p, time.Now())
	require.NoError(t, err)
	require.Empty(t, due, "the completion reset the clock")

	found, err := store.RetractCompletion(ctx, c.ID, p, promptID, time.Now())
	require.NoError(t, err)
	require.True(t, found)

	due, err = store.DueChores(ctx, p, time.Now())
	require.NoError(t, err)
	require.Len(t, due, 1, "retraction put it back")
}

func TestRetractionLeavesTheEventInTheLog(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	promptID, err := store.RecordPrompt(ctx, p, "9", "digest", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)

	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack", time.Now()))
	_, err = store.RetractCompletion(ctx, c.ID, p, promptID, time.Now())
	require.NoError(t, err)

	var total, live int
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select count(*), count(*) filter (where retracted_at is null)
		   from events where chore_id = $1`, c.ID).Scan(&total, &live))
	require.Equal(t, 1, total, "the event is still there")
	require.Equal(t, 0, live, "but it no longer counts")
}

func TestRetractingNothingIsNotAnError(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	promptID, err := store.RecordPrompt(ctx, p, "9", "digest", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)

	found, err := store.RetractCompletion(ctx, c.ID, p, promptID, time.Now())
	require.NoError(t, err, "an un-tap with nothing to undo is a no-op, not a failure")
	require.False(t, found)
}

// The finding this round of review caught: retraction must be a state
// assertion, not a counter. Campfire's webhook delivery carries no event id
// and can retry a tap, so a second RetractCompletion call against the same
// prompt has to land in the same place as the first — nothing live since
// that prompt — rather than walking back to an earlier completion the user
// never touched.
func TestSecondRetractionDoesNotReachAnEarlierCompletion(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack", time.Now().Add(-time.Hour)))

	promptID, err := store.RecordPrompt(ctx, p, "9", "digest", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)

	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack", time.Now()))

	found, err := store.RetractCompletion(ctx, c.ID, p, promptID, time.Now())
	require.NoError(t, err)
	require.True(t, found, "the completion after the prompt was live")

	found, err = store.RetractCompletion(ctx, c.ID, p, promptID, time.Now())
	require.NoError(t, err)
	require.False(t, found, "a retried un-tap must not reach back to the earlier completion")

	var live int
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select count(*) from events where chore_id = $1 and retracted_at is null`,
		c.ID).Scan(&live))
	require.Equal(t, 1, live, "the completion from before the prompt is still live")
}

// The finding from the second round of review: RecordCompletion is not
// itself idempotent, so a retried "done" delivery can leave two live
// completions both at or after the same prompt's sent_at. Retracting only
// the most recent of those is still not a state assertion — a second call
// would find the older one still live and retract it too, landing in
// {both retracted} after two calls instead of {both retracted} after one.
// The whole window has to clear in a single statement so a second call
// finds nothing left to retract.
func TestRetractionClearsEveryCompletionInTheWindow(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	backdateChore(t, store, c.ID, 15*24*time.Hour)

	promptID, err := store.RecordPrompt(ctx, p, "9", "digest", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)

	// Two live completions in the same window, as a retried "done" delivery
	// would leave behind.
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack", time.Now()))
	require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "ack", time.Now()))

	due, err := store.DueChores(ctx, p, time.Now())
	require.NoError(t, err)
	require.Empty(t, due, "either completion reset the clock")

	found, err := store.RetractCompletion(ctx, c.ID, p, promptID, time.Now())
	require.NoError(t, err)
	require.True(t, found)

	var live int
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select count(*) from events where chore_id = $1 and retracted_at is null`,
		c.ID).Scan(&live))
	require.Zero(t, live, "both completions in the window are retracted, not just the newest")

	due, err = store.DueChores(ctx, p, time.Now())
	require.NoError(t, err)
	require.Len(t, due, 1, "nothing live since the prompt, so the clock falls back to creation")

	// A second call — the retry — must be a true no-op: nothing left live in
	// the window, so it changes nothing and reports nothing found.
	found, err = store.RetractCompletion(ctx, c.ID, p, promptID, time.Now())
	require.NoError(t, err)
	require.False(t, found, "a second call finds nothing left live in the window")

	require.NoError(t, store.Pool().QueryRow(ctx,
		`select count(*) from events where chore_id = $1 and retracted_at is null`,
		c.ID).Scan(&live))
	require.Zero(t, live, "still nothing live — the retry changed nothing")
}

// A tap carries a position, and the position resolves through a person-scoped
// prompt — so this is defence in depth rather than the only guard. It goes in
// because "unreachable" now depends on a decision rather than on structure.
func TestCompletionRefusesAnotherPersonsChore(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	a := owner(t, store)
	b, err := store.SeedOwner(ctx, "someone-else", nil)
	require.NoError(t, err)

	theirs, err := store.UpsertChore(ctx, b, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	require.Error(t, store.RecordCompletion(ctx, theirs.ID, a, "ack", time.Now()))

	var n int
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select count(*) from events where chore_id = $1`, theirs.ID).Scan(&n))
	require.Zero(t, n)
}
