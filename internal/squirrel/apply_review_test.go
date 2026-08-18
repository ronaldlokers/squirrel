//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// liveNudge delivers a nudge carrying one chore and a live ✅, and returns the
// chore. Its message id is "m-nudge" so a test can say which button was closed.
func liveNudge(t *testing.T, store *squirrel.Store, personID int64) squirrel.Chore {
	t.Helper()
	ctx := context.Background()

	c, err := store.UpsertChore(ctx, personID, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	promptID, err := store.RecordPrompt(ctx, personID, "9", "nudge", time.Now(), nil,
		[]squirrel.Chore{c})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, promptID, "m-nudge", time.Now()))
	return c
}

// Looking at your own notes must not cost you the day's chore.
//
// closePrevious keeps exactly one live *button* surface. A pile listing carries
// no buttons by design, so closing the nudge on its account disabled the ✅ —
// in production Update always sends disabled: true — and bought nothing. Phase
// 4 spent a fix round on this exact shape when the piggyback nudge destroyed
// the `?` list.
func TestListingThePileLeavesTheNudgesButtonAlone(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	liveNudge(t, store, p)
	insertItem(t, store, p, "buy milk")

	chat, got := chatRecorder("m-notes")
	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(ctx, itemOf("!notes"), &p))

	require.Len(t, *got, 1)
	require.Empty(t, (*got)[0].updates,
		"a buttonless list must close nothing — the nudge's ✅ is still the day's only way to mark it by tap")
}

func TestSearchingLeavesTheNudgesButtonAlone(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	liveNudge(t, store, p)
	insertItem(t, store, p, "the boiler thing")

	chat, got := chatRecorder("m-find")
	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(ctx, itemOf("!find boiler"), &p))

	require.Len(t, *got, 1)
	require.Empty(t, (*got)[0].updates)
}

// A surface that does carry buttons still closes the one before it. The fix
// must not disable closePrevious generally — that would leave two live lists
// whose numbers mean different things.
func TestAButtonedSurfaceStillClosesThePreviousOne(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	liveNudge(t, store, p)

	chat, got := chatRecorder("m-query")
	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(ctx, itemOf("?"), &p))

	require.Len(t, *got, 1)
	require.Equal(t, []string{"m-nudge"}, (*got)[0].updates,
		"two live numbered button surfaces would make a tapped 1 ambiguous")
}

// Squirrel must never say nothing is outstanding while something is. Every
// other surface depends on believing what it says, and a bare `done` is the
// shortest thing you can type.
func TestBareDoneStillFindsTheChoreAfterListingThePile(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	liveNudge(t, store, p)
	insertItem(t, store, p, "buy milk")

	chat, _ := chatRecorder("m-notes")
	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(ctx, itemOf("!notes"), &p))

	reply := triage(t, store, p, "done")
	require.Contains(t, reply, "vacuum",
		"a buttonless list between the nudge and the answer must not hide the chore")
	require.NotContains(t, reply, "Nothing outstanding")

	due, err := store.DueChores(ctx, p, time.Now())
	require.NoError(t, err)
	require.Empty(t, due, "and it must actually have been completed")
}

// A typed position still means the newest list. Only "the one thing
// outstanding" reaches back past a buttonless list, because that question was
// never about the newest message.
func TestATypedPositionStillMeansTheNewestList(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	liveNudge(t, store, p)
	insertItem(t, store, p, "buy milk")

	chat, _ := chatRecorder("m-notes")
	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(ctx, itemOf("!notes"), &p))

	require.Contains(t, triage(t, store, p, "done 1"), "buy milk")

	chores, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Len(t, chores, 1, "the chore is untouched: line 1 was the note")
}

// The interval must come from what was typed, never from the note. Before this
// the note supplied the missing unit: `!chore 1 every` against "week groceries"
// parsed as "every week groceries" and created a weekly chore nobody asked for.
func TestPromotingNeverBorrowsTheUnitFromTheNote(t *testing.T) {
	for _, tc := range []struct{ note, cmd string }{
		{"week groceries", "!chore 1 every"},
		{"days off in march", "!chore 1 every 2"},
		{"months are long", "!chore 1 every"},
	} {
		t.Run(tc.note, func(t *testing.T) {
			store := withStore(t)
			ctx := context.Background()
			p := owner(t, store)

			pileOf(t, store, p, tc.note)
			reply := triage(t, store, p, tc.cmd)

			chores, err := store.ActiveChores(ctx, p)
			require.NoError(t, err)
			require.Empty(t, chores,
				"an incomplete interval must not become a chore on a schedule nobody typed")
			require.Contains(t, reply, "How often")
		})
	}
}

// The pile and the evening list have to agree about what a note is. An
// attachment-only Campfire message lands as an empty row, which CapturesSince
// has always filtered and the pile did not — so the pile printed a blank
// numbered line for it.
func TestAnEmptyItemIsNotALineInThePile(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	insertItem(t, store, p, "buy milk")
	insertItem(t, store, p, "")

	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "buy milk", items[0].RawText)

	captures, err := store.CapturesSince(ctx, p, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, captures, 1, "the two surfaces must agree, which is the point")
}

// noSuchLine means "that surface is gone". Saying it about a line printed a
// second ago is the one thing its own doc comment forbids.
func TestStopAgainstANoteLineSaysItIsANote(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	pileOf(t, store, p, "buy milk")

	reply := triage(t, store, p, "stop 1")
	require.Contains(t, reply, "not a chore")
	require.NotContains(t, reply, "don't have a line")
}

func TestStopAgainstAChoreLineStillStopsIt(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	liveNudge(t, store, p)

	require.Contains(t, triage(t, store, p, "stop 1"), "Stopped")

	chores, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Empty(t, chores)
}

// "!notes to self: the boiler man comes tuesday" printed the pile and dropped
// the sentence. The spec names "notes to self" as its motivating example for
// why commands need a prefix at all, so this is the one shape that must not
// vanish quietly.
func TestNotesWithAnArgumentDoesNotSilentlyDiscardIt(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	insertItem(t, store, p, "buy milk")

	chat, got := chatRecorder("m-1")
	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(ctx, itemOf("!notes to self: the boiler man comes tuesday"), &p))

	text := (*got)[0].message.Text
	require.NotContains(t, text, "1. buy milk",
		"printing the pile and dropping the sentence is the worst of both")
	require.Contains(t, text, "!notes", "say what the commands are")
}
