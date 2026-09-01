package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// An appointment is a ticket, not a card. It is the one thing in this product
// you can be late for, and it looked exactly like a note about a rattle.
func TestAnAppointmentIsATicket(t *testing.T) {
	f := &fakeStore{upcoming: []squirrel.Moment{
		{ID: 1, Label: "dentist", Starts: now().Add(3 * time.Hour), Travel: 20 * time.Minute},
	}}
	drew := drewIn(t, f, "at")
	require.NotEmpty(t, drew)
	require.Contains(t, string(drew[len(drew)-1].Shown), `"kind":"at"`)

	body := drawnAs(t, "at")
	require.Contains(t, body, "kat")
	require.Contains(t, body, `class="notch"`, "the ticket has no notch")
}

// A task wears the notebook's page tab, which is the device this system
// already owns for a row that was decided on.
func TestATaskWearsThePageTab(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{task(1, "book the MOT", squirrel.ItemOpen)}}
	drew := drewIn(t, f, "tasks")
	require.NotEmpty(t, drew)

	require.Contains(t, string(drew[len(drew)-1].Shown), `"kind":"task"`)
	require.Contains(t, drawnAs(t, "task"), `class="pagetab"`)
}

// Something set aside is recessed rather than raised. It is the one body that
// bends "cream card stock, never white", and it bends it on purpose.
func TestSomethingSetAsideIsNotStock(t *testing.T) {
	f := &fakeStore{aside: []squirrel.HeldItem{{
		ID: 5, Text: "the referral", State: squirrel.ItemWaiting, Kind: squirrel.ItemNote,
	}}}
	drew := drewIn(t, f, "held")
	require.NotEmpty(t, drew)

	require.Contains(t, string(drew[len(drew)-1].Shown), `"kind":"held"`)
	require.Contains(t, drawnAs(t, "held"), "kheld")
}

// A chore keeps the class it has always had, because pile.js binds the chore
// keys to it and the screen going did not take the thing on it.
func TestAChoreKeepsItsOwnClass(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 1, Name: "the bins", EveryDays: 7, SinceDays: 6, Active: true},
	}}
	drew := drewIn(t, f, "chores")
	require.NotEmpty(t, drew)
	require.Contains(t, string(drew[len(drew)-1].Shown), `"kind":"chore"`)
	require.Contains(t, drawnAs(t, "chore"), "turncard kchore chore")
}

// Every kind that has a body is a different word in the markup. A test that
// only checked one would pass with the other five collapsed back into one.
// Each kind is still told apart at a glance, and on the board that job belongs
// to the holder: the colour down a strip's left edge says which bay it is in.
// The four rooms that used to draw four kinds of card are those four bays since
// 1 September 2026.
func TestTheKindsAreDistinguishable(t *testing.T) {
	f := &fakeStore{
		items: []squirrel.Item{
			note(1, "kaas", squirrel.ItemOpen),
			task(2, "book the MOT", squirrel.ItemOpen),
		},
		chores:   []squirrel.Chore{{ID: 3, Name: "the bins", EveryDays: 7, Active: true}},
		upcoming: []squirrel.Moment{{ID: 4, Label: "dentist", Starts: now().Add(3 * time.Hour)}},
	}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	seen := map[string]bool{}
	for _, holder := range []string{"h-notes", "h-chores", "h-tasks", "h-agenda"} {
		require.Contains(t, body, `class="strip `+holder, "%s draws no strip", holder)
		require.False(t, seen[holder], "%s is drawn twice", holder)
		seen[holder] = true
	}
}

// drawnAs renders one turn carrying a card of that kind, so a test can look at
// the markup a kind produces rather than at the JSON that asks for it.
func drawnAs(t *testing.T, kind string) string {
	t.Helper()
	return thread(t, &fakeStore{turns: []squirrel.Turn{{
		ID: 1, Who: squirrel.SpeakerBuddy, Words: "here",
		Shown: []byte(`{"cards":[{"kind":"` + kind + `","title":"a thing","meta":"a line","take":"the letter"}]}`),
	}}})
}
