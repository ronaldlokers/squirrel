package web

import (
	"strings"
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
func TestTheKindsAreDistinguishable(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range []struct {
		where string
		f     *fakeStore
	}{
		{"at", &fakeStore{upcoming: []squirrel.Moment{{ID: 1, Label: "dentist", Starts: now().Add(3 * time.Hour)}}}},
		{"tasks", &fakeStore{items: []squirrel.Item{task(1, "book the MOT", squirrel.ItemOpen)}}},
		{"chores", &fakeStore{chores: []squirrel.Chore{{ID: 1, Name: "the bins", EveryDays: 7, Active: true}}}},
		{"held", &fakeStore{aside: []squirrel.HeldItem{{ID: 5, Text: "the referral", State: squirrel.ItemWaiting}}}},
	} {
		drew := drewIn(t, c.f, c.where)
		require.NotEmpty(t, drew)
		shown := string(drew[len(drew)-1].Shown)
		for _, kind := range []string{"at", "task", "chore", "held"} {
			if strings.Contains(shown, `"kind":"`+kind+`"`) {
				require.False(t, seen[kind], "%s claims a kind another already used", c.where)
				seen[kind] = true
			}
		}
	}
	require.Len(t, seen, 4, "four kinds went in and %d came out distinguishable", len(seen))
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
