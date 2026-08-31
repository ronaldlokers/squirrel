//go:build browser

package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The time goes under the face, and the words stay beside it.
//
// It sat in the words' own column first, which pushed every run down by the
// height of an avatar; moved to the gutter on its own it read as a label on
// whatever came next. Both were only visible in a browser — the markup was
// correct each time.
func TestBrowserTheTimeDoesNotPushTheWordsOffTheFacesLine(t *testing.T) {
	now = func() time.Time { return at(t, "2026-08-31 14:00") }
	t.Cleanup(func() { now = time.Now })

	f := aPile()
	f.turns = []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerYou, Words: "the boiler code", SaidAt: at(t, "2026-08-31 09:12")},
		{ID: 2, Who: squirrel.SpeakerBuddy, Words: "Down it goes.", SaidAt: at(t, "2026-08-31 09:12")},
	}
	srv := screen(t, f)
	c := browserAt(t, srv, "/r/notes")
	c.navigate(t, srv.URL+"/r/notes")
	c.until(t, "the times", `document.querySelectorAll(".whensaid").length >= 2`)

	require.Equal(t, true, c.eval(t, `
		return [...document.querySelectorAll(".turn")].filter(t => t.querySelector(".whoat")).every(t => {
			const face = t.querySelector(".buddyface, .youface");
			const words = t.querySelector(".bub, .said");
			if (!face || !words) return true;
			const f = face.getBoundingClientRect(), w = words.getBoundingClientRect();
			// Overlapping bands rather than equal tops: a bubble is taller than
			// a face and sits on the same line without sharing an edge.
			return w.top < f.bottom && f.top < w.bottom;
		})`),
		"the words dropped below the face, so the time took a row of its own")

	require.Equal(t, true, c.eval(t, `
		return [...document.querySelectorAll(".turn")].filter(t => t.querySelector(".whensaid")).every(t => {
			const face = t.querySelector(".buddyface, .youface");
			const when = t.querySelector(".whensaid");
			if (!face) return true;
			const f = face.getBoundingClientRect(), s = when.getBoundingClientRect();
			// Under the face and within its column, which is what makes it read
			// as belonging to the run that starts here.
			return s.top >= f.bottom - 1 && Math.abs((s.left + s.width / 2) - (f.left + f.width / 2)) < 12;
		})`),
		"the time is not under the face it belongs to")
}

// The day is a seam across the conversation, not something either speaker said.
func TestBrowserTheDayIsCentredAcrossTheConversation(t *testing.T) {
	now = func() time.Time { return at(t, "2026-08-31 14:00") }
	t.Cleanup(func() { now = time.Now })

	f := aPile()
	f.turns = []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerYou, Words: "the boiler code", SaidAt: at(t, "2026-08-30 09:12")},
		{ID: 2, Who: squirrel.SpeakerYou, Words: "and today's", SaidAt: at(t, "2026-08-31 09:12")},
	}
	srv := screen(t, f)
	c := browserAt(t, srv, "/r/notes")
	c.navigate(t, srv.URL+"/r/notes")
	c.until(t, "the days", `document.querySelectorAll(".whenday").length >= 2`)

	// The text's box, never the element's: a full-width block is centred on the
	// thread whichever way its text is aligned, so measuring the element proves
	// nothing at all. This test passed with text-align: left before it measured
	// a Range.
	require.Equal(t, true, c.eval(t, `
		const thread = document.getElementById("thread").getBoundingClientRect();
		return [...document.querySelectorAll(".whenday")].every(d => {
			const r = document.createRange();
			r.selectNodeContents(d);
			const b = r.getBoundingClientRect();
			return Math.abs((b.left + b.width / 2) - (thread.left + thread.width / 2)) < 4;
		})`),
		"a day divider is not centred on the conversation")
}
