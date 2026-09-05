//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestDeleteLatestCoachAnswerRemovesOnlyTheMostRecent(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	personID := coachPerson(t, store, "ronald", "1")

	now := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	require.NoError(t, store.RecordCoachAnswer(ctx, personID, squirrel.CoachAnswer{
		Kind: "sheet", Model: "gpt-5.6-luna", Prompt: "earlier", Reply: "earlier reply",
		Used: true, At: now.Add(-time.Hour),
	}))
	require.NoError(t, store.RecordCoachAnswer(ctx, personID, squirrel.CoachAnswer{
		Kind: "sheet", Model: "gpt-5.6-luna", Prompt: "latest", Reply: "latest reply",
		Used: true, At: now,
	}))

	deleted, err := store.DeleteLatestCoachAnswer(ctx, personID)
	require.NoError(t, err)
	require.True(t, deleted)

	var remaining []string
	rows, err := store.Pool().Query(ctx,
		`select reply from coach_answers where person_id = $1`, personID)
	require.NoError(t, err)
	for rows.Next() {
		var reply string
		require.NoError(t, rows.Scan(&reply))
		remaining = append(remaining, reply)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []string{"earlier reply"}, remaining)
}

func TestDeleteLatestCoachAnswerWithNothingRecordedDoesNothing(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	personID := coachPerson(t, store, "ronald", "1")

	deleted, err := store.DeleteLatestCoachAnswer(ctx, personID)
	require.NoError(t, err)
	require.False(t, deleted)
}

func TestDeleteLatestCoachAnswerOnlyTouchesYourOwn(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	ronald := coachPerson(t, store, "ronald", "1")
	someoneElse := coachPerson(t, store, "guest", "2")

	require.NoError(t, store.RecordCoachAnswer(ctx, someoneElse, squirrel.CoachAnswer{
		Kind: "sheet", Model: "gpt-5.6-luna", Prompt: "theirs", Reply: "their reply", Used: true,
	}))

	deleted, err := store.DeleteLatestCoachAnswer(ctx, ronald)
	require.NoError(t, err)
	require.False(t, deleted, "deleting the owner's last answer removed a guest's")

	var stillThere bool
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select exists (select 1 from coach_answers where person_id = $1)`, someoneElse).
		Scan(&stillThere))
	require.True(t, stillThere)
}

func TestPurgeCoachAnswersBeforeRemovesOnlyOlderRows(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	personID := coachPerson(t, store, "ronald", "1")

	cutoff := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, store.RecordCoachAnswer(ctx, personID, squirrel.CoachAnswer{
		Kind: "sheet", Model: "gpt-5.6-luna", Prompt: "old", Reply: "old reply",
		Used: true, At: cutoff.Add(-time.Hour),
	}))
	require.NoError(t, store.RecordCoachAnswer(ctx, personID, squirrel.CoachAnswer{
		Kind: "sheet", Model: "gpt-5.6-luna", Prompt: "recent", Reply: "recent reply",
		Used: true, At: cutoff.Add(time.Hour),
	}))

	n, err := store.PurgeCoachAnswersBefore(ctx, cutoff)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	var remaining []string
	rows, err := store.Pool().Query(ctx,
		`select reply from coach_answers where person_id = $1`, personID)
	require.NoError(t, err)
	for rows.Next() {
		var reply string
		require.NoError(t, rows.Scan(&reply))
		remaining = append(remaining, reply)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []string{"recent reply"}, remaining)
}
