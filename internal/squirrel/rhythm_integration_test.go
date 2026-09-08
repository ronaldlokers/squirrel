//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/stretchr/testify/require"
)

func TestWhenYouUsuallyDoAThingIsReadFromWhenYouActuallyDidIt(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "bins out", 7*24*time.Hour, 24*time.Hour)
	require.NoError(t, err)

	sunday := time.Date(2026, 8, 30, 9, 30, 0, 0, time.UTC)
	for _, at := range []time.Time{sunday, sunday.AddDate(0, 0, -7), sunday.AddDate(0, 0, -14)} {
		require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "board", at))
	}

	usually, err := store.WhenYouUsuallyDo(ctx, p)
	require.NoError(t, err)

	got, found := usually[c.ID]
	require.True(t, found, "three completions on three Sunday mornings taught it nothing")
	require.Equal(t, time.Sunday, got.Weekday)
	require.Equal(t, squirrel.Morning, got.Part)
}

func TestTwoCompletionsAreNotAHabit(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "descale the kettle", 14*24*time.Hour, 24*time.Hour)
	require.NoError(t, err)

	sunday := time.Date(2026, 8, 30, 9, 30, 0, 0, time.UTC)
	for _, at := range []time.Time{sunday, sunday.AddDate(0, 0, -14)} {
		require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "board", at))
	}

	usually, err := store.WhenYouUsuallyDo(ctx, p)
	require.NoError(t, err)
	require.NotContains(t, usually, c.ID, "it claimed a habit from two completions")
}

func TestARetractedCompletionIsNotEvidenceOfAHabit(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "water the plants", 7*24*time.Hour, 24*time.Hour)
	require.NoError(t, err)

	sunday := time.Date(2026, 8, 30, 9, 30, 0, 0, time.UTC)
	promptID, err := store.RecordPrompt(ctx, p, "9", "nudge", sunday.Add(-time.Hour), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	for _, at := range []time.Time{sunday, sunday.AddDate(0, 0, -7), sunday.AddDate(0, 0, -14)} {
		require.NoError(t, store.RecordCompletion(ctx, c.ID, p, "board", at))
	}
	took, err := store.RetractCompletion(ctx, c.ID, p, promptID, sunday.Add(time.Hour))
	require.NoError(t, err)
	require.True(t, took, "nothing was retracted, so this test proves nothing")

	usually, err := store.WhenYouUsuallyDo(ctx, p)
	require.NoError(t, err)
	require.NotContains(t, usually, c.ID,
		"a completion you took back still counts toward when you usually do a thing")
}

func TestWhatYouUsuallyDoIsOnlyYours(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	mine := owner(t, store)
	theirs, err := store.PersonForLogin(ctx, "sub-someone-else", "someone-else")
	require.NoError(t, err)

	c, err := store.UpsertChore(ctx, theirs, "their chore", 7*24*time.Hour, 24*time.Hour)
	require.NoError(t, err)
	sunday := time.Date(2026, 8, 30, 9, 30, 0, 0, time.UTC)
	for _, at := range []time.Time{sunday, sunday.AddDate(0, 0, -7), sunday.AddDate(0, 0, -14)} {
		require.NoError(t, store.RecordCompletion(ctx, c.ID, theirs, "board", at))
	}

	usually, err := store.WhenYouUsuallyDo(ctx, mine)
	require.NoError(t, err)
	require.Empty(t, usually, "another person's habits reached this one's board")
}
