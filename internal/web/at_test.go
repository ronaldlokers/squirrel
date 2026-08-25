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
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
		Spool:    sp,
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

// callFragment is a press made by the script rather than by the browser's own
// form machinery: same URL, same body, one header.
func (m *realMux) callFragment(t *testing.T, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest("POST", target, strings.NewReader(body))
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "a-token"})
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
	body := landsInTheThread(t, withMoment(aMoment(3*time.Hour, "keys, wallet")))

	require.Contains(t, body, "dentist")
	require.Contains(t, body, "keys, wallet")
	require.NotContains(t, body, "LEAVING", "hours out, there is nothing to press")
}

func TestLeavingIsOfferedInsideTheWindow(t *testing.T) {
	body := landsInTheThread(t, withMoment(aMoment(10*time.Minute, "")))

	require.Contains(t, body, "LEAVING")
}

func TestTheNotesPointingAtItAreShown(t *testing.T) {
	f := withMoment(aMoment(3*time.Hour, ""))
	f.attached = []squirrel.Item{{ID: 9, RawText: "the referral letter", ReceivedAt: now()}}

	body := landsInTheThread(t, f)
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
	body := landsInTheThread(t, f)

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
// The one screen the agenda still has. The list became a turn on 24 August
// 2026; this is the page a notification lands on, and it stays until phase 4.
func TestTappingTheWarningOpensTheAppointment(t *testing.T) {
	f := withMoment(aMoment(3*time.Hour, "keys, wallet"))
	one := landsInTheThread(t, f)
	require.Contains(t, one, "dentist")
	require.Contains(t, one, "keys, wallet")
}

func TestWhatIsComingListsTheSoonestFirst(t *testing.T) {
	f := &fakeStore{upcoming: []squirrel.Moment{
		{ID: 4, Label: "dentist", Starts: now().Add(2 * time.Hour), Travel: 15 * time.Minute, Ready: 10 * time.Minute},
		{ID: 5, Label: "school run", Starts: now().Add(30 * time.Hour), Travel: 15 * time.Minute, Ready: 10 * time.Minute},
	}}
	// Soonest first, in the turn the door draws now.
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=at"))
	body := string(f.appended[1].Shown)

	require.Less(t, strings.Index(body, "dentist"), strings.Index(body, "school run"))
}

// Never a count, and never a word about being behind. Everything here is still
// ahead of you, which is the only reason this list is allowed to exist.
func TestWhatIsComingCountsWhatIsAheadAndScoldsNobody(t *testing.T) {
	f := &fakeStore{upcoming: []squirrel.Moment{
		{ID: 4, Label: "dentist", Starts: now().Add(2 * time.Hour), Travel: 15 * time.Minute, Ready: 10 * time.Minute},
		{ID: 5, Label: "school run", Starts: now().Add(30 * time.Hour), Travel: 15 * time.Minute, Ready: 10 * time.Minute},
	}}
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=at"))
	said := strings.ToLower(f.appended[1].Words)
	drawn := strings.ToLower(string(f.appended[1].Shown))

	// Buddy counts what is ahead — permitted since 24 August 2026 — and the
	// counting is the only number here.
	require.Contains(t, said, "2 things have a time")
	for _, banned := range []string{"late", "overdue", "you have", "behind"} {
		require.NotContains(t, said, banned)
		require.NotContains(t, drawn, banned)
	}
}

func TestNothingComingIsAnAbsenceAndNotAnEncouragement(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/open", strings.NewReader("where=at"))
	body := strings.ToLower(f.appended[1].Words)

	require.Contains(t, body, "when something has a time you can be late for")
	require.NotContains(t, body, "plan")
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

// Opening one draws it with what to take and the notes pointing at it.
func TestOpeningAFixedPointDrawsItsNotes(t *testing.T) {
	f := withMoment(aMoment(3*time.Hour, "keys, wallet"))
	f.attached = []squirrel.Item{
		{ID: 7, RawText: "the referral letter", State: squirrel.ItemOpen},
	}
	routed(t, f).call(t, "POST", "/at/open", strings.NewReader("id=4"))

	require.Len(t, f.appended, 2)
	shown := string(f.appended[1].Shown)
	require.Contains(t, shown, "dentist")
	require.Contains(t, shown, "take keys, wallet")
	require.Contains(t, shown, "the referral letter")
}

// A fixed point that is not yours draws nothing and says nothing.
func TestAFixedPointThatIsNotYoursDrawsNothing(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/at/open", strings.NewReader("id=99"))

	require.Empty(t, f.appended)
}

// A note goes back to the pile, and the going back is said.
func TestDetachingANoteIsSaid(t *testing.T) {
	f := withMoment(aMoment(3*time.Hour, ""))
	f.attached = []squirrel.Item{{ID: 7, RawText: "the referral letter", State: squirrel.ItemOpen}}
	routed(t, f).call(t, "POST", "/at/4/detach", strings.NewReader("id=7"))

	require.Equal(t, []int64{7}, f.detached)
	require.Len(t, f.appended, 2)
	require.Contains(t, f.appended[1].Words, "pile")
}

// The question offers a month to pick a day out of, and a time.
func TestAskingForADayOffersAMonthAndTimes(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/at/new", strings.NewReader("label=dentist"))

	require.Len(t, f.appended, 2)
	shown := string(f.appended[1].Shown)
	require.Contains(t, shown, `"month"`)
	require.Contains(t, shown, `"times"`)
	require.Contains(t, shown, "14:30")
}

// Answering makes the appointment on the day that was chosen, through the same
// parser a typed sentence goes through.
func TestAnsweringMakesItOnThatDay(t *testing.T) {
	f := &fakeStore{}
	day := now().AddDate(0, 0, 3)
	routed(t, f).call(t, "POST", "/at/make", strings.NewReader(
		"label=dentist&day="+day.Format("2006-01-02")+"&at=14:30"))

	require.Len(t, f.moments, 1)
	require.Equal(t, "dentist", f.moments[0].Label)
	require.Equal(t, day.Day(), f.moments[0].Starts.Day())
	require.Equal(t, 14, f.moments[0].Starts.Hour())
	require.Equal(t, 30, f.moments[0].Starts.Minute())
}

// The picker and a typed sentence agree about what a time is. Asserted on the
// hour and minute, not on a string.
func TestThePickerAndTheSentenceAgreeAboutTheTime(t *testing.T) {
	typed, ok := squirrel.ParseMoment("at 14:30 dentist", now())
	require.True(t, ok)

	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/at/make", strings.NewReader(
		"label=dentist&day="+typed.Starts.Format("2006-01-02")+"&at=14:30"))

	require.Len(t, f.moments, 1)
	require.Equal(t, typed.Starts, f.moments[0].Starts)
}

// A time nobody offered does nothing, and the time is a real one: 25:99 proves
// nothing, because the parser refuses that on its own. 03:00 is a time this
// picker does not draw, and pressing it is something only a hand-made post can
// do — which is exactly what arriving from a form means.
func TestATimeThatWasNeverOfferedDoesNothing(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/at/make", strings.NewReader(
		"label=dentist&day="+now().AddDate(0, 0, 1).Format("2006-01-02")+"&at=03:00"))

	require.Empty(t, f.moments)
	require.Empty(t, f.appended)
}

// And a day in the past does nothing either: the picker offers none, and an
// appointment you are already late for is the one thing this list may not hold.
func TestADayInThePastDoesNothing(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/at/make", strings.NewReader(
		"label=dentist&day="+now().AddDate(0, 0, -2).Format("2006-01-02")+"&at=14:30"))

	require.Empty(t, f.moments)
}

// Turning to another month asks again rather than writing an appointment: it
// is not an answer, it is turning a page.
func TestTurningTheMonthAsksAgainAndMakesNothing(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/at/new", strings.NewReader(
		"label=dentist&month="+now().AddDate(0, 1, 0).Format("2006-01")))

	require.Empty(t, f.moments)
	require.Len(t, f.appended, 1, "turning a page is not something you said")
	require.Contains(t, string(f.appended[0].Shown), `"month"`)
}

// routedSplitting is a real mux with a coach that will split, for the routes
// that carry a wildcard or a form the shared testMux cannot post.
func routedSplitting(t *testing.T, f *fakeStore, pieces ...string) *realMux {
	t.Helper()
	c := &fakeCoach{pieces: pieces, splittable: true}
	m := &realMux{mux: http.NewServeMux()}
	require.NoError(t, Mount(m, f, c.options(Options{
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
		Spool:    &fakeSpool{},
	})))
	return m
}

// The notification's own URL keeps working, and lands in the conversation.
//
// One sent last week is still on a lock screen, so the link may never 404 — and
// what it opens is the appointment at the live edge rather than a page of its
// own.
func TestTheNotificationsURLLandsInTheConversation(t *testing.T) {
	f := withMoment(aMoment(20*time.Minute, "keys, wallet"))
	w := routed(t, f).call(t, "GET", "/at/4", nil)

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/", w.Header().Get("Location"))
	require.Len(t, f.appended, 2)
	require.Contains(t, string(f.appended[1].Shown), "take keys, wallet")
	require.Contains(t, string(f.appended[1].Shown), "LEAVING")
}

// A fixed point that is not yours is not written into anyone's conversation.
func TestANotificationForSomethingThatIsNotYoursWritesNothing(t *testing.T) {
	f := &fakeStore{}
	w := routed(t, f).call(t, "GET", "/at/99", nil)

	require.Equal(t, 404, w.Code)
	require.Empty(t, f.appended)
}

// landsInTheThread taps the notification's URL and renders the conversation it
// wrote — which is what a person tapping a leave-by warning actually gets.
func landsInTheThread(t *testing.T, f *fakeStore) string {
	t.Helper()
	// A fresh reading, so Buddy does not ask how you are on arrival and become
	// the live edge himself — which would take LEAVING off the appointment you
	// just tapped a warning about.
	if f.checkin == nil {
		f.checkin = &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: now()}
	}
	m := routed(t, f)
	m.call(t, "GET", "/at/4", nil)
	f.turns, f.appended = append(f.turns, f.appended...), nil
	return m.call(t, "GET", "/", nil).Body.String()
}
