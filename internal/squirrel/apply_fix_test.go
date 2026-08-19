//go:build integration

package squirrel_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// A thought typed one-handed arrives half-written or autocorrected into
// something else. Until now the only remedy was to drop it and say it again,
// which cost the arrival time and the place in the pile.
func TestFixChangesWhatANoteSays(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	pileOf(t, store, p, "the boler makes a noise")

	reply := triage(t, store, p, "!fix 1 the boiler makes a noise on tuesdays")
	require.Contains(t, reply, "boiler makes a noise on tuesdays")

	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "the boiler makes a noise on tuesdays", items[0].RawText)
}

// The two ways of correcting a thought are the same write, so they cannot come
// to mean different things.
func TestFixKeepsEverythingElseAboutTheNote(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	pileOf(t, store, p, "meter readng 48213")
	before, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, before, 1)

	triage(t, store, p, "!fix 1 meter reading 48213")

	after, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, after, 1)
	require.Equal(t, before[0].ID, after[0].ID)
	require.Equal(t, before[0].ReceivedAt, after[0].ReceivedAt)
	require.Equal(t, before[0].State, after[0].State)
}

// A chore is not a note, and its name changes by saying it again with an
// interval — which is the one write that already exists for that.
func TestFixRefusesAChoreLine(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	choresOf(t, store, p, "bins out")
	triage(t, store, p, "?")

	reply := triage(t, store, p, "!fix 1 bins in")

	require.Contains(t, reply, "chore")
}

// Without words there is no correction, and an empty note is not a thing this
// product has.
func TestFixWithoutWordsAsksForThem(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	pileOf(t, store, p, "buy milk")

	reply := triage(t, store, p, "!fix 1")

	require.Contains(t, reply, "What should line 1 say")
}
