package web

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func aBoardStoreWithATaskOffer() *fakeStore {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferTask, RefID: 3, Text: "vet about the booster"}
	return f
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

func TestTooBigWithNothingBrokenDownFallsBackToTheFixedLine(t *testing.T) {
	f := aBoardStoreWithATaskOffer()
	c := &fakeCoach{}
	m := mountedWith(t, f, c)

	w := post(t, m, "/board/now", url.Values{"act": {"stuck"}, "why": {"big"}, "kind": {"task"}, "id": {"3"}})

	require.Equal(t, 1, c.broke)
	require.Equal(t, "/?stuck=big", w.Header().Get("Location"))
}

func TestOnlyTooBigAsksTheBoardForABreakdown(t *testing.T) {
	for _, why := range []string{"how", "boring"} {
		f := aBoardStoreWithATaskOffer()
		c := breaksInto(&fakeCoach{}, "one", "two")
		m := mountedWith(t, f, c)

		post(t, m, "/board/now", url.Values{"act": {"stuck"}, "why": {why}, "kind": {"task"}, "id": {"3"}})
		require.Zero(t, c.broke, "%q asked for a breakdown", why)
	}
}

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
