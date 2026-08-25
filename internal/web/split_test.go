package web

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Nothing here is automatic and nothing is silent. The press is only drawn on
// a note that looks like several things, the pieces are only written after a
// second press, and what you actually typed is never rewritten.

func splitting(c *fakeCoach, pieces ...string) *fakeCoach {
	c.pieces = pieces
	c.splittable = true
	return c
}

func aDump() *fakeStore {
	return &fakeStore{items: []squirrel.Item{
		note(1, "ring the vet, put the bins out and do the tax", squirrel.ItemOpen),
	}}
}

// With no coach at all the pile is exactly the screen it was.
func TestTheCardOffersNothingWithNoCoach(t *testing.T) {
	require.NotContains(t, mounted(t, aDump()).call(t, "GET", "/", nil).Body.String(),
		"this is more than one thing")
}

// The pieces travel in the form that renders them. Nothing is stored, so there
// is no pending proposal anywhere to expire — it lasts exactly as long as the
// page it is on.
func TestTheProposalTravelsInTheForm(t *testing.T) {
	// The pieces travel in the form, so one cannot be applied without the
	// press — and they are drawn from the turn's own record of what was
	// proposed rather than asked for again.
	f := aDump()
	f.checkin = &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: now()}
	m := routedSplitting(t, f, "ring the vet", "put the bins out")
	m.call(t, "POST", "/pile/split", strings.NewReader("act=propose&id=1&from=thread"))
	f.turns, f.appended = append(f.turns, f.appended...), nil
	body := m.call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `name="piece" value="ring the vet"`)
	require.Contains(t, body, `name="piece" value="put the bins out"`)
}

// The second press is the one that writes, and what you typed is kept rather
// than dropped: a machine's reading of your words must never be the only
// surviving version of them.
func TestKeepingWritesThePiecesAndKeepsTheOriginal(t *testing.T) {
	f := aDump()

	w := mounted(t, f).call(t, "POST", "/pile/split", strings.NewReader(
		"act=keep&id=1&piece=ring+the+vet&piece=put+the+bins+out"))

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, []string{"ring the vet", "put the bins out"}, f.inserted)
	require.Equal(t, map[int64]squirrel.ItemState{1: squirrel.ItemKept}, f.states)

	// It goes through the same transition every other exit uses, and what is
	// said about it is said the same way — see TestKeepingASplitInTheThread.
	require.Equal(t, "/", w.Header().Get("Location"))
}

// One piece is not a split, and a form that arrives with one is a form that
// was edited. Nothing is written.
func TestKeepingOnePieceWritesNothing(t *testing.T) {
	f := aDump()
	mounted(t, f).call(t, "POST", "/pile/split",
		strings.NewReader("act=keep&id=1&piece=ring+the+vet"))

	require.Empty(t, f.inserted)
	require.Empty(t, f.states)
}

// A coach that cannot answer leaves the pile exactly as it was, and says
// nothing about having tried.
func TestACoachThatCannotSplitChangesNothing(t *testing.T) {
	f := aDump()
	c := &fakeCoach{splittable: true}

	w := mountedWith(t, f, c).call(t, "POST", "/pile/split", strings.NewReader("act=propose&id=1"))
	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/", w.Header().Get("Location"))
	require.Empty(t, f.inserted)
}

// The pieces are ordinary notes. Nothing marks them as a machine's doing,
// because by the time they exist a button was pressed to say they are yours.
func TestThePiecesAreOrdinaryNotes(t *testing.T) {
	f := aDump()
	mounted(t, f).call(t, "POST", "/pile/split",
		strings.NewReader("act=keep&id=1&piece=ring+the+vet&piece=put+the+bins+out"))

	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()
	require.NotContains(t, body, "suggested")
	require.NotContains(t, body, "split from")
}

// The proposal, in the conversation. Nothing is written until the press: the
// pieces are words in a turn, and a proposal in scrollback has lost its button.
func TestProposingASplitInTheThread(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		note(9, "call the vet and book the car in", squirrel.ItemOpen),
	}}
	routedSplitting(t, f, "call the vet", "book the car in").call(t, "POST", "/pile/split",
		strings.NewReader("id=9&act=propose&from=thread"))

	require.Len(t, f.appended, 2)
	shown := string(f.appended[1].Shown)
	require.Contains(t, shown, "call the vet")
	require.Contains(t, shown, "book the car in")
	require.Empty(t, f.inserted, "nothing is written until the press")
	require.Empty(t, f.states)
}

// Pressing it writes the pieces, keeps the note, and hands you the next one.
func TestKeepingASplitInTheThread(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		note(9, "call the vet and book the car in", squirrel.ItemOpen),
		note(8, "meter reading", squirrel.ItemOpen),
	}}
	routedSplitting(t, f, "call the vet", "book the car in").call(t, "POST", "/pile/split", strings.NewReader(
		"id=9&act=keep&from=thread&piece=call+the+vet&piece=book+the+car+in"))

	require.Len(t, f.inserted, 2)
	require.Equal(t, squirrel.ItemKept, f.states[9], "the note itself is kept, not dropped")
	// And hands you a note again. It is one of the pieces, because a piece is
	// an ordinary note and the newest open note is what the pile hands you —
	// the deck would have shown the same thing.
	require.Len(t, f.appended, 3)
	require.Contains(t, string(f.appended[2].Shown), `"cards"`)
}

// A note it reads as one thing says so rather than answering with silence.
func TestANoteThatIsOneThingSaysSo(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(9, "milk", squirrel.ItemOpen)}}
	m := routed(t, f) // no splitter configured
	m.call(t, "POST", "/pile/split", strings.NewReader("id=9&act=propose&from=thread"))

	require.Empty(t, f.appended, "with no splitter there is nothing to say")
}

// Both are pinned in the conversation now: the press is drawn only on a note
// that looks like several things, and nothing is written when a split is
// proposed — see TestProposingASplitInTheThread.
