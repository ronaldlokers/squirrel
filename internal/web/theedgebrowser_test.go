//go:build browser

package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Acting on the list changes the list, without leaving the room.
//
// The task you have just done was still on the screen asking, because the
// room's list was written into the conversation and a press answers below it.
// The tasks rather than the chores: a chore stays on its list when you do it —
// it is still a chore — where a task you have done leaves the open ones, which
// is a difference this can see.
func TestBrowserActingOnTheListChangesTheList(t *testing.T) {
	f := &fakeStore{
		checkin: &squirrel.Checkin{Mood: squirrel.MoodCalm, SaidAt: time.Now()},
		items: []squirrel.Item{
			task(1, "send the form back", squirrel.ItemOpen),
			task(2, "collect the parcel", squirrel.ItemOpen),
		},
	}
	srv := screen(t, f)
	c := browserAt(t, srv, "/r/tasks")
	c.navigate(t, srv.URL+"/r/tasks")
	c.until(t, "the tasks", `document.querySelectorAll("#edge article").length === 2`)

	c.eval(t, `const b = [...document.querySelectorAll("#edge form button")]
		.find(x => x.textContent.trim().toLowerCase().startsWith("did"));
		b.closest("form").requestSubmit(b); return 1`)
	c.eval(t, `return new Promise(r => setTimeout(() => r(1), 1500))`)

	require.Equal(t, "/r/tasks", c.eval(t, `return location.pathname`))
	require.Positive(t, c.eval(t, `return document.querySelectorAll("#thread .turn").length`),
		"the press said nothing in the conversation")
	require.Equal(t, float64(1), c.eval(t, `return document.querySelectorAll("#edge article").length`),
		"the task you just did is still on the list asking")
}

// One place carries the controls. With the list below the conversation, a live
// turn in the scrollback would be a second set of buttons for the same room.
func TestBrowserOnlyTheEdgeCarriesTheControls(t *testing.T) {
	f := &fakeStore{
		checkin: &squirrel.Checkin{Mood: squirrel.MoodCalm, SaidAt: time.Now()},
		chores: []squirrel.Chore{
			{ID: 1, Name: "bins out", Every: 7 * 24 * time.Hour, EveryDays: 7, SinceDays: 8, Active: true, EverDone: true},
		},
		turns: []squirrel.Turn{{
			ID: 1, Who: squirrel.SpeakerBuddy, Words: "1 comes back.",
			Shown: []byte(`{"cards":[{"kind":"chore","title":"an older drawing","acts":[{"label":"DID IT","action":"/chores/act"}]}]}`),
		}},
	}
	srv := screen(t, f)
	c := browserAt(t, srv, "/r/chores")
	c.navigate(t, srv.URL+"/r/chores")
	c.until(t, "the chores", `!!document.querySelector("#edge .chore")`)

	require.Equal(t, float64(0), c.eval(t, `return document.querySelectorAll("#thread form").length`),
		"the scrollback still carries controls")
}
