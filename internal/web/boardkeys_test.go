//go:build browser

package web

import (
	"testing"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/stretchr/testify/require"
)

// Letters act and arrows move, on the strip you are focused in. The board drew
// a key on every stamp from the day it was mounted and read none of them; the
// rooms it replaced had this, and DESIGN.md's keyboard-first path is not a
// promise a screen may make and not keep.
func TestBrowserTheBoardsKeysFollowFocus(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 1, PersonID: 1, Name: "bins out", Active: true, EverDone: true, Every: 14 * 24 * time.Hour, EveryDays: 14, SinceDays: 3},
		{ID: 2, PersonID: 1, Name: "water the ferns", Active: true, Every: 7 * 24 * time.Hour, EveryDays: 7},
	}}
	srv := screen(t, f)
	c := browserAt(t, srv, "/?bay=chores")
	c.navigate(t, srv.URL+"/?bay=chores")
	c.until(t, "the chores to arrive", `!!document.querySelector(".strip.h-chores")`)

	// Nothing focused: the first press says where you are rather than acting on
	// something you did not choose.
	c.key(t, "d")
	c.until(t, "the first chore to take focus",
		`document.activeElement.closest(".strip")?.querySelector(".words").textContent.trim().startsWith("bins out")`)

	// And the arrows move between strips rather than acting on one.
	c.key(t, "ArrowDown")
	c.until(t, "the second chore to take focus",
		`document.activeElement.closest(".strip")?.querySelector(".words").textContent.trim().startsWith("water the ferns")`)

	// The second press of the same letter acts on the strip it is in, which is
	// the whole of what a key is for.
	c.key(t, "d")
	c.until(t, "the strike", `!!document.querySelector(".strip.struck")`)
	require.Eventually(t, func() bool { return len(f.completed) == 1 },
		4*time.Second, 50*time.Millisecond, "the key did not act on the chore")
}

// A letter with nothing to act on does nothing at all, rather than pressing
// something that happens to be first.
func TestBrowserALetterNoBayAnswersDoesNothing(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 1, PersonID: 1, Name: "bins out", Active: true, Every: 7 * 24 * time.Hour, EveryDays: 7},
	}}
	srv := screen(t, f)
	c := browserAt(t, srv, "/?bay=chores")
	c.navigate(t, srv.URL+"/?bay=chores")
	c.until(t, "the chores to arrive", `!!document.querySelector(".strip.h-chores")`)

	c.key(t, "q")
	require.Equal(t, "BODY", c.eval(t, `return document.activeElement.tagName`),
		"a letter nothing answers to moved the focus")
}
