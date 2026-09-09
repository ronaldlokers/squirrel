//go:build browser

package web

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBrowserTheLidHoldsThreeAndDoesNotGrow(t *testing.T) {
	srv := screen(t, everyScreen())
	c := browserAt(t, srv, "/")
	touching(t, c)

	for _, w := range []int{320, 375, 390, 430} {
		c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
			"width": w, "height": 844, "deviceScaleFactor": 1, "mobile": true,
		})
		c.navigate(t, srv.URL+"/")
		c.until(t, "the bar", `!!document.querySelector(".ops .rail")`)

		require.Equal(t, float64(3), c.eval(t,
			`return document.querySelectorAll(".ops .rail .chip").length`),
			"%dpx: the lid holds a number of icons that is not three", w)

		require.Greater(t, c.eval(t,
			`return Math.round(document.querySelector(".ops .rail .find").getBoundingClientRect().width)`),
			float64(120), "%dpx: the find field is too narrow to type in", w)

		require.Equal(t, true, c.eval(t, `
			const rail = document.querySelector(".ops .rail");
			return rail.scrollWidth <= rail.clientWidth + 1;
		`), "%dpx: the lid holds more than it fits", w)
	}
}
