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

func aMoment(in time.Duration, bring string) *squirrel.Moment {
	return &squirrel.Moment{
		ID: 4, Label: "dentist", Starts: now().Add(in),
		Travel: 15 * time.Minute, Ready: 10 * time.Minute, Bring: bring,
	}
}

func TestWhatIsComingListsTheSoonestFirst(t *testing.T) {
	f := &fakeStore{upcoming: []squirrel.Moment{
		{ID: 4, Label: "dentist", Starts: now().Add(2 * time.Hour), Travel: 15 * time.Minute, Ready: 10 * time.Minute},
		{ID: 5, Label: "school run", Starts: now().Add(30 * time.Hour), Travel: 15 * time.Minute, Ready: 10 * time.Minute},
	}}
	body := mounted(t, f).call(t, "GET", "/?bay=agenda", nil).Body.String()

	require.Less(t, strings.Index(body, "dentist"), strings.Index(body, "school run"))
}

func TestWhatIsComingCountsWhatIsAheadAndScoldsNobody(t *testing.T) {
	f := &fakeStore{upcoming: []squirrel.Moment{
		{ID: 4, Label: "dentist", Starts: now().Add(2 * time.Hour), Travel: 15 * time.Minute, Ready: 10 * time.Minute},
		{ID: 5, Label: "school run", Starts: now().Add(30 * time.Hour), Travel: 15 * time.Minute, Ready: 10 * time.Minute},
	}}
	drawn := strings.ToLower(mounted(t, f).call(t, "GET", "/?bay=agenda", nil).Body.String())

	require.Contains(t, drawn, `the agenda <span class="n">2</span>`)
	for _, banned := range []string{"late", "overdue", "you have", "behind"} {
		require.NotContains(t, drawn, banned)
	}
}

func TestNothingComingIsAnAbsenceAndNotAnEncouragement(t *testing.T) {
	body := strings.ToLower(mounted(t, &fakeStore{}).call(t, "GET", "/?bay=agenda", nil).Body.String())

	require.Contains(t, body, "the agenda")
	require.NotContains(t, body, `the agenda <span class="n">`)
	for _, banned := range []string{"plan", "nothing coming", "all clear"} {
		require.NotContains(t, body, banned)
	}
}

func TestTheAgendaRackDrawsWhatIsComing(t *testing.T) {
	m := aMoment(3*time.Hour, "keys, wallet")
	f := withUpcoming(*m)

	shown := mounted(t, f).call(t, "GET", "/?bay=agenda", nil).Body.String()

	require.Contains(t, shown, "dentist")
	require.Contains(t, shown, `class="strip h-agenda`, "it is not drawn as an appointment")
	require.Contains(t, shown, markOfMoment(*m, now()), "the strip does not say when")
}

func TestLeavingIsAbsentOutsideTheWindow(t *testing.T) {
	far := withUpcoming(*aMoment(3*time.Hour, ""))
	farShown := mounted(t, far).call(t, "GET", "/?bay=agenda", nil).Body.String()
	require.NotContains(t, farShown, "leaving")

	near := withUpcoming(*aMoment(20*time.Minute, ""))
	nearShown := mounted(t, near).call(t, "GET", "/?bay=agenda", nil).Body.String()
	require.Contains(t, nearShown, "leaving")
	require.Contains(t, nearShown, "leave "+near.upcoming[0].LeaveAt().Format("15:04"))
}

func TestAnEmptyAgendaSaysSoWithoutEncouraging(t *testing.T) {
	body := strings.ToLower(mounted(t, &fakeStore{}).call(t, "GET", "/?bay=agenda", nil).Body.String())

	require.Contains(t, body, "the agenda")
	require.NotContains(t, body, `the agenda <span class="n">`, "a sign counted what is not there")
	for _, nag := range []string{"why not", "get started", "add your first"} {
		require.NotContains(t, body, nag)
	}
}

func withUpcoming(ms ...squirrel.Moment) *fakeStore {
	return &fakeStore{upcoming: ms}
}

func TestTheNotificationsURLLandsOnTheAgenda(t *testing.T) {
	w := routed(t, withMoment(aMoment(20*time.Minute, "keys, wallet"))).call(t, "GET", "/at/4", nil)

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/?bay=agenda", w.Header().Get("Location"))
}

func TestANotificationForAnythingLandsOnTheAgenda(t *testing.T) {
	w := routed(t, &fakeStore{}).call(t, "GET", "/at/99", nil)

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/?bay=agenda", w.Header().Get("Location"))
}
