package web

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// answered is a store with a fresh check-in, because the offer only appears
// once the question has been answered — asking how you are and handing you a
// job in the same breath is the interruption this product exists to reduce.
func answered(offer *squirrel.Offer) *fakeStore {
	return &fakeStore{
		checkin: &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: time.Now()},
		offer:   offer,
	}
}

var aTask = &squirrel.Offer{
	Kind:    squirrel.OfferTask,
	RefID:   7,
	Text:    "ring the vet about the booster",
	Because: "you decided this on tuesday",
}

func TestHomeOffersOneThingWithItsClause(t *testing.T) {
	body := mounted(t, answered(aTask)).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, "ring the vet about the booster")
	require.Contains(t, body, "you decided this on tuesday")
	require.Contains(t, body, "DID IT")
}

// The hardest rule in the product, on the newest surface. One thing means one.
func TestTheOfferIsNeverAListAndNeverACount(t *testing.T) {
	body := mounted(t, answered(aTask)).call(t, "GET", "/", nil).Body.String()

	require.Equal(t, 1, strings.Count(body, `class="offer"`))
	require.Equal(t, 1, strings.Count(body, `value="did"`))
	require.NotContains(t, body, "and more")
	require.NotContains(t, body, "left")
}

// Absent rather than empty. Having nothing to be handed is a normal state, and
// a reassuring sentence in its place would be the product deciding you ought
// to be busy.
func TestNothingToOfferRendersNoRegionAtAll(t *testing.T) {
	body := mounted(t, answered(nil)).call(t, "GET", "/", nil).Body.String()

	require.NotContains(t, body, `class="offer"`)
	require.NotContains(t, body, "nothing")
}

// The question comes first: with no fresh reading the region is the check-in,
// so there is still exactly one interactive thing above the doors.
func TestWithNoFreshReadingTheRegionAsksInstead(t *testing.T) {
	body := mounted(t, &fakeStore{offer: aTask}).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, "how do you feel?")
	require.NotContains(t, body, `class="offer"`)
}

// A picker that cannot answer must not take down a page that rendered without
// one for the product's whole life.
func TestTheOfferFailingLeavesHomeStanding(t *testing.T) {
	s := answered(aTask)
	s.err = errTest
	w := mounted(t, s).call(t, "GET", "/", nil)

	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "the pile")
}

// Saying you are wiped must not be a wall.
func TestALowDayOffersTheWayThrough(t *testing.T) {
	s := answered(aTask)
	s.gated = true

	body := mounted(t, s).call(t, "GET", "/", nil).Body.String()
	require.NotContains(t, body, `class="offer"`)
	require.Contains(t, body, "anyway")

	body = mounted(t, s).call(t, "GET", "/?anyway=1", nil).Body.String()
	require.Contains(t, body, "ring the vet about the booster")
	require.NotContains(t, body, "show me something anyway",
		"the way through does not offer itself again once taken")
}

func TestDidItCompletesTheOfferedThing(t *testing.T) {
	s := answered(aTask)
	w := post(t, mounted(t, s), "/now/act", url.Values{
		"kind": {"task"}, "id": {"7"}, "act": {"did"},
	})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/", w.Header().Get("Location"))
	require.Equal(t, []string{"did:task"}, s.answers)
}

// One press, no consequence, no follow-up question.
func TestNotNowRefusesAndNothingElse(t *testing.T) {
	s := answered(aTask)
	post(t, mounted(t, s), "/now/act", url.Values{
		"kind": {"task"}, "id": {"7"}, "act": {"later"},
	})

	require.Equal(t, []int64{7}, s.refused)
	require.Equal(t, []string{"later:task"}, s.answers)
}

// The same timer row `!timer` and the chores screen write, labelled with the
// offer's own words.
func TestTenMinutesStartsTheBodyDouble(t *testing.T) {
	s := answered(aTask)
	post(t, mounted(t, s), "/now/act", url.Values{
		"kind": {"task"}, "id": {"7"}, "act": {"start"},
		"minutes": {"10"}, "label": {"ring the vet about the booster"},
	})

	require.NotNil(t, s.timer)
	require.Equal(t, "ring the vet about the booster", s.timer.Label)
	require.Equal(t, []string{"started:task"}, s.answers)
}

// A kind that was never offered is a lookup miss rather than a default branch
// someone later fills in with something destructive.
func TestAnUnknownKindWritesNothing(t *testing.T) {
	s := answered(aTask)
	w := post(t, mounted(t, s), "/now/act", url.Values{
		"kind": {"whatever"}, "id": {"7"}, "act": {"did"},
	})

	require.Equal(t, 303, w.Code)
	require.Empty(t, s.answers)
}

// You are already on it, so there is nothing to press: the lid carries the one
// control this needs, which is the way to stop.
func TestARunningTimerOfferCarriesNoButtons(t *testing.T) {
	body := mounted(t, answered(&squirrel.Offer{
		Kind: squirrel.OfferTimer, Text: "the kitchen", Because: "you are on this",
	})).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, "the kitchen")
	require.NotContains(t, body, `class="oacts"`)
}

// A breadcrumb names what you were on rather than a row Squirrel can act on,
// so it offers the way back into it and nothing else.
func TestTheBreadcrumbOffersOnlyTheWayBackIn(t *testing.T) {
	body := mounted(t, answered(&squirrel.Offer{
		Kind: squirrel.OfferAgain, Text: "the kitchen",
		Because: "you were on this a little while ago",
	})).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, "PICK IT UP")
	require.NotContains(t, body, "DID IT")
	require.Contains(t, body, "not now")
}
