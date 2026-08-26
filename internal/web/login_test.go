package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// fakeSessions stands in for the store's half of a session. What is being
// tested here is the flow, not the SQL — that is proved against Postgres in
// internal/squirrel.
type fakeSessions struct {
	opened  []squirrel.Session
	tokens  [][]byte
	ended   [][]byte
	live    map[string]squirrel.Session
	openErr error
}

func newFakeSessions() *fakeSessions {
	return &fakeSessions{live: map[string]squirrel.Session{}}
}

func (f *fakeSessions) OpenSession(_ context.Context, personID int64, sub string, token []byte, _ time.Time, _ time.Duration) error {
	if f.openErr != nil {
		return f.openErr
	}
	s := squirrel.Session{PersonID: personID, Sub: sub, ExpiresAt: time.Now().Add(time.Hour)}
	f.opened = append(f.opened, s)
	f.tokens = append(f.tokens, token)
	f.live[string(token)] = s
	return nil
}

func (f *fakeSessions) SessionFor(_ context.Context, token []byte, _ time.Time) (squirrel.Session, bool, error) {
	s, ok := f.live[string(token)]
	return s, ok, nil
}

func (f *fakeSessions) EndSession(_ context.Context, token []byte) error {
	f.ended = append(f.ended, token)
	delete(f.live, string(token))
	return nil
}

func TestAPageWithNoSessionGoesToTheGate(t *testing.T) {
	m := mounted(t, &fakeStore{})
	w := m.callAnonymously(t, "GET", "/moods", nil)

	require.Equal(t, http.StatusSeeOther, w.Code)
	to, err := url.Parse(w.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "/auth", to.Path)
	require.Equal(t, "/moods", to.Query().Get("next"))
}

// A POST does not redirect. A form that followed a 303 to the gate would post
// its words into a login screen and lose them.
func TestAPostWithNoSessionIsRefusedRatherThanRedirected(t *testing.T) {
	m := mounted(t, &fakeStore{})
	w := m.callAnonymously(t, "POST", "/capture", strings.NewReader("text=the+boiler"))

	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Empty(t, w.Body.String(), "a refusal described what it was refusing")
}

// And neither does a fragment. thread.js would paste the gate into the
// conversation as a turn.
func TestAFragmentWithNoSessionIsRefusedRatherThanRedirected(t *testing.T) {
	m := mounted(t, &fakeStore{})
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Thread", "fragment")
	w := httptest.NewRecorder()
	m.routes["GET /{$}"](w, r)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Empty(t, w.Header().Get("Location"))
}

// The gate itself is reachable with no session, or there is no way in.
func TestTheGateIsOutsideTheGuard(t *testing.T) {
	m := mounted(t, &fakeStore{})
	require.Equal(t, http.StatusOK, m.callAnonymously(t, "GET", "/auth", nil).Code)
}

func TestAGoodCallbackOpensASessionAndSendsYouOn(t *testing.T) {
	sess := newFakeSessions()
	m, idp := mountedWithAGate(t, &fakeStore{}, sess)
	_ = idp

	w := m.callback(t, "/chores")

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/chores", w.Header().Get("Location"))
	require.Len(t, sess.opened, 1)
	require.Equal(t, "sub-123", sess.opened[0].Sub)
	require.NotNil(t, cookieNamed(w, sessionCookie))
}

// The cookie a browser is given cannot be read by script, cannot travel over
// plain HTTP, and does not go out on a cross-site request.
func TestTheSessionCookieIsLockedDown(t *testing.T) {
	sess := newFakeSessions()
	m, _ := mountedWithAGate(t, &fakeStore{}, sess)

	set := cookieNamed(m.callback(t, "/"), sessionCookie)
	require.NotNil(t, set)

	require.True(t, set.HttpOnly, "script can read the session cookie")
	require.True(t, set.Secure, "the session cookie travels over plain HTTP")
	// Lax rather than Strict: the callback arrives as a top-level navigation
	// from Authentik, and Strict drops the cookie on exactly that hop.
	require.Equal(t, http.SameSiteLaxMode, set.SameSite)
	require.Equal(t, "/", set.Path)
}

// The token in the cookie is not the token in the store. A database dump must
// not be a set of live sessions.
func TestTheCookieIsNotWhatIsStored(t *testing.T) {
	sess := newFakeSessions()
	m, _ := mountedWithAGate(t, &fakeStore{}, sess)

	set := cookieNamed(m.callback(t, "/"), sessionCookie)
	require.NotNil(t, set)

	require.NotEmpty(t, set.Value)
	require.Len(t, sess.tokens, 1)
	require.NotEqual(t, set.Value, string(sess.tokens[0]), "the cookie's own bytes were stored")
	require.Equal(t, hashOf(set.Value), sess.tokens[0], "the stored token is not the cookie's hash")
}

// A callback whose state does not match the cookie is not our redirect.
func TestACallbackWithTheWrongStateIsRefused(t *testing.T) {
	sess := newFakeSessions()
	m, _ := mountedWithAGate(t, &fakeStore{}, sess)

	r := httptest.NewRequest("GET", "/auth/callback?code=a-code&state=somebody-elses", nil)
	r.AddCookie(&http.Cookie{Name: stateCookie, Value: startedValue(t, "the-state", "the-verifier", "/")})
	w := httptest.NewRecorder()
	m.routes["GET /auth/callback"](w, r)

	require.Empty(t, sess.opened, "a session was opened for somebody else's redirect")
	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Contains(t, w.Header().Get("Location"), "said=down")
}

// A callback with no state cookie at all is refused too — a bare
// /auth/callback?code=... typed by anybody.
//
// Defended twice on purpose: the missing cookie is caught, and if it were not,
// there would be no state to compare the query's against. Removing either
// check alone leaves this passing.
func TestACallbackWithNoStateCookieIsRefused(t *testing.T) {
	sess := newFakeSessions()
	m, _ := mountedWithAGate(t, &fakeStore{}, sess)

	r := httptest.NewRequest("GET", "/auth/callback?code=a-code&state=anything", nil)
	w := httptest.NewRecorder()
	m.routes["GET /auth/callback"](w, r)

	require.Empty(t, sess.opened, "a session was opened with no state to check against")
}

// An account Authentik authenticated and Squirrel will not admit lands on the
// gate saying so, and opens nothing.
func TestARefusedAccountLandsOnTheGate(t *testing.T) {
	sess := newFakeSessions()
	m, idp := mountedWithAGate(t, &fakeStore{}, sess)
	idp.claims["groups"] = []any{"somebody-elses-app"}

	w := m.callback(t, "/")

	require.Empty(t, sess.opened, "a refused account got a session")
	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Contains(t, w.Header().Get("Location"), "said=no")
}

// A session that cannot be written is not a login. Failing open here would be
// a person who appears to be signed in and is not.
func TestASessionThatCannotBeWrittenIsNotALogin(t *testing.T) {
	sess := newFakeSessions()
	sess.openErr = context.DeadlineExceeded
	m, _ := mountedWithAGate(t, &fakeStore{}, sess)

	w := m.callback(t, "/")

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Contains(t, w.Header().Get("Location"), "said=down")
	require.Nil(t, cookieNamed(w, sessionCookie),
		"a cookie was set for a session that was never opened")
}

func TestStartingALoginSendsYouToAuthentik(t *testing.T) {
	sess := newFakeSessions()
	m, idp := mountedWithAGate(t, &fakeStore{}, sess)

	r := httptest.NewRequest("POST", "/auth/in", strings.NewReader("next=/chores"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "http://"+r.Host)
	w := httptest.NewRecorder()
	m.routes["POST /auth/in"](w, r)

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Contains(t, w.Header().Get("Location"), idp.URL)
	require.NotNil(t, cookieNamed(w, stateCookie))
}

func TestSigningOutEndsTheSessionAndLandsOnTheGate(t *testing.T) {
	sess := newFakeSessions()
	m, _ := mountedWithAGate(t, &fakeStore{}, sess)
	cookie := m.signIn(t)

	r := httptest.NewRequest("POST", "/auth/out", nil)
	r.Header.Set("Origin", "http://"+r.Host)
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	m.routes["POST /auth/out"](w, r)

	require.Len(t, sess.ended, 1)
	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/auth?said=out", w.Header().Get("Location"))
	cleared := cookieNamed(w, sessionCookie)
	require.NotNil(t, cleared)
	require.Empty(t, cleared.Value, "signing out left the cookie in place")
}

// And signing out twice is not a failure page.
func TestSigningOutWithNoSessionIsFine(t *testing.T) {
	sess := newFakeSessions()
	m, _ := mountedWithAGate(t, &fakeStore{}, sess)

	r := httptest.NewRequest("POST", "/auth/out", nil)
	r.Header.Set("Origin", "http://"+r.Host)
	w := httptest.NewRecorder()
	m.routes["POST /auth/out"](w, r)

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/auth?said=out", w.Header().Get("Location"))
}

func TestASessionIsWhoYouAreForEveryRequestAfterwards(t *testing.T) {
	sess := newFakeSessions()
	m, _ := mountedWithAGate(t, &fakeStore{}, sess)
	cookie := m.signIn(t)

	var saw int64
	m.routes["GET /auth/whoami"] = guard(m.opts, func(_ http.ResponseWriter, r *http.Request) {
		saw, _ = personOf(r)
	})
	r := httptest.NewRequest("GET", "/auth/whoami", nil)
	r.AddCookie(cookie)
	m.routes["GET /auth/whoami"](httptest.NewRecorder(), r)

	require.Equal(t, int64(1), saw)
}

// Mount refuses without a required group. Every other missing value degrades
// to less product; this one degrades to more access.
func TestMountRefusesWithNoRequiredGroup(t *testing.T) {
	// signedInOptions rather than testOptions: a mount refusal has to fail on
	// the thing being tested, and testOptions carries no spool — which Mount
	// refuses first, so the test would pass with the group check deleted.
	opts := signedInOptions()
	opts.RequiredGroup = ""
	require.Error(t, Mount(newTestMux(), &fakeStore{}, opts),
		"the pile mounted with nothing deciding who may read it")
}

// And without a way to read a session, which would make guard refuse every
// request forever.
func TestMountRefusesWithNoSessions(t *testing.T) {
	opts := signedInOptions()
	opts.Sessions = nil
	require.Error(t, Mount(newTestMux(), &fakeStore{}, opts))
}

// mountedWithAGate is the screen with a real way in: a fake Authentik that
// signs real ID tokens, and a session store the test can inspect.
func mountedWithAGate(t *testing.T, f *fakeStore, sess *fakeSessions) (*testMux, *fakeIdP) {
	t.Helper()
	idp := anIdP(t)
	gate := aGate(t, idp, "squirrel-users")

	m := newTestMux()
	opts := signedInOptions()
	opts.Gate = gate
	opts.Sessions = newSessions(sess, cacheFor, cacheMost)
	require.NoError(t, Mount(m, f, opts))
	m.opts = opts
	return m, idp
}

// startedValue packs a login in progress the way beginHandler does.
func startedValue(t *testing.T, state, verifier, next string) string {
	t.Helper()
	return started(state, verifier, next)
}

// callback walks the way back from Authentik with a state that matches.
func (m *testMux) callback(t *testing.T, next string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest("GET", "/auth/callback?code=a-code&state=the-state", nil)
	r.AddCookie(&http.Cookie{Name: stateCookie, Value: started("the-state", "the-verifier", next)})
	w := httptest.NewRecorder()
	m.routes["GET /auth/callback"](w, r)
	return w
}

// signIn walks all the way in and hands back the cookie a browser would hold.
func (m *testMux) signIn(t *testing.T) *http.Cookie {
	t.Helper()
	w := m.callback(t, "/")
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie && c.Value != "" {
			return c
		}
	}
	t.Fatal("signing in set no session cookie")
	return nil
}

// cookieNamed is the one the response set, or nil. Header().Get returns only
// the first Set-Cookie, and the callback sets two.
func cookieNamed(w *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == name && c.Value != "" {
			return c
		}
	}
	// A cleared cookie still counts as one that was set, for the tests about
	// signing out.
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}
