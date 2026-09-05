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
	require.Equal(t, "/r/everything", w.Header().Get("Location"))

	// Through the spool, not straight to Postgres. Durable before anything
	// says it was kept, which is what the room's captures have always had and
	// this one did not.
	require.Len(t, sp.written, 1)
	require.Equal(t, "ask the garage about the rattle", sp.written[0].Text)
	require.Equal(t, squirrel.ScreenTransport, sp.written[0].Transport)

	// Whose it is, in the transport's own vocabulary — the drain resolves the
	// owner from this, and a capture that resolves to nobody is a note that
	// belongs to no one. It is the session's sub;
	// TestACaptureIsKeptUnderTheSubThatTypedIt is the one that proves it
	// comes from there rather than from anywhere else that says "ronald".
	require.NotNil(t, sp.written[0].SenderID)
	require.Equal(t, "ronald", *sp.written[0].SenderID)
}

// The one word Buddy says back, and it names no place you are behind.
//
// It is a turn now rather than a paragraph inside the slot: the answer lives
// in the conversation, like everything else the screen says.
func TestTheSlotSaysKeptAndNothingElse(t *testing.T) {
	f := &fakeStore{}
	post(t, mountedSpooling(t, f, &fakeSpool{}), "/capture", url.Values{"text": {"a thought"}})

	require.Len(t, f.appended, 2)

	// Against the pool, not the word: the wording is chosen from the day, and
	// "kept" is only one of them. Asserting the literal failed on every day
	// that picked "On the shelf." — the third saying-shaped flake in this
	// suite, after the one that failed after 21:00 and the one that failed
	// between midnight and two.
	require.Contains(t, squirrel.Sayings(squirrel.SayingKept), f.appended[1].Words)

	said := strings.ToLower(f.appended[1].Words)
	for _, total := range []string{"1 note", "added", "in the pile now", "to review"} {
		require.NotContains(t, said, total)
	}
}

func TestAFailedCaptureKeepsTheWords(t *testing.T) {
	f := &fakeStore{}
	m := mountedSpooling(t, f, &fakeSpool{err: errTest})

	w := post(t, m, "/capture", url.Values{"text": {"the boiler makes a noise"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/r/everything", w.Header().Get("Location"))

	// The words are in the conversation and Buddy says they were not kept.
	// The old shape carried them back in the address bar to be re-rendered
	// inside the slot, which needed a home screen to come back to; the promise
	// is the same and the record is where it is kept now.
	require.Len(t, f.appended, 2)
	require.Equal(t, "the boiler makes a noise", f.appended[0].Words,
		"the words are kept rather than disappearing")
	require.Contains(t, f.appended[1].Words, "Not kept")
}

// A thought that reads like a command is still a thought. In a chat room the
// only thing separating the two is the words; the slot has no commands to be
// confused with, so it never interprets. Without the payload marker these
// would be stored and then hidden by the pile's own definition of a note —
// a thought lost silently.
func TestTheSlotNeverReadsAThoughtAsACommand(t *testing.T) {
	for _, text := range []string{"done 2", "!notes", "every day vacuum", "?"} {
		sp := &fakeSpool{}
		w := post(t, mountedSpooling(t, &fakeStore{}, sp), "/capture", url.Values{"text": {text}})

		require.Equal(t, 303, w.Code, text)
		require.Len(t, sp.written, 1, text)
		require.Equal(t, text, sp.written[0].Text, text)
		// Verbatim into the spool, and the drain will not apply it: a capture
		// with no conversation has nowhere to answer into, so nothing runs
		// Match over it. That is what keeps the slot a slot.
		require.Nil(t, sp.written[0].ConversationID, text)
	}
}

func TestAnEmptySlotDoesNothing(t *testing.T) {
	f := &fakeStore{}
	w := post(t, mounted(t, f), "/capture", url.Values{"text": {"   "}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/r/everything", w.Header().Get("Location"))
	require.Empty(t, f.items, "whitespace is not a thought")
}

// What you said comes back on the screen as text, whatever you typed. It comes
// back as a turn, which the thread renders on every load, forever.
func TestTheSlotEscapesWhatItGivesBack(t *testing.T) {
	body := mounted(t, &fakeStore{turns: []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerYou, Words: "<script>alert(1)</script>"},
	}}).call(t, "GET", "/r/everything", nil).Body.String()

	require.NotContains(t, body, "<script>alert(1)</script>")
	require.Contains(t, body, "&lt;script&gt;")
}

// Held is a third state, not a flavour of the other two: the words are safe,
// which failure is not, and they are not in the pile yet, which kept is. It is
// also the one answer that is not a turn, because a turn needs a database and
// this is what happens when there is no network to reach one.
func TestTheSlotSaysWhenWordsAreHeld(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/r/everything?held=1", nil).Body.String()

	require.Contains(t, body, "No network")
	require.Contains(t, body, "goes in when you are back")
	require.NotContains(t, body, "Not kept", "nothing has gone wrong")
}

// signedInAs resolves any cookie to one person with one sub, so a test can
// tell a capture written under the session's sub from one written under
// anything else that happens to say "ronald".
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

// This is the trap the design's section 4 names. The screen does not write to
// Postgres — it spools, with a sender string, and the drain resolves the
// owner from that string rather than from the session. So the sender has to be
// the sub, and a second person's notes land belonging to nobody if it is
// anything else.
//
// The sub travels with the row as well as the person, because a capture keeps
// transport vocabulary the same way the room's do.
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
	r.Header.Set("X-Thread", "fragment")
	w := httptest.NewRecorder()
	m.routes[best](w, r)
	return w
}

func TestACaptureNobodyCanRecordStillSaysSo(t *testing.T) {
	f := &fakeStore{err: errTest}
	m := mountedSpooling(t, f, &fakeSpool{err: errTest})

	w := postFragment(t, m, "/capture", url.Values{"text": {"the boiler makes a noise"}})

	require.Equal(t, 200, w.Code)
	require.NotEmpty(t, w.Body.String(),
		"an empty answer is how the script is told there was nothing to keep")
	require.Contains(t, w.Body.String(), "cannot reach its memory",
		"the refusal has to reach the person even when it cannot be recorded")
	require.Contains(t, w.Body.String(), "the boiler makes a noise",
		"and the words come back with it")
	require.Empty(t, w.Header().Get("X-Kept"),
		"nothing was kept, so the box does not clear")
}

func TestAWriteThatNeverAnswersStillAnswersThePerson(t *testing.T) {
	f := &fakeStore{blockInsert: make(chan struct{})}
	m := mountedSpooling(t, f, &fakeSpool{})

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() { done <- postFragment(t, m, "/capture", url.Values{"text": {"the boiler makes a noise"}}) }()

	select {
	case w := <-done:
		require.Equal(t, 200, w.Code)
		require.Contains(t, w.Body.String(), "cannot reach its memory")
		require.Empty(t, w.Header().Get("X-Kept"))
	case <-time.After(keepingTakesAtMost + 5*time.Second):
		t.Fatal("the write never gave up, so neither did the person")
	}
}
