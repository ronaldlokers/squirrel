//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// TestDefineDoesNotCloseTheDigest is the phase-level acceptance criterion
// from the design doc: a chore definition is a standalone surface — one
// chore, never numbered — and must never retire the digest's buttons, no
// matter how soon after the digest it lands.
func TestDefineDoesNotCloseTheDigest(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	vac, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	digestID, err := store.RecordPrompt(ctx, p, "9", "digest", time.Now(), nil, []squirrel.Chore{vac})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, digestID, "d-1", time.Now()))

	chat, sent := chatRecorder("d-2")
	a := squirrel.NewApplier(store, nil, chat, nil)
	require.NoError(t, a.Apply(ctx, itemOf("every 2 weeks: dust"), &p))

	require.Len(t, *sent, 1)
	require.Empty(t, (*sent)[0].updates, "a definition must not close the digest's buttons")
}

// TestQueryClosesThePreviousNumberedPrompt is the other half: a fresh
// numbered surface — an on-demand "?" here, same as a new digest — does
// close the numbered one before it.
func TestQueryClosesThePreviousNumberedPrompt(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	vac, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	digestID, err := store.RecordPrompt(ctx, p, "9", "digest", time.Now(), nil, []squirrel.Chore{vac})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, digestID, "d-1", time.Now()))

	chat, sent := chatRecorder("d-2")
	a := squirrel.NewApplier(store, nil, chat, nil)
	require.NoError(t, a.Apply(ctx, itemOf("?"), &p))

	require.Len(t, *sent, 1)
	require.Equal(t, []string{"d-1"}, (*sent)[0].updates,
		"the query closes the digest's own message id")
}
