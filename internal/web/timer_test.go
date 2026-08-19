package web

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestStartingATimerFromAChore(t *testing.T) {
	f := &fakeStore{}

	w := post(t, mounted(t, f), "/timer",
		url.Values{"minutes": {"10"}, "label": {"the kitchen"}, "from": {"chores"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/chores", w.Header().Get("Location"))
	require.NotNil(t, f.timer)
	require.Equal(t, "the kitchen", f.timer.Label)
	require.Equal(t, 10*time.Minute, f.timer.Ends.Sub(f.timer.Started))
}

// Running, it rides the lid on every screen — you started it in order to go
// and do something, and wandering to the pile should not lose it.
func TestARunningTimerShowsOnEveryScreen(t *testing.T) {
	ends := time.Now().Add(6*time.Minute + 12*time.Second)
	f := &fakeStore{
		timer:  &squirrel.Timer{Label: "the kitchen", Started: time.Now(), Ends: ends},
		items:  []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)},
		chores: []squirrel.Chore{{ID: 1, Name: "bins out", Active: true, Every: 14 * 24 * time.Hour, EveryDays: 14}},
	}
	m := mounted(t, f)

	for _, url := range []string{"/", "/pile", "/chores", "/kept"} {
		body := m.call(t, "GET", url, nil).Body.String()
		require.Contains(t, body, "the kitchen", url)
		require.Contains(t, body, `class="running"`, url)
	}
}

// The one number in this product that counts down, and it counts down rather
// than up because it is a fact about a thing you chose to start.
func TestTheStripSaysWhatIsLeft(t *testing.T) {
	f := &fakeStore{timer: &squirrel.Timer{
		Label: "the kitchen", Started: time.Now(), Ends: time.Now().Add(6*time.Minute + 12*time.Second),
	}}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, "06:1")
}

// Stopping halfway is a normal ending, and it leaves nothing behind.
func TestStoppingATimerLeavesNothing(t *testing.T) {
	f := &fakeStore{timer: &squirrel.Timer{
		Label: "the kitchen", Started: time.Now(), Ends: time.Now().Add(time.Minute),
	}}

	w := post(t, mounted(t, f), "/timer", url.Values{"stop": {"1"}, "from": {"pile"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/pile", w.Header().Get("Location"), "back where you pressed it")
	require.Nil(t, f.timer)
}

// No timer, no strip. There is nothing to say about not having started one.
func TestNoTimerNoStrip(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/", nil).Body.String()

	require.NotContains(t, body, `class="running"`)
}

// Three hours is not a body double, it is an afternoon.
func TestATimerHasBounds(t *testing.T) {
	for _, mins := range []string{"0", "-5", "1000", "nope"} {
		f := &fakeStore{}
		post(t, mounted(t, f), "/timer", url.Values{"minutes": {mins}, "label": {"x"}})
		require.Nil(t, f.timer, mins)
	}
}
