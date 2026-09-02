package web

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// pile.css is a static file behind a long cache and cannot know what day it
// is, so the two things that move without being read — the stamp's lean and
// where the room's light falls — arrive as custom properties on the body.
// Between the picker and the stylesheet there is nothing but this attribute,
// and it is the part with no other test: the numbers are held in
// internal/squirrel, and the CSS reads them or does not.
func TestTheDayReachesTheStylesheet(t *testing.T) {
	day := time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)
	was := now
	now = func() time.Time { return day }
	t.Cleanup(func() { now = was })

	body := mounted(t, &fakeStore{}).call(t, "GET", "/r/everything", nil).Body.String()

	require.Contains(t, body,
		fmt.Sprintf("--tilt: %ddeg", squirrel.Tilt(day)),
		"the stamp's angle never left the server")
	require.Contains(t, body,
		fmt.Sprintf("--light: %d%%", squirrel.Light(day)),
		"the light never left the server")
}

// The light is the field, which is behind all thirteen, and the properties are
// set once on the body for that reason. A screen that got the sentences but
// not these would be lit differently from the screen beside it.
func TestEveryScreenGetsTheSameDay(t *testing.T) {
	day := time.Date(2026, 9, 3, 20, 0, 0, 0, time.UTC)
	was := now
	now = func() time.Time { return day }
	t.Cleanup(func() { now = was })

	want := fmt.Sprintf("--light: %d%%", squirrel.Light(day))
	for _, path := range []string{"/", "/?bay=chores", "/r/everything"} {
		body := mounted(t, &fakeStore{}).call(t, "GET", path, nil).Body.String()
		require.Contains(t, body, want, "%s is lit from somewhere else", path)
	}
}
