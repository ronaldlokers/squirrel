package web

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Which choice is in effect is a state, not a colour.
//
// Options with identical roles separated only by a purple fill are unreachable
// by a screen reader and painted away in forced colours. The chips were four
// buttons carrying aria-pressed; the picker is two radio groups, where checked
// is the state and the browser says it without being asked — which is why this
// asserts on the input rather than on an ARIA attribute that would be wrong on
// a radio.
func TestTheCurrentIntervalSaysSoAndNotOnlyInPurple(t *testing.T) {
	// A fresh reading, so Buddy does not ask how you are on the render and
	// become the live edge himself — which takes the controls off the picker.
	f := &fakeStore{
		chores: []squirrel.Chore{{ID: 1, Name: "bins out", Every: 7 * 24 * time.Hour,
			EveryDays: 7, SinceDays: 6, Active: true, EverDone: true}},
		checkin: &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: now()},
	}

	m := routed(t, f)
	m.call(t, "POST", "/open", strings.NewReader("where=chores"))
	m.call(t, "POST", "/chores/often", strings.NewReader("id=1"))
	f.turns, f.appended = append(f.turns, f.appended...), nil
	body := m.call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `<input type="radio" name="count" value="1" checked>`,
		"the number in effect does not say so")
	require.Contains(t, body, `<input type="radio" name="unit" value="weeks" checked>`,
		"the unit in effect does not say so")
	require.Contains(t, body, `<input type="radio" name="unit" value="days">`,
		"the ones not in effect do not say so either")
}
