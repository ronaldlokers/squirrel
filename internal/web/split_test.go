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

// The press is on the card, and only on a note worth asking about.
func TestTheCardOffersToSplitOnlyWhenItLooksLikeSeveralThings(t *testing.T) {
	c := splitting(&fakeCoach{}, "ring the vet", "put the bins out")
	require.Contains(t, mountedWith(t, aDump(), c).call(t, "GET", "/pile", nil).Body.String(),
		"this is more than one thing")

	quiet := &fakeCoach{}
	require.NotContains(t, mountedWith(t, aDump(), quiet).call(t, "GET", "/pile", nil).Body.String(),
		"this is more than one thing")
}

// With no coach at all the pile is exactly the screen it was.
func TestTheCardOffersNothingWithNoCoach(t *testing.T) {
	require.NotContains(t, mounted(t, aDump()).call(t, "GET", "/pile", nil).Body.String(),
		"this is more than one thing")
}

// Asking proposes, and proposing writes nothing.
func TestProposingWritesNothing(t *testing.T) {
	f := aDump()
	c := splitting(&fakeCoach{}, "ring the vet", "put the bins out", "do the tax")

	body := mountedWith(t, f, c).
		call(t, "POST", "/pile/split", strings.NewReader("act=propose&id=1")).Body.String()

	require.Contains(t, body, "IS THIS WHAT YOU MEANT")
	require.Contains(t, body, "put the bins out")
	// The note it came from is still on the card, unchanged, because the
	// question cannot be answered without both.
	require.Contains(t, body, "ring the vet, put the bins out and do the tax")
	require.Empty(t, f.inserted, "a proposal wrote something")
	require.Empty(t, f.states, "a proposal moved the note")
}

// The pieces travel in the form that renders them. Nothing is stored, so there
// is no pending proposal anywhere to expire — it lasts exactly as long as the
// page it is on.
func TestTheProposalTravelsInTheForm(t *testing.T) {
	c := splitting(&fakeCoach{}, "ring the vet", "put the bins out")
	body := mountedWith(t, aDump(), c).
		call(t, "POST", "/pile/split", strings.NewReader("act=propose&id=1")).Body.String()

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

	// It goes through the same transition every other exit uses, so undo works
	// on it exactly as it works on anything else.
	require.Contains(t, w.Header().Get("Location"), "undo=1")
	require.Contains(t, w.Header().Get("Location"), "was=open")
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
	require.Equal(t, "/pile", w.Header().Get("Location"))
	require.Empty(t, f.inserted)
}

// The pieces are ordinary notes. Nothing marks them as a machine's doing,
// because by the time they exist a button was pressed to say they are yours.
func TestThePiecesAreOrdinaryNotes(t *testing.T) {
	f := aDump()
	mounted(t, f).call(t, "POST", "/pile/split",
		strings.NewReader("act=keep&id=1&piece=ring+the+vet&piece=put+the+bins+out"))

	body := mounted(t, f).call(t, "GET", "/pile", nil).Body.String()
	require.NotContains(t, body, "suggested")
	require.NotContains(t, body, "split from")
}
