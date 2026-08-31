//go:build browser

// Things that are only wrong on the screen.
//
// Both of these looked right in the source — a width that was stated, a role
// that was documented — and what the browser drew was a staggered column and a
// label in body text.
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

// The readings are a grid, and a grid whose rows start in different places is
// not one.
//
// The defect this pins outlived the shape it was written for. `.aday` was a
// flex row, so each day's name sized itself and pushed its own face: "thursday
// 20 august" started its reading 47px further right than "today" did. The week
// labels can do exactly the same thing to the days beside them — "this week"
// is wider than "21 jul" — which is why `.weekrow .wl` states a width and does
// not flex.
func TestBrowserEveryWeekOfReadingsStartsInTheSameColumn(t *testing.T) {
	srv := screen(t, aFortnight())
	c := browserAt(t, srv, "/")
	c.navigate(t, srv.URL+"/")
	// A turn rather than a page since 31 August 2026, so it is asked for the
	// way a person asks for it.
	c.until(t, "the settings panel", `!!document.querySelector('form[action="/me/moods"]')`)
	c.eval(t, `const f = document.querySelector('form[action="/me/moods"]');
		f.requestSubmit(f.querySelector("button")); return 1`)
	c.until(t, "the readings", `document.querySelectorAll(".weekrow").length === 6`)

	starts := numbers(t, c, "return ("+lefts+`)(".weekrow .dots")`)
	require.Len(t, starts, 6, "the screen did not draw six weeks")

	for i, at := range starts {
		require.Equal(t, starts[0], at,
			"week %d starts at %vpx and the first starts at %vpx: the week labels are setting the column",
			i+1, at, starts[0])
	}
}

// `.lead` is the small line that says what the thing under it is. It was six
// scoped rules and no definition, so every site none of them reached rendered
// at the document's own size — larger and lighter than the thing it labels.
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

	// There is one screen, and its meta labels are the picker's — which arrives
	// by pressing HOW OFTEN on a chore.
	openChores(t, c, srv)
	c.eval(t, `document.querySelector('article.chore form[action="/chores/often"] button').click()`)
	c.until(t, "the question", `!!document.querySelector(".pick")`)

	found := c.eval(t, `
		const body = parseFloat(getComputedStyle(document.body).fontSize);
		return Array.from(document.querySelectorAll(".lead")).map(el => ({
			text: el.textContent.trim(),
			size: parseFloat(getComputedStyle(el).fontSize),
			body: body,
		}));`)

	labels, ok := found.([]any)
	require.True(t, ok, "the page did not answer with a list of labels")
	require.NotEmpty(t, labels, "nothing on the screen drew a meta label at all")

	for _, l := range labels {
		label := l.(map[string]any)
		require.Less(t, label["size"], label["body"],
			"%s is %vpx against body text at %vpx: it is not wearing the meta role",
			label["text"], label["size"], label["body"])
	}
}
