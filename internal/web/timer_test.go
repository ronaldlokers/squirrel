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
	require.Equal(t, "/r/everything", w.Header().Get("Location"))
	require.NotNil(t, f.timer)
	require.Equal(t, "the kitchen", f.timer.Label)
	require.Equal(t, 10*time.Minute, f.timer.Ends.Sub(f.timer.Started))
}

func TestARunningTimerShowsOnEveryScreen(t *testing.T) {
	ends := time.Now().Add(6*time.Minute + 12*time.Second)
	f := &fakeStore{
		timer:  &squirrel.Timer{Label: "the kitchen", Started: time.Now(), Ends: ends},
		items:  []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)},
		chores: []squirrel.Chore{{ID: 1, Name: "bins out", Active: true, Every: 14 * 24 * time.Hour, EveryDays: 14}},
	}
	m := mounted(t, f)

	for _, url := range []string{"/", "/r/everything"} {
		body := m.call(t, "GET", url, nil).Body.String()
		require.Contains(t, body, "the kitchen", url)
		require.Contains(t, body, "the kitchen", url)
	}
}

// The one number in this product that counts down, and it counts down rather
// than up because it is a fact about a thing you chose to start.
func TestTheStripSaysWhatIsLeft(t *testing.T) {
	f := &fakeStore{timer: &squirrel.Timer{
		Label: "the kitchen", Started: time.Now(), Ends: time.Now().Add(6*time.Minute + 12*time.Second),
	}}
	body := mounted(t, f).call(t, "GET", "/r/everything", nil).Body.String()

	require.Contains(t, body, "06:1")
}

// Stopping halfway is a normal ending, and it leaves nothing behind.
func TestStoppingATimerLeavesNothing(t *testing.T) {
	f := &fakeStore{timer: &squirrel.Timer{
		Label: "the kitchen", Started: time.Now(), Ends: time.Now().Add(time.Minute),
	}}

	// "from" names the screen you pressed it on, and almost nothing is a
	// screen any more: it answers with the conversation, which is where the
	// press was made. The few that are still screens keep their own way back.
	w := post(t, mounted(t, f), "/timer", url.Values{"stop": {"1"}, "from": {"moods"}})

	require.Equal(t, 303, w.Code)
	// The readings were the last screen that was not a conversation and they
	// are a turn now, so there is nowhere else a timer can have been stopped
	// from — every way back is the conversation.
	require.Equal(t, "/r/everything", w.Header().Get("Location"), "back where you pressed it")
	require.Nil(t, f.timer)
}

// No timer, no strip. There is nothing to say about not having started one.
func TestNoTimerNoStrip(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/r/everything", nil).Body.String()

	require.NotContains(t, body, `class="running"`)
}

func TestATimerHasBounds(t *testing.T) {
	for _, mins := range []string{"0", "-5", "1000", "nope"} {
		f := &fakeStore{}
		post(t, mounted(t, f), "/timer", url.Values{"minutes": {mins}, "label": {"x"}})
		require.Nil(t, f.timer, mins)
	}
}
