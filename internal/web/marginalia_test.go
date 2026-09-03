//go:build browser

package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestBrowserTheLineSitsInTheMarginOfItsOwnStrip(t *testing.T) {
	f := &fakeStore{
		items: []squirrel.Item{
			{ID: 1, RawText: "boiler service code is 4471", State: squirrel.ItemOpen, Kind: squirrel.ItemNote, ReceivedAt: time.Now()},
		},
		noticed: []squirrel.Noticed{
			{ID: 9, Kind: "note", RefID: 1, Words: "The code you need for this is in the other note."},
		},
	}
	c := browserAt(t, screen(t, f), "/board")
	c.until(t, "the line", `!!document.querySelector('.strip .seen')`)

	num := func(expr string) float64 {
		return c.eval(t, `const s = document.querySelector('.strip:has(.seen)'),
				w = s.querySelector('.what'), n = s.querySelector('.seen');
			const sb = s.getBoundingClientRect(), wb = w.getBoundingClientRect(),
				nb = n.getBoundingClientRect();
			const textOf = el => { const r = document.createRange();
				r.selectNodeContents(el); return r.getBoundingClientRect(); };
			const wt = textOf(w), nt = textOf(n);
			return `+expr).(float64)
	}
	below := num(`nb.top - wb.bottom`)
	inside := num(`sb.bottom - nb.bottom`)
	indent := num(`nt.left - wt.left`)
	smaller := num(`parseFloat(getComputedStyle(w).fontSize) - parseFloat(getComputedStyle(n).fontSize)`)
	past := num(`nb.right - sb.right`)

	require.Positive(t, below, "the line is not under the words it is about")
	require.Positive(t, inside, "the line hangs out of the bottom of its strip")
	require.Positive(t, indent, "the line is not set in from the strip's own words")
	require.Positive(t, smaller, "the line is set as large as the thing it is about")
	require.Negative(t, past, "the line runs past the right edge of the strip")
}

func TestBrowserRefusingALineTakesItOffTheStrip(t *testing.T) {
	f := &fakeStore{
		items: []squirrel.Item{
			{ID: 1, RawText: "boiler service code is 4471", State: squirrel.ItemOpen, Kind: squirrel.ItemNote, ReceivedAt: time.Now()},
		},
		noticed: []squirrel.Noticed{
			{ID: 9, Kind: "note", RefID: 1, Words: "The code you need for this is in the other note."},
		},
	}
	c := browserAt(t, screen(t, f), "/board")
	c.until(t, "the line", `!!document.querySelector('.strip .seen')`)

	f.noticed = nil
	c.eval(t, `const f = document.querySelector('.strip .seen form'); f.requestSubmit(); return 1`)

	c.until(t, "the strip without its line", `!document.querySelector('.strip .seen')`)
	require.Equal(t, []int64{9}, f.unuseful, "nothing was refused")
	require.Contains(t, c.eval(t, `return document.body.innerText`), "boiler service code is 4471",
		"refusing the line took the strip with it")
}
