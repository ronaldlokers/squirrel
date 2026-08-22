package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Setting a note aside is a disposition like the other four, and was the only
// one with no way back.
//
// DONE, KEEP, DROP and A TASK each leave a stamp, a held card and a focused
// PUT IT BACK, and without script a banner on the next page rebuilt from the
// redirect. The three why-not chips posted and redirected with nothing at all,
// so the card just went. It was recoverable from /held the whole time; it did
// not look it, and on a bad day a note that vanishes is a note you have lost.
func TestSettingANoteAsideOffersItBack(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "chase the landlord", squirrel.ItemOpen)}}
	m := mounted(t, f)

	to := m.call(t, "POST", "/held/act",
		strings.NewReader("aside=waiting&id=1&from=%2Fpile")).Header().Get("Location")

	body := m.call(t, "GET", to, nil).Body.String()

	require.Contains(t, body, "PUT IT BACK",
		"the note was set aside and nothing offered it back")
	require.Contains(t, body, "waiting on someone",
		"and nothing said what had happened to it")
}

// The way back is not the way the other four come back.
//
// `open` is a state a note can be moved to; a note that was set aside is
// picked back up, which is its own verb on its own route. The banner has to
// post to the one that exists.
func TestTheWayBackFromSetAsideIsPickingItUp(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "chase the landlord", squirrel.ItemOpen)}}
	m := mounted(t, f)

	to := m.call(t, "POST", "/held/act",
		strings.NewReader("aside=someday&id=1&from=%2Fpile")).Header().Get("Location")
	body := m.call(t, "GET", to, nil).Body.String()

	require.Contains(t, body, `action="/held/act"`,
		"the way back posts where a set-aside note cannot be moved from")
	require.Contains(t, body, `value="back"`)

	// And pressing it actually picks the note back up.
	q, err := url.Parse(to)
	require.NoError(t, err)
	m.call(t, "POST", "/held/act",
		strings.NewReader(url.Values{
			"id": {"1"}, "act": {"back"}, "from": {q.Query().Get("from")},
		}.Encode()))

	require.Equal(t, []int64{1}, f.unheld)
}

// The stamp on the card and the line on the next page are the same words for
// the same press. With script one appears, without it the other does, and a
// person who uses both must not meet two vocabularies for one action.
func TestTheCardAndThePageSayTheSameThingAboutSettingAside(t *testing.T) {
	js, err := staticFS.ReadFile("static/pile.js")
	require.NoError(t, err)

	for state, said := range map[string]string{
		"waiting": "waiting on someone",
		"blocked": "blocked on a thing",
		"someday": "someday",
	} {
		require.Equal(t, said, saidWords[state], "the page's word for %s", state)
		require.Contains(t, string(js), said, "the card's word for %s", state)
	}
}

// Stopping is offered wherever work happens, not only where triage does.
//
// /enough was linked from the deck's foot and nowhere else, so leaving was
// normal if you were triaging and unmentioned if you were marking tasks done
// or answering chores. An evening spent on the tasks is just as much a
// session, and Principle 3 says leaving one must never look like failure —
// which it does when the only screen with an exit is the one you were not on.
func TestEverySessionScreenOffersAWayToStop(t *testing.T) {
	f := &fakeStore{
		items: []squirrel.Item{
			note(1, "ring the vet", squirrel.ItemOpen),
			task(2, "send the form back", squirrel.ItemOpen),
			task(3, "collect the parcel", squirrel.ItemDone),
			note(4, "the bike rack", squirrel.ItemKept),
		},
		chores: []squirrel.Chore{{ID: 1, Name: "bins out", EveryDays: 7, SinceDays: 6, Active: true}},
		aside: []squirrel.HeldItem{{
			ID: 5, Text: "chase the landlord", State: squirrel.ItemWaiting, Kind: squirrel.ItemNote,
		}},
	}
	m := mounted(t, f)

	for _, screen := range []string{"/pile", "/tasks", "/tasks/done", "/chores", "/kept", "/held"} {
		body := m.call(t, "GET", screen, nil).Body.String()
		require.Contains(t, body, "stop whenever you like",
			"%s is a screen you can spend an evening on with no way to stop", screen)
		require.Contains(t, body, `href="/enough"`, "%s", screen)
	}
}
