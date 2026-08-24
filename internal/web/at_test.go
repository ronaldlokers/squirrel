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
	return routedSpooling(t, f, &fakeSpool{})
}

// routedSpooling is the same mount with a spool the test can look inside. The
// dock writes there rather than straight to the pile, exactly as /capture
// does, so a test about the dock is a test about what reached the spool.
func routedSpooling(t *testing.T, f *fakeStore, sp *fakeSpool) *realMux {
	t.Helper()
	m := &realMux{mux: http.NewServeMux()}
	require.NoError(t, Mount(m, f, Options{
		IdentityHeader: "X-Authentik-Username", Identity: "ronald",
		Owner: func() int64 { return 1 }, Spool: sp,
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

// callFragment is a press made by the script rather than by the browser's own
// form machinery: same URL, same body, one header.
func (m *realMux) callFragment(t *testing.T, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest("POST", target, strings.NewReader(body))
	r.Header.Set("X-Authentik-Username", "ronald")
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "http://"+r.Host)
	r.Header.Set("X-Thread", "fragment")
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
	require.Contains(t, list, "the agenda")
	require.Contains(t, list, "dentist")

	one := m.call(t, "GET", "/at/4", nil).Body.String()
	require.Contains(t, one, "dentist")
	require.Contains(t, one, "keys, wallet")
}

func TestWhatIsComingListsTheSoonestFirst(t *testing.T) {
	f := &fakeStore{upcoming: []squirrel.Moment{
		{ID: 4, Label: "dentist", Starts: now().Add(2 * time.Hour), Travel: 15 * time.Minute, Ready: 10 * time.Minute},
		{ID: 5, Label: "school run", Starts: now().Add(30 * time.Hour), Travel: 15 * time.Minute, Ready: 10 * time.Minute},
	}}
	body := routed(t, f).call(t, "GET", "/at", nil).Body.String()

	require.Less(t, strings.Index(body, "dentist"), strings.Index(body, "school run"))
	require.Contains(t, body, `href="/at/4"`)
}

// Never a count, and never a word about being behind. Everything here is still
// ahead of you, which is the only reason this list is allowed to exist.
func TestWhatIsComingCountsNothingAndScoldsNobody(t *testing.T) {
	f := &fakeStore{upcoming: []squirrel.Moment{
		{ID: 4, Label: "dentist", Starts: now().Add(2 * time.Hour), Travel: 15 * time.Minute, Ready: 10 * time.Minute},
		{ID: 5, Label: "school run", Starts: now().Add(30 * time.Hour), Travel: 15 * time.Minute, Ready: 10 * time.Minute},
	}}
	body := strings.ToLower(routed(t, f).call(t, "GET", "/at", nil).Body.String())

	for _, banned := range []string{"late", "overdue", "2 coming", "you have"} {
		require.NotContains(t, body, banned)
	}
}

func TestNothingComingIsAnAbsenceAndNotAnEncouragement(t *testing.T) {
	body := routed(t, &fakeStore{}).call(t, "GET", "/at", nil).Body.String()

	require.Contains(t, body, "when something has a time you can be late for")
	require.NotContains(t, strings.ToLower(body), "plan")
}

// The agenda arrives as cards, and each says when to leave in the core's own
// words — so the card, chat and the notification cannot drift apart about it.
func TestOpeningTheAgendaDrawsWhatIsComing(t *testing.T) {
	m := aMoment(3*time.Hour, "keys, wallet")
	f := withUpcoming(*m)
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=at"))

	require.Len(t, f.appended, 2)
	shown := string(f.appended[1].Shown)
	require.Contains(t, shown, "dentist")
	require.Contains(t, shown, `"place":"the agenda"`)
	require.Contains(t, shown, squirrel.LeaveWords(*m))
}

// LEAVING only inside the window. Outside it there is nothing to press: the
// appointment is not yet something you can act on, and a button that closes a
// thing three hours early is one that gets pressed by accident.
func TestLeavingIsAbsentOutsideTheWindow(t *testing.T) {
	far := withUpcoming(*aMoment(3*time.Hour, ""))
	routed(t, far).call(t, "POST", "/open", strings.NewReader("where=at"))
	require.NotContains(t, string(far.appended[1].Shown), "LEAVING")

	near := withUpcoming(*aMoment(20*time.Minute, ""))
	routed(t, near).call(t, "POST", "/open", strings.NewReader("where=at"))
	require.Contains(t, string(near.appended[1].Shown), "LEAVING")
}

// An absence, not an encouragement. Nothing here says you ought to be making
// plans, and nothing counts what is not there.
func TestAnEmptyAgendaSaysSoWithoutEncouraging(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=at"))

	require.Len(t, f.appended, 2)
	require.Contains(t, strings.ToLower(f.appended[1].Words), "when something has a time you can be late for")
	for _, nag := range []string{"why not", "get started", "add your first", "0"} {
		require.NotContains(t, strings.ToLower(f.appended[1].Words), nag)
	}
}

func withUpcoming(ms ...squirrel.Moment) *fakeStore {
	return &fakeStore{upcoming: ms}
}
