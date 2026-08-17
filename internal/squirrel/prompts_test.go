//go:build integration

package squirrel_test

import (
	"context"
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

	_, err = store.RecordPrompt(ctx, p, "9", "digest", time.Now(), nil, []squirrel.Chore{bins, vac})
	require.NoError(t, err)

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
	_, err = store.RecordPrompt(ctx, p, "9", "digest", sentAt, nil, []squirrel.Chore{bins, vac})
	require.NoError(t, err)

	outstanding, err := store.OutstandingLines(ctx, p)
	require.NoError(t, err)
	require.Len(t, outstanding, 2)

	require.NoError(t, store.RecordCompletion(ctx, bins.ID, p, "ack", sentAt.Add(time.Minute)))

	outstanding, err = store.OutstandingLines(ctx, p)
	require.NoError(t, err)
	require.Len(t, outstanding, 1)
	require.Equal(t, vac.ID, outstanding[0].ID)
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

	_, err = store.RecordPrompt(ctx, p, "9", "digest", time.Now().Add(-time.Hour), nil, []squirrel.Chore{bins, vac})
	require.NoError(t, err)
	_, err = store.RecordPrompt(ctx, p, "9", "query", time.Now(), nil, []squirrel.Chore{vac})
	require.NoError(t, err)

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

	_, err = store.RecordPrompt(ctx, a, "9", "digest", time.Now().Add(-time.Hour), nil, []squirrel.Chore{aChore})
	require.NoError(t, err)
	_, err = store.RecordPrompt(ctx, b, "9", "digest", time.Now(), nil, []squirrel.Chore{bChore})
	require.NoError(t, err)

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

	_, err = store.RecordPrompt(ctx, p, "9", "digest", time.Now().Add(-time.Hour), nil, []squirrel.Chore{bins})
	require.NoError(t, err)
	_, err = store.RecordPrompt(ctx, p, "9", "define", time.Now(), nil, []squirrel.Chore{vac})
	require.NoError(t, err)

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
