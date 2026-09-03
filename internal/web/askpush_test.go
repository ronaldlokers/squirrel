//go:build browser

// Whether the one control that can turn push on is ever visible, and whether it
// can say what it is.
//
// It was not visible for the whole life of the feature. The template shipped
// the button with `hidden` on it, and `pile.js` set `hidden = true` in the
// branch that declines to offer it and attached a click listener in the branch
// that offers it — without ever taking `hidden` off. So the offer could not be
// accepted, permission was never requested, and `push_subscriptions` held zero
// rows on a production database that had been running the feature for weeks.
//
// The test that covered that asserted `id="askPush"` was in the markup, which
// was true and told nobody anything. DESIGN.md carries the rule it needed,
// written for the opposite direction: "when something is meant to disappear,
// assert that it is invisible, not that it is unmarked." A thing meant to
// appear wants the same assertion the other way up, and it has to come from a
// browser — `hidden` plus the global `[hidden] { display: none !important }` is
// a cascade result, not a string in a template.
//
// The control is a setting in the panel now rather than a one-shot button, so
// there is a second thing to prove: that it says which way it is set.
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
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
		PushKey:  "BKtestkey",
		Spool:    &fakeSpool{},
	}))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "a-token"})
		m.mux.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// The setting is inside a disclosure, so it is opened before it is measured:
// a control nobody has opened to is not the thing under test.
// A desktop width, because below 980px the rail is the room sheet's contents
// and a control inside a closed sheet is display:none whatever its own
// attribute says — which is the exact confusion this file exists to refuse.
// The settings are a page of their own since 3 September 2026, so opening them
// is going there rather than opening a disclosure.
func openSettings(t *testing.T, c *cdp, srv *httptest.Server) {
	t.Helper()
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 1280, "height": 900, "deviceScaleFactor": 1, "mobile": false,
	})
	c.navigate(t, srv.URL+"/me")
	c.eval(t, `return new Promise(r => setTimeout(() => r(1), 120))`)
}

// permitted grants the permission this browser would otherwise be asked for, so a
// test about the "on" state is not really a test about "not asked yet".
func permitted(t *testing.T, c *cdp, origin string) {
	t.Helper()
	c.send(t, "Browser.setPermission", map[string]any{
		"permission": map[string]any{"name": "notifications"},
		"setting":    "granted",
		"origin":     origin,
	})
}

func TestTheWayToTurnPushOnCanBeSeen(t *testing.T) {
	srv := pushScreen(t, aPile())
	c := browserAt(t, srv, "/r/everything")
	openSettings(t, c, srv)

	require.Equal(t, "default", c.eval(t, `return Notification.permission`),
		"this browser has already answered, so the test would pass for the wrong reason")

	require.Equal(t, false, c.eval(t, `return document.getElementById("pushon").hidden`),
		"the way to turn notifications on is in the markup but cannot be seen")

	// Not just the attribute: `[hidden]` is enforced globally with `!important`,
	// so the property and the pixels are two different claims.
	require.Equal(t, true, c.eval(t, `
		const el = document.getElementById("pushon");
		const r = el.getBoundingClientRect();
		return r.width > 0 && r.height > 0 && getComputedStyle(el).display !== "none";`),
		"it takes up no space on the page")
}

// And it stays absent where there is nothing behind it: a control that cannot
// change anything is furniture, which is the rule the original code was
// enforcing correctly in the branch it got right.
func TestTheWayToTurnPushOnIsAbsentWithoutAKey(t *testing.T) {
	srv := screen(t, aPile())
	c := browserAt(t, srv, "/r/everything")
	openSettings(t, c, srv)

	require.Equal(t, nil, c.eval(t, `return document.getElementById("pushbit")`),
		"no key, no setting at all")
}

// A setting says which way it is set. The old control could not: it hid itself
// the moment it was answered, either way, so "on" and "refused" looked the same
// from the outside — which is nothing at all.
func TestTheSettingSaysWhetherItIsOn(t *testing.T) {
	f := aPile()
	f.notifying = true
	srv := pushScreen(t, f)
	c := browserAt(t, srv, "/r/everything")
	permitted(t, c, srv.URL)
	openSettings(t, c, srv)

	require.Contains(t, c.eval(t, `return document.getElementById("pushsays").textContent`), "On.",
		"the panel does not say that notifications are on")
	require.Equal(t, false, c.eval(t, `return document.getElementById("pushoff").hidden`),
		"there is no way to turn them off")
	require.Equal(t, true, c.eval(t, `return document.getElementById("pushon").hidden`),
		"it offers to turn on something that is already on")
}

// And the way back once a browser has refused. Issue #147: a browser that has
// been told no will not ask again and this site cannot make it, so a no was the
// end of it with nothing said.
func TestARefusalSaysWhereTheSwitchIs(t *testing.T) {
	srv := pushScreen(t, aPile())
	c := browserAt(t, srv, "/r/everything")
	c.send(t, "Browser.setPermission", map[string]any{
		"permission": map[string]any{"name": "notifications"},
		"setting":    "denied",
		"origin":     srv.URL,
	})
	openSettings(t, c, srv)

	require.Contains(t, c.eval(t, `return document.getElementById("pushsays").textContent`),
		"Turn notifications on for this site",
		"a refusal says nothing about where the switch is")
	require.Equal(t, true, c.eval(t, `return document.getElementById("pushon").hidden`),
		"it offers a button the browser will not honour")
}

// And no key, no sentence: there is nothing to turn on.
func TestNoKeyMeansNothingIsSaidAboutNotifications(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/r/everything", nil).Body.String()

	require.NotContains(t, body, "Notifications")
}
