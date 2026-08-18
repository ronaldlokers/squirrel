package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestTheCardCarriesWhatTheScriptNeeds(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/pile", nil).Body.String()

	for _, hook := range []string{`id="card"`, `data-id="1"`, `id="stamp"`, `id="undoRow"`, `data-act="done"`} {
		require.Contains(t, body, hook, "the script hangs off %s", hook)
	}
}

func TestEveryActionIsAFormSubmissionNotAScriptHook(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/pile", nil).Body.String()

	require.Contains(t, body, `method="post"`)
	require.Contains(t, body, `action="/pile/act"`)
	require.NotContains(t, body, "onclick=",
		"behaviour lives in pile.js; a page that needs inline handlers is a page that fails without them")
	require.Equal(t, strings.Count(body, `name="act"`), strings.Count(body, "<button class=\"btn\""),
		"every action button submits a value the server understands")
}

// The chore chips are the same rule one level down: each submits an interval
// the server can read, to the route that reads it.
func TestTheChoreDisclosureSubmitsWithoutScript(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "bins out", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/pile", nil).Body.String()

	require.Contains(t, body, "<details")
	require.Equal(t, 4, strings.Count(body, `name="every"`))
	require.Equal(t, 4, strings.Count(body, `formaction="/pile/chore"`))
}

// The comp replaces the actions row with the interval row while you choose.
// The scriptless page cannot hide anything, so the way back out is rendered
// hidden and only exists once pile.js is there to work it.
func TestTheNeverMindChipIsAnEnhancement(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "bins out", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/pile", nil).Body.String()

	require.Contains(t, body, `class="chip back"`)
	require.Contains(t, body, `data-close="chore"`)
	require.Contains(t, body, `type="button"`,
		"a button that is not a submit cannot post a half-made chore")
}
