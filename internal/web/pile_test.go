package web

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// TestTheRouteTable pins the shape of the screen rather than describing it. The
// table is the spec's table, and a route that moves has to move here first.
func TestTheRouteTable(t *testing.T) {
	m := mounted(t, &fakeStore{})

	for _, route := range []string{
		"GET /{$}",
		"GET /pile",
		"POST /capture",
		"POST /mood",
		"POST /pile/act",
		"POST /pile/chore",
		"POST /pile/fix",
		"GET /chores",
		"GET /kept",
		"POST /chores/act",
		"GET /pile/chores",
		"GET /manifest.webmanifest",
		"GET /sw.js",
		"GET /static/",
	} {
		require.Contains(t, m.routes, route, "the route table lost %s", route)
	}
	require.Len(t, m.routes, 14, "a route was added without being pinned here")
}

// The chores screen lived at /pile/chores for its whole life, and a bookmark
// that dies quietly is worse than a redirect nobody notices.
func TestTheOldChoresURLRedirects(t *testing.T) {
	w := mounted(t, &fakeStore{}).call(t, "GET", "/pile/chores", nil)

	require.Equal(t, http.StatusMovedPermanently, w.Code)
	require.Equal(t, "/chores", w.Header().Get("Location"))
}

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

	// The rule is about the fact, not the digit. A bare digit test reads the
	// lid's SVG geometry and the asset stamp — a content hash, which is free to
	// contain "40" and did — so what is pinned here is every shape a total
	// could take in prose, plus the cap that makes "there is more" true.
	lower := strings.ToLower(body)
	for _, total := range []string{"41 notes", "40 notes", "41 more", "of 41", "(41)", "1 of "} {
		require.NotContains(t, lower, total)
	}
	require.NotContains(t, body, "note number 2", "the deck shows one card")
	require.Contains(t, lower, "more")
}

func TestEmptyPileDoesNotCelebrate(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/pile", nil).Body.String()

	require.Contains(t, body, "nothing in the pile")
	for _, forbidden := range []string{"well done", "all done", "congrat", "🎉", "streak"} {
		require.NotContains(t, strings.ToLower(body), forbidden)
	}
}

// The slot lives on home and nowhere else. A capture box above the deck is the
// inbox shape this product refuses: what you are adding, directly over what
// you have not dealt with.
//
// The deck does have a field now, and the distinction is exact rather than
// cosmetic — it corrects a note that already exists, so it is bound to that
// note's id and it can never bring a new one into being. What is banned here
// is a way to create, not a way to type.
func TestPileHasNoCaptureBox(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "kaas", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/pile", nil).Body.String()

	require.NotContains(t, body, `action="/capture"`, "capture belongs to home")
	// Every field you can type a sentence into on this screen names the note
	// it is correcting.
	require.Equal(t, strings.Count(body, "<textarea"),
		strings.Count(body, `action="/pile/fix"`),
		"a field on the deck edits one note, and says which")
	require.Contains(t, body, `<input type="hidden" name="id" value="1">`)
}

func TestPileFailsVisiblyWhenTheDatabaseIsDown(t *testing.T) {
	f := &fakeStore{err: errTest}
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

// The routes are registered before the server listens, so they can be reached
// before Postgres has ever answered and before anyone knows whose pile this
// is. That is the database being unreachable, and the screen says so.
func TestPileWaitsVisiblyForItsOwner(t *testing.T) {
	m := newTestMux()
	require.NoError(t, Mount(m, &fakeStore{}, Options{
		IdentityHeader: "X-Authentik-Username", Identity: "ronald",
		Owner: func() int64 { return 0 },
	}))

	w := m.call(t, "GET", "/pile", nil)
	require.Equal(t, 503, w.Code)
	require.Contains(t, w.Body.String(), "cannot reach")
}

func TestMountRefusesWithoutAnOwner(t *testing.T) {
	require.Error(t, Mount(newTestMux(), &fakeStore{}, Options{
		IdentityHeader: "X-Authentik-Username", Identity: "ronald",
	}))
}
