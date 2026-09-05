//go:build browser

package web

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBrowserTheFindFieldShowsItIsFocused(t *testing.T) {
	srv := screen(t, aRackOfNotes())
	c := browserAt(t, srv, "/board")
	c.until(t, "the find field", `!!document.querySelector('.ops .find input')`)
	c.eval(t, `document.querySelector('.ops .find input').focus(); return 1`)
	require.Equal(t, "2px", c.eval(t, `return getComputedStyle(document.querySelector('.ops .find')).borderWidth`),
		"the board's find field carries no border to colour when it is focused")
	require.Equal(t, "rgb(255, 138, 43)", c.eval(t, `return getComputedStyle(document.querySelector('.ops .find')).borderColor`),
		"the board's find field does not take the focus colour")

	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 2, "mobile": true,
	})
	c.navigate(t, srv.URL+"/board")
	c.until(t, "the find field", `!!document.querySelector('.ops .find input')`)
	c.eval(t, `document.querySelector('.ops .find input').focus(); return 1`)
	require.Equal(t, "rgb(255, 138, 43)", c.eval(t, `return getComputedStyle(document.querySelector('.ops .find')).borderColor`),
		"the phone's find field keeps its resting colour while focused")
}

func TestBrowserTheRoomsFindFieldShowsItIsFocused(t *testing.T) {
	srv := screen(t, aRackOfNotes())
	c := browserAt(t, srv, "/r/everything")
	c.until(t, "the find field", `!!document.querySelector('.lid .find input')`)
	c.eval(t, `document.querySelector('.lid .find input').focus(); return 1`)
	require.Equal(t, "rgb(255, 138, 43)", c.eval(t, `return getComputedStyle(document.querySelector('.lid .find')).borderColor`),
		"the room's find field does not take the focus colour")
}
