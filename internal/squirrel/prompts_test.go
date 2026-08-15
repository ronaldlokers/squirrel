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
