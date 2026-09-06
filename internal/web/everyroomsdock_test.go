//go:build browser

package web

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBrowserYourFaceIsRoundEverywhere(t *testing.T) {
	f := &fakeStore{whoName: "Ronald Lokers", whoFace: []byte("\x89PNG\r\n\x1a\n")}
	srv := screen(t, f)
	c := browserAt(t, srv, "/")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 1280, "height": 900, "deviceScaleFactor": 1, "mobile": false,
	})
	round := `return [...document.querySelectorAll(".youface img, .chip.face img")].every(i => {
			const r = getComputedStyle(i).borderRadius;
			const own = r === "999px" || parseFloat(r) >= i.getBoundingClientRect().width / 2;
			const p = getComputedStyle(i.parentElement).borderRadius;
			return own || p === "999px" || parseFloat(p) >= i.getBoundingClientRect().width / 2;
		})`

	c.navigate(t, srv.URL+"/")
	c.until(t, "the face", `document.querySelectorAll(".chip.face img").length === 1`)
	require.Equal(t, true, c.eval(t, round), "a picture of you is square on the board")

	c.navigate(t, srv.URL+"/me")
	c.until(t, "the faces", `document.querySelectorAll(".youface img, .chip.face img").length === 2`)
	require.Equal(t, true, c.eval(t, round), "a picture of you is square on the settings page")
}
