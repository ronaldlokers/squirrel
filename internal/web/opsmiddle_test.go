//go:build browser

package web

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBrowserTheFindFieldIsTheMiddleOfTheBar(t *testing.T) {
	srv := screen(t, aPile())
	c := browserAt(t, srv, "/")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 1280, "height": 800, "deviceScaleFactor": 1, "mobile": false,
	})
	c.navigate(t, srv.URL+"/")
	c.until(t, "the bar", `document.querySelector(".ops .rail .find input") !== null`)

	bar := box(t, c, ".ops")
	find := box(t, c, ".ops .rail .find")
	clock := box(t, c, ".ops .clock")

	middle := (bar["left"] + bar["right"]) / 2
	require.Less(t, find["left"], middle,
		"the find field starts at %vpx and the bar's middle is %vpx: the middle of the bar is bare",
		find["left"], middle)
	require.Greater(t, find["right"], middle,
		"the find field ends at %vpx before the bar's middle at %vpx: the middle of the bar is bare",
		find["right"], middle)
	require.Less(t, clock["right"], middle,
		"the clock ends at %vpx, past the bar's middle at %vpx: the clock left the brand's side",
		clock["right"], middle)
}
