//go:build browser

package web

import (
	"testing"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/stretchr/testify/require"
)

func TestBrowserTheOpenedStripsLettersAct(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "boiler service code is 4471", squirrel.ItemOpen)}}
	srv := screen(t, f)
	c := browserAt(t, srv, "/?open=1")
	c.navigate(t, srv.URL+"/?open=1")
	c.until(t, "the opened strip to arrive", `!!document.querySelector(".strip.opened")`)

	c.eval(t, `document.querySelector(".strip.opened .stamp").focus(); return true;`)
	c.key(t, "x")

	require.Eventually(t, func() bool { return f.states[1] != "" },
		4*time.Second, 50*time.Millisecond,
		"the opened strip draws D and nothing reads it")
}

// A result that already left the pile carries the way back, and Z is what
// presses it. The shelves were where this was proved until 9 September 2026,
// when the notes became a wall and the shelves went with the triage.
func TestBrowserTheWayBackLetterActs(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "meter reading 48213", squirrel.ItemDropped)}}
	srv := screen(t, f)
	c := browserAt(t, srv, "/?find=meter")
	c.navigate(t, srv.URL+"/?find=meter")
	c.until(t, "the result to arrive", `!!document.querySelector(".strip .stamp")`)

	c.eval(t, `document.querySelector(".strip .stamp").focus(); return true;`)
	c.key(t, "z")

	require.Eventually(t, func() bool { return f.states[1] == squirrel.ItemOpen },
		4*time.Second, 50*time.Millisecond,
		"the way back draws Z and nothing reads it")
}
