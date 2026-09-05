package web

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestThePulledStripOffersNotThisOneForATask(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferTask, RefID: 3, Text: "send the meter reading"}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `value="wrong"`)
	require.Contains(t, body, "not this one")
}

func TestThePulledStripOffersNotThisOneForAChore(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferChore, RefID: 7, Text: "bins out"}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `value="wrong"`)
}

func TestThePulledStripHasNoNotThisOneForABreadcrumb(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferAgain, Text: "the kitchen"}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.NotContains(t, body, `value="wrong"`)
}

func TestThePulledStripHasNoNotThisOneForARunningTimer(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferTimer, Text: "the kitchen", Because: "you are on this"}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.NotContains(t, body, `value="wrong"`)
}

func TestPressingNotThisOneRecordsItDistinctlyFromADeferral(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferChore, RefID: 7, Text: "bins out"}
	m := mounted(t, f)

	w := m.call(t, "POST", "/board/now", strings.NewReader("act=wrong&kind=chore&id=7"))

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, []int64{7}, f.markedWrong, "the wrong pick was not recorded")
	require.Empty(t, f.refused, "a wrong pick was recorded as a deferral instead")
	require.Contains(t, f.answers, "wrong:chore")
}

func TestNotThisOneCallsNoModel(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferChore, RefID: 7, Text: "bins out"}
	c := &fakeCoach{reply: "should never be seen"}
	m := mountedWith(t, f, c)

	m.call(t, "POST", "/board/now", strings.NewReader("act=wrong&kind=chore&id=7"))

	require.Empty(t, c.asked, "marking a pick wrong must never cost a model call")
}
