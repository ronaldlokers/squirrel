//go:build integration

package squirrel_test

import (
	"context"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// pileOf lists the pile so a numbered surface exists, then returns an Applier
// ready for the triage command under test. Every triage test needs both, and
// doing it by hand each time invites a test that resolves against no surface
// at all and passes for the wrong reason.
func pileOf(t *testing.T, store *squirrel.Store, personID int64, notes ...string) {
	t.Helper()
	ctx := context.Background()

	for _, n := range notes {
		insertItem(t, store, personID, n)
	}
	// Its own message id per call, for the reason triage's own comment gives:
	// a fixed one means two pilings in a single test store the same
	// external_message_id, and the unique index correctly refuses the second.
	chat, _ := chatRecorder(strconv.FormatInt(replyIDs.Add(1), 10))
	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(ctx, itemOf("!notes"), &personID))
}

func stateOf(t *testing.T, store *squirrel.Store, itemID int64) string {
	t.Helper()
	var state string
	require.NoError(t, store.Pool().QueryRow(context.Background(),
		`select state from items where id = $1`, itemID).Scan(&state))
	return state
}

// replyIDs hands every reply its own message id.
//
// A fixed id was fine while at most one reply per test recorded a prompt.
// Since the hand-off, a completion's own reply carries a numbered surface too,
// so two calls in one test would store the same external_message_id twice and
// the unique index would — correctly — refuse it. Campfire's ids are unique;
// the helper's were the thing that was not.
var replyIDs atomic.Int64

func triage(t *testing.T, store *squirrel.Store, personID int64, text string) string {
	t.Helper()
	chat, got := chatRecorder(strconv.FormatInt(replyIDs.Add(1), 10))
	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(context.Background(), itemOf(text), &personID))
	require.Len(t, *got, 1)
	return (*got)[0].message.Text
}

func TestDoneAgainstANoteLineMarksItDone(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	pileOf(t, store, p, "buy milk")
	id := lineItemID(t, store, p, 1)

	reply := triage(t, store, p, "done 1")
	require.Equal(t, "done", stateOf(t, store, id))
	require.Contains(t, reply, "buy milk")
}

func TestKeepAgainstANoteLineKeepsIt(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	pileOf(t, store, p, "boiler serial is 44Q")
	id := lineItemID(t, store, p, 1)

	reply := triage(t, store, p, "keep 1")
	require.Equal(t, "kept", stateOf(t, store, id))
	require.Contains(t, reply, "44Q")
}

func TestDropAgainstANoteLineDropsIt(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	pileOf(t, store, p, "that thing I no longer care about")
	id := lineItemID(t, store, p, 1)

	triage(t, store, p, "drop 1")
	require.Equal(t, "dropped", stateOf(t, store, id))
}

func TestATriagedNoteLeavesThePile(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	pileOf(t, store, p, "buy milk", "the boiler thing")
	triage(t, store, p, "done 1")

	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, items, 1)
}

// The spec is explicit that the evening window is a question about
// received_at, not about state: the evening message reports what you told
// Squirrel, not what is outstanding. Nothing else in this plan would catch
// someone helpfully filtering triaged notes out of it.
func TestATriagedNoteStillAppearsInThatEveningsCaptures(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	pileOf(t, store, p, "buy milk")
	triage(t, store, p, "done 1")

	captures, err := store.CapturesSince(ctx, p, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Contains(t, captures, "buy milk",
		"the evening message reports what you said, not what is still outstanding")
}

// A tap is a state assertion, not a delta, so the same command twice is the
// same state twice.
func TestARepeatedTriageIsANoOp(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	pileOf(t, store, p, "buy milk")
	id := lineItemID(t, store, p, 1)

	triage(t, store, p, "done 1")
	triage(t, store, p, "done 1")
	require.Equal(t, "done", stateOf(t, store, id))
}

// Every transition reverses. `keep 1` after `done 1` is a correction, not an
// error, and the pile is where corrections happen most.
func TestATriageCanBeCorrected(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	pileOf(t, store, p, "buy milk")
	id := lineItemID(t, store, p, 1)

	triage(t, store, p, "drop 1")
	require.Equal(t, "dropped", stateOf(t, store, id))
	triage(t, store, p, "keep 1")
	require.Equal(t, "kept", stateOf(t, store, id))
}

func TestDoneAgainstAChoreLineStillCompletesTheChore(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	promptID, err := store.RecordPrompt(ctx, p, "9", "query", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, promptID, "m-0", time.Now()))

	reply := triage(t, store, p, "done 1")
	require.Contains(t, reply, "vacuum")
	require.Contains(t, reply, "next in")

	due, err := store.DueChores(ctx, p, time.Now())
	require.NoError(t, err)
	require.Empty(t, due, "completing it must have reset the chore's clock")
}

// `keep 2` aimed at a chore is a real mistake. A bot that silently does
// nothing looks broken in exactly the way that stops you trusting it.
func TestKeepAgainstAChoreLineSaysSo(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	promptID, err := store.RecordPrompt(ctx, p, "9", "query", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, promptID, "m-0", time.Now()))

	reply := triage(t, store, p, "keep 1")
	require.Contains(t, strings.ToLower(reply), "chore")
}

// The regression guard for the naming decision: `drop N` is the numbered form
// and touches a note, while a bare `nvm` still undoes a chore the matcher made
// from what was meant as a note. They share an intent kind and are told apart
// by position alone.
func TestNvmStillUndoesAChoreAndDoesNotTouchNotes(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	chat, _ := chatRecorder("m-1")
	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(ctx, itemOf("every 2 weeks: vacuum"), &p))

	pileOf(t, store, p, "buy milk")
	id := lineItemID(t, store, p, 1)

	triage(t, store, p, "nvm")

	chores, err := store.ActiveChores(ctx, p)
	require.NoError(t, err)
	require.Empty(t, chores, "nvm undoes the chore")
	require.Equal(t, "open", stateOf(t, store, id), "and leaves every note alone")
}

func TestTriageBeyondTheLastLineSaysSo(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	pileOf(t, store, p, "buy milk")

	for _, cmd := range []string{"done 7", "keep 7", "drop 7"} {
		require.Contains(t, triage(t, store, p, cmd), "line 7", "for %q", cmd)
	}
}

// A long note is quoted back trimmed, sliced by rune rather than by byte.
// Phase 2 crash-looped the pod on a chore name containing Ⱥ because a byte
// slice cut a rune in half.
//
// The input is one ASCII byte followed by three-byte runes, deliberately: with
// an all-Ⱥ string the 60-byte cut lands exactly on a rune boundary and a
// byte-slicing implementation passes. The single leading "x" shifts every
// boundary by one so byte 60 falls inside a rune, which is what makes this
// test able to fail at all.
func TestALongNoteIsShortenedWithoutCuttingARune(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	long := "x" + strings.Repeat("あ", 100)
	pileOf(t, store, p, long)

	reply := triage(t, store, p, "done 1")
	require.True(t, utf8.ValidString(reply), "a byte slice would leave a half rune here")
	require.True(t, strings.HasSuffix(reply, "…"))
	require.Less(t, len([]rune(reply)), 100)
}
