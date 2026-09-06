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
	c.key(t, "d")

	require.Eventually(t, func() bool { return f.states[1] != "" },
		4*time.Second, 50*time.Millisecond,
		"the opened strip draws D and nothing reads it")
}

func TestBrowserTheShelvesLetterActs(t *testing.T) {
	f := &fakeStore{
		items: []squirrel.Item{note(1, "meter reading 48213", squirrel.ItemWaiting)},
		aside: []squirrel.HeldItem{
			{ID: 1, Text: "meter reading 48213", State: squirrel.ItemWaiting},
		},
	}
	srv := screen(t, f)
	c := browserAt(t, srv, "/?shelf=held")
	c.navigate(t, srv.URL+"/?shelf=held")
	c.until(t, "the shelf to arrive", `!!document.querySelector(".strip .stamp")`)

	c.eval(t, `document.querySelector(".strip .stamp").focus(); return true;`)
	c.key(t, "z")

	require.Eventually(t, func() bool { return f.states[1] == squirrel.ItemOpen },
		4*time.Second, 50*time.Millisecond,
		"the shelf draws Z and nothing reads it")
}
