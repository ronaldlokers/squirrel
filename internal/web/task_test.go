package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A TASK said the wrong thing about itself and offered no way back.
//
// It is the one answer on the card that moves a note between two screens
// rather than ending it — and it was the one with no word of its own in either
// vocabulary, so the stamp fell through to the default and a note promoted to
// a task announced itself as "marked done". The four dispositions were audited
// when set-aside was fixed; the fifth answer was not.
func TestPromotingANoteSaysItBecameATask(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(3, "ring the vet", squirrel.ItemOpen)}}
	routed(t, f).call(t, "POST", "/pile/act", strings.NewReader("id=3&act=task&from=thread"))

	require.NotEmpty(t, f.appended)
	// What you said carries the verb; the reply's wording varies by the day.
	// The defect this pins is that the fifth answer had no word of its own and
	// fell through to "marked done" — so what must never happen is the wrong
	// verb, on either side of the exchange.
	require.Contains(t, f.appended[0].Words, "a task",
		"the note was promoted and nothing true was said about it")
	require.Contains(t, squirrel.Sayings(squirrel.SayingDecided), f.appended[1].Words)
	require.NotContains(t, f.appended[0].Words, "done")
	require.NotContains(t, f.appended[1].Words, "done")
}

// And the way back is its own verb.
//
// The note's state never moved — what changed was its kind — so `act=open`
// undoes nothing at all. Putting it back means making it a note again, which
// is the same lesson setting-aside landed on: some transitions come back by
// their own route rather than by being moved to a state.
func TestTheWayBackFromATaskIsMakingItANoteAgain(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(3, "ring the vet", squirrel.ItemOpen)}}
	m := routed(t, f)

	m.call(t, "POST", "/pile/act", strings.NewReader("id=3&act=task&from=thread"))
	shown := string(f.appended[1].Shown)

	require.Contains(t, shown, "put it back",
		"a promotion offered no way to change your mind")
	require.Contains(t, shown, `"act":"note"`,
		"the way back moves the note's state, which is not what changed")

	m.call(t, "POST", "/pile/act", strings.NewReader("id=3&act=note&from=thread"))
	require.Equal(t, squirrel.ItemNote, f.items[0].Kind,
		"putting it back did not make it a note again")
}

// Undoing is not itself a thing to be offered an undo for.
func TestPuttingATaskBackOffersNoFurtherUndo(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{task(3, "ring the vet", squirrel.ItemOpen)}}
	m := mounted(t, f)

	to := m.call(t, "POST", "/pile/act",
		strings.NewReader("id=3&act=note")).Header().Get("Location")

	q, err := url.Parse(to)
	require.NoError(t, err)
	require.Empty(t, q.Query().Get("undo"))
}

// The card's stamp and the next page's line are the same words for the same
// press, with script and without it.
func TestTheCardAndThePageAgreeAboutATask(t *testing.T) {
	js, err := staticFS.ReadFile("static/pile.js")
	require.NoError(t, err)

	require.Equal(t, "now a task", saidWords["task"])
	require.Contains(t, string(js), `said: "now a task"`)
}
