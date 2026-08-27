package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

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

	// One screen, since every place became a message.
	body := m.call(t, "GET", "/", nil).Body.String()
	require.Contains(t, body, `href="/enough"`,
		"the conversation is a screen you can spend an evening on with no way to stop")
	require.Contains(t, body, said, "it does not say the way out in today's words")
}
