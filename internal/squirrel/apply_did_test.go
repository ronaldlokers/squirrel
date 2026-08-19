//go:build integration

package squirrel_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Completing a chore from chat took two steps and a number held in your head:
// ask ?, read the list, type done 1. The listing was load-bearing for the
// wrong reason — it was where the number came from, not where the decision was
// made.
func TestDidCompletesAChoreByName(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	choresOf(t, store, p, "bins out")
	backdate(t, store, "bins out", 20)
	require.Equal(t, []string{"bins out"}, dueNames(t, store, p))

	reply := triage(t, store, p, "!did bins out")

	require.Contains(t, reply, "bins out")
	require.Empty(t, dueNames(t, store, p), "it is done, so it stops being due")
}

// "bins" for "bins out" is what a person types. Refusing it to be strict about
// a name they chose themselves is pedantry with a cost.
func TestDidAcceptsAnUnambiguousPartialName(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	choresOf(t, store, p, "bins out")
	backdate(t, store, "bins out", 20)

	triage(t, store, p, "!did bins")

	require.Empty(t, dueNames(t, store, p))
}

// Two chores that both say it is not a guess to make on someone's behalf.
func TestDidRefusesAnAmbiguousName(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	choresOf(t, store, p, "bins out")
	choresOf(t, store, p, "bins washed")
	backdate(t, store, "bins out", 20)

	reply := triage(t, store, p, "!did bins")

	require.Contains(t, reply, "bins out")
	require.Contains(t, reply, "bins washed")
	require.Equal(t, []string{"bins out"}, dueNames(t, store, p), "nothing was recorded")
}

// The line number still works, because it is what ? just printed.
func TestDidTakesALineNumber(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	choresOf(t, store, p, "bins out")
	backdate(t, store, "bins out", 20)
	triage(t, store, p, "?")

	triage(t, store, p, "!did 1")

	require.Empty(t, dueNames(t, store, p))
}

// A name that is not a chore names what you do have. An error that lists the
// alternatives is a recovery rather than a refusal.
func TestDidNamesWhatYouHave(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	choresOf(t, store, p, "bins out")

	reply := triage(t, store, p, "!did the washing")

	require.Contains(t, reply, "bins out")
}

// The same completion the tap records, through the same call, so the two ways
// of saying it cannot come to mean different things.
func TestDidAndTheTapAgree(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	choresOf(t, store, p, "bins out")
	backdate(t, store, "bins out", 20)

	triage(t, store, p, "!did bins out")

	chores, err := store.ActiveChores(t.Context(), p)
	require.NoError(t, err)
	require.Len(t, chores, 1)
	require.True(t, chores[0].EverDone, "a completion is a completion")
	require.Less(t, chores[0].SinceDays, 1)
}
