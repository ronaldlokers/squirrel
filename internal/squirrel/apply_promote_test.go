//go:build integration

package squirrel_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Two writes, one intent. Both are asserted, because a chore created without
// the note leaving the pile means the next triage makes the same chore again,
// and a note marked done without a chore loses the thing entirely.
func TestPromotingANoteCreatesTheChoreAndMarksTheNoteDone(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	pileOf(t, store, p, "water the plants")
	id := lineItemID(t, store, p, 1)

	reply := triage(t, store, p, "!chore 1 every 2 weeks")

	chores, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Len(t, chores, 1)
	require.Equal(t, "water the plants", chores[0].Name, "the note's own text is the name")
	require.Equal(t, 14, chores[0].EveryDays)

	require.Equal(t, "done", stateOf(t, store, id),
		"the note did its job by becoming something that comes back on its own")
	require.Contains(t, reply, "water the plants")
}

func TestPromotingAcceptsTheIntervalFormsParseEveryKnows(t *testing.T) {
	for _, tc := range []struct {
		arg  string
		days int
	}{
		{"every day", 1},
		{"every 3 days", 3},
		{"every week", 7},
		{"every 2 weeks", 14},
		{"every month", 30},
	} {
		t.Run(tc.arg, func(t *testing.T) {
			store := withStore(t)
			ctx := context.Background()
			p := owner(t, store)

			pileOf(t, store, p, "water the plants")
			triage(t, store, p, "!chore 1 "+tc.arg)

			chores, err := store.ActiveChores(ctx, p)
			require.NoError(t, err)
			require.Len(t, chores, 1)
			require.Equal(t, tc.days, chores[0].EveryDays)
		})
	}
}

// The interval is the whole point of promoting: without one there is nothing
// to schedule, and guessing a default would put a chore on the calendar that
// nobody chose.
func TestPromotingWithoutAnIntervalAsksForOne(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	pileOf(t, store, p, "water the plants")
	id := lineItemID(t, store, p, 1)

	reply := triage(t, store, p, "!chore 1")
	require.Contains(t, strings.ToLower(reply), "how often")

	chores, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Empty(t, chores)
	require.Equal(t, "open", stateOf(t, store, id), "and the note stays in the pile")
}

func TestPromotingWithoutALineNumberAsksForOne(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	pileOf(t, store, p, "water the plants")

	reply := triage(t, store, p, "!chore every 2 weeks")
	require.Contains(t, strings.ToLower(reply), "which line")

	chores, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Empty(t, chores)
}

func TestPromotingAChoreLineSaysItIsAlreadyOne(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	promptID, err := store.RecordPrompt(ctx, p, "9", "query", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, promptID, "m-0", time.Now()))

	reply := triage(t, store, p, "!chore 1 every 2 weeks")
	require.Contains(t, strings.ToLower(reply), "already a chore")

	chores, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Len(t, chores, 1, "no second chore from the same line")
}

func TestPromotingBeyondTheLastLineSaysSo(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	pileOf(t, store, p, "water the plants")

	require.Contains(t, triage(t, store, p, "!chore 7 every 2 weeks"), "line 7")

	chores, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Empty(t, chores)
}

// A promoted note leaves the pile, so the same note cannot be promoted twice
// into two chores that mean the same thing.
func TestAPromotedNoteLeavesThePile(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	// Inserted oldest-first; the pile lists newest-first, so line 1 is the
	// second argument. Worth being explicit about — the first version of this
	// test assumed line 1 was the first note passed and failed on it.
	pileOf(t, store, p, "water the plants", "buy milk")
	triage(t, store, p, "!chore 1 every 2 weeks")

	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "water the plants", items[0].RawText,
		"the promoted note left the pile, and only it")
}

// The note's text is the chore's name verbatim, including any shape the
// matcher would otherwise have read as an instruction. Promotion is explicit,
// so nothing here needs to guess.
func TestPromotingKeepsTheNotesTextVerbatim(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	pileOf(t, store, p, "every day i think about leaving")
	triage(t, store, p, "!chore 1 every week")

	chores, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Len(t, chores, 1)
	require.Equal(t, "every day i think about leaving", chores[0].Name)
	require.Equal(t, 7, chores[0].EveryDays, "the interval typed wins over the one in the text")
}
