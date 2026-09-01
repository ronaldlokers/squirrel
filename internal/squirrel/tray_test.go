//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/stretchr/testify/require"
)

func TestTheTrayHoldsWhatLeftTheBoardSince(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	at := time.Now()

	kept := insertItem(t, store, p, "boiler service code is 4471")
	dropped := insertItem(t, store, p, "the thing about the bike lights")
	yesterday := insertItem(t, store, p, "an old one")
	insertItem(t, store, p, "kaas")

	require.NoError(t, store.SetItemState(ctx, kept, squirrel.ItemKept, at.Add(-2*time.Hour)))
	require.NoError(t, store.SetItemState(ctx, dropped, squirrel.ItemDropped, at.Add(-1*time.Hour)))
	require.NoError(t, store.SetItemState(ctx, yesterday, squirrel.ItemDone, at.Add(-30*time.Hour)))

	left, err := store.TriagedSince(ctx, p, at.Add(-8*time.Hour))
	require.NoError(t, err)

	require.Len(t, left, 2, "only what left today, and not what is still open")
	require.Equal(t, "the thing about the bike lights", left[0].RawText)
	require.Equal(t, squirrel.ItemDropped, left[0].State)
	require.Equal(t, "boiler service code is 4471", left[1].RawText)
	require.Equal(t, squirrel.ItemKept, left[1].State)
}
