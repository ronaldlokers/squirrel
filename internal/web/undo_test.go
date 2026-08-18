package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// With JavaScript the undo lives on the card during the hold. Without it there
// is no card left to hold — the write has landed and the pile has moved on —
// so the redirect carries the undo and the next page renders it.
func TestTheUndoOutlivesTheCard(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(2, "buy milk", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/pile?undo=1&was=open&state=done", nil).Body.String()

	require.Contains(t, body, "PUT IT BACK")
	require.Contains(t, body, "marked done")
	require.Contains(t, body, `name="id" value="1"`)
	require.Contains(t, body, `name="act" value="open"`, "undo is the transition back to what it was")
}

// The last note in the pile leaves an empty deck behind, which is exactly when
// an undo is most likely to be wanted.
func TestTheUndoSurvivesAnEmptyPile(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/pile?undo=1&was=open&state=drop", nil).Body.String()

	require.Contains(t, body, "PUT IT BACK")
	require.Contains(t, body, "nothing in the pile")
}

// A triage from the results list comes back to the results list, so the undo
// has to carry the search with it.
func TestTheUndoKeepsTheSearch(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "the boiler", squirrel.ItemDone)}}
	body := mounted(t, f).call(t, "GET", "/pile?q=boiler&undo=1&was=open&state=done", nil).Body.String()

	require.Contains(t, body, "PUT IT BACK")
	require.Contains(t, body, `name="q" value="boiler"`)
}

func TestAnUnreadableUndoIsNoUndo(t *testing.T) {
	for _, query := range []string{
		"",
		"?undo=abc&was=open&state=done",
		"?undo=1&was=nonsense&state=done",
		"?undo=1&state=done",
		"?undo=0&was=open&state=done",
	} {
		body := mounted(t, &fakeStore{}).call(t, "GET", "/pile"+query, nil).Body.String()
		require.NotContains(t, body, "PUT IT BACK", "query %q", query)
	}
}

// The whole round trip: a transition answers with a redirect, and the page that
// redirect names carries the way back.
func TestTheRedirectAfterATransitionCarriesTheUndo(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	m := mounted(t, f)

	w := post(t, m, "/pile/act", url.Values{"id": {"1"}, "act": {"drop"}})
	require.Equal(t, 303, w.Code)

	target := w.Header().Get("Location")
	require.True(t, strings.HasPrefix(target, "/pile?"), "landed at %q", target)

	body := m.call(t, "GET", target, nil).Body.String()
	require.Contains(t, body, "PUT IT BACK")
	require.Contains(t, body, "dropped")
}

// Promotion is the one transition whose undo is not simply the state it was:
// the chore it created stays, because a chore is not a note's state.
func TestTheUndoAfterAPromotionPutsTheNoteBack(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "bins out", squirrel.ItemOpen)}}
	m := mounted(t, f)

	w := post(t, m, "/pile/chore", url.Values{"id": {"1"}, "every": {"every week"}})
	body := m.call(t, "GET", w.Header().Get("Location"), nil).Body.String()

	require.Contains(t, body, "now a chore")
	require.Contains(t, body, `name="act" value="open"`)
}
