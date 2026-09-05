package web

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Steps live on the strip they are about, opened. Pressing "too big" on the
// pulled strip is where the breakdown starts; the sequence itself never shows
// up as a card, a page or a list — only as the one thing to do next, under the
// strip whose task it is.

func aBoardStoreWithATaskOffer() *fakeStore {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferTask, RefID: 3, Text: "vet about the booster"}
	return f
}

func breaksInto(c *fakeCoach, steps ...string) *fakeCoach {
	c.steps = steps
	return c
}

func TestPressingTooBigOnTheBoardOpensTheTaskItWasAbout(t *testing.T) {
	f := aBoardStoreWithATaskOffer()
	c := breaksInto(&fakeCoach{}, "find the vet phone number", "book the appointment")
	m := mountedWith(t, f, c)

	w := post(t, m, "/board/now", url.Values{"act": {"stuck"}, "why": {"big"}, "kind": {"task"}, "id": {"3"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/?open=3", w.Header().Get("Location"))
}

func TestTheOpenedTaskShowsOneStepAndNeverTheList(t *testing.T) {
	f := aBoardStoreWithATaskOffer()
	c := breaksInto(&fakeCoach{}, "find the vet phone number", "book the appointment")
	m := mountedWith(t, f, c)

	post(t, m, "/board/now", url.Values{"act": {"stuck"}, "why": {"big"}, "kind": {"task"}, "id": {"3"}})
	body := m.call(t, "GET", "/?open=3", nil).Body.String()

	require.Contains(t, body, "find the vet phone number")
	require.NotContains(t, body, "book the appointment")
}

// A model that broke nothing down took the fixed line down with it: the
// ladder's own sentence is the floor, and pressing too big must still land
// somewhere with it on when there is nothing to open a strip onto.
func TestTooBigWithNothingBrokenDownFallsBackToTheFixedLine(t *testing.T) {
	f := aBoardStoreWithATaskOffer()
	c := &fakeCoach{}
	m := mountedWith(t, f, c)

	w := post(t, m, "/board/now", url.Values{"act": {"stuck"}, "why": {"big"}, "kind": {"task"}, "id": {"3"}})

	require.Equal(t, 1, c.broke)
	require.Equal(t, "/?stuck=big", w.Header().Get("Location"))
}

// The other three blockers have answers that are not a sequence.
func TestOnlyTooBigAsksTheBoardForABreakdown(t *testing.T) {
	for _, why := range []string{"how", "boring"} {
		f := aBoardStoreWithATaskOffer()
		c := breaksInto(&fakeCoach{}, "one", "two")
		m := mountedWith(t, f, c)

		post(t, m, "/board/now", url.Values{"act": {"stuck"}, "why": {why}, "kind": {"task"}, "id": {"3"}})
		require.Zero(t, c.broke, "%q asked for a breakdown", why)
	}
}

// The step belongs to the task it was about, not to whatever else you happen
// to open next — a sequence is one thing at a time for the person, and it
// must not bleed onto an unrelated strip.
func TestAStepDoesNotShowUnderAnUnrelatedStrip(t *testing.T) {
	f := aBoardStoreWithATaskOffer()
	c := breaksInto(&fakeCoach{}, "find the vet phone number")
	m := mountedWith(t, f, c)

	post(t, m, "/board/now", url.Values{"act": {"stuck"}, "why": {"big"}, "kind": {"task"}, "id": {"3"}})
	body := m.call(t, "GET", "/?open=1", nil).Body.String()

	require.NotContains(t, body, "find the vet phone number")
}

func TestFinishingTheStepOnTheStripMovesToTheNext(t *testing.T) {
	f := aBoardStoreWithATaskOffer()
	c := breaksInto(&fakeCoach{}, "find the vet phone number", "book the appointment")
	m := mountedWith(t, f, c)

	post(t, m, "/board/now", url.Values{"act": {"stuck"}, "why": {"big"}, "kind": {"task"}, "id": {"3"}})

	w := post(t, m, "/steps", url.Values{"act": {"done"}, "id": {"1"}, "from": {"/?open=3"}})
	require.Equal(t, 303, w.Code)
	require.Equal(t, "/?open=3", w.Header().Get("Location"))

	body := m.call(t, "GET", "/?open=3", nil).Body.String()
	require.Contains(t, body, "book the appointment")
	require.Contains(t, body, "the last one")
}

func TestForgettingTheStepsFromTheStripCostsOnePress(t *testing.T) {
	f := aBoardStoreWithATaskOffer()
	c := breaksInto(&fakeCoach{}, "find the vet phone number")
	m := mountedWith(t, f, c)

	post(t, m, "/board/now", url.Values{"act": {"stuck"}, "why": {"big"}, "kind": {"task"}, "id": {"3"}})
	post(t, m, "/steps", url.Values{"act": {"clear"}, "from": {"/?open=3"}})

	body := m.call(t, "GET", "/?open=3", nil).Body.String()
	require.NotContains(t, body, "find the vet phone number")
}

func TestAStepOnTheStripNeverSaysHowManyAreLeft(t *testing.T) {
	f := aBoardStoreWithATaskOffer()
	c := breaksInto(&fakeCoach{}, "find the vet phone number", "book the appointment", "confirm the time")
	m := mountedWith(t, f, c)

	post(t, m, "/board/now", url.Values{"act": {"stuck"}, "why": {"big"}, "kind": {"task"}, "id": {"3"}})
	body := m.call(t, "GET", "/?open=3", nil).Body.String()

	require.Contains(t, body, "find the vet phone number")
	for _, count := range []string{"of 3", "1/3", "step 1", "1 of"} {
		require.NotContains(t, body, count)
	}
}

func TestAnUnreadableStepIsNoStepOnTheStrip(t *testing.T) {
	f := aBoardStoreWithATaskOffer()
	c := breaksInto(&fakeCoach{}, "find the vet phone number")
	m := mountedWith(t, f, c)

	post(t, m, "/board/now", url.Values{"act": {"stuck"}, "why": {"big"}, "kind": {"task"}, "id": {"3"}})
	f.err = errTest

	require.NotPanics(t, func() {
		body := m.call(t, "GET", "/?open=3", nil).Body.String()
		require.NotContains(t, body, "find the vet phone number")
	})
}
