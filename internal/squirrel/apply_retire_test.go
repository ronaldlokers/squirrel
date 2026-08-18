//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// choresOf seeds a chore the way a promotion would, so a retire test does not
// depend on the promotion path to make its subject.
func choresOf(t *testing.T, store *squirrel.Store, personID int64, names ...string) {
	t.Helper()
	for _, name := range names {
		_, err := store.UpsertChore(context.Background(), personID, name,
			14*24*time.Hour, squirrel.DefaultTolerance(14*24*time.Hour))
		require.NoError(t, err)
	}
}

func activeNames(t *testing.T, store *squirrel.Store, personID int64) []string {
	t.Helper()
	chores, err := store.ActiveChores(context.Background(), personID)
	require.NoError(t, err)
	names := []string{}
	for _, c := range chores {
		names = append(names, c.Name)
	}
	return names
}

// A chore you no longer want is otherwise permanent: it comes back on its own
// forever, and the only way to stop it was a database. Retiring is the way out
// that "every state transition is reversible" already implies.
func TestRetireByNameStopsAChoreComingBack(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	choresOf(t, store, p, "bins out", "water the plants")

	reply := triage(t, store, p, "!retire bins out")

	require.Contains(t, reply, "bins out")
	require.Equal(t, []string{"water the plants"}, activeNames(t, store, p))
}

// The name is matched the way the unique index matches it, so the case you
// typed is not the thing that decides.
func TestRetireByNameIgnoresCase(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	choresOf(t, store, p, "bins out")

	triage(t, store, p, "!retire BINS OUT")

	require.Empty(t, activeNames(t, store, p))
}

// The numbered surface is the other way in: whatever printed line 1 is what
// line 1 means, the same rule done/keep/drop follow.
func TestRetireByLineNumber(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	choresOf(t, store, p, "bins out")
	// `?` prints a line for every active chore, due or not.
	triage(t, store, p, "?")

	reply := triage(t, store, p, "!retire 1")

	require.Contains(t, reply, "bins out")
	require.Empty(t, activeNames(t, store, p))
}

// Retiring is not deleting. The chore's history stays, and saying it again
// brings it back — which is what makes this safe to do on a whim.
func TestARetiredChoreComesBackIfYouSayItAgain(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	choresOf(t, store, p, "bins out")
	before, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)

	triage(t, store, p, "!retire bins out")
	choresOf(t, store, p, "bins out")

	after, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Len(t, after, 1)
	require.Equal(t, before[0].ID, after[0].ID, "the same chore, not a second one")
}

func TestRetireSaysWhatItCannotFind(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	choresOf(t, store, p, "bins out")

	reply := triage(t, store, p, "!retire the boiler")

	require.Contains(t, reply, "the boiler")
	require.Contains(t, reply, "bins out", "say what there is instead of only what there is not")
	require.Equal(t, []string{"bins out"}, activeNames(t, store, p))
}

func TestRetireWithNothingToRetire(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	reply := triage(t, store, p, "!retire bins out")

	require.Contains(t, reply, "no chores")
}

func TestRetireNeedsToKnowWhich(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	choresOf(t, store, p, "bins out")

	reply := triage(t, store, p, "!retire")

	require.Contains(t, reply, "Which")
	require.Equal(t, []string{"bins out"}, activeNames(t, store, p))
}

// A number that names a note is a real mistake — the same shape as `keep 2`
// against a chore, and answered the same way rather than ignored. The chore
// here is what makes the test about line resolution: with none at all, the
// truer answer is that there is nothing to retire, and that is what it says.
func TestRetireAgainstANoteLineSaysSo(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	choresOf(t, store, p, "bins out")
	pileOf(t, store, p, "buy milk")

	reply := triage(t, store, p, "!retire 1")

	require.Contains(t, reply, "note")
	require.Equal(t, []string{"bins out"}, activeNames(t, store, p))
}

func TestRetireWithNoChoresSaysThatFirst(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	pileOf(t, store, p, "buy milk")

	require.Contains(t, triage(t, store, p, "!retire 1"), "no chores")
}

func TestHelpMentionsRetire(t *testing.T) {
	require.Contains(t, squirrel.HelpMessage().Text, "!retire")
}
