//go:build browser

// The chores' own picker, and the keys it was not given.
package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func twoChores() *fakeStore {
	f := aPile()
	f.chores = []squirrel.Chore{
		{ID: 1, Name: "bins out", Every: 7 * 24 * time.Hour, EveryDays: 7, SinceDays: 6, Active: true, EverDone: true},
		{ID: 2, Name: "water the plants", Every: 4 * 24 * time.Hour, EveryDays: 4, SinceDays: 9, Active: true, EverDone: true},
	}
	return f
}

// Looking something up had a keyboard path and keeping a thought did not, so
// the first thing home asked of a keyboard user was to reach for the mouse —
// for the one act this product calls sacred.
func TestBrowserAKeyReachesTheSlotOnHome(t *testing.T) {
	f := aPile()
	srv := screen(t, f)
	c := browserAt(t, srv, "/r/everything")

	c.key(t, "t")

	require.Equal(t, "TEXTAREA", c.eval(t, `return document.activeElement.tagName`))
	require.Equal(t, true, c.eval(t, `return !!document.activeElement.closest(".slot")`),
		"t on home did not reach the slot")
}

// Triage left the conversation on 2 September 2026. The card at the live edge
// came from the four rooms' lists, and those are the board's racks now: a strip
// is answered where it lies, with the same letters, proved in
// TestBrowserTheBoardsKeysFollowFocus. His room is a conversation, and a
// conversation is not where the pile is worked.
