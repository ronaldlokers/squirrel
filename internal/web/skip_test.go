package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestSkippingShowsTheNextOldestNote(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		note(3, "the third thought", squirrel.ItemOpen),
		note(2, "the second thought", squirrel.ItemOpen),
		note(1, "the first thought", squirrel.ItemOpen),
	}}
	body := mounted(t, f).call(t, "GET", "/pile?after=3", nil).Body.String()

	require.Contains(t, body, "the second thought")
	require.NotContains(t, body, "the third thought", "the skipped note is behind you")
}

// Nothing older is not an empty pile: everything skipped past is still there,
// and the page has to say which of the two it is.
func TestTheEndOfTheSkippedPileIsNotAnEmptyPile(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/pile?after=1", nil).Body.String()

	require.NotContains(t, body, "nothing in the pile")
	require.Contains(t, body, "start at the top")
	require.Contains(t, body, `href="/pile"`)
}

func TestAnEmptyPileStillReadsAsEmptyWithACursor(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/pile", nil).Body.String()
	require.Contains(t, body, "nothing in the pile")
}

// Triaging while skipped keeps your place: the redirect carries the cursor the
// same way it carries a search.
func TestATransitionKeepsWhereYouWere(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		note(2, "the second thought", squirrel.ItemOpen),
		note(1, "the first thought", squirrel.ItemOpen),
	}}
	m := mounted(t, f)

	w := post(t, m, "/pile/act", url.Values{"id": {"1"}, "act": {"done"}, "after": {"2"}})

	require.Equal(t, 303, w.Code)
	require.Contains(t, w.Header().Get("Location"), "after=2")
}

func TestPromotionKeepsWhereYouWere(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "bins out", squirrel.ItemOpen)}}
	m := mounted(t, f)

	w := post(t, m, "/pile/chore", url.Values{"id": {"1"}, "every": {"every week"}, "after": {"7"}})

	require.Contains(t, w.Header().Get("Location"), "after=7")
}

// The card carries the cursor into every form on it, or acting on a skipped-to
// note would silently take you back to the top.
func TestTheCardCarriesTheCursor(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		note(2, "the second thought", squirrel.ItemOpen),
		note(1, "the first thought", squirrel.ItemOpen),
	}}
	body := mounted(t, f).call(t, "GET", "/pile?after=2", nil).Body.String()

	require.Contains(t, body, `name="after" value="2"`)
	require.Equal(t, 1, strings.Count(body, `data-skip="1"`),
		"the deck says which note space moves past")
}

// A cursor is a number from an address bar. Nonsense in it is no cursor, not an
// error page.
func TestAnUnreadableCursorIsNoCursor(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(9, "buy milk", squirrel.ItemOpen)}}
	for _, q := range []string{"?after=abc", "?after=", "?after=-4"} {
		body := mounted(t, f).call(t, "GET", "/pile"+q, nil).Body.String()
		require.Contains(t, body, "buy milk", "query %q", q)
	}
}
