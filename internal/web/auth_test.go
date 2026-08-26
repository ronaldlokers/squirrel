package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func testOptions() Options {
	return Options{
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
	}
}

func TestGuardAllowsASession(t *testing.T) {
	reached := false
	h := guard(testOptions(), func(http.ResponseWriter, *http.Request) { reached = true })

	r := httptest.NewRequest("GET", "/pile", nil)
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "a-token"})
	w := httptest.NewRecorder()
	h(w, r)

	require.True(t, reached)
	require.Equal(t, http.StatusOK, w.Code)
}

// A header is not a session. This is the whole of what changed on 25 August
// 2026, and it is worth a test of its own: X-Authentik-Username used to be the
// entire authentication, and a deployment that still had the outpost in front
// must not keep working by accident after the middleware comes off.
func TestTheIdentityHeaderIsNotASession(t *testing.T) {
	reached := false
	h := guard(testOptions(), func(http.ResponseWriter, *http.Request) { reached = true })

	r := httptest.NewRequest("GET", "/pile", nil)
	r.Header.Set("X-Authentik-Username", "ronald")
	w := httptest.NewRecorder()
	h(w, r)

	require.False(t, reached, "a header signed somebody in")
	require.Equal(t, http.StatusSeeOther, w.Code)
}

// A cookie the store does not know is nobody.
func TestGuardRefusesACookieNobodyOpened(t *testing.T) {
	opts := testOptions()
	opts.Sessions = newSessions(&nobodyEver{}, cacheFor, cacheMost)
	reached := false
	h := guard(opts, func(http.ResponseWriter, *http.Request) { reached = true })

	r := httptest.NewRequest("POST", "/pile/act", nil)
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "made up"})
	w := httptest.NewRecorder()
	h(w, r)

	require.False(t, reached, "a made-up cookie reached a handler")
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Empty(t, w.Body.String(), "a refusal says nothing about what is behind it")
}

// Options.Owner was a process-global atomic.Int64 and opts.person() read it in
// forty-nine places. That cannot survive two people.
func TestWhoIsAskingComesFromTheRequest(t *testing.T) {
	r := withWho(httptest.NewRequest("GET", "/", nil), 7, "sub-7")

	id, ok := personOf(r)
	require.True(t, ok)
	require.Equal(t, int64(7), id)
	require.Equal(t, "sub-7", subOf(r))
}

// A request nobody has been put on is nobody. It must not fall back to a
// process-wide owner, which is how a second person would silently read the
// first one's pile.
func TestARequestWithNobodyOnItIsNobody(t *testing.T) {
	_, ok := personOf(httptest.NewRequest("GET", "/", nil))
	require.False(t, ok, "a request nobody was put on resolved to somebody")
	require.Empty(t, subOf(httptest.NewRequest("GET", "/", nil)))
}

// A person id of zero is nobody too, whatever was put on the request. Owner()
// is zero until Postgres answers, and zero is not a person.
func TestAPersonOfZeroIsNobody(t *testing.T) {
	_, ok := personOf(withWho(httptest.NewRequest("GET", "/", nil), 0, "sub"))
	require.False(t, ok, "person zero was treated as somebody")
}

// The guard is what puts them there. Every handler downstream reads the
// request rather than the process, so a guard that forgets is a guard that
// serves nobody's pile to everybody.
func TestTheGuardPutsThemOnTheRequest(t *testing.T) {
	var saw int64
	var sub string
	h := guard(testOptions(), func(_ http.ResponseWriter, r *http.Request) {
		saw, _ = personOf(r)
		sub = subOf(r)
	})
	r := httptest.NewRequest("GET", "/pile", nil)
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "a-token"})
	h(httptest.NewRecorder(), r)

	require.Equal(t, int64(1), saw, "the guard let a request through with nobody on it")
	require.Equal(t, "ronald", sub)
}

// And a guard that cannot say who is asking refuses rather than passing the
// request along for a handler to discover the same thing.
//
// This is the one place in the product that fails closed on an unreachable
// Postgres. Everywhere else the failure costs a feature; here the alternative
// is guessing who is asking.
func TestTheGuardRefusesWhenItCannotSayWhoIsAsking(t *testing.T) {
	reached := false
	opts := testOptions()
	opts.Sessions = newSessions(&unwellStore{}, cacheFor, cacheMost)
	h := guard(opts, func(http.ResponseWriter, *http.Request) { reached = true })

	r := httptest.NewRequest("POST", "/pile/act", nil)
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "a-token"})
	w := httptest.NewRecorder()
	h(w, r)

	require.False(t, reached, "a request reached a handler with nobody established")
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

// A session whose person is zero is nobody, whatever the store said. Nothing
// should write such a row, and a guard that trusts it is a guard that hands
// every handler a person id of zero to scope by.
func TestGuardRefusesASessionWithNobodyInIt(t *testing.T) {
	reached := false
	opts := testOptions()
	opts.Sessions = newSessions(&sessionForNobody{}, cacheFor, cacheMost)
	h := guard(opts, func(http.ResponseWriter, *http.Request) { reached = true })

	r := httptest.NewRequest("GET", "/pile", nil)
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "a-token"})
	h(httptest.NewRecorder(), r)

	require.False(t, reached, "a session belonging to nobody reached a handler")
}
