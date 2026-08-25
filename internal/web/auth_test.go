package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func testOptions() Options {
	return Options{
		IdentityHeader: "X-Authentik-Username",
		Identity:       "ronald",
		Owner:          func() int64 { return 1 },
	}
}

func TestGuardAllowsTheConfiguredIdentity(t *testing.T) {
	reached := false
	h := guard(testOptions(), func(http.ResponseWriter, *http.Request) { reached = true })

	r := httptest.NewRequest("GET", "/pile", nil)
	r.Header.Set("X-Authentik-Username", "ronald")
	w := httptest.NewRecorder()
	h(w, r)

	require.True(t, reached)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestGuardRefusesEveryoneElse(t *testing.T) {
	for _, name := range []string{"", "someone", "Ronald "} {
		reached := false
		h := guard(testOptions(), func(http.ResponseWriter, *http.Request) { reached = true })

		r := httptest.NewRequest("GET", "/pile", nil)
		if name != "" {
			r.Header.Set("X-Authentik-Username", name)
		}
		w := httptest.NewRecorder()
		h(w, r)

		require.False(t, reached, "handler ran for identity %q", name)
		require.Equal(t, http.StatusForbidden, w.Code, "identity %q", name)
		require.Empty(t, w.Body.String(), "a refusal says nothing about what is behind it")
	}
}

// Who is asking is a fact about the request, not about the process.
//
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
	r.Header.Set("X-Authentik-Username", "ronald")
	h(httptest.NewRecorder(), r)

	require.Equal(t, int64(1), saw, "the guard let a request through with nobody on it")
	require.Equal(t, "ronald", sub)
}

// And a guard that cannot say who is asking refuses rather than passing the
// request along for a handler to discover the same thing.
func TestTheGuardRefusesWhenTheOwnerIsNotKnown(t *testing.T) {
	reached := false
	opts := testOptions()
	opts.Owner = func() int64 { return 0 }
	h := guard(opts, func(http.ResponseWriter, *http.Request) { reached = true })

	r := httptest.NewRequest("GET", "/pile", nil)
	r.Header.Set("X-Authentik-Username", "ronald")
	w := httptest.NewRecorder()
	h(w, r)

	require.False(t, reached, "a request with no owner reached a handler")
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}
