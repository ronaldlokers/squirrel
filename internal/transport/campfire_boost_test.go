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
	require.Equal(t,
		"http://campfire/rooms/7/3-abc/messages/42/boosts",
		transport.BoostURL("http://campfire", "/rooms/7/3-abc/messages", "42"))

	// A trailing slash on the base must not double up.
	require.Equal(t,
		"http://campfire/rooms/7/3-abc/messages/42/boosts",
		transport.BoostURL("http://campfire/", "/rooms/7/3-abc/messages", "42"))
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
