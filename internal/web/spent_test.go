package web

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func mountedSpending(t *testing.T, f *fakeStore, spent, ceiling string, ok bool) *testMux {
	t.Helper()
	m := newTestMux()
	require.NoError(t, Mount(m, f, Options{
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
		Location: time.Local,
		Spent: func(context.Context, int64) (string, string, bool) {
			return spent, ceiling, ok
		},
	}))
	return m
}

func TestTheOneCountThisProductPermitsIsOnAScreen(t *testing.T) {
	body := mountedSpending(t, &fakeStore{}, "€2.40", "€10", true).
		call(t, "GET", "/me", nil).Body.String()

	require.Contains(t, body, "€2.40 of €10",
		"the ceiling is invisible until the month it is reached, which is what the exception exists to prevent")
	require.Contains(t, body, "What Buddy has cost this month")
}

func TestWithNoCoachNothingReportsOnWhatItCost(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/me", nil).Body.String()

	require.NotContains(t, body, "What Buddy has cost this month",
		"a build with no key reports the cost of something that cannot be called")
}

func TestACostThatCannotBeReadIsNotDrawnAsZero(t *testing.T) {
	body := mountedSpending(t, &fakeStore{}, "", "", false).
		call(t, "GET", "/me", nil).Body.String()

	require.NotContains(t, body, "What Buddy has cost this month",
		"a cost that could not be read was drawn as if it had been")
}

func TestTheCountIsOnTheOneScreenAndNoOther(t *testing.T) {
	m := mountedSpending(t, aBoardStore(), "€2.40", "€10", true)

	for _, where := range []string{"/", "/?bay=chores", "/?bay=tasks", "/?bay=agenda"} {
		body := m.call(t, "GET", where, nil).Body.String()
		require.NotContains(t, strings.ToLower(body), "cost this month",
			"%s reports a running cost, and the exception is for one screen only", where)
		require.NotContains(t, body, "€2.40", "%s draws money", where)
	}
}
