package squirrel_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

type writable bool

func (w writable) Writable() bool { return bool(w) }

func listen(t *testing.T, s *squirrel.Server) string {
	t.Helper()
	port, err := s.Listen("127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		require.NoError(t, s.Shutdown(ctx))
	})
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}

func TestServerHealthyWhenSpoolWritable(t *testing.T) {
	base := listen(t, squirrel.NewServer(writable(true)))

	res, err := http.Get(base + "/healthz")
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, "ok", string(body))
}

// Deliberately not a Postgres check. An unready pod stops receiving webhooks,
// and Campfire never retries, so that would turn a survivable outage into
// permanent data loss.
func TestServerUnhealthyOnlyWhenSpoolUnwritable(t *testing.T) {
	base := listen(t, squirrel.NewServer(writable(false)))

	res, err := http.Get(base + "/healthz")
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusServiceUnavailable, res.StatusCode)
}

// http.NotFound sets a Content-Type, and Campfire uploads any non-200 that
// carries one into the room as a file attachment.
func TestServerUnroutedPathSendsNoContentType(t *testing.T) {
	base := listen(t, squirrel.NewServer(writable(true)))

	res, err := http.Post(base+"/transports/campfire/", "application/json", nil)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusNotFound, res.StatusCode)
	require.Empty(t, res.Header.Get("Content-Type"))
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Empty(t, body)
}

func TestServerMountsAPostRoute(t *testing.T) {
	s := squirrel.NewServer(writable(true))
	s.Post("/transports/fake", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(body)
	})
	base := listen(t, s)

	res, err := http.Post(base+"/transports/fake", "text/plain", stringReader("hello"))
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, "hello", string(body))
}

// A handler that writes nothing must produce no Content-Type. This is the
// property the whole silence contract rests on, and it is only observable
// over a real socket.
func TestServerHeaderlessHandlerSendsNoContentType(t *testing.T) {
	s := squirrel.NewServer(writable(true))
	s.Post("/transports/silent", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	base := listen(t, s)

	res, err := http.Post(base+"/transports/silent", "text/plain", nil)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Empty(t, res.Header.Get("Content-Type"))
}

// A panicking handler must not kill the process or emit a body Campfire would
// upload.
func TestServerSurvivesAPanickingHandler(t *testing.T) {
	s := squirrel.NewServer(writable(true))
	s.Post("/transports/boom", func(w http.ResponseWriter, r *http.Request) {
		panic("handler exploded")
	})
	base := listen(t, s)

	res, err := http.Post(base+"/transports/boom", "text/plain", nil)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusInternalServerError, res.StatusCode)
	require.Empty(t, res.Header.Get("Content-Type"))
}

// A handler that sets a Content-Type before panicking must not leak it into
// the recovered 500 — Campfire uploads any non-200 that carries a content
// type into the room as a file attachment.
func TestServerPanicAfterContentTypeSendsNoContentType(t *testing.T) {
	s := squirrel.NewServer(writable(true))
	s.Post("/transports/boom-with-header", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		panic("handler exploded after setting a header")
	})
	base := listen(t, s)

	res, err := http.Post(base+"/transports/boom-with-header", "text/plain", nil)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusInternalServerError, res.StatusCode)
	require.Empty(t, res.Header.Get("Content-Type"))
}

// A registered route answering the wrong method still goes through
// mux.Handler, which returns an empty pattern on a method mismatch just as it
// does for an unrouted path — so the wrapper's no-Content-Type guard covers
// this case too. This header family has produced three defects across two
// builds, so it is worth pinning down explicitly rather than trusting the
// unrouted-path test to imply it.
func TestServerWrongMethodOnARegisteredRouteSendsNoContentType(t *testing.T) {
	s := squirrel.NewServer(writable(true))
	s.Post("/transports/fake", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "hello")
	})
	base := listen(t, s)

	res, err := http.Get(base + "/transports/fake")
	require.NoError(t, err)
	defer res.Body.Close()

	require.Empty(t, res.Header.Get("Content-Type"))
}

func TestServerReportsTheBoundPort(t *testing.T) {
	s := squirrel.NewServer(writable(true))
	port, err := s.Listen("127.0.0.1:0")
	require.NoError(t, err)
	require.Greater(t, port, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, s.Shutdown(ctx))
}

func TestServerRejectsAnOccupiedPort(t *testing.T) {
	first := squirrel.NewServer(writable(true))
	port, err := first.Listen("127.0.0.1:0")
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		first.Shutdown(ctx)
	}()

	second := squirrel.NewServer(writable(true))
	_, err = second.Listen(fmt.Sprintf("127.0.0.1:%d", port))
	require.Error(t, err)
}

func stringReader(s string) io.Reader { return strings.NewReader(s) }
