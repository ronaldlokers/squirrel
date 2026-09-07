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

func TestWhatWasRefusedOnTheBoardReachesTheModel(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	require.NotEmpty(t, url, "TEST_DATABASE_URL is required — see docs/testing.md")

	ctx := context.Background()
	store, err := squirrel.OpenStore(ctx, url)
	require.NoError(t, err)
	t.Cleanup(store.Close)
	require.NoError(t, store.Migrate(ctx))

	personID, err := store.SeedOwner(ctx, "wiring-refused", nil)
	require.NoError(t, err)
	_, err = store.Pool().Exec(ctx, `delete from noticed where person_id = $1`, personID)
	require.NoError(t, err)

	require.Empty(t, nowFor(ctx, store, personID, time.Now()).LandedBadly)

	at := time.Now()
	require.NoError(t, store.Notice(ctx, personID, "note", 1,
		"you have written this down three times", at))
	lines, err := store.WhatWasNoticed(ctx, personID)
	require.NoError(t, err)
	require.Len(t, lines, 1)

	refused, err := store.NotUseful(ctx, personID, lines[0].ID, at)
	require.NoError(t, err)
	require.True(t, refused)

	got := nowFor(ctx, store, personID, time.Now()).LandedBadly

	require.Equal(t, []string{"you have written this down three times"}, got,
		"the model is not shown what the board refused")
}
