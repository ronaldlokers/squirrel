package web

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The body double was retired on 9 September 2026. These pin its absence rather
// than its behaviour, because a feature that half-goes is worse than one that
// stays: a control still drawn against a store that no longer answers is a
// press that does nothing.
func TestNoScreenOffersATimer(t *testing.T) {
	f := aBoardStore()
	// A chore that wants you today, so the phone's head is drawn: it is the
	// one place a countdown used to hang, and a fixture without a head would
	// let it come back unnoticed.
	f.chores = append(f.chores, squirrel.Chore{
		ID: 8, Name: "water the plants", Active: true, EverDone: true,
		Every: 24 * time.Hour, EveryDays: 1, SinceDays: 1,
	})
	f.usually = map[int64]squirrel.Usually{8: {Part: squirrel.Morning}}
	m := mounted(t, f)

	for _, where := range []string{"/", "/notes", "/?open=3", "/?find=boiler", "/?stuck=1", "/?stuck=boring"} {
		body := m.call(t, "GET", where, nil).Body.String()
		for _, gone := range []string{"stopTimer", "running", "MIN<", "catch me if I lose track", "still on this?"} {
			require.NotContains(t, body, gone, "%s still offers %q", where, gone)
		}
	}
}

func TestThereIsNoTimerRoute(t *testing.T) {
	m := mounted(t, aBoardStore())

	require.NotContains(t, m.routes, "POST /timer", "the timer route is still mounted")

	for _, act := range []string{"start", "stop", "timer", "hush"} {
		w := m.call(t, "POST", "/board/now", strings.NewReader("act="+act+"&kind=chore&id=7&minutes=5&label=x"))
		require.Equal(t, 303, w.Code, "%s answered %d", act, w.Code)
	}
}

func TestTheLadderIsASentenceAndNothingToPress(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferTask, RefID: 3, Text: "vet about the booster"}
	m := mounted(t, f)

	body := m.call(t, "GET", "/?stuck=boring", nil).Body.String()

	require.Contains(t, body, "short go", "the ladder lost its sentence with its timer")
	require.NotContains(t, body, `value="timer"`, "the ladder still has something to press")
}
