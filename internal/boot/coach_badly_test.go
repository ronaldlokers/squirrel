//go:build integration

package boot

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The wiring, which nothing else notices going missing.
//
// The store can record what did not land, and the prompt can show it to the
// model. Between them is one line in `nowFor`, and removing it compiles
// cleanly and fails no unit test — the store keeps its rows, the prompt keeps
// its formatting, and the model simply stops being told. That is exactly the
// shape of defect this project has learnt to test for: a fix that no test
// would notice being reverted is an unpinned fix.
func TestWhatDidNotLandReachesTheModel(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	require.NotEmpty(t, url, "TEST_DATABASE_URL is required — see docs/testing.md")

	ctx := context.Background()
	store, err := squirrel.OpenStore(ctx, url)
	require.NoError(t, err)
	t.Cleanup(store.Close)
	require.NoError(t, store.Migrate(ctx))

	personID, err := store.SeedOwner(ctx, "wiring-badly", nil)
	require.NoError(t, err)

	// Nothing said yet: the model is told nothing rather than "nothing".
	require.Empty(t, nowFor(ctx, store, personID, time.Now()).LandedBadly)

	require.NoError(t, store.RecordCoachAnswer(ctx, personID, squirrel.CoachAnswer{
		Kind: "sheet", Model: "test", Prompt: "what now",
		Reply: "you have done this three times this week", Used: true,
	}))
	marked, err := store.LandedBadlyLatest(ctx, personID, time.Now())
	require.NoError(t, err)
	require.True(t, marked)

	got := nowFor(ctx, store, personID, time.Now()).LandedBadly

	require.Equal(t, []string{"you have done this three times this week"}, got,
		"the model is not shown what did not land here")
}
