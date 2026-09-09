//go:build browser

package web

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBrowserFindingSomethingOnAPhoneShowsWhatMatched(t *testing.T) {
	srv := screen(t, everyScreen())
	c := browserAt(t, srv, "/")
	touching(t, c)

	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 1, "mobile": true,
	})
	c.navigate(t, srv.URL+"/?find=the")
	c.until(t, "the board", `!!document.querySelector(".racks")`)

	require.NotEqual(t, "none", c.eval(t,
		`return getComputedStyle(document.querySelector(".racks.found")).display`),
		"the phone was given a blank screen where the matches were")

	shown := c.eval(t, `return [...document.querySelectorAll(".racks.found .strip:not(.back)")]
		.filter(s => s.getBoundingClientRect().height > 0).length`)
	require.Greater(t, shown.(float64), float64(0), "nothing that matched was drawn")

	require.Greater(t, c.eval(t,
		`return Math.round(document.querySelector(".racks.found .strip.back").getBoundingClientRect().height)`),
		float64(0), "there is no way back to the board")

	require.Equal(t, false, c.eval(t,
		`const d = document.documentElement; return d.scrollWidth > d.clientWidth`),
		"the matches push the phone sideways")

	require.Equal(t, true, c.eval(t, `
		const d = document.querySelector(".deck");
		d.scrollTop = 99999;
		const last = [...document.querySelectorAll(".racks.found .strip")].pop();
		return last.getBoundingClientRect().bottom <= window.innerHeight;
	`), "the last thing that matched cannot be reached")
}
