package web

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestTickingTheBoxWhenPickingUpTheBreadcrumbArmsTheRamp(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferAgain, Text: "the kitchen"}
	m := mounted(t, f)

	post(t, m, "/board/now", url.Values{
		"act": {"start"}, "kind": {"again"}, "minutes": {"10"}, "label": {"the kitchen"}, "ramp": {"1"},
	})

	require.Equal(t, []bool{true}, f.armed)
}

func TestNotTickingTheBoxWhenPickingUpTheBreadcrumbLeavesTheRampUnarmed(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferAgain, Text: "the kitchen"}
	m := mounted(t, f)

	post(t, m, "/board/now", url.Values{
		"act": {"start"}, "kind": {"again"}, "minutes": {"10"}, "label": {"the kitchen"},
	})

	require.Empty(t, f.armed)
}

func TestTickingTheBoxOnTheLaddersTimerArmsTheRamp(t *testing.T) {
	f := aBoardStore()
	f.offer = &squirrel.Offer{Kind: squirrel.OfferTask, RefID: 3, Text: "send the meter reading"}
	m := mounted(t, f)

	post(t, m, "/board/now", url.Values{
		"act": {"timer"}, "minutes": {"5"}, "label": {"send the meter reading"}, "ramp": {"1"},
	})

	require.Equal(t, []bool{true}, f.armed)
}

func TestTheBoardShowsTheRampBannerWhenOneIsDue(t *testing.T) {
	f := aBoardStore()
	f.hasRamp = true
	f.ramp = squirrel.Timer{Label: "the tax return"}
	m := mountedWhere(t, f, time.UTC)
	was := now
	t.Cleanup(func() { now = was })
	now = func() time.Time { return time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC) }

	body := m.call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, "the tax return")
	require.Equal(t, 1, f.rampSaid)
}

func TestTheRampStaysSilentAtNight(t *testing.T) {
	f := aBoardStore()
	f.hasRamp = true
	f.ramp = squirrel.Timer{Label: "the tax return"}
	m := mountedWhere(t, f, time.UTC)
	was := now
	t.Cleanup(func() { now = was })
	now = func() time.Time { return time.Date(2026, 9, 7, 23, 0, 0, 0, time.UTC) }

	body := m.call(t, "GET", "/", nil).Body.String()

	require.NotContains(t, body, "the tax return")
	require.Equal(t, 0, f.rampSaid, "it spoke first at a bad moment")
}

func TestTheRampStaysSilentOnALowCapacityDay(t *testing.T) {
	f := aBoardStore()
	f.hasRamp = true
	f.ramp = squirrel.Timer{Label: "the tax return"}
	f.capacity = squirrel.CapacityLow
	m := mountedWhere(t, f, time.UTC)
	was := now
	t.Cleanup(func() { now = was })
	now = func() time.Time { return time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC) }

	body := m.call(t, "GET", "/", nil).Body.String()

	require.NotContains(t, body, "the tax return")
	require.Equal(t, 0, f.rampSaid, "it spoke first on a low day")
}

func TestLeaveMeAloneCallsHushRamp(t *testing.T) {
	f := aBoardStore()
	m := mounted(t, f)

	w := post(t, m, "/board/now", url.Values{"act": {"hush"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, 1, f.hushed)
}
