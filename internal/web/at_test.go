package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

type realMux struct{ mux *http.ServeMux }

func (m *realMux) Get(pattern string, h http.HandlerFunc)  { m.mux.HandleFunc("GET "+pattern, h) }
func (m *realMux) Post(pattern string, h http.HandlerFunc) { m.mux.HandleFunc("POST "+pattern, h) }

func routed(t *testing.T, f *fakeStore) *realMux {
	t.Helper()
	return routedSpooling(t, f, &fakeSpool{})
}

func routedSpooling(t *testing.T, f *fakeStore, sp *fakeSpool) *realMux {
	t.Helper()
	f.kept = sp
	m := &realMux{mux: http.NewServeMux()}
	require.NoError(t, Mount(m, f, Options{
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
		Location: time.Local,
	}))
	return m
}

func (m *realMux) call(t *testing.T, method, target string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, target, body)
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "a-token"})
	if method == "POST" {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", "http://"+r.Host)
	}
	w := httptest.NewRecorder()
	m.mux.ServeHTTP(w, r)
	return w
}

func aMoment(in time.Duration, bring ...string) *squirrel.Moment {
	m := &squirrel.Moment{
		ID: 4, Label: "dentist", Starts: now().Add(in),
		Travel: 15 * time.Minute, Ready: 10 * time.Minute,
	}
	if len(bring) > 0 {
		m.Bring = bring[0]
	}
	return m
}

func TestWhatIsComingListsTheSoonestFirst(t *testing.T) {
	f := &fakeStore{upcoming: []squirrel.Moment{
		{ID: 4, Label: "dentist", Starts: now().Add(2 * time.Hour), Travel: 15 * time.Minute, Ready: 10 * time.Minute},
		{ID: 5, Label: "school run", Starts: now().Add(30 * time.Hour), Travel: 15 * time.Minute, Ready: 10 * time.Minute},
	}}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.Less(t, strings.Index(body, "dentist"), strings.Index(body, "school run"))
}

func TestWhatIsComingCountsWhatIsAheadAndScoldsNobody(t *testing.T) {
	f := &fakeStore{upcoming: []squirrel.Moment{
		{ID: 4, Label: "dentist", Starts: now().Add(2 * time.Hour), Travel: 15 * time.Minute, Ready: 10 * time.Minute},
		{ID: 5, Label: "school run", Starts: now().Add(30 * time.Hour), Travel: 15 * time.Minute, Ready: 10 * time.Minute},
	}}
	drawn := strings.ToLower(mounted(t, f).call(t, "GET", "/", nil).Body.String())

	require.Contains(t, drawn, `what is coming <span class="n">2</span>`)
	for _, banned := range []string{"late", "overdue", "you have", "behind"} {
		require.NotContains(t, drawn, banned)
	}
}

func TestNothingComingIsAnAbsenceAndNotAnEncouragement(t *testing.T) {
	body := strings.ToLower(mounted(t, &fakeStore{}).call(t, "GET", "/", nil).Body.String())

	require.Contains(t, body, "nothing in the diary")
	require.NotContains(t, body, `what is coming <span class="n">`)
	for _, banned := range []string{"plan", "nothing coming", "all clear"} {
		require.NotContains(t, body, banned)
	}
}

// A fixed point is the one real deadline this product holds, so its time is
// set as a printed figure rather than as a mark in the corner of a strip.
func TestTheSidebarDrawsWhatIsComing(t *testing.T) {
	m := aMoment(3*time.Hour, "keys, wallet")
	shown := mounted(t, withUpcoming(*m)).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, shown, "dentist")
	require.Contains(t, shown, `class="attime`, "the time is not set as a figure")
	require.Contains(t, shown, m.Starts.Format("15:04"), "it does not say when")
}

// Inside the window where leaving matters, and only then, it comes out of the
// list and sits above the dial. It leaves the list when it does: a fixed point
// drawn twice in one column is the duplication the picker was cured of.
func TestOnlyWhatIsInsideItsWindowIsHoisted(t *testing.T) {
	far := mounted(t, withUpcoming(*aMoment(3*time.Hour, ""))).call(t, "GET", "/", nil).Body.String()
	require.NotContains(t, far, `class="hoist"`)

	near := withUpcoming(*aMoment(20 * time.Minute))
	shown := mounted(t, near).call(t, "GET", "/", nil).Body.String()
	require.Contains(t, shown, `class="hoist"`, "the world is calling and the column did not move")
	require.Contains(t, shown, "leave "+near.upcoming[0].LeaveAt().Format("15:04"))
	require.Equal(t, 1, strings.Count(shown, `class="atlabel">dentist`),
		"the same appointment is drawn twice in one column")
}

func TestAnEmptyDiarySaysSoWithoutEncouraging(t *testing.T) {
	body := strings.ToLower(mounted(t, &fakeStore{}).call(t, "GET", "/", nil).Body.String())

	require.Contains(t, body, "nothing in the diary")
	require.NotContains(t, body, `what is coming <span class="n">`, "a sign counted what is not there")
	for _, nag := range []string{"why not", "get started", "add your first"} {
		require.NotContains(t, body, nag)
	}
}

func withUpcoming(ms ...squirrel.Moment) *fakeStore {
	return &fakeStore{upcoming: ms}
}

func TestTheNotificationsURLLandsOnTheBoard(t *testing.T) {
	w := routed(t, withMoment(aMoment(20*time.Minute, "keys, wallet"))).call(t, "GET", "/at/4", nil)

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/", w.Header().Get("Location"))
}

func TestANotificationForAnythingLandsOnTheBoard(t *testing.T) {
	w := routed(t, &fakeStore{}).call(t, "GET", "/at/99", nil)

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/", w.Header().Get("Location"))
}
