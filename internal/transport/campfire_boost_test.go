package transport_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/ronaldlokers/squirrel/internal/transport"
)

func TestBoostURL(t *testing.T) {
	got, ok := transport.BoostURL("http://campfire", "/rooms/7/3-abc/messages", "42")
	require.True(t, ok)
	require.Equal(t, "http://campfire/rooms/7/3-abc/messages/42/boosts", got)

	// A trailing slash on the base must not double up.
	got, ok = transport.BoostURL("http://campfire/", "/rooms/7/3-abc/messages", "42")
	require.True(t, ok)
	require.Equal(t, "http://campfire/rooms/7/3-abc/messages/42/boosts", got)
}

// A room path rides in on the webhook payload, not configuration — treat it
// as attacker-controlled. The bare "@" case is the sharpest: composed against
// a real base URL it moves the host, not just the path.
func TestBoostURLRejectsUnsafeRoomPaths(t *testing.T) {
	cases := map[string]string{
		"userinfo @ attack, moves the host": "@evil.com/rooms",
		"protocol-relative //":              "//evil.com/rooms",
		"traversal segment":                 "/rooms/../../../admin",
		"query string":                      "/rooms/7/key/messages?x=1",
		"no leading slash":                  "rooms/7/key/messages",
		"embedded whitespace":               "/rooms/7/key/mess ages",
	}

	for name, roomPath := range cases {
		t.Run(name, func(t *testing.T) {
			_, ok := transport.BoostURL("http://campfire.internal", roomPath, "42")
			require.False(t, ok, "roomPath %q must be rejected", roomPath)
		})
	}
}

// The specific attack the userinfo check exists for: composed naively, this
// pair produces a URL whose host is evil.com and whose userinfo is the real
// host — a different destination entirely.
func TestBoostURLCatchesTheUserinfoHostSwap(t *testing.T) {
	_, ok := transport.BoostURL("http://campfire.campfire.svc.cluster.local", "@evil.com/rooms", "42")
	require.False(t, ok)
}

type boostRecorder struct {
	mu   sync.Mutex
	got  []string
	body []string
}

func (r *boostRecorder) add(path, body string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.got = append(r.got, path)
	r.body = append(r.body, body)
}

func (r *boostRecorder) paths() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.got...)
}

func boostStub(t *testing.T, status int) (string, *boostRecorder) {
	t.Helper()
	rec := &boostRecorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rec.add(r.URL.Path, string(body))
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, rec
}

// The common path: no message posted, a squirrel reacted onto the message.
func TestStoredBoostsInsteadOfReplying(t *testing.T) {
	base, rec := boostStub(t, http.StatusCreated)

	mount := &oneMount{}
	sink := &recordingSink{outcome: squirrel.Stored}
	cfg := config()
	cfg.BaseURL = base

	_, err := transport.NewCampfire(cfg).Start(context.Background(), sink, mount)
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	mount.h(rw, httptest.NewRequest(http.MethodPost, "/transports/campfire", strings.NewReader(payload)))

	require.Equal(t, http.StatusOK, rw.Code)
	require.Empty(t, rw.Header().Get("Content-Type"), "no message is posted")
	require.Empty(t, rw.Body.String())

	require.Eventually(t, func() bool { return len(rec.paths()) == 1 },
		2*time.Second, 20*time.Millisecond)
	require.Equal(t, "/rooms/7/3-abc/messages/42/boosts", rec.paths()[0])
	require.Equal(t, "🐿️", rec.body[0])
}

// Failure is the one case that must not be a quiet reaction.
func TestFailedStillPostsAMessage(t *testing.T) {
	base, rec := boostStub(t, http.StatusCreated)

	mount := &oneMount{}
	sink := &recordingSink{outcome: squirrel.Failed}
	cfg := config()
	cfg.BaseURL = base

	_, err := transport.NewCampfire(cfg).Start(context.Background(), sink, mount)
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	mount.h(rw, httptest.NewRequest(http.MethodPost, "/transports/campfire", strings.NewReader(payload)))

	require.Equal(t, http.StatusOK, rw.Code)
	require.Contains(t, rw.Body.String(), "resend")
	require.Empty(t, rec.paths(), "no boost on failure")
}

func TestIgnoredBoostsNothing(t *testing.T) {
	base, rec := boostStub(t, http.StatusCreated)

	mount := &oneMount{}
	sink := &recordingSink{outcome: squirrel.Ignored}
	cfg := config()
	cfg.BaseURL = base

	_, err := transport.NewCampfire(cfg).Start(context.Background(), sink, mount)
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	mount.h(rw, httptest.NewRequest(http.MethodPost, "/transports/campfire", strings.NewReader(payload)))

	require.Empty(t, rw.Header().Get("Content-Type"))
	require.Empty(t, rec.paths())
}

// A boost that never succeeds costs nothing: the capture is already durable and
// the daily digest is the backstop.
func TestBoostFailureDoesNotAffectTheResponse(t *testing.T) {
	base, _ := boostStub(t, http.StatusInternalServerError)

	mount := &oneMount{}
	sink := &recordingSink{outcome: squirrel.Stored}
	cfg := config()
	cfg.BaseURL = base

	_, err := transport.NewCampfire(cfg).Start(context.Background(), sink, mount)
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	mount.h(rw, httptest.NewRequest(http.MethodPost, "/transports/campfire", strings.NewReader(payload)))

	require.Equal(t, http.StatusOK, rw.Code)
	require.Empty(t, rw.Header().Get("Content-Type"))
	require.Len(t, sink.seen, 1, "the capture still happened")
}

// Fail-open reaches the boost too: an envelope we could not read has no path
// and no message id, so there is no receipt and still no dropped capture.
func TestNoBoostWithoutARoomPath(t *testing.T) {
	base, rec := boostStub(t, http.StatusCreated)

	mount := &oneMount{}
	sink := &recordingSink{outcome: squirrel.Stored}
	cfg := config()
	cfg.BaseURL = base

	_, err := transport.NewCampfire(cfg).Start(context.Background(), sink, mount)
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	mount.h(rw, httptest.NewRequest(http.MethodPost, "/transports/campfire", strings.NewReader("not json")))

	require.Equal(t, http.StatusOK, rw.Code)
	require.Len(t, sink.seen, 1)
	require.Empty(t, rec.paths())
}

// Without a base url there is nowhere to send a boost, and that must degrade
// to phase-1 behaviour rather than to a crash.
func TestNoBoostWithoutABaseURL(t *testing.T) {
	mount := &oneMount{}
	sink := &recordingSink{outcome: squirrel.Stored}

	_, err := transport.NewCampfire(config()).Start(context.Background(), sink, mount)
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	require.NotPanics(t, func() {
		mount.h(rw, httptest.NewRequest(http.MethodPost, "/transports/campfire", strings.NewReader(payload)))
	})
	require.Equal(t, http.StatusOK, rw.Code)
}

// A room.path crafted to move the host must still be treated as fail-open,
// exactly like a missing room.path: the capture stores, the response is
// unaffected, and only the boost is refused. BoostURL's rejection happens
// synchronously — before the retry goroutine is ever spawned — so there is
// nothing to wait for here; if that ever changed to a race, this assertion
// would need require.Eventually instead of an immediate check.
func TestStoredWithAnUnsafeRoomPathBoostsNothing(t *testing.T) {
	base, rec := boostStub(t, http.StatusCreated)

	mount := &oneMount{}
	sink := &recordingSink{outcome: squirrel.Stored}
	cfg := config()
	cfg.BaseURL = base

	_, err := transport.NewCampfire(cfg).Start(context.Background(), sink, mount)
	require.NoError(t, err)

	malicious := strings.Replace(payload,
		`"path": "/rooms/7/3-abc/messages"`, `"path": "@evil.com/rooms"`, 1)
	require.NotEqual(t, payload, malicious, "the replacement must actually apply")

	rw := httptest.NewRecorder()
	mount.h(rw, httptest.NewRequest(http.MethodPost, "/transports/campfire", strings.NewReader(malicious)))

	require.Equal(t, http.StatusOK, rw.Code)
	require.Empty(t, rw.Header().Get("Content-Type"))
	require.Empty(t, rw.Body.String())
	require.Len(t, sink.seen, 1, "the capture still stored")
	require.Empty(t, rec.paths(), "no boost anywhere for an unsafe room path")
}
