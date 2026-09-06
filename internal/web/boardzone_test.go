package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func mountedWhere(t *testing.T, f *fakeStore, where *time.Location) *testMux {
	t.Helper()
	m := newTestMux()
	require.NoError(t, Mount(m, f, Options{
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
		Location: where,
	}))
	return m
}

func TestTheBoardsClockIsWhereYouAreAndNotWhereTheProcessIs(t *testing.T) {
	amsterdam, err := time.LoadLocation("Europe/Amsterdam")
	require.NoError(t, err)

	was := now
	t.Cleanup(func() { now = was })
	// 23:30 UTC on a summer evening is half past one the next morning in
	// Amsterdam: a different clock, a different day and a different month.
	now = func() time.Time { return time.Date(2026, time.June, 30, 23, 30, 0, 0, time.UTC) }

	body := mountedWhere(t, aBoardStore(), amsterdam).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, "01:30", "the clock is the container's, not yours")
	require.Contains(t, body, "Wednesday 1 July", "the day is the container's, not yours")
	require.NotContains(t, body, "23:30")
	require.NotContains(t, body, "Tuesday 30 June")
}

func TestTheBoardFallsBackToUTCRatherThanToWhateverTheProcessHas(t *testing.T) {
	was := now
	t.Cleanup(func() { now = was })
	now = func() time.Time { return time.Date(2026, time.June, 30, 23, 30, 0, 0, time.UTC) }

	body := mountedWhere(t, aBoardStore(), nil).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, "23:30",
		"with nowhere named the screen must say UTC, which is a clock you can recognise as wrong")
}
