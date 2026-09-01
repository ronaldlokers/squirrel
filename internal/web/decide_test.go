package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// withOffer is a pile with a fresh reading, because home shows the check-in
// rather than an offer until there is one — see now_test.go.
func withOffer(o *squirrel.Offer) *fakeStore {
	return &fakeStore{
		checkin: &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: time.Now()},
		offer:   o,
	}
}

// The model chooses among what the picker found, and what it chooses is
// rendered by the same card with the same three buttons. Nothing downstream
// knows which produced it — that is the point.

func TestTheModelCanChooseSomethingElse(t *testing.T) {
	f := withOffer(&squirrel.Offer{
		Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet", Because: "you decided this one",
	})
	c := &fakeCoach{decision: &fakeDecision{
		kind: "chore", refID: 3, text: "put the bins out", because: "they go out tonight",
	}}

	body := mountedWith(t, f, c).call(t, "GET", "/r/everything", nil).Body.String()

	require.Contains(t, body, "put the bins out")
	require.Contains(t, body, "they go out tonight")
	require.NotContains(t, body, "ring the vet")
	require.Equal(t, []string{"task"}, c.picked, "the model was not shown the picker's answer")
}

// A coach that has nothing to say leaves the picker's answer exactly as it
// was. This is the shipping state most of the time and it has to be the quiet
// one.
func TestTheModelDecliningLeavesThePickerAlone(t *testing.T) {
	f := withOffer(&squirrel.Offer{
		Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet", Because: "you decided this one",
	})
	body := mountedWith(t, f, &fakeCoach{}).call(t, "GET", "/r/everything", nil).Body.String()

	require.Contains(t, body, "ring the vet")
	require.Contains(t, body, "you decided this one")
}

// A running timer is a thing you are already doing, and a fixed point is the
// one thing here the world imposed rather than the product suggested. A model
// second-guessing either would be overruling a rule with an opinion.
func TestTheModelIsNotAskedAboutATimerOrAFixedPoint(t *testing.T) {
	for _, kind := range []squirrel.OfferKind{squirrel.OfferTimer, squirrel.OfferMoment} {
		f := withOffer(&squirrel.Offer{Kind: kind, RefID: 1, Text: "the kitchen"})
		c := &fakeCoach{decision: &fakeDecision{
			kind: "task", refID: 7, text: "ring the vet", because: "anything",
		}}

		body := mountedWith(t, f, c).call(t, "GET", "/r/everything", nil).Body.String()

		require.Empty(t, c.picked, "the model was asked about %s", kind)
		require.Contains(t, body, "the kitchen")
		require.NotContains(t, body, "ring the vet")
	}
}

// Nothing to hand over is a normal state and the model is not invited to find
// something. It would be answering a different question, and this region is
// absent rather than empty when there is nothing.
func TestTheModelIsNotAskedWhenThePickerFoundNothing(t *testing.T) {
	c := &fakeCoach{decision: &fakeDecision{
		kind: "task", refID: 7, text: "ring the vet", because: "anything",
	}}
	body := mountedWith(t, withOffer(nil), c).call(t, "GET", "/r/everything", nil).Body.String()

	require.Empty(t, c.picked)
	require.NotContains(t, body, "ring the vet")
}

// An offer that cannot say why it is the offer is a demand, whichever of the
// two produced it.
func TestAChoiceWithNoClauseIsNotUsed(t *testing.T) {
	f := withOffer(&squirrel.Offer{
		Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet", Because: "you decided this one",
	})
	c := &fakeCoach{decision: &fakeDecision{kind: "chore", refID: 3, text: "put the bins out"}}

	body := mountedWith(t, f, c).call(t, "GET", "/r/everything", nil).Body.String()
	require.Contains(t, body, "ring the vet")
}

// The buttons act on what is on the card. A chosen offer that kept the
// picker's id would mark the wrong thing done, which is the one way this could
// do real damage.
func TestTheButtonsActOnWhatWasChosen(t *testing.T) {
	f := withOffer(&squirrel.Offer{
		Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet", Because: "you decided this one",
	})
	c := &fakeCoach{decision: &fakeDecision{
		kind: "chore", refID: 3, text: "put the bins out", because: "they go out tonight",
	}}

	body := mountedWith(t, f, c).call(t, "GET", "/r/everything", nil).Body.String()
	require.Contains(t, body, `name="kind" value="chore"`)
	require.Contains(t, body, `name="id" value="3"`)
}
