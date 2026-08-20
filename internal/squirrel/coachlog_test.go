//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func coachPerson(t *testing.T, store *squirrel.Store, name, externalID string) int64 {
	t.Helper()
	id, err := store.SeedOwner(context.Background(), name,
		[]squirrel.IdentitySeed{{Transport: "campfire", ExternalID: externalID}})
	require.NoError(t, err)
	return id
}

func TestCoachSpendSumsThisMonthOnly(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	personID := coachPerson(t, store, "ronald", "1")

	now := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	month := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	for _, a := range []squirrel.CoachAnswer{
		// Last month. Present so the window is actually tested rather than
		// "the sum of every row that exists".
		{Model: "gpt-5.6-luna", CostMicros: 9_000_000, At: now.AddDate(0, -1, 0)},
		{Model: "gpt-5.6-luna", CostMicros: 640, Used: true, At: now.Add(-time.Hour)},
		// A rejected reply. Still paid for, so still counted.
		{Model: "gpt-5.6-luna", CostMicros: 360, Used: false, At: now.Add(-time.Minute)},
	} {
		require.NoError(t, store.RecordCoachAnswer(ctx, personID, a))
	}

	spent, err := store.CoachSpentSince(ctx, personID, month)
	require.NoError(t, err)
	require.Equal(t, int64(1_000), spent)
}

// A month with nothing in it sums to zero rather than failing to scan a null.
// The budget fails closed on an error, so a null here would take the coach off
// for everyone who had not used it yet.
func TestCoachSpendIsZeroBeforeAnythingIsSaid(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	personID := coachPerson(t, store, "ronald", "1")

	spent, err := store.CoachSpentSince(ctx, personID, time.Now().AddDate(0, -1, 0))
	require.NoError(t, err)
	require.Equal(t, int64(0), spent)
}

// One person's spend is not another's.
func TestCoachSpendIsPerPerson(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	mine := coachPerson(t, store, "ronald", "1")
	theirs := coachPerson(t, store, "someone-else", "2")

	now := time.Now()
	require.NoError(t, store.RecordCoachAnswer(ctx, theirs,
		squirrel.CoachAnswer{Model: "gpt-5.6-luna", CostMicros: 5_000_000, At: now}))

	spent, err := store.CoachSpentSince(ctx, mine, now.AddDate(0, 0, -1))
	require.NoError(t, err)
	require.Equal(t, int64(0), spent)
}

// The whole exchange is kept, by decision: it is what makes a bad answer
// inspectable afterwards and changing the model in configuration evaluable.
func TestCoachAnswerKeepsWhatWasSaid(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	personID := coachPerson(t, store, "ronald", "1")

	require.NoError(t, store.RecordCoachAnswer(ctx, personID, squirrel.CoachAnswer{
		Kind:      "sheet",
		Model:     "gpt-5.6-luna",
		Prompt:    "It is 14:00, capacity ok.",
		Reply:     "Start with the envelope.",
		InTokens:  2_000,
		OutTokens: 200,
		Used:      true,
		At:        time.Now(),
	}))

	var kind, model, prompt, reply string
	var in, out int
	var used bool
	require.NoError(t, store.Pool().QueryRow(ctx, `
		select kind, model, prompt, reply, in_tokens, out_tokens, used
		  from coach_answers where person_id = $1`, personID).
		Scan(&kind, &model, &prompt, &reply, &in, &out, &used))

	require.Equal(t, "sheet", kind)
	require.Equal(t, "gpt-5.6-luna", model)
	require.Equal(t, "It is 14:00, capacity ok.", prompt)
	require.Equal(t, "Start with the envelope.", reply)
	require.Equal(t, 2_000, in)
	require.Equal(t, 200, out)
	require.True(t, used)
}
