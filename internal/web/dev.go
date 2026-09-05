//go:build dev

package web

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
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
	watch := watching(webDir)
	m.routes.HandleFunc("GET /dev/redraw", watch.serve)

	fmt.Printf("the screen is at http://%s\n", addr)
	fmt.Printf("serving %s — edit a template or the stylesheet and the screen redraws\n", webDir)
	fmt.Println("nothing here is real: no database, no model, no spool")
	fmt.Printf("one rack at a time: http://%s/?only=chores\n", addr)
	return http.ListenAndServe(addr, redrawing(m.routes))
}

const redrawScript = `<script>
(function () {
  var source = new EventSource("/dev/redraw");
  source.onmessage = function () { location.reload(); };
})();
</script>`

func redrawing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/dev/redraw" {
			next.ServeHTTP(w, r)
			return
		}
		held := &heldPage{ResponseWriter: w, body: &bytes.Buffer{}}
		next.ServeHTTP(held, r)
		body := held.body.Bytes()
		if strings.HasPrefix(held.Header().Get("Content-Type"), "text/html") {
			if at := bytes.LastIndex(body, []byte("</body>")); at >= 0 {
				body = append(append(append([]byte{}, body[:at]...), redrawScript...), body[at:]...)
			}
		}
		held.Header().Del("Content-Length")
		if held.code == 0 {
			held.code = http.StatusOK
		}
		w.WriteHeader(held.code)
		_, _ = w.Write(body)
	})
}

type heldPage struct {
	http.ResponseWriter
	body *bytes.Buffer
	code int
}

func (h *heldPage) WriteHeader(code int) { h.code = code }

func (h *heldPage) Write(b []byte) (int, error) {
	if h.code == 0 {
		h.code = http.StatusOK
	}
	return h.body.Write(b)
}

type watcher struct {
	dir     string
	mu      sync.Mutex
	waiting map[chan struct{}]struct{}
}

func watching(dir string) *watcher {
	w := &watcher{dir: dir, waiting: map[chan struct{}]struct{}{}}
	go w.look()
	return w
}

func (w *watcher) look() {
	was := w.stamps()
	for range time.Tick(250 * time.Millisecond) {
		now := w.stamps()
		if now == was {
			continue
		}
		was = now
		w.mu.Lock()
		for c := range w.waiting {
			close(c)
			delete(w.waiting, c)
		}
		w.mu.Unlock()
	}
}

func (w *watcher) stamps() string {
	var b strings.Builder
	_ = filepath.WalkDir(w.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		switch filepath.Ext(path) {
		case ".html", ".css", ".js":
		default:
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		fmt.Fprintf(&b, "%s:%d:%d\n", path, info.ModTime().UnixNano(), info.Size())
		return nil
	})
	return b.String()
}

func (w *watcher) serve(rw http.ResponseWriter, r *http.Request) {
	flusher, ok := rw.(http.Flusher)
	if !ok {
		http.Error(rw, "no flushing here", http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-store")
	rw.WriteHeader(http.StatusOK)
	flusher.Flush()

	changed := make(chan struct{})
	w.mu.Lock()
	w.waiting[changed] = struct{}{}
	w.mu.Unlock()

	select {
	case <-changed:
		fmt.Fprint(rw, "data: redraw\n\n")
		flusher.Flush()
	case <-r.Context().Done():
		w.mu.Lock()
		delete(w.waiting, changed)
		w.mu.Unlock()
	}
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
		Ask:           devAsk,
	}
}

func devAsk(_ context.Context, _ int64, _, _, _, _ string) (Answer, error) {
	return Answer{Text: "This has waited three days already. A short call would close it."}, nil
}

type devSessions struct{}

func (devSessions) SessionFor(context.Context, []byte, time.Time) (squirrel.Session, bool, error) {
	return squirrel.Session{PersonID: 1, Sub: "dev"}, true, nil
}
func (devSessions) OpenSession(context.Context, int64, string, []byte, time.Time, time.Duration) error {
	return nil
}
func (devSessions) EndSession(context.Context, []byte) error { return nil }
