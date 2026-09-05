package web

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestOnTheBoardARunningTimerOfferCarriesNoButtons(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferTimer, Text: "the kitchen", Because: "you are on this"}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, "the kitchen")
	require.NotContains(t, body, `name="act" value="did"`)
	require.NotContains(t, body, `name="act" value="later"`)
	require.NotContains(t, body, `name="act" value="stuck"`)
}

func TestOnTheBoardTheBreadcrumbOffersOnlyTheWayBackIn(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{
		Kind: squirrel.OfferAgain, Text: "the kitchen",
		Because: "you were on this a little while ago",
	}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, "pick it up")
	require.Contains(t, body, "not now")
	require.NotContains(t, body, `name="act" value="did"`)
	require.NotContains(t, body, `name="act" value="stuck"`)
}

func TestPickingUpTheBreadcrumbStartsATenMinuteTimer(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferAgain, Text: "the kitchen"}
	m := mounted(t, f)

	w := post(t, m, "/board/now", url.Values{
		"act": {"start"}, "kind": {"again"}, "minutes": {"10"}, "label": {"the kitchen"},
	})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/", w.Header().Get("Location"))
	require.NotNil(t, f.timer)
	require.Equal(t, "the kitchen", f.timer.Label)
	require.Equal(t, 10*time.Minute, f.timer.Ends.Sub(f.timer.Started))
	require.Equal(t, []string{"started:again"}, f.answers)
}

func TestStoppingARunningTimerFromTheBoardLeavesNothing(t *testing.T) {
	f := aBoardStore()
	f.timer = &squirrel.Timer{Label: "the kitchen", Started: time.Now(), Ends: time.Now().Add(time.Minute)}
	m := mounted(t, f)

	w := post(t, m, "/board/now", url.Values{"act": {"stop"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/", w.Header().Get("Location"))
	require.Nil(t, f.timer)
}

func TestTheTickingTimerCarriesItsOwnStopControl(t *testing.T) {
	f := aBoardStore()
	f.timer = &squirrel.Timer{Label: "the kitchen", Started: time.Now(), Ends: time.Now().Add(6*time.Minute + 12*time.Second)}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `<input type="hidden" name="act" value="stop">`)
	require.Contains(t, body, "stop")
}

func TestTooBigOffersFiveMinutesAndNothingElse(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferTask, RefID: 3, Text: "send the meter reading"}
	body := mounted(t, f).call(t, "GET", "/?stuck=too+big", nil).Body.String()

	require.Contains(t, body, squirrel.UnstuckFor(squirrel.BlockerBig).Line)
	require.Contains(t, body, `<input type="hidden" name="minutes" value="5">`)
	require.Contains(t, body, "5 MIN<")
	require.NotContains(t, body, `name="act" value="did"`)
	require.NotContains(t, body, `name="act" value="stuck"`)
}

func TestStartingTheLaddersTimerFromTheBoard(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferTask, RefID: 3, Text: "send the meter reading"}
	m := mounted(t, f)

	w := post(t, m, "/board/now", url.Values{
		"act": {"timer"}, "minutes": {"5"}, "label": {"send the meter reading"},
	})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/", w.Header().Get("Location"))
	require.NotNil(t, f.timer)
	require.Equal(t, "send the meter reading", f.timer.Label)
	require.Equal(t, 5*time.Minute, f.timer.Ends.Sub(f.timer.Started))
}

func TestDontKnowHowOffersNoTimerControl(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferTask, RefID: 3, Text: "send the meter reading"}
	body := mounted(t, f).call(t, "GET", "/?stuck=don%27t+know+how", nil).Body.String()

	require.Contains(t, body, squirrel.UnstuckFor(squirrel.BlockerHow).Line)
	require.NotContains(t, body, "MIN<")
}

func TestBogusMinutesFromTheLadderStartsNothing(t *testing.T) {
	f := aBoardStore()
	m := mounted(t, f)

	post(t, m, "/board/now", url.Values{"act": {"timer"}, "minutes": {"not a number"}, "label": {"x"}})

	require.Nil(t, f.timer)
}
