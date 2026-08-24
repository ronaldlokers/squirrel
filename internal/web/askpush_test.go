//go:build browser

// Whether the one control that can turn push on is ever visible.
//
// It was not, for the whole life of the feature. The template ships the button
// with `hidden` on it, and `pile.js` set `hidden = true` in the branch that
// declines to offer it and attached a click listener in the branch that offers
// it — without ever taking `hidden` off. So the offer could not be accepted,
// permission was never requested, and `push_subscriptions` held zero rows on a
// production database that had been running the feature for weeks.
//
// The existing test asserted `id="askPush"` was in the markup, which was true
// and told nobody anything. DESIGN.md already carries the rule this needed,
// written for the opposite direction: "when something is meant to disappear,
// assert that it is invisible, not that it is unmarked." A thing meant to
// appear wants the same assertion the other way up, and it has to come from a
// browser — `hidden` plus the global `[hidden] { display: none !important }` is
// a cascade result, not a string in a template.
package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// pushScreen is the screen with a key to subscribe with, over a real socket.
func pushScreen(t *testing.T, f *fakeStore) *httptest.Server {
	t.Helper()
	m := &serveMux{mux: http.NewServeMux()}
	require.NoError(t, Mount(m, f, Options{
		IdentityHeader: "X-Authentik-Username", Identity: "ronald",
		PushKey: "BKtestkey",
		Owner:   func() int64 { return 1 }, Spool: &fakeSpool{},
	}))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("X-Authentik-Username", "ronald")
		m.mux.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestTheWayToTurnPushOnCanBeSeen(t *testing.T) {
	srv := pushScreen(t, aPile())
	c := browserAt(t, srv, "/")

	require.Equal(t, "default", c.eval(t, `return Notification.permission`),
		"this browser has already answered, so the test would pass for the wrong reason")

	require.Equal(t, false, c.eval(t, `return document.getElementById("askPush").hidden`),
		"the offer to turn push on is in the markup but cannot be seen, so it cannot be accepted")

	// Not just the attribute: `[hidden]` is enforced globally with `!important`,
	// so the property and the pixels are two different claims.
	require.Equal(t, true, c.eval(t, `
		const el = document.getElementById("askPush");
		const r = el.getBoundingClientRect();
		return r.width > 0 && r.height > 0 && getComputedStyle(el).display !== "none";`),
		"it takes up no space on the page")
}

// And it stays hidden where there is nothing behind it: a control that cannot
// change anything is furniture, which is the rule the original code was
// enforcing correctly in the branch it got right.
func TestTheWayToTurnPushOnIsAbsentWithoutAKey(t *testing.T) {
	srv := screen(t, aPile())
	c := browserAt(t, srv, "/")

	require.Equal(t, nil, c.eval(t, `return document.getElementById("askPush")`),
		"no key, no button at all")
}
