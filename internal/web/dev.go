//go:build dev

package web

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The screen on a port, with made-up contents.
//
// It exists because everything this product looks like is compiled in:
// templates and static files are both go:embed, so editing pile.css does
// nothing to a running process. That left three things with nowhere to run —
// impeccable's live mode, the design detector's overlay, and any test of the
// service worker by hand, which needs a real origin and a real network to cut.
//
// All of it is behind the `dev` build tag, and that is the safety argument
// rather than a convention: a binary built without the tag does not contain
// EnableDevelopment, so nothing in it can set devDir, and the checks that read
// devDir are simply never true. This file is also where the fakes live,
// because Gate and sessions are package types a cmd/ could not construct.

// EnableDevelopment serves templates and static files from dir — the
// internal/web directory of a checkout — and stops caching either.
func EnableDevelopment(dir string) { devDir = dir }

// DevServe mounts the screen against store and listens. It never returns
// except on error.
func DevServe(addr, webDir string, store Store) error {
	EnableDevelopment(webDir)

	m := &devMux{routes: http.NewServeMux()}
	if err := Mount(m, store, devOptions()); err != nil {
		return fmt.Errorf("mounting the screen: %w", err)
	}
	fmt.Printf("the screen is at http://%s\n", addr)
	fmt.Printf("serving %s — edit a template or the stylesheet and refresh\n", webDir)
	fmt.Println("nothing here is real: no database, no model, no spool")
	return http.ListenAndServe(addr, m.routes)
}

type devMux struct{ routes *http.ServeMux }

func (m *devMux) Get(pattern string, h http.HandlerFunc)  { m.routes.HandleFunc("GET "+pattern, h) }
func (m *devMux) Post(pattern string, h http.HandlerFunc) { m.routes.HandleFunc("POST "+pattern, h) }

// devOptions is enough to satisfy Mount's refusals. Every one of them is a
// real guard against a half-configured deploy; none of them is meaningful here,
// because guard short-circuits before any of it is consulted.
func devOptions() Options {
	return Options{
		RequiredGroup: "dev",
		Location:      time.Local,
		Gate:          &Gate{},
		Sessions:      NewSessions(devSessions{}),
		Login:         func(context.Context, string, string) (int64, error) { return 1, nil },
		Spool:         devSpool{},
	}
}

type devSessions struct{}

func (devSessions) SessionFor(context.Context, []byte, time.Time) (squirrel.Session, bool, error) {
	return squirrel.Session{PersonID: 1, Sub: "dev"}, true, nil
}
func (devSessions) OpenSession(context.Context, int64, string, []byte, time.Time, time.Duration) error {
	return nil
}
func (devSessions) EndSession(context.Context, []byte) error { return nil }

// A spool that accepts and forgets. The screen only asks whether it is
// writable and whether a write succeeded.
type devSpool struct{}

func (devSpool) Write(squirrel.Capture) (string, error) { return "dev", nil }
func (devSpool) Writable() bool                         { return true }
