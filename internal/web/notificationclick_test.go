//go:build browser

package web

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// What a notification tap actually does, run rather than read.
//
// The worker has always meant to send you to the front door, and its comment
// says so — but it only managed it when nothing was open. `focus()` raises a
// window exactly as it was left, so a screen already open put you back wherever
// you last were — halfway up this morning's conversation — which is the
// opposite of "a page that says what is true now".
//
// The handler is lifted out of sw.js and run in the page with the two things
// it touches faked, because the alternative — a registered worker, a real push
// and a real notification — is not something a test can press. What is under
// test is the handler's own logic, and that is all of it.
const runNotificationClick = `
	const src = await (await fetch("/sw.js")).text();

	// Only the handler. Evaluating the whole file would register a fetch
	// listener over the page it is being tested in.
	const from = src.indexOf('self.addEventListener("notificationclick"');
	if (from < 0) throw new Error("sw.js has no notificationclick handler");

	window.__went = [];
	let focused = 0;
	const client = {
		url: location.origin + AT,
		navigate(to) { window.__went.push(to); this.url = new URL(to, location.origin).href; return Promise.resolve(this); },
		focus() { focused++; return Promise.resolve(this); },
	};
	self.clients = {
		matchAll: async () => OPEN ? [client] : [],
		openWindow: async (to) => { window.__went.push("opened " + to); },
	};

	eval(src.slice(from));

	const event = new Event("notificationclick");
	event.notification = { close() {}, data: DATA };
	let settled;
	event.waitUntil = (p) => { settled = p; };
	self.dispatchEvent(event);
	await settled;

	return { went: window.__went, focused, ended: new URL(client.url).pathname };`

func notificationClick(t *testing.T, c *cdp, at string, open bool) map[string]any {
	t.Helper()
	setup := "const DATA = undefined; const AT = " + `"` + at + `"; const OPEN = `
	if open {
		setup += "true;"
	} else {
		setup += "false;"
	}
	got := c.eval(t, setup+runNotificationClick)
	out, ok := got.(map[string]any)
	require.True(t, ok, "the handler answered %#v", got)
	return out
}

// A window left on another screen is taken to the front door before it is
// raised.
func TestBrowserTappingANotificationReachesTheFrontDoor(t *testing.T) {
	c, _ := open(t, aPile())

	got := notificationClick(t, c, "/moods", true)

	require.Equal(t, "/", got["ended"],
		"the tap raised a window still sitting on %v", got["ended"])
	require.Equal(t, float64(1), got["focused"], "and it is raised, not merely navigated")
}

// A window already at the front door is left exactly as it is.
//
// Navigating would reload it, and home is the one screen carrying a thought
// that has not been written down yet. A notification is not allowed to throw
// that away to tell you something.
func TestBrowserTappingANotificationDoesNotReloadTheFrontDoor(t *testing.T) {
	c, _ := open(t, aPile())

	got := notificationClick(t, c, "/", true)

	require.Empty(t, got["went"], "the front door was navigated over itself, discarding the slot")
	require.Equal(t, float64(1), got["focused"])
}

func TestBrowserTappingANotificationWithNothingOpenOpensTheFrontDoor(t *testing.T) {
	c, _ := open(t, aPile())

	got := notificationClick(t, c, "/moods", false)

	require.Equal(t, []any{"opened /"}, got["went"])
}

// The front door was right while there was nowhere better to go, and its
// reasoning is kept: a link to something already done is worse than a page
// saying what is true now. A fixed point inside its leave-by window is the one
// thing here that cannot be stale — the notification and the window are the
// same fact.
func TestBrowserTappingALeaveByNotificationLandsOnTheFixedPoint(t *testing.T) {
	c, _ := open(t, aPile())

	setup := `const DATA = { url: "/at/4" }; const AT = "/moods"; const OPEN = true;`
	got, ok := c.eval(t, setup+runNotificationClick).(map[string]any)
	require.True(t, ok)

	require.Equal(t, "/at/4", got["ended"],
		"the tap raised a window still sitting on %v", got["ended"])
	require.Equal(t, float64(1), got["focused"])
}

// And a window already on that URL is left alone, exactly as the front door is.
//
// The rule is about the URL rather than about "/", which is what this pins that
// its sibling above cannot: the two are the same comparison, and a special case
// for the front door would pass that one and fail this.
func TestBrowserTappingItAgainDoesNotReloadTheFixedPoint(t *testing.T) {
	c, _ := open(t, aPile())

	setup := `const DATA = { url: "/at/4" }; const AT = "/at/4"; const OPEN = true;`
	got, ok := c.eval(t, setup+runNotificationClick).(map[string]any)
	require.True(t, ok)

	require.Empty(t, got["went"], "it was navigated over itself, discarding the slot")
	require.Equal(t, float64(1), got["focused"])
}
