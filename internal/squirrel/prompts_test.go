//go:build integration

package squirrel_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestRecordPromptNumbersLines(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	bins, err := store.UpsertChore(ctx, p, "bin day", oneWeek, oneDay)
	require.NoError(t, err)
	vac, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	promptID, err := store.RecordPrompt(ctx, p, "9", "digest", time.Now(), nil, []squirrel.Chore{bins, vac})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, promptID, "m-1", time.Now()))

	got, ok, err := store.ChoreAtPosition(ctx, p, 2)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, vac.ID, got.ID)

	_, ok, err = store.ChoreAtPosition(ctx, p, 7)
	require.NoError(t, err)
	require.False(t, ok)
}

// The unique index is the idempotency, not application logic.
func TestRecordPromptRefusesASecondDigestForOneDate(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	day := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)

	_, err := store.RecordPrompt(ctx, p, "9", "digest", time.Now(), &day, nil)
	require.NoError(t, err)

	_, err = store.RecordPrompt(ctx, p, "9", "digest", time.Now(), &day, nil)
	require.ErrorIs(t, err, squirrel.ErrDigestAlreadySent)
}

func TestOutstandingExcludesCompletedLines(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	bins, err := store.UpsertChore(ctx, p, "bin day", oneWeek, oneDay)
	require.NoError(t, err)
	vac, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	sentAt := time.Now()
	promptID, err := store.RecordPrompt(ctx, p, "9", "digest", sentAt, nil, []squirrel.Chore{bins, vac})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, promptID, "m-1", time.Now()))

	outstanding, err := store.OutstandingLines(ctx, p)
	require.NoError(t, err)
	require.Len(t, outstanding, 2)

	require.NoError(t, store.RecordCompletion(ctx, bins.ID, p, "ack", sentAt.Add(time.Minute)))

	outstanding, err = store.OutstandingLines(ctx, p)
	require.NoError(t, err)
	require.Len(t, outstanding, 1)
	require.Equal(t, vac.ID, outstanding[0].ID)
}

// Before the fix, OutstandingLines' "not exists" subquery checked only
// e.occurred_at >= p.sent_at, with no e.retracted_at is null predicate —
// baselineCTE and CompletedSince both carry it, this third reader of events
// was missed. So a completion that was later retracted (an un-tap) still
// counted as "exists", and OutstandingLines kept treating the chore as
// resolved even though DueChores correctly saw it as overdue again. A typed
// bare "done" answered "Nothing outstanding" for a chore the digest still
// listed.
func TestOutstandingLinesSeesARetractedCompletion(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	vac, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	sentAt := time.Now()
	promptID, err := store.RecordPrompt(ctx, p, "9", "digest", sentAt, nil, []squirrel.Chore{vac})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, promptID, "m-1", time.Now()))

	require.NoError(t, store.RecordCompletion(ctx, vac.ID, p, "tap", sentAt.Add(time.Minute)))

	outstanding, err := store.OutstandingLines(ctx, p)
	require.NoError(t, err)
	require.Empty(t, outstanding, "completed since the prompt, so nothing outstanding yet")

	retracted, err := store.RetractCompletion(ctx, vac.ID, p, promptID, sentAt.Add(2*time.Minute))
	require.NoError(t, err)
	require.True(t, retracted)

	outstanding, err = store.OutstandingLines(ctx, p)
	require.NoError(t, err)
	require.Len(t, outstanding, 1, "the retraction must make the chore outstanding again")
	require.Equal(t, vac.ID, outstanding[0].ID)

	require.NoError(t, squirrel.NewApplier(store, send, squirrel.Chat{}, nil).Apply(ctx, itemOf("done"), &p))
	require.Len(t, *got, 1)
	require.Contains(t, (*got)[0].text, "vacuum", "a bare done must resolve to the un-tapped chore")
}

// A later prompt supersedes an earlier one: numbering always addresses the
// most recent list.
func TestPositionsComeFromTheMostRecentPrompt(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	bins, err := store.UpsertChore(ctx, p, "bin day", oneWeek, oneDay)
	require.NoError(t, err)
	vac, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	digest, err := store.RecordPrompt(ctx, p, "9", "digest", time.Now().Add(-time.Hour), nil, []squirrel.Chore{bins, vac})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, digest, "m-1", time.Now()))
	query, err := store.RecordPrompt(ctx, p, "9", "query", time.Now(), nil, []squirrel.Chore{vac})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, query, "m-2", time.Now()))

	got, ok, err := store.ChoreAtPosition(ctx, p, 1)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, vac.ID, got.ID)
}

// person_id is the only thing keeping one person's "done 2" off another
// person's chore. B's prompt is sent after A's on purpose: if the predicate
// were dropped, resolution would fall back to the globally most recent
// prompt and hand A person B's chore, not merely find nothing.
func TestPositionsDoNotCrossPersons(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	a := owner(t, store)
	b, err := store.SeedOwner(ctx, "someone-else", nil)
	require.NoError(t, err)

	aChore, err := store.UpsertChore(ctx, a, "bin day", oneWeek, oneDay)
	require.NoError(t, err)
	bChore, err := store.UpsertChore(ctx, b, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	aPrompt, err := store.RecordPrompt(ctx, a, "9", "digest", time.Now().Add(-time.Hour), nil, []squirrel.Chore{aChore})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, aPrompt, "m-1", time.Now()))
	bPrompt, err := store.RecordPrompt(ctx, b, "9", "digest", time.Now(), nil, []squirrel.Chore{bChore})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, bPrompt, "m-2", time.Now()))

	got, ok, err := store.ChoreAtPosition(ctx, a, 1)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, aChore.ID, got.ID)
}

// A definition confirmation carries a button, so it is a prompt — but it is not
// a numbered surface, and `done 1` must keep meaning the morning digest's first
// line rather than the chore that was just defined.
func TestDefineDoesNotStealTheNumbering(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	bins, err := store.UpsertChore(ctx, p, "bin day", oneWeek, oneDay)
	require.NoError(t, err)
	vac, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	digest, err := store.RecordPrompt(ctx, p, "9", "digest", time.Now().Add(-time.Hour), nil, []squirrel.Chore{bins})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, digest, "m-1", time.Now()))
	define, err := store.RecordPrompt(ctx, p, "9", "define", time.Now(), nil, []squirrel.Chore{vac})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, define, "m-2", time.Now()))

	got, ok, err := store.ChoreAtPosition(ctx, p, 1)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, bins.ID, got.ID, "the digest still owns position 1")
}

func TestPromptByMessageIDIsScopedByPerson(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	a := owner(t, store)
	b, err := store.SeedOwner(ctx, "someone-else", nil)
	require.NoError(t, err)

	c, err := store.UpsertChore(ctx, b, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	id, err := store.RecordPrompt(ctx, b, "9", "digest", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, id, "m-77", time.Now()))

	_, ok, err := store.PromptByMessageID(ctx, a, "m-77")
	require.NoError(t, err)
	require.False(t, ok, "one person's tap must not reach another's prompt")

	_, ok, err = store.PromptByMessageID(ctx, b, "m-77")
	require.NoError(t, err)
	require.True(t, ok)
}

func TestPreviousNumberedPromptSkipsDefines(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	digest, err := store.RecordPrompt(ctx, p, "9", "digest", time.Now().Add(-2*time.Hour), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, digest, "m-1", time.Now()))

	define, err := store.RecordPrompt(ctx, p, "9", "define", time.Now().Add(-time.Hour), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, define, "m-2", time.Now()))

	current, err := store.RecordPrompt(ctx, p, "9", "query", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)

	prev, ok, err := store.PreviousNumberedPrompt(ctx, p, current)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "m-1", prev.ExternalMessageID, "the define is not a numbered surface")
}

// replyFor commits a query prompt before the message carrying it is sent. If
// the send then fails, apply returns early — no MarkPromptSent, no
// closePrevious — so the row is committed but delivered_at stays null and no
// button for it ever reaches the room. Before the fix, latestPrompt and
// ChoreAtPosition selected the newest numbered prompt with no delivery
// predicate, so that phantom row became "current" for typed positions anyway,
// while the room's actual buttons still pointed at the last prompt that
// really went out. Typed "done 1" and a tap on button 1 then resolved to
// different chores.
func TestChoreAtPositionIgnoresAnUndeliveredQuery(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	// "apple errand" sorts before "vacuum", so ActiveChores — what a "?" query
	// prompt is built from — puts it at position 1 and vacuum at position 2.
	// Only vacuum goes in the digest, at position 1. If ChoreAtPosition ever
	// reads the failed query as "current", position 1 resolves to the wrong
	// chore; that mismatch is what makes this test mean something rather than
	// passing by coincidence.
	apple, err := store.UpsertChore(ctx, p, "apple errand", oneWeek, oneDay)
	require.NoError(t, err)
	vac, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	_ = apple

	digest, err := store.RecordPrompt(ctx, p, "9", "digest", time.Now().Add(-time.Hour), nil, []squirrel.Chore{vac})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, digest, "m-1", time.Now()))

	failingSend := func(context.Context, string, string) error { return errors.New("send failed") }
	require.Error(t, squirrel.NewApplier(store, failingSend, squirrel.Chat{}, nil).Apply(ctx, itemOf("?"), &p))

	var undelivered int
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select count(*) from prompts where person_id = $1 and kind = 'query' and delivered_at is null`,
		p).Scan(&undelivered))
	require.Equal(t, 1, undelivered, "the failed query prompt was committed but never marked delivered")

	got, ok, err := store.ChoreAtPosition(ctx, p, 1)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, vac.ID, got.ID, "typed done 1 must still resolve against the last delivered prompt")
}

// A prompt whose send failed has no message id, so there is nothing to disable
// and nothing to resolve a tap against.
func TestPreviousNumberedPromptIgnoresUnsentOnes(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	_, err = store.RecordPrompt(ctx, p, "9", "digest", time.Now().Add(-time.Hour), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	current, err := store.RecordPrompt(ctx, p, "9", "query", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)

	_, ok, err := store.PreviousNumberedPrompt(ctx, p, current)
	require.NoError(t, err)
	require.False(t, ok)
}
