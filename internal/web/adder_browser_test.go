//go:build browser

package web

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBrowserTheWriterAsksOnlyWhatTheKindNeeds(t *testing.T) {
	srv := screen(t, aBoardStore())
	c := browserAt(t, srv, "/?bay=daily&rhythm=defrost+the+freezer")
	c.until(t, "the writer", `!!document.querySelector(".addform")`)

	shown := func(sel string) float64 {
		return c.eval(t, fmt.Sprintf(
			`return document.querySelector(%q).getBoundingClientRect().height`, sel)).(float64)
	}

	require.Positive(t, shown(".rhythmasks"), "it comes back and nothing asks how often")
	require.Zero(t, shown(".whenasks"), "it comes back and the writer is asking for a date")

	c.eval(t, `document.querySelector('.kinds input[value="notes"]').click(); return 1`)
	require.Zero(t, shown(".rhythmasks"), "a thought is being asked how often it comes back")
	require.Zero(t, shown(".whenasks"), "a thought is being asked for a date")

	c.eval(t, `document.querySelector('.kinds input[value="agenda"]').click(); return 1`)
	require.Positive(t, shown(".whenasks"), "at a time and nothing asks when")
	require.Zero(t, shown(".rhythmasks"), "at a time and the writer is asking for a rhythm")
}

func TestBrowserEscapeShutsTheWriterAndKeepsNothing(t *testing.T) {
	srv := screen(t, aBoardStore())
	c := browserAt(t, srv, "/")
	c.until(t, "the board", `!!document.querySelector(".addbar .plus")`)

	c.eval(t, `document.querySelector(".addbar .words").value = "half a thought"; return 1`)
	c.eval(t, `document.querySelector(".addbar .plus").click(); return 1`)
	c.until(t, "the writer", `!!document.querySelector(".adder.open")`)
	require.Equal(t, "half a thought", c.eval(t, `return document.querySelector(".addform .words").value`),
		"the words did not travel with the press, so they have to be typed twice")

	c.key(t, "Escape")

	c.until(t, "the writer to shut", `!document.querySelector(".adder.open")`)
	require.NotContains(t, c.eval(t, `return location.search`), "half",
		"the words survive the way out, which is a draft being kept")
}
