//go:build browser

package web

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func atWidth(t *testing.T, c *cdp, srv string, w int) {
	t.Helper()
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": w, "height": 900, "deviceScaleFactor": 1, "mobile": w <= 620,
	})
	c.navigate(t, srv+"/")
	c.until(t, "the board", `!!document.querySelector(".racks")`)
}

func columnsOf(t *testing.T, c *cdp) int {
	t.Helper()
	got := c.eval(t, `return getComputedStyle(document.querySelector(".racks"))
		.gridTemplateColumns.split(" ").length`)
	n, ok := got.(float64)
	require.True(t, ok, "the grid answered %#v", got)
	return int(n)
}

// A tablet is a desk you hold, and it was getting the desk's layout at half the
// desk's width: four racks and a sidebar need about 1240px before a rack is
// wide enough to read a chore beside its mark, and an iPad has 1180 in
// landscape and 834 in portrait.
func TestBrowserATabletGetsTwoRacksAcrossAndTheSidebarBelow(t *testing.T) {
	srv := screen(t, aDay())
	c := browserAt(t, srv, "/")

	for _, w := range []int{834, 1024, 1180} {
		atWidth(t, c, srv.URL, w)
		require.Equal(t, 2, columnsOf(t, c), "%dpx draws a different number of racks", w)
		require.Equal(t, "1 / -1", c.eval(t,
			`const d = document.querySelector(".racks > .dial");
			 return getComputedStyle(d).gridColumn`),
			"%dpx leaves the sidebar in a column of its own", w)
	}

	atWidth(t, c, srv.URL, 1440)
	require.Equal(t, 5, columnsOf(t, c), "the desk lost a rack or its sidebar")
}

// Nothing is clipped at any width the board is drawn at. A strip's mark used to
// take a max-content column beside the words, so EVERY 365 DAYS starved them
// and `overflow: hidden` cut the word in half rather than wrapping it.
func TestBrowserNoWordIsClippedAtAnyWidth(t *testing.T) {
	f := aDay()
	// One word longer than any rack is wide. Wrapping cannot break it at a
	// space, so without `overflow-wrap` it runs under the mark and out of the
	// strip, and `overflow: hidden` cuts it rather than showing it.
	f.chores = append(f.chores, squirrel.Chore{
		ID: 9, Name: "sortthelaundryandputitawayproperly", Active: true, EverDone: true,
		Every: 7 * 24 * time.Hour, EveryDays: 7, SinceDays: 7,
	})
	srv := screen(t, f)
	c := browserAt(t, srv, "/")

	// 1240 and 1300 are the desk with its racks at their narrowest — four of
	// them plus a sidebar in barely enough room, which is the only width
	// where the container rule is what stands between a mark and a cut word.
	for _, w := range []int{390, 700, 834, 1024, 1180, 1240, 1300, 1440, 1920} {
		atWidth(t, c, srv.URL, w)
		clipped := c.eval(t, `return JSON.stringify([...document.querySelectorAll(
			".strip .what, .strip .mark, .hang .says, .fold .says, .baysign")]
			.filter(el => el.offsetParent !== null &&
				(el.scrollWidth > el.clientWidth + 1 || el.scrollHeight > el.clientHeight + 1))
			.map(el => el.className + ": " + el.textContent.trim().slice(0, 30)))`)
		require.Equal(t, "[]", fmt.Sprint(clipped), "at %dpx something is cut off", w)

		// And the words are not squeezed to one letter a line. A cut word is
		// what the clipping check catches; this catches the other half, where
		// the mark takes its max-content column and leaves the words 40px.
		narrow := c.eval(t, `return JSON.stringify([...document.querySelectorAll(".strip .what")]
			.filter(el => el.offsetParent !== null && el.getBoundingClientRect().width < 90)
			.map(el => Math.round(el.getBoundingClientRect().width) + "px: " +
				el.textContent.trim().slice(0, 24)))`)
		require.Equal(t, "[]", fmt.Sprint(narrow),
			"at %dpx the words are squeezed into a gutter", w)

		// The one field that is always on screen has to stay wide enough to
		// type in. It is the first thing the bar squeezes when something is
		// added to it.
		require.Greater(t, c.eval(t,
			`return document.querySelector(".ops .find input").getBoundingClientRect().width`).(float64),
			float64(120), "at %dpx the find field is too narrow to type in", w)
	}
}
