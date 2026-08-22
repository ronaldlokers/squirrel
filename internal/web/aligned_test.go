//go:build browser

// Five things that are only wrong on the screen.
//
// Each of these looked right in the source: a min-width that was stated, a
// role that was documented, a panel that hangs from its control, a
// destructive press that was deliberately not full width, two list cards
// built from the same stock. What the browser drew was a staggered column, a
// label in body text, a panel two pixels off the side of a phone, an ending
// eight pixels under a doing, and two lists whose words start four pixels
// apart.
//
// So, like the cascade tests, these ask the browser for the computed value and
// compare against a sibling rather than against a number wherever the point is
// that two things must match.
package web

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// lefts is the left edge of every element matching sel, rounded, in document
// order.
const lefts = `(sel) => Array.from(document.querySelectorAll(sel))
	.map(el => Math.round(el.getBoundingClientRect().left))`

// edges is one element's box, rounded, so a test can compare two of them.
const edges = `(sel) => {
	const el = document.querySelector(sel);
	if (!el) throw new Error("no such element: " + sel);
	const b = el.getBoundingClientRect();
	return {left: Math.round(b.left), right: Math.round(b.right),
	        top: Math.round(b.top), bottom: Math.round(b.bottom)};
}`

func numbers(t *testing.T, c *cdp, expression string) []float64 {
	t.Helper()
	raw, ok := c.eval(t, expression).([]any)
	require.True(t, ok, "%s did not come back as a list", expression)
	out := make([]float64, 0, len(raw))
	for _, v := range raw {
		n, ok := v.(float64)
		require.True(t, ok, "%#v is not a number", v)
		out = append(out, n)
	}
	return out
}

func box(t *testing.T, c *cdp, sel string) map[string]float64 {
	t.Helper()
	raw, ok := c.eval(t, fmt.Sprintf("return (%s)(%q)", edges, sel)).(map[string]any)
	require.True(t, ok, "no box for %s", sel)
	out := map[string]float64{}
	for k, v := range raw {
		n, ok := v.(float64)
		require.True(t, ok, "%s of %s is not a number", k, sel)
		out[k] = n
	}
	return out
}

// A fortnight of readings whose day names are deliberately not the same
// length: "today" and "yesterday" are two words the renderer special-cases,
// and everything older is "wednesday 19 august".
func aFortnight() *fakeStore {
	now := time.Now()
	return &fakeStore{readings: []squirrel.Checkin{
		{Mood: squirrel.MoodCalm, SaidAt: now},
		{Mood: squirrel.MoodLow, SaidAt: now.AddDate(0, 0, -1)},
		{Mood: squirrel.MoodGood, SaidAt: now.AddDate(0, 0, -3)},
		{Mood: squirrel.MoodFrazzled, SaidAt: now.AddDate(0, 0, -4)},
	}}
}

// The readings are a column, and a column that starts in a different place on
// every row is a stagger.
//
// `.aday` was a flex row, so each day's name sized itself and pushed its own
// face: "thursday 20 august" started its reading 47px further right than
// "today" did. A `min-width` was stated where the row is built and a second,
// smaller one later in the file outranked it — so the floor that was supposed
// to hold the column together was 8em, and a date is wider than that.
func TestBrowserEveryMoodReadingStartsInTheSameColumn(t *testing.T) {
	srv := screen(t, aFortnight())
	c := browserAt(t, srv, "/moods")

	starts := numbers(t, c, "return ("+lefts+`)(".aday .saidwhat")`)
	require.Len(t, starts, 4, "the screen did not draw four readings")

	for i, at := range starts {
		require.Equal(t, starts[0], at,
			"reading %d starts at %vpx and the first starts at %vpx: the day names are setting the column",
			i+1, at, starts[0])
	}
}

// `.lead` is the small line that says what the thing under it is, and the
// stylesheet says so: "the meta role, set in cream at 11.5px everywhere it is
// used". It was six scoped rules and no definition, so the three sites none of
// them reached — the search results' two group labels, COMES BACK inside a
// chore's interval row, RIGHT NOW above Buddy's context — rendered at the
// document's own size, larger and lighter than the thing they label.
//
// Against the body rather than against a number: the defect is not that a
// label was 16px, it is that it was the same size as running text.
func TestBrowserNoMetaLabelIsAsLargeAsBodyText(t *testing.T) {
	f := aPile()
	f.chores = []squirrel.Chore{{
		ID: 1, Name: "bins out", Every: 7 * 24 * time.Hour,
		EveryDays: 7, SinceDays: 6, Active: true, EverDone: true,
	}}
	f.checkin = &squirrel.Checkin{Mood: squirrel.MoodCalm, SaidAt: time.Now()}
	f.offer = &squirrel.Offer{
		Kind: squirrel.OfferChore, RefID: 1, Text: "bins out", Because: "it is bin day",
	}
	srv := screen(t, f)
	c := browserAt(t, srv, "/chores")

	for _, path := range []string{"/chores", "/buddy", "/pile?q=bins"} {
		c.navigate(t, srv.URL+path)

		found := c.eval(t, `
			const body = parseFloat(getComputedStyle(document.body).fontSize);
			return Array.from(document.querySelectorAll(".lead")).map(el => ({
				text: el.textContent.trim(),
				size: parseFloat(getComputedStyle(el).fontSize),
				body: body,
			}));`)

		labels, ok := found.([]any)
		require.True(t, ok, "%s did not answer with a list of labels", path)
		require.NotEmpty(t, labels, "%s drew no meta label at all", path)

		for _, l := range labels {
			label := l.(map[string]any)
			require.Less(t, label["size"], label["body"],
				"%s on %s is %vpx against body text at %vpx: it is not wearing the meta role",
				label["text"], path, label["size"], label["body"])
		}
	}
}

// Both lid panels end on the same edge, and it is inside the margin.
//
// Each hangs from the right edge of its own control, and the search icon is
// not the last control in the lid. So on a 390px phone the 320px field started
// two pixels from the side of the screen — outside the margin every other
// thing on the page keeps — and ended 54px short of the edge the menu's own
// panel uses.
func TestBrowserBothLidPanelsEndWhereThePageEnds(t *testing.T) {
	c := lid(t)

	c.eval(t, `document.querySelector("details.where > summary").click()`)
	c.until(t, "the map", `(`+shown+`)(".wherelist")`)
	menu := box(t, c, ".wherelist")

	c.eval(t, `document.querySelector("details.findbox > summary").click()`)
	c.until(t, "the field", `(`+shown+`)(".findbox .find")`)
	find := box(t, c, ".findbox .find")

	require.Equal(t, menu["right"], find["right"],
		"the field's panel ends at %vpx and the menu's at %vpx", find["right"], menu["right"])

	// The card is the page's own left margin, and the panel must not sit
	// outside it.
	card := box(t, c, "#card")
	require.GreaterOrEqual(t, find["left"], card["left"],
		"the field's panel starts at %vpx, outside the %vpx margin the page keeps",
		find["left"], card["left"])
}

// STOP ASKING is the one press on the chores that ends something, and on a
// phone the two-column grid put it directly under DID IT — same column, eight
// pixels down, which is the gap between any two cells. Nothing chose that
// distance for it.
//
// Against the grid's own gap rather than against a number: the point is that
// it is deliberately further away than the next cell would be.
func TestBrowserTheEndingIsNotTheNextThingUnderTheDoing(t *testing.T) {
	f := aPile()
	f.chores = []squirrel.Chore{{
		ID: 1, Name: "bins out", Every: 7 * 24 * time.Hour,
		EveryDays: 7, SinceDays: 6, Active: true, EverDone: true,
	}}
	srv := screen(t, f)
	c := browserAt(t, srv, "/chores")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 780, "deviceScaleFactor": 0, "mobile": true,
	})
	c.navigate(t, srv.URL+"/chores")

	did := box(t, c, ".chore .abtn")
	often := box(t, c, ".chore .often > summary")
	stop := box(t, c, ".chore .abtn.stop")

	require.Equal(t, did["left"], stop["left"],
		"this test is about the column under DID IT and STOP ASKING is no longer in it")

	cell := often["left"] - did["right"]
	require.Greater(t, stop["top"]-did["bottom"], cell,
		"STOP ASKING sits %vpx under DID IT, which is the %vpx the grid puts between any two cells",
		stop["top"]-did["bottom"], cell)
}

// Two lists of the same rows, one tab apart, inset their words by the same
// amount. The set-aside card is quieter than a card in the deck on purpose —
// less shadow, less weight — and it was also narrower, by four pixels nobody
// chose.
func TestBrowserTheSetAsideInsetsItsWordsLikeEveryOtherList(t *testing.T) {
	f := aPile()
	f.items = append(f.items, note(9, "ask about the bike rack", squirrel.ItemKept))
	f.aside = []squirrel.HeldItem{{
		ID: 10, Text: "chase the landlord about the window",
		State: squirrel.ItemWaiting, Because: "waiting on him", Kind: squirrel.ItemTask,
	}}
	srv := screen(t, f)
	c := browserAt(t, srv, "/kept")

	shelf := style(t, c, ".rcard", "padding-left")

	c.navigate(t, srv.URL+"/held")
	setAside := style(t, c, ".aside", "padding-left")

	require.Equal(t, shelf, setAside,
		"the shelf insets its words by %s and the set-aside by %s", shelf, setAside)
}

// A group of hits and a group of set-aside rows are the same idea — a label
// over a run of cards saying what the run is — and they are one tab apart.
// Search's two were not styled at all, so the same job was done in 12.5px
// amber caps on one screen and 16px sentence case on the other.
func TestBrowserSearchGroupsItsHitsTheWayTheSetAsideGroupsIts(t *testing.T) {
	f := aPile()
	f.chores = []squirrel.Chore{{
		ID: 1, Name: "bins out", Every: 7 * 24 * time.Hour,
		EveryDays: 7, SinceDays: 6, Active: true, EverDone: true,
	}}
	f.aside = []squirrel.HeldItem{{
		ID: 10, Text: "chase the landlord about the window",
		State: squirrel.ItemWaiting, Because: "waiting on him", Kind: squirrel.ItemTask,
	}}
	srv := screen(t, f)
	c := browserAt(t, srv, "/held")

	for _, prop := range []string{"font-size", "letter-spacing", "text-transform", "color"} {
		c.navigate(t, srv.URL+"/held")
		aside := style(t, c, ".every .heldgroup", prop)

		c.navigate(t, srv.URL+"/pile?q=bins")
		hits := style(t, c, ".results .lead", prop)

		require.Equal(t, aside, hits,
			"the set-aside's group label has %s %s and search's has %s", prop, aside, hits)
	}
}
