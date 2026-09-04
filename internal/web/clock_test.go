package web

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func formWith(t *testing.T, values url.Values) string {
	t.Helper()
	r := httptest.NewRequest("POST", "/board/new", strings.NewReader(values.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, r.ParseForm())
	return clockFrom(r)
}

// A native time input renders in the browser's locale and no attribute makes
// it say 24 hours, so the hour and the minute are two fields of our own.
func TestTheClockIsBuiltFromTheHourAndTheMinute(t *testing.T) {
	require.Equal(t, "14:30", formWith(t, url.Values{"hour": {"14"}, "minute": {"30"}}))
	require.Equal(t, "09:05", formWith(t, url.Values{"hour": {"9"}, "minute": {"5"}}))
}

// The old field still works: something on a phone's back button may post it.
func TestAWholeTimeStillCounts(t *testing.T) {
	require.Equal(t, "14:30", formWith(t, url.Values{"time": {"14:30"}}))
	require.Equal(t, "14:30", formWith(t,
		url.Values{"time": {"14:30"}, "hour": {"09"}, "minute": {"05"}}))
}

func TestHalfAClockIsNoClock(t *testing.T) {
	require.Empty(t, formWith(t, url.Values{"hour": {"14"}}))
	require.Empty(t, formWith(t, url.Values{"minute": {"30"}}))
	require.Empty(t, formWith(t, url.Values{}))
}
