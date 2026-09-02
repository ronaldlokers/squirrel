package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPressComesBackToTheRoomItWasMadeIn(t *testing.T) {
	for _, room := range []string{"at", "chores", "tasks", "notes"} {
		t.Run(room, func(t *testing.T) {
			f := &fakeStore{}
			body := url.Values{"room": {room}, "label": {""}}.Encode()
			res := routed(t, f).call(t, "POST", "/at/new", strings.NewReader(body))

			require.Equal(t, 303, res.Code)
			require.Equal(t, theBays[room], res.Header().Get("Location"),
				"a press made about %s does not come back to its bay", room)
		})
	}
}

func TestAPressFromBuddysRoomComesBackToTheFrontDoor(t *testing.T) {
	f := &fakeStore{}
	body := url.Values{"room": {"everything"}, "label": {""}}.Encode()
	res := routed(t, f).call(t, "POST", "/at/new", strings.NewReader(body))

	require.Equal(t, "/r/everything", res.Header().Get("Location"))
}

func TestATimeOfDayTakesAnyTimeOnTheClock(t *testing.T) {
	for _, at := range []string{"00:00", "09:00", "11:15", "14:30", "23:59"} {
		require.True(t, aTimeOfDay(at), at)
	}
	for _, at := range []string{"", "9:00", "24:00", "12:60", "1130", "11:1", "aa:bb", "11:15:00"} {
		require.False(t, aTimeOfDay(at), at)
	}
}

func TestAnAppointmentCanBeAtATimeNoChipOffers(t *testing.T) {
	f := &fakeStore{}
	body := url.Values{
		"room": {"at"}, "label": {"dentist"},
		"day": {now().AddDate(0, 0, 1).Format("2006-01-02")}, "at": {"11:15"},
	}.Encode()
	res := routed(t, f).callFragment(t, "/at/make", body)

	require.Equal(t, 200, res.Code)
	require.Len(t, f.moments, 1, "11:15 is a time on the clock and was not kept")
	require.Equal(t, 11, f.moments[0].Starts.Hour())
	require.Equal(t, 15, f.moments[0].Starts.Minute())
}

func TestTurningTheMonthReplacesTheQuestionInPlace(t *testing.T) {
	f := &fakeStore{}
	body := url.Values{
		"room": {"at"}, "label": {"dentist"},
		"month": {"2026-12"}, "turn": {"7"},
	}.Encode()
	res := routed(t, f).callFragment(t, "/at/new", body)

	require.Equal(t, "turn-7", res.Header().Get("X-Replaces"),
		"the answer does not say which turn it replaces, so the script appends it")
	require.Contains(t, res.Body.String(), `id="turn-7"`)
	require.Contains(t, res.Body.String(), "December 2026")
	require.Empty(t, f.appended, "paging a calendar was written into the conversation")
}

func TestAskingForADayIsKept(t *testing.T) {
	f := &fakeStore{}
	body := url.Values{"room": {"everything"}, "label": {"dentist"}}.Encode()
	res := routed(t, f).callFragment(t, "/at/new", body)

	require.Empty(t, res.Header().Get("X-Replaces"),
		"the first question replaces nothing; there is nothing there yet")
	require.NotEmpty(t, f.appended, "the question was not kept")
}
