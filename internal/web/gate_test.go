package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func atTheGate(t *testing.T, query string) string {
	t.Helper()
	w := httptest.NewRecorder()
	gateHandler()(w, httptest.NewRequest("GET", "/auth"+query, nil))
	require.Equal(t, http.StatusOK, w.Code)
	return w.Body.String()
}

// One screen, four states. They are the same moment arrived at four ways
// rather than four pages nobody drew.
func TestTheGateHasFourStates(t *testing.T) {
	for _, s := range []struct{ query, says, button string }{
		{"", "", "LET ME IN"},
		{"?said=out", "you are signed out", "LET ME IN"},
		{"?said=no", "that account cannot use Squirrel", "LET ME IN"},
		{"?said=down", "I cannot reach the way in just now", "TRY AGAIN"},
	} {
		body := atTheGate(t, s.query)
		require.Contains(t, body, s.button, s.query)
		if s.says != "" {
			require.Contains(t, body, s.says, s.query)
		}
	}
}

// The cold state says nothing. A first arrival is not an error and must not
// read as one.
func TestTheColdGateSaysNothing(t *testing.T) {
	body := atTheGate(t, "")
	for _, said := range []string{"signed out", "cannot use Squirrel", "cannot reach"} {
		require.NotContains(t, body, said, "the first thing anybody sees reads as a failure")
	}
}

// A refusal says the account cannot use Squirrel and never which group it
// lacks. That is a fact about the Authentik, not about them, and naming it
// tells a stranger how to ask for the right thing.
func TestARefusalNamesNoGroup(t *testing.T) {
	body := strings.ToLower(atTheGate(t, "?said=no"))
	require.NotContains(t, body, "squirrel-users")
	require.NotContains(t, body, "group")
}

// Where you were going travels, so signing in puts you back where you were
// rather than at the top of the pile.
func TestTheGateCarriesWhereYouWereGoing(t *testing.T) {
	body := atTheGate(t, "?next="+url.QueryEscape("/chores"))
	require.Contains(t, body, `value="/chores"`)
}

// And only if it is somewhere this server serves. A value that arrives in a
// URL is a place a stranger can type.
func TestTheGateWillNotSendYouSomewhereElse(t *testing.T) {
	for _, bad := range []string{"https://example.com/", "//example.com/", "javascript:alert(1)"} {
		body := atTheGate(t, "?next="+url.QueryEscape(bad))
		require.NotContains(t, body, "example.com", bad)
		require.NotContains(t, body, "javascript:", bad)
	}
}

// An unknown said is the cold state rather than an empty sentence or a crash.
// The value arrives in a URL and anybody can type one.
func TestAnUnknownSaidIsTheColdGate(t *testing.T) {
	require.Contains(t, atTheGate(t, "?said=nonsense"), "LET ME IN")
}

// The way in is a POST, because it writes: it sets a state cookie and a PKCE
// verifier. It also means a prefetch or a crawler cannot begin a login.
func TestTheWayInIsAPost(t *testing.T) {
	body := atTheGate(t, "")
	require.Contains(t, body, `method="post"`)
	require.Contains(t, body, `action="/auth/in"`)
}

// The gate carries none of the app's frame. Every link in the lid goes
// somewhere a signed-out person cannot reach, and offering them is offering a
// bounce back to here.
func TestTheGateCarriesNoMenu(t *testing.T) {
	body := atTheGate(t, "")
	require.NotContains(t, body, `class="lid"`)
	require.NotContains(t, body, `href="/chores"`)
}
