package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// withOffer is a pile with a fresh reading, because the board shows the
// check-in rather than an offer until there is one — see now_test.go.
func withOffer(o *squirrel.Offer) *fakeStore {
	return &fakeStore{
		checkin: &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: time.Now()},
		offer:   o,
	}
}

// The picker chooses, and the picker says why.
//
// A model was allowed to replace the picker's answer on both surfaces between
// 3 and 4 September 2026 — the board paid for it on the first draw of a newly
// picked thing, and the conversation paid whenever it was opened. Deciding is
// a tool loop of up to three round trips, and NOT TODAY invalidates the cached
// decision by design, so the press that means "not this one" waited seconds
// for the next card.
//
// What the product notices is written in the margin now, once a day, where
// nothing waits for it. No surface can spend a call to draw itself: Decide is
// gone from the Coach interface, so there is nothing left to ask.

func TestTheBoardShowsWhatThePickerChose(t *testing.T) {
	f := withOffer(&squirrel.Offer{
		Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet", Because: "you decided this one",
	})
	c := &fakeCoach{}

	body := mountedWith(t, f, c).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, "ring the vet")
	require.Contains(t, body, "you decided this one")
}

func TestTheConversationShowsWhatThePickerChose(t *testing.T) {
	f := withOffer(&squirrel.Offer{
		Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet", Because: "you decided this one",
	})
	c := &fakeCoach{}

	body := mountedWith(t, f, c).call(t, "GET", "/r/everything", nil).Body.String()

	require.Contains(t, body, "ring the vet")
}

// And the buttons act on it, which is the half that would silently rot: an
// offer whose words came from one row and whose id came from another is a
// press that answers something you were not shown.
func TestTheButtonsActOnWhatWasChosen(t *testing.T) {
	f := withOffer(&squirrel.Offer{
		Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet", Because: "you decided this one",
	})

	body := mountedWith(t, f, &fakeCoach{}).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `name="kind" value="task"`)
	require.Contains(t, body, `name="id" value="7"`)
}
