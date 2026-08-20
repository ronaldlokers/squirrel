//go:build integration

package squirrel_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The third promotion, beside !chore. A note is what you said; a task is what
// you decided.
func TestTaskPromotesANoteByNumber(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	pileOf(t, store, p, "ring the vet about the booster")
	reply := triage(t, store, p, "!task 1")
	require.Contains(t, reply, "ring the vet")

	pile, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Empty(t, pile, "deciding takes it out of the pile")

	tasks, _, err := store.Tasks(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
}

// Words rather than a number makes one outright — the same number-or-name
// shape !did and !snooze already use.
func TestTaskMakesOneFromWords(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	triage(t, store, p, "!task book the car in")

	tasks, _, err := store.Tasks(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.Equal(t, "book the car in", tasks[0].RawText)

	pile, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Empty(t, pile, "a task is not also a note")
}

func TestUntaskReturnsItToThePile(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	pileOf(t, store, p, "ring the vet")
	triage(t, store, p, "!task 1")
	triage(t, store, p, "!tasks")
	triage(t, store, p, "!untask 1")

	pile, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, pile, 1)
}

// The existing verb, unchanged: archiving a task is the same transition it
// performs on a note.
func TestDoneArchivesATask(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	triage(t, store, p, "!task ring the vet")
	triage(t, store, p, "!tasks")
	triage(t, store, p, "done 1")

	open, _, err := store.Tasks(ctx, p, 10)
	require.NoError(t, err)
	require.Empty(t, open)

	archived, _, err := store.ArchivedTasks(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, archived, 1)
	require.Equal(t, "ring the vet", archived[0].RawText)
}

// Never a count, on the surface where a count is most tempting.
func TestTheTaskListNeverCounts(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	for _, w := range []string{"one thing", "another thing", "a third thing"} {
		triage(t, store, p, "!task "+w)
	}
	reply := triage(t, store, p, "!tasks")

	for _, total := range []string{"3 tasks", "3 things", "(3)", "you have 3", "3 left"} {
		require.NotContains(t, reply, total)
	}
}

// Absence, not encouragement. An empty task list is a normal state.
func TestNoTasksSaysSoWithoutInstructing(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	reply := triage(t, store, p, "!tasks")

	require.Contains(t, reply, "Nothing decided")
	for _, nag := range []string{"add ", "start by", "you should", "why not"} {
		require.NotContains(t, strings.ToLower(reply), nag)
	}
}

// A chore already comes back on its own; deciding it is not a thing that means
// anything.
func TestTaskRefusesAChoreLine(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	choresOf(t, store, p, "bins out")
	triage(t, store, p, "?")

	require.Contains(t, triage(t, store, p, "!task 1"), "chore")
}
