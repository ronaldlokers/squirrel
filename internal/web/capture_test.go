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

func TestTheSlotKeepsAThought(t *testing.T) {
	sp := &fakeSpool{}
	m := mountedSpooling(t, &fakeStore{}, sp)

	w := post(t, m, "/capture", url.Values{"text": {"ask the garage about the rattle"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/", w.Header().Get("Location"))

	require.Len(t, sp.written, 1)
	require.Equal(t, "ask the garage about the rattle", sp.written[0].Text)
	require.Equal(t, squirrel.ScreenTransport, sp.written[0].Transport)

	require.NotNil(t, sp.written[0].SenderID)
	require.Equal(t, "ronald", *sp.written[0].SenderID)
}

func TestAFailedCaptureFailsVisiblyAndKeepsNothingSilently(t *testing.T) {
	f := &fakeStore{}
	m := mountedSpooling(t, f, &fakeSpool{err: errTest})

	w := post(t, m, "/capture", url.Values{"text": {"the boiler makes a noise"}})

	require.Equal(t, 503, w.Code,
		"a capture that could not be recorded must say so rather than pretend it landed")
}

func TestTheSlotNeverReadsAThoughtAsACommand(t *testing.T) {
	for _, text := range []string{"done 2", "!notes", "every day vacuum", "?"} {
		sp := &fakeSpool{}
		w := post(t, mountedSpooling(t, &fakeStore{}, sp), "/capture", url.Values{"text": {text}})

		require.Equal(t, 303, w.Code, text)
		require.Len(t, sp.written, 1, text)
		require.Equal(t, text, sp.written[0].Text, text)
		require.Nil(t, sp.written[0].ConversationID, text)
	}
}

func TestAnEmptySlotDoesNothing(t *testing.T) {
	f := &fakeStore{}
	w := post(t, mounted(t, f), "/capture", url.Values{"text": {"   "}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/", w.Header().Get("Location"))
	require.Empty(t, f.items, "whitespace is not a thought")
}

type signedInAs struct {
	personID int64
	sub      string
}

func (s signedInAs) SessionFor(context.Context, []byte, time.Time) (squirrel.Session, bool, error) {
	return squirrel.Session{
		PersonID: s.personID, Sub: s.sub, ExpiresAt: time.Now().Add(time.Hour),
	}, true, nil
}

func (signedInAs) OpenSession(context.Context, int64, string, []byte, time.Time, time.Duration) error {
	return nil
}
func (signedInAs) EndSession(context.Context, []byte) error { return nil }

func TestACaptureIsKeptUnderTheSubThatTypedIt(t *testing.T) {
	f, sp := &fakeStore{}, &fakeSpool{}
	f.kept = sp
	m := newTestMux()
	opts := signedInOptions()
	opts.Sessions = newSessions(signedInAs{personID: 7, sub: "sub-seven"}, cacheFor, cacheMost)
	require.NoError(t, Mount(m, f, opts))

	m.call(t, "POST", "/capture", strings.NewReader("text=the+boiler"))

	require.Len(t, sp.written, 1)
	require.NotNil(t, sp.written[0].SenderID)
	require.Equal(t, "sub-seven", *sp.written[0].SenderID,
		"the capture was kept under somebody else's name")
}

func postFragment(t *testing.T, m *testMux, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	best := m.route(t, "POST", path)
	r := httptest.NewRequest("POST", path, strings.NewReader(form.Encode()))
	setPathValues(r, best, path)
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "a-token"})
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "http://"+r.Host)
	w := httptest.NewRecorder()
	m.routes[best](w, r)
	return w
}

func TestACaptureNobodyCanRecordStillSaysSo(t *testing.T) {
	f := &fakeStore{err: errTest}
	m := mountedSpooling(t, f, &fakeSpool{err: errTest})

	w := postFragment(t, m, "/capture", url.Values{"text": {"the boiler makes a noise"}})

	require.Equal(t, 503, w.Code)
	require.Contains(t, w.Body.String(), "cannot reach its memory")
}

func TestAWriteThatNeverAnswersStillAnswersThePerson(t *testing.T) {
	f := &fakeStore{blockInsert: make(chan struct{})}
	m := mountedSpooling(t, f, &fakeSpool{})

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() { done <- postFragment(t, m, "/capture", url.Values{"text": {"the boiler makes a noise"}}) }()

	select {
	case w := <-done:
		require.Equal(t, 503, w.Code)
		require.Contains(t, w.Body.String(), "cannot reach its memory")
	case <-time.After(keepingTakesAtMost + 5*time.Second):
		t.Fatal("the write never gave up, so neither did the person")
	}
}
