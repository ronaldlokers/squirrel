package web

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestARunningTimerShowsOnEveryScreen(t *testing.T) {
	ends := time.Now().Add(6*time.Minute + 12*time.Second)
	f := &fakeStore{
		timer:  &squirrel.Timer{Label: "the kitchen", Started: time.Now(), Ends: ends},
		items:  []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)},
		chores: []squirrel.Chore{{ID: 1, Name: "bins out", Active: true, Every: 14 * 24 * time.Hour, EveryDays: 14}},
	}
	m := mounted(t, f)

	for _, path := range []string{"/", "/me"} {
		body := m.call(t, "GET", path, nil).Body.String()
		require.Contains(t, body, "the kitchen", path)
	}
}

func TestTheStripSaysWhatIsLeft(t *testing.T) {
	f := &fakeStore{timer: &squirrel.Timer{
		Label: "the kitchen", Started: time.Now(), Ends: time.Now().Add(6*time.Minute + 12*time.Second),
	}}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, "06:1")
}

func TestStoppingATimerLeavesNothing(t *testing.T) {
	f := &fakeStore{timer: &squirrel.Timer{
		Label: "the kitchen", Started: time.Now(), Ends: time.Now().Add(time.Minute),
	}}

	w := post(t, mounted(t, f), "/timer", url.Values{"stop": {"1"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/", w.Header().Get("Location"))
	require.Nil(t, f.timer)
}

func TestNoTimerNoStrip(t *testing.T) {
	require.NotContains(t, mounted(t, &fakeStore{}).call(t, "GET", "/", nil).Body.String(),
		`class="ticking"`)
	require.NotContains(t, mounted(t, &fakeStore{}).call(t, "GET", "/me", nil).Body.String(),
		`class="running"`)
}
