//go:build browser

package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestBrowserTheLiveEdgeWearsItsLetters(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "the bike rack", squirrel.ItemOpen)}}
	srv := screen(t, f)
	c := browserAt(t, srv, "/r/notes")
	c.navigate(t, srv.URL+"/r/notes")
	c.until(t, "the card", `!!document.querySelector('input[name="act"]')`)

	require.Equal(t, []any{"D", "K", "X", "T"}, c.eval(t, `
		return ["done", "keep", "drop", "task"].map(act =>
			document.querySelector('input[name="act"][value="' + act + '"]').form
				.querySelector("button .key")?.textContent ?? null)`),
		"the four letters are not on the four buttons they press")

	require.Equal(t, true, c.eval(t, `
		const last = document.querySelector("#thread .turn:last-child");
		return [...document.querySelectorAll(".key")].every(k =>
			k.closest(".turn") === last && k.getAttribute("aria-hidden") === "true")`),
		"a keycap is outside the live edge, or is not hidden from a screen reader")
}

func TestBrowserTheIntervalQuestionWearsItsLetters(t *testing.T) {
	f := &fakeStore{
		chores: []squirrel.Chore{{ID: 1, Name: "bins out", Every: 7 * 24 * time.Hour,
			EveryDays: 7, SinceDays: 6, Active: true, EverDone: true}},
		checkin: &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: time.Now()},
	}
	srv := screen(t, f)
	c := atChores(t, srv)
	c.eval(t, `document.querySelector('article.chore form[action="/chores/often"] button').click()`)
	c.until(t, "the question", `!!document.querySelector(".pick .key")`)

	require.Equal(t, true, c.eval(t, `
		return [...document.querySelectorAll('.pick input[name="count"]')].every(r =>
			r.closest(".chip").querySelector(".key")?.textContent === r.value)`),
		"a count chip does not wear the digit that chooses it")

	require.Equal(t, []any{"D", "W", "M"}, c.eval(t, `
		return [...document.querySelectorAll('.pick input[name="unit"]')].map(r =>
			r.closest(".chip").querySelector(".key")?.textContent ?? null)`),
		"a unit chip does not wear its own first letter")

	require.Equal(t, "↵", c.eval(t,
		`return document.querySelector(".pick .make .key").textContent`),
		"the answering button does not wear Enter")

	require.Equal(t, float64(0), c.eval(t, `
		return document.querySelectorAll('.pick input[name="day"]').length === 0 ? 0 :
			[...document.querySelectorAll('.pick input[name="day"]')]
				.filter(r => r.closest(".chip").querySelector(".key")).length`),
		"the day row wears letters, and no key answers it")
}

func TestBrowserAPhoneHasNoKeycaps(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "the bike rack", squirrel.ItemOpen)}}
	srv := screen(t, f)
	c := browserAt(t, srv, "/r/notes")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 0, "mobile": true,
	})
	c.navigate(t, srv.URL+"/r/notes")
	c.until(t, "the card", `!!document.querySelector('input[name="act"]')`)

	require.Equal(t, true, c.eval(t, `
		const caps = [...document.querySelectorAll(".key")];
		return caps.length > 0 && caps.every(k => getComputedStyle(k).display === "none")`),
		"a thumb cannot press D, and the letters are on screen anyway")
}
