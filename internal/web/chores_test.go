package web

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func chore(id int64, name string, everyDays, sinceDays int) squirrel.Chore {
	return squirrel.Chore{
		ID: id, PersonID: 1, Name: name, Active: true,
		Every:     time.Duration(everyDays) * 24 * time.Hour,
		EveryDays: everyDays,
		SinceDays: sinceDays,
	}
}

func TestChoresListsWhatComesBack(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		chore(1, "bins out", 14, 3),
		chore(2, "water the plants", 7, 1),
	}}
	body := mounted(t, f).call(t, "GET", "/pile/chores", nil).Body.String()

	require.Contains(t, body, "bins out")
	require.Contains(t, body, "water the plants")
	require.Contains(t, body, "every 14 days")
}

// The rule that governs the pile governs this too: a chore may say when it was
// last done, never how many are waiting or how far behind anything is.
func TestChoresNeverCounts(t *testing.T) {
	chores := []squirrel.Chore{}
	for i := int64(1); i <= 7; i++ {
		chores = append(chores, chore(i, "chore "+string(rune('a'+i)), 7, 30))
	}
	body := mounted(t, &fakeStore{chores: chores}).call(t, "GET", "/pile/chores", nil).Body.String()

	lower := strings.ToLower(body)
	for _, forbidden := range []string{"7 chores", "overdue", "behind", "streak", "% ", "of 7"} {
		require.NotContains(t, lower, forbidden)
	}
}

func TestTheEmptyChoreListDoesNotNag(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/pile/chores", nil).Body.String()

	require.Contains(t, body, "nothing comes back")
	for _, forbidden := range []string{"should", "why not", "add one"} {
		require.NotContains(t, strings.ToLower(body), forbidden)
	}
}

func TestMarkingAChoreDoneRecordsIt(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{chore(1, "bins out", 14, 20)}}
	w := post(t, mounted(t, f), "/pile/chores/act", url.Values{"id": {"1"}, "act": {"done"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, []int64{1}, f.completed)
}

func TestRetiringAChoreFromTheScreen(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{chore(1, "bins out", 14, 3)}}
	w := post(t, mounted(t, f), "/pile/chores/act", url.Values{"id": {"1"}, "act": {"retire"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, []int64{1}, f.retired)
}

// The same interval vocabulary the card uses, so the two surfaces cannot drift
// into meaning different things by "every 2 weeks".
func TestChangingHowOftenAChoreComesBack(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{chore(1, "bins out", 14, 3)}}
	w := post(t, mounted(t, f), "/pile/chores/act",
		url.Values{"id": {"1"}, "every": {"every week"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "bins out", f.reinterval.name)
	require.Equal(t, 7*24*time.Hour, f.reinterval.every)
}

// An id from a form is a number a stranger could have typed, so it is checked
// against what this person actually has rather than trusted.
func TestActingOnAChoreThatIsNotYours(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{chore(1, "bins out", 14, 3)}}
	w := post(t, mounted(t, f), "/pile/chores/act", url.Values{"id": {"99"}, "act": {"retire"}})

	require.Equal(t, 404, w.Code)
	require.Empty(t, f.retired)
}

func TestChoresRefusesAnUnknownAction(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{chore(1, "bins out", 14, 3)}}
	w := post(t, mounted(t, f), "/pile/chores/act", url.Values{"id": {"1"}, "act": {"delete"}})

	require.Equal(t, 400, w.Code)
	require.Empty(t, f.retired)
}

// The two screens have to be reachable from each other, or the chores are as
// invisible as they were before.
func TestThePileAndTheChoresLinkToEachOther(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	m := mounted(t, f)

	require.Contains(t, m.call(t, "GET", "/pile", nil).Body.String(), `href="/pile/chores"`)
	require.Contains(t, m.call(t, "GET", "/pile/chores", nil).Body.String(), `href="/pile"`)
}

func TestChoresFailsVisiblyWhenTheDatabaseIsDown(t *testing.T) {
	f := &fakeStore{err: errTest}
	w := mounted(t, f).call(t, "GET", "/pile/chores", nil)

	require.Equal(t, 503, w.Code)
	require.Contains(t, w.Body.String(), "cannot reach")
}
