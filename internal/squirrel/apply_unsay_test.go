//go:build integration

package squirrel_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestUnsayDeletesTheLastCoachExchange(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	require.NoError(t, store.RecordCoachAnswer(context.Background(), p, squirrel.CoachAnswer{
		Kind: "sheet", Model: "gpt-5.6-luna", Prompt: "help", Reply: "start with the envelope",
		Used: true,
	}))

	reply := triage(t, store, p, "!unsay")
	require.Contains(t, reply, "Unsaid")

	var remaining int
	require.NoError(t, store.Pool().QueryRow(context.Background(),
		`select count(*) from coach_answers where person_id = $1`, p).Scan(&remaining))
	require.Zero(t, remaining)
}

func TestUnsayWithNothingToUnsaySaysSo(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	reply := triage(t, store, p, "!unsay")
	require.Contains(t, reply, "nothing to unsay")
}
