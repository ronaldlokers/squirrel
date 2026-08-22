package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Which chip is in effect is a state, not a colour.
//
// Four buttons, identical text, identical roles, separated only by a purple
// fill: unreachable by a screen reader and painted away in forced colours.
func TestTheCurrentIntervalSaysSoAndNotOnlyInPurple(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 1, Name: "bins out", Every: 7 * 24 * time.Hour, EveryDays: 7, SinceDays: 6, Active: true, EverDone: true},
	}}

	body := mounted(t, f).call(t, "GET", "/chores", nil).Body.String()

	require.Contains(t, body, `class="chip current" aria-pressed="true" name="every" value="every week"`,
		"the chip in effect does not say so")
	require.Contains(t, body, `class="chip" aria-pressed="false" name="every" value="every day"`,
		"the chips not in effect do not say so either")
}
