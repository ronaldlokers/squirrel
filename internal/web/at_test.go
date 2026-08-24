package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// These routes carry a wildcard, so they need a real ServeMux.
//
// The shared testMux matches by prefix and does not know what `{id}` is: it
// answered `/at/99` with the handler for `/at`, and `r.PathValue` through it is
// always empty. A test that cannot tell those apart cannot test either.
type realMux struct{ mux *http.ServeMux }

func (m *realMux) Get(pattern string, h http.HandlerFunc)  { m.mux.HandleFunc("GET "+pattern, h) }
func (m *realMux) Post(pattern string, h http.HandlerFunc) { m.mux.HandleFunc("POST "+pattern, h) }

func routed(t *testing.T, f *fakeStore) *realMux {
	t.Helper()
	m := &realMux{mux: http.NewServeMux()}
	require.NoError(t, Mount(m, f, Options{
		IdentityHeader: "X-Authentik-Username", Identity: "ronald",
		Owner: func() int64 { return 1 }, Spool: &fakeSpool{},
	}))
	return m
}

func (m *realMux) call(t *testing.T, method, target string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, target, body)
	r.Header.Set("X-Authentik-Username", "ronald")
	if method == "POST" {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", "http://"+r.Host)
	}
	w := httptest.NewRecorder()
	m.mux.ServeHTTP(w, r)
	return w
}

func aMoment(in time.Duration, bring string) *squirrel.Moment {
	return &squirrel.Moment{
		ID: 4, Label: "dentist", Starts: now().Add(in),
		Travel: 15 * time.Minute, Ready: 10 * time.Minute, Bring: bring,
	}
}

func TestAFixedPointShowsWhenToLeaveAndWhatToTake(t *testing.T) {
	body := routed(t, withMoment(aMoment(3*time.Hour, "keys, wallet"))).
		call(t, "GET", "/at/4", nil).Body.String()

	require.Contains(t, body, "dentist")
	require.Contains(t, body, "keys, wallet")
	require.NotContains(t, body, "LEAVING", "hours out, there is nothing to press")
}

func TestLeavingIsOfferedInsideTheWindow(t *testing.T) {
	body := routed(t, withMoment(aMoment(10*time.Minute, ""))).
		call(t, "GET", "/at/4", nil).Body.String()

	require.Contains(t, body, "LEAVING")
}

func TestTheNotesPointingAtItAreShown(t *testing.T) {
	f := withMoment(aMoment(3*time.Hour, ""))
	f.attached = []squirrel.Item{{ID: 9, RawText: "the referral letter", ReceivedAt: now()}}

	body := routed(t, f).call(t, "GET", "/at/4", nil).Body.String()
	require.Contains(t, body, "the referral letter")
}

// Anything typed here is an ordinary note that happens to point at this
// appointment. No picker, because a picker needs a browsable list of
// appointments to pick from — here the appointment is the page you are on.
func TestTypingIntoAFixedPointKeepsANotePointingAtIt(t *testing.T) {
	f := withMoment(aMoment(3*time.Hour, ""))
	m := routed(t, f)

	form := url.Values{"words": {"the referral letter"}}
	res := m.call(t, "POST", "/at/4/note", strings.NewReader(form.Encode()))

	require.Equal(t, 303, res.Code)
	require.Len(t, f.items, 1)
	require.Equal(t, "the referral letter", f.items[0].RawText)
	require.Equal(t, squirrel.ItemNote, f.items[0].Kind, "it is an ordinary note")
	require.Equal(t, []int64{4}, f.attachedTo, "it points at the appointment it was typed on")
	require.Equal(t, []int64{f.items[0].ID}, f.attachedItems, "and it is that note that was pointed")
}

// Every transition here reverses, and this is the reversal.
func TestANotePutBackLeavesTheFixedPoint(t *testing.T) {
	f := withMoment(aMoment(3*time.Hour, ""))
	f.attached = []squirrel.Item{{ID: 9, RawText: "the referral letter", ReceivedAt: now()}}

	form := url.Values{"id": {"9"}}
	res := routed(t, f).call(t, "POST", "/at/4/detach", strings.NewReader(form.Encode()))

	require.Equal(t, 303, res.Code)
	require.Equal(t, []int64{9}, f.detached)
}

// The hardest rule in the product, on its newest surface.
func TestAFixedPointNeverCountsAnything(t *testing.T) {
	f := withMoment(aMoment(3*time.Hour, ""))
	f.attached = []squirrel.Item{
		{ID: 9, RawText: "one", ReceivedAt: now()},
		{ID: 10, RawText: "two", ReceivedAt: now()},
		{ID: 11, RawText: "three", ReceivedAt: now()},
	}
	body := routed(t, f).call(t, "GET", "/at/4", nil).Body.String()

	require.NotContains(t, body, "3 notes")
	require.NotContains(t, body, "three notes")
}

// A fixed point that is not yours is one that does not exist.
func TestSomebodyElsesFixedPointIsNotFound(t *testing.T) {
	res := routed(t, withMoment(aMoment(3*time.Hour, ""))).call(t, "GET", "/at/99", nil)
	require.Equal(t, 404, res.Code)
}

// The contrast walk cannot tell a clean screen from an empty one, so this says
// the two new paths render something. Both were added to that walk's list, and
// a page with nothing on it would have passed it silently.
func TestBothNewScreensRenderSomething(t *testing.T) {
	f := withMoment(aMoment(3*time.Hour, "keys, wallet"))
	f.upcoming = []squirrel.Moment{*f.moment}
	m := routed(t, f)

	list := m.call(t, "GET", "/at", nil).Body.String()
	require.Contains(t, list, "what is coming")
	require.Contains(t, list, "dentist")

	one := m.call(t, "GET", "/at/4", nil).Body.String()
	require.Contains(t, one, "dentist")
	require.Contains(t, one, "keys, wallet")
}
