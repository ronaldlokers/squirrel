package web

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestPileShowsTheNewestOpenNote(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		note(1, "the boiler makes a noise on tuesdays", squirrel.ItemOpen),
		note(2, "buy milk", squirrel.ItemOpen),
		note(3, "boiler service is booked", squirrel.ItemDone),
	}}
	body := mounted(t, f).call(t, "GET", "/pile", nil).Body.String()

	require.Contains(t, body, "the boiler makes a noise on tuesdays")
	require.NotContains(t, body, "boiler service is booked", "a triaged note is not in the pile")
}

func TestPileNeverEmitsACount(t *testing.T) {
	items := []squirrel.Item{}
	for i := int64(1); i <= 41; i++ {
		items = append(items, note(i, "note number "+strconv.FormatInt(i, 10), squirrel.ItemOpen))
	}
	body := mounted(t, &fakeStore{items: items}).call(t, "GET", "/pile", nil).Body.String()

	require.NotContains(t, body, "41")
	require.NotContains(t, body, "40")
	// The rule is about the fact, not the digit: the page may say there is more.
	require.Contains(t, strings.ToLower(body), "more")
}

func TestEmptyPileDoesNotCelebrate(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/pile", nil).Body.String()

	require.Contains(t, body, "nothing in the pile")
	for _, forbidden := range []string{"well done", "all done", "congrat", "🎉", "streak"} {
		require.NotContains(t, strings.ToLower(body), forbidden)
	}
}

func TestPileHasNoCaptureBox(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "kaas", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/pile", nil).Body.String()

	require.NotContains(t, body, `name="text"`)
	require.NotContains(t, body, "<textarea")
	// Exactly one field you can type into, and it is the search field. The
	// hidden fields are excluded because they carry which note a form is
	// about; a note's id is not a place a thought can be entered, which is what
	// this rule is protecting.
	typeable := strings.Count(body, "<input") - strings.Count(body, `<input type="hidden"`)
	require.Equal(t, 1, typeable,
		"exactly one field you can type into, and it is the search field")
}

func TestPileFailsVisiblyWhenTheDatabaseIsDown(t *testing.T) {
	f := &fakeStore{err: errors.New("connection refused")}
	w := mounted(t, f).call(t, "GET", "/pile", nil)

	require.Equal(t, 503, w.Code)
	require.Contains(t, w.Body.String(), "cannot reach")
}

func TestSearchCrossesEveryState(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		note(1, "the boiler makes a noise", squirrel.ItemOpen),
		note(2, "boiler service is booked", squirrel.ItemDone),
		note(3, "boiler insurance thing", squirrel.ItemDropped),
		note(4, "boiler meter reading 48213", squirrel.ItemKept),
	}}
	body := mounted(t, f).call(t, "GET", "/pile?q=boiler", nil).Body.String()

	require.Contains(t, body, "IN THE PILE")
	require.Contains(t, body, "DONE")
	require.Contains(t, body, "DROPPED")
	require.Contains(t, body, "KEPT")
}

func TestSearchSaysThereIsMoreWithoutSayingHowMuch(t *testing.T) {
	items := []squirrel.Item{}
	for i := int64(1); i <= 9; i++ {
		items = append(items, note(i, "boiler "+strconv.FormatInt(i, 10), squirrel.ItemOpen))
	}
	body := mounted(t, &fakeStore{items: items}).call(t, "GET", "/pile?q=boiler", nil).Body.String()

	require.Contains(t, strings.ToLower(body), "more")
	// The list is capped, and the page says so in words. A bare digit test
	// would match the geometry in the lid's own SVG, so what is pinned here is
	// every shape a total could take in prose.
	for _, total := range []string{"9 results", "9 notes", "9 more", "of 9", "(9)"} {
		require.NotContains(t, strings.ToLower(body), total)
	}
	require.NotContains(t, body, "boiler 7", "the cap is what makes \"there is more\" true")
}

func TestSearchWithNoHitsSaysSo(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/pile?q=boiler", nil).Body.String()

	require.Contains(t, body, "nothing says")
}

func TestSearchEscapesTheQuery(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{}}
	body := mounted(t, f).call(t, "GET", "/pile?q=%3Cscript%3Ealert(1)%3C%2Fscript%3E", nil).Body.String()

	require.NotContains(t, body, "<script>alert(1)</script>")
	require.Contains(t, body, "&lt;script&gt;")
}
