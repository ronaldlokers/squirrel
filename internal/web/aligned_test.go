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
	c := browserAt(t, srv, "/")

	// The thread is first and is not navigated to: its meta labels are the
	// picker's, and the picker arrives by pressing HOW OFTEN.
	for _, path := range []string{"", "/buddy"} {
		if path == "" {
			openChores(t, c, srv)
			c.eval(t, `document.querySelector('article.chore form[action="/chores/often"] button').click()`)
			c.until(t, "the question", `!!document.querySelector(".pick")`)
		} else {
			c.navigate(t, srv.URL+path)
		}

		found := c.eval(t, `
			const body = parseFloat(getComputedStyle(document.body).fontSize);
			return Array.from(document.querySelectorAll(".lead")).map(el => ({
				text: el.textContent.trim(),
				size: parseFloat(getComputedStyle(el).fontSize),
				body: body,
			}));`)

		labels, ok := found.([]any)
		require.True(t, ok, "%s did not answer with a list of labels", path)
		require.NotEmpty(t, labels, "%q drew no meta label at all", path)

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

	// The thread is the page's own left margin now that the card is gone, and
	// the panel must not sit outside it.
	card := box(t, c, "#thread")
	require.GreaterOrEqual(t, find["left"], card["left"],
		"the field's panel starts at %vpx, outside the %vpx margin the page keeps",
		find["left"], card["left"])
}

// Two lists of the same rows, one tab apart, inset their words by the same
// amount. The set-aside card is quieter than a card in the deck on purpose —
// less shadow, less weight — and it was also narrower, by four pixels nobody
// chose.
// TestBrowserTheSetAsideInsetsItsWordsLikeEveryOtherList was retired on
// 25 August 2026. It measured `.rcard` on the shelf against `.aside` on the
// set-aside page, and both pages went the same day: the two are cards in the
// conversation now and draw from `.turncard`, which is one rule rather than
// two that could drift apart. The appearance snapshot records it.

// TestBrowserTheEndingIsNotTheNextThingUnderTheDoing was retired on 24 August
// 2026. It measured STOP ASKING against the disclosure that used to sit beside
// it in the chore card; the disclosure is a turn now, and the three buttons sit
// in one row whose spacing the appearance snapshot records.

// This compared two group labels — search's and the set-aside's — because they
// were the same role in two places and had drifted apart. The conversation has
// no group labels: results are cards, and a card's line is the card's own Meta
// at 11.5px rather than a heading over a group at 12.5px.
//
// Retired rather than repointed. Forcing a card's line to match a group's
// heading would be making two different roles the same size to keep a test,
// which is the opposite of what it was for.
