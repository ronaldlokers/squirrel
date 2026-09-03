package web

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// withPush mounts the screen as it is when a VAPID pair is configured.
func withPush(t *testing.T, f *fakeStore) *testMux {
	t.Helper()
	m := newTestMux()
	require.NoError(t, Mount(m, f, Options{
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
		PushKey:  "BKtestkey",
		Spool:    &fakeSpool{},
	}))
	return m
}

// A route that always answers 400 teaches the client to stop asking, so it is
// not there at all until there is a key behind it.
func TestSubscribingIsNotEvenARouteWithoutAKey(t *testing.T) {
	require.NotContains(t, mounted(t, &fakeStore{}).routes, "POST /push/subscribe")
	require.Contains(t, withPush(t, &fakeStore{}).routes, "POST /push/subscribe")
}

func TestABrowserCanSayWhereToReachIt(t *testing.T) {
	f := &fakeStore{}
	w := postJSON(t, withPush(t, f), "/push/subscribe",
		`{"endpoint":"https://push.example/abc","keys":{"p256dh":"aaa","auth":"bbb"}}`)

	require.Equal(t, 204, w.Code)
	require.Equal(t, []string{"https://push.example/abc"}, f.subscribed)
}

// This is a refusal to store something that could never be sent to, not a
// permission check — the identity already made one.
func TestASubscriptionThatCouldNeverBeSentToIsRefused(t *testing.T) {
	for _, body := range []string{
		`{"endpoint":"http://push.example/abc","keys":{"p256dh":"a","auth":"b"}}`, // not https
		`{"endpoint":"https://push.example/abc","keys":{"p256dh":"","auth":"b"}}`, // no key
		`{"endpoint":"","keys":{"p256dh":"a","auth":"b"}}`,
		`not json at all`,
	} {
		f := &fakeStore{}
		w := postJSON(t, withPush(t, f), "/push/subscribe", body)
		require.Equal(t, 400, w.Code, body)
		require.Empty(t, f.subscribed, body)
	}
}

// The key reaches the page, because the script needs it to subscribe at all —
// and nothing is offered when there is none.
// The setting moved to its own page on 3 September 2026, and the key follows
// it: the key is on every screen because the script that uses it is, and the
// control is where the setting lives.
func TestTheKeyIsOnThePageOnlyWhenThereIsOne(t *testing.T) {
	with := withPush(t, &fakeStore{}).call(t, "GET", "/r/everything", nil).Body.String()
	require.Contains(t, with, `data-push-key="BKtestkey"`)
	require.Contains(t, withPush(t, &fakeStore{}).call(t, "GET", "/me", nil).Body.String(),
		`id="pushbit"`, "no key on the page means no setting to use it")

	without := mounted(t, &fakeStore{}).call(t, "GET", "/r/everything", nil).Body.String()
	require.NotContains(t, without, "data-push-key")
	require.NotContains(t, mounted(t, &fakeStore{}).call(t, "GET", "/me", nil).Body.String(),
		`id="pushbit"`)
}

func postJSON(t *testing.T, m *testMux, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return m.call(t, "POST", path, strings.NewReader(body))
}
