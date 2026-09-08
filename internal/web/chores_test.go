package web

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func chore(id int64, name string, everyDays, sinceDays int) squirrel.Chore {
	return squirrel.Chore{
		ID: id, PersonID: 1, Name: name, Active: true, EverDone: true,
		Every:     time.Duration(everyDays) * 24 * time.Hour,
		EveryDays: everyDays,
		SinceDays: sinceDays,
	}
}

// The chores are three racks now rather than one bay, so what these hold the
// screen to is the whole of the board and not a section of it. What is being
// guarded has not moved: the rhythm in words, no counting, no arithmetic, and
// nothing about how long a thing has been waiting.
func theChores(t *testing.T, f *fakeStore) string {
	t.Helper()
	if f.checkin == nil {
		f.checkin = &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: now()}
	}
	return routed(t, f).call(t, "GET", "/", nil).Body.String()
}

func TestChoresListsWhatComesBack(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		chore(1, "bins out", 14, 3),
		chore(2, "water the plants", 7, 1),
	}}
	body := theChores(t, f)

	require.Contains(t, body, "bins out")
	require.Contains(t, body, "water the plants")
	require.Contains(t, body, "every 2 weeks", "the rhythm as a person says it, not arithmetic")
}

func TestChoresNeverCounts(t *testing.T) {
	chores := []squirrel.Chore{}
	for i := int64(1); i <= 7; i++ {
		chores = append(chores, chore(i, "chore "+string(rune('a'+i)), 7, 30))
	}
	body := theChores(t, &fakeStore{chores: chores})

	lower := strings.ToLower(body)
	for _, forbidden := range []string{"7 chores", "overdue", "behind", "streak", "of 7"} {
		require.NotContains(t, lower, forbidden)
	}
}

func TestTheEmptyChoreListDoesNotNag(t *testing.T) {
	body := strings.ToLower(theChores(t, &fakeStore{}))

	for _, rack := range []string{"daily", "weekly", "seldom"} {
		require.Contains(t, body, rack)
		require.NotContains(t, body, rack+` <span class="n">`, "a sign counted what is not there")
	}
	for _, forbidden := range []string{"should", "why not", "add one"} {
		require.NotContains(t, body, forbidden)
	}
}

func TestAChoreNeverDoneSaysOnlyItsRhythm(t *testing.T) {
	never := chore(1, "descale the shower head", 30, 400)
	never.EverDone = false
	body := theChores(t, &fakeStore{chores: []squirrel.Chore{never}})

	require.Contains(t, body, "every month")
	require.NotContains(t, strings.ToLower(body), "last done")
	require.NotContains(t, body, "a while back")
}

func TestChoresFailsVisiblyWhenTheDatabaseIsDown(t *testing.T) {
	f := &fakeStore{choresErr: errTest}
	body := theChores(t, f)

	require.Contains(t, body, "cannot reach the chores")
}

func TestTheChoresScreenNeverPrintsADayCount(t *testing.T) {
	body := strings.ToLower(theChores(t, &fakeStore{chores: []squirrel.Chore{
		chore(1, "bins out", 7, 3),
		chore(2, "the filter", 30, 45),
	}}))

	require.Contains(t, body, "every week")
	require.Contains(t, body, "every month")
	for _, arithmetic := range []string{"3 days ago", "45 days", "days ago", "last done"} {
		require.NotContains(t, body, arithmetic)
	}
}
