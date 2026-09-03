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
	c := browserAt(t, srv, "/me")
	// Pressed where it lives, which is the settings page since 3 September
	// 2026, and read where it answers, which is his room. That is a full
	// navigation, and it is only testable because the fake store reads back
	// what it was told — it did not until this test asked it to.
	c.navigate(t, srv.URL+"/me")
	c.until(t, "the settings", `!!document.querySelector('form[action="/me/moods"]')`)
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

// The meta labels this measured arrived by pressing HOW OFTEN on a chore in a
// room, and that question is four rhythm stamps beside a field now. What it
// protected — that a label is never as large as the text it labels — is the
// board's type ramp, where a mark is 11px against a strip's 14.5 and the
// appearance snapshot records both.
