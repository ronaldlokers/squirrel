package squirrel

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// WritableChecker is the only thing the health endpoint needs from a spool.
type WritableChecker interface {
	Writable() bool
}

type Server struct {
	mux      *http.ServeMux
	spool    WritableChecker
	http     *http.Server
	listener net.Listener
	done     chan struct{}
}

func NewServer(w WritableChecker) *Server {
	s := &Server{mux: http.NewServeMux(), spool: w, done: make(chan struct{})}

	// Liveness and readiness both, and deliberately not a database check. A
	// readiness probe that failed on a Postgres outage would remove this pod
	// from its Service; webhook delivery would then fail, and Campfire does
	// not retry. That converts a survivable outage into permanent data loss.
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		if !s.spool.Writable() {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, "spool unwritable")
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "ok")
	})

	return s
}

// Post registers a transport's route. A transport that polls never calls it.
func (s *Server) Post(pattern string, h http.HandlerFunc) {
	s.mux.HandleFunc("POST "+pattern, h)
}

// handler wraps the mux so that an unmatched route answers with no body and no
// Content-Type. http.NotFound would set one, and Campfire uploads any non-200
// carrying a Content-Type into the room as a file attachment — the same defect
// that got a whole HTTP adapter replaced in the TypeScript build.
//
// The recover does the same job for a panicking handler, which would otherwise
// emit Go's default 500 page into the room.
func (s *Server) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("handler panicked", "panic", rec, "path", r.URL.Path)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		if _, pattern := s.mux.Handler(r); pattern == "" {
			slog.Warn("unrouted request", "method", r.Method, "path", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		s.mux.ServeHTTP(w, r)
	})
}

// Listen binds and starts serving, returning the actual bound port. It fails
// loudly on a bind error rather than hanging.
func (s *Server) Listen(addr string) (int, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return 0, fmt.Errorf("listening on %s: %w", addr, err)
	}
	s.listener = listener
	s.http = &http.Server{
		Handler:           s.handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		defer close(s.done)
		if err := s.http.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server stopped", "error", err)
		}
	}()

	return listener.Addr().(*net.TCPAddr).Port, nil
}

// Shutdown stops accepting and waits for in-flight requests, so a rollout does
// not sever a webhook Campfire will never retry.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.http == nil {
		return nil
	}
	err := s.http.Shutdown(ctx)
	<-s.done
	return err
}
