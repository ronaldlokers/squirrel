package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

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

	// The wording varies by the day now, so this asks for the offer rather
	// than for one phrasing of it: the door, and today's words on it.
	said := squirrel.Say(squirrel.SayingStop, time.Now())
	require.NotEmpty(t, said)

	// One screen, since every place became a message. The shelf and the
	// set-aside were the last two with a lid of their own.
	for _, screen := range []string{"/"} {
		body := m.call(t, "GET", screen, nil).Body.String()
		require.Contains(t, body, `href="/enough"`,
			"%s is a screen you can spend an evening on with no way to stop", screen)
		require.Contains(t, body, said, "%s does not say the way out in today's words", screen)
	}
}
