package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Every kind that has a body is a different word in the markup. A test that
// only checked one would pass with the other five collapsed back into one.
// Each kind is still told apart at a glance, and on the board that job belongs
// to the holder: the colour down a strip's left edge says which bay it is in.
// The four rooms that used to draw four kinds of card are those four bays since
// 1 September 2026.
// A ticket, a page tab and a chore's own class were three tests here, and they
// described cards the four object rooms drew. Those rooms are the board's bays
// since 2 September 2026 and the board tells the four kinds apart by the
// holder, which is what this asserts.
func TestTheKindsAreDistinguishable(t *testing.T) {
	f := &fakeStore{
		items: []squirrel.Item{
			note(1, "kaas", squirrel.ItemOpen),
			task(2, "book the MOT", squirrel.ItemOpen),
		},
		chores:   []squirrel.Chore{{ID: 3, Name: "the bins", EveryDays: 7, Active: true}},
		upcoming: []squirrel.Moment{{ID: 4, Label: "dentist", Starts: now().Add(3 * time.Hour)}},
	}
	m := mounted(t, f)

	require.Contains(t, m.call(t, "GET", "/?bay=notes", nil).Body.String(),
		`class="strip h-notes`, "h-notes draws no strip")

	board := m.call(t, "GET", "/", nil).Body.String()
	require.Contains(t, board, `class="strip h-weekly`, "a chore draws in no rack's colour")
	require.Contains(t, board, `class="strip h-once`, "a thing you do one time draws in no rack's colour")
	require.NotContains(t, board, "h-tasks",
		"a thing you do one time still wears its own colour, so the merge only went halfway")
}
