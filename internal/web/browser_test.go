//go:build browser

// These tests run the real pile.js in a real browser, because everything it
// does is invisible to Go: the stamp, the interval question, the keys, and
// search that answers as you type. Two of the three defects this screen has
// shipped were in that file, and both were found by opening a browser by hand.
//
// They are behind a build tag because they need a browser on the machine, and
// `make test` must keep needing nothing at all. Run them with
// `make test-browser`; CI runs them on every push.
package web

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"time"
)

// browserBinary finds something to drive. GitHub's runners ship Chrome; this
// machine has chromium; both answer to the same flags.
func browserBinary(t *testing.T) string {
	t.Helper()
	if set := os.Getenv("BROWSER"); set != "" {
		return set
	}
	for _, name := range []string{"google-chrome", "chromium", "chromium-browser", "google-chrome-stable"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	t.Skip("no browser found; set BROWSER to run these")
	return ""
}

// serveMux adapts http.ServeMux to what Mount wants, so these tests exercise
// the same routing the real server does rather than a map of handlers.
type serveMux struct{ mux *http.ServeMux }

func (m *serveMux) Get(pattern string, h http.HandlerFunc)  { m.mux.HandleFunc("GET "+pattern, h) }
func (m *serveMux) Post(pattern string, h http.HandlerFunc) { m.mux.HandleFunc("POST "+pattern, h) }

// screen stands the whole thing up over a real socket, signed in the way the
// session middleware signs a request in. No Postgres: what is under test is the
// script, and the fake store is the same one the rest of this package's tests
// use.
func screen(t *testing.T, f *fakeStore) *httptest.Server {
	return screenWith(t, f, nil)
}

// screenWith is screen plus a coach, for the tests that need one behind the
// sheet. Without it the sheet already shows "which of these is it", which
// makes a test looking for an answer pass before anything has been sent.
func screenWith(t *testing.T, f *fakeStore, c *fakeCoach) *httptest.Server {
	t.Helper()

	opts := Options{
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
		Location: time.Local,
	}
	if c != nil {
		opts = c.options(opts)
	}

	m := &serveMux{mux: http.NewServeMux()}
	require.NoError(t, Mount(m, f, opts))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "a-token"})
		if r.Method == http.MethodPost && r.Header.Get("Origin") == "" {
			r.Header.Set("Origin", "http://"+r.Host)
		}
		m.mux.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// open starts a browser, points it at the pile, and hands back something to
// drive it with.
func open(t *testing.T, f *fakeStore) (*cdp, *httptest.Server) {
	return openWith(t, f, nil)
}

func openWith(t *testing.T, f *fakeStore, coach *fakeCoach) (*cdp, *httptest.Server) {
	t.Helper()

	srv := screenWith(t, f, coach)
	return browserAt(t, srv, "/r/everything"), srv
}

// browserAt is the browser half on its own, for the tests that stand a screen
// up themselves — the camera needs somewhere to put a photograph, which is not
// something every test wants to have to say.
func browserAt(t *testing.T, srv *httptest.Server, path string) *cdp {
	t.Helper()

	port := freePort(t)

	// Not t.TempDir: a browser writes to its profile until the moment it dies,
	// and Go's cleanup runs first and fails the test for a directory that was
	// merely still in use. This one is removed after the process is gone, and
	// a leftover in /tmp is not worth failing a test over.
	profile, err := os.MkdirTemp("", "squirrel-browser-")
	require.NoError(t, err)

	cmd := exec.Command(browserBinary(t),
		"--headless", "--disable-gpu", "--no-sandbox",
		"--no-first-run", "--no-default-browser-check",
		"--disable-features=Translate",
		// A CI container's /dev/shm is 64MB and Chrome will die on it rather
		// than say so. Everything else here is a background service that has
		// nowhere to phone home to on a runner.
		"--disable-dev-shm-usage", "--disable-extensions",
		"--disable-background-networking", "--disable-sync",
		"--user-data-dir="+profile,
		fmt.Sprintf("--remote-debugging-port=%d", port),
		"about:blank")

	// Kept so that a browser which dies on startup can say why. Without this
	// the only symptom is a port that never opens, which is the least
	// informative failure a test can have.
	var said bytes.Buffer
	cmd.Stdout, cmd.Stderr = &said, &said
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		_ = os.RemoveAll(profile)
	})

	c := dialCDP(t, port, &said)
	c.send(t, "Page.enable", nil)
	c.send(t, "Runtime.enable", nil)
	c.navigate(t, srv.URL+path)
	return c
}

func aPile() *fakeStore {
	return &fakeStore{items: []squirrel.Item{
		note(3, "the boiler makes a noise", squirrel.ItemOpen),
		note(2, "meter reading 48213", squirrel.ItemOpen),
		note(1, "ask about the bins", squirrel.ItemOpen),
	}}
}

// The chores rack's keys are the board's keys, and they are proved in
// TestBrowserTheBoardsKeysFollowFocus: letters act on the strip you are focused
// in, arrows move between strips, and a letter nothing answers to does nothing.
// The where this exercised is a rack since 2 September 2026, and a rack's
// selection model is the platform's own focus, which is what the where used too.

// Five answers, five equal drawings. They were cut from one sheet at one
// scale, and each carries a different amount of decoration outside its tile —
// sparks, leaves, a scribble, a zzz — so sizing them by width would make the
// busiest one the smallest. Sized by height, they read as five answers rather
// than as one louder than the others.
func TestBrowserTheFacesAreAllOneSize(t *testing.T) {
	c, srv := open(t, aPile())
	c.navigate(t, srv.URL+"/r/everything")

	heights := c.eval(t, `return [...document.querySelectorAll(".face img")]
		.map(i => Math.round(i.getBoundingClientRect().height))`)
	require.Len(t, heights, 5)
	for _, h := range heights.([]any) {
		require.Equal(t, heights.([]any)[0], h, "every face the same height")
	}
}

// The five labels have to fit their cells on a phone without pushing the page
// sideways. This is here because the first version bought the fit with a 10px
// type step that is not on the ramp — and the documented Meta size turned out
// to fit anyway, with a third of the cell to spare.
func TestBrowserTheFaceLabelsFitAPhone(t *testing.T) {
	c, srv := open(t, aPile())
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 2, "mobile": true,
	})
	c.navigate(t, srv.URL+"/r/everything")

	require.Equal(t, float64(0), c.eval(t, `
		return document.documentElement.scrollWidth - document.documentElement.clientWidth`),
		"nothing may push the page sideways")
	require.Equal(t, true, c.eval(t, `
		return [...document.querySelectorAll(".face")].every(f => {
			const span = f.querySelector("span");
			const r = document.createRange(); r.selectNodeContents(span);
			return r.getBoundingClientRect().width <= f.getBoundingClientRect().width;
		})`), "every label fits its own cell")
}

// degreesOf resolves an angle the way the browser will, so the test compares
// degrees rather than the strings they were written as.
func degreesOf(t *testing.T, c *cdp, angle any) any {
	t.Helper()
	return c.eval(t, fmt.Sprintf(`
		const el = document.createElement("div");
		el.style.transform = "rotate(%v)";
		document.body.appendChild(el);
		const m = new DOMMatrix(getComputedStyle(el).transform);
		el.remove();
		return Math.round(Math.atan2(m.b, m.a) * 180 / Math.PI);
	`, angle))
}

// The lid's field, on the thread. It posts and the answer arrives as a turn —
// the deck's search-as-you-type would fetch a page and paste it over the
// conversation, so it stands aside here.
// Searching from his room answered inside it until 3 September 2026, as a turn
// carrying hits. His room has the board's own bar now, and its field is the
// board's: a word typed there leaves the room and answers on the board, where
// what matched takes the racks' place. That is one search rather than two, and
// it is a real change rather than a tidy-up — the conversation can no longer
// show you a result without leaving it.
//
// What replaced this is TestBrowserTheFieldIsThereWithoutBeingAskedFor and the
// board's own what-matched tests.

// Triage left the conversation on 2 September 2026. The card at the live edge
// came from the four rooms' lists, and those are the board's racks now: a strip
// is answered where it lies, with the same letters, proved in
// TestBrowserTheBoardsKeysFollowFocus. His where is a conversation, and a
// conversation is not where the pile is worked.

// visible is whether an element is actually shown, rather than merely present:
// checkVisibility, because a hidden element keeps its geometry and a bounding
// rect says every one of them is on screen.
const visible = `(sel) => {
	const el = document.querySelector(sel);
	return !!el && el.checkVisibility({
		checkOpacity: true, checkVisibilityCSS: true, contentVisibilityAuto: true,
	});
}`

func layer(t *testing.T, v any) int {
	t.Helper()
	s, ok := v.(string)
	require.True(t, ok, "z-index came back as %T", v)
	n, err := strconv.Atoi(s)
	require.NoError(t, err, "z-index %q is not a number", s)
	return n
}

// A redirecting press went with the shelf chip on 2 September 2026, and the
// shelves themselves went into the notes rack on 3 September: there is no press
// that redirects and no page for the script to paste a whole document into.
// TestTheNotesShowEverythingWithTheUndecidedFirst is where they are proved.

// The worker holding a capture is the nearest honest substitute for a spool, and
// this is the test that it actually holds.
//
// The server is closed rather than the network emulated: CDP's offline emulation
// applies to the page's network stack and not the worker's, so the first version
// passed while the POST reached the server and came back "kept".
func waitForTheWorker(t *testing.T, c *cdp, url string) {
	t.Helper()
	c.until(t, "the worker to be ready", `
		(async () => { await navigator.serviceWorker.ready; return true })()`)
	if c.eval(t, `return !!navigator.serviceWorker.controller`) == true {
		return
	}
	c.navigate(t, url)
	c.until(t, "the worker to be controlling the page", `!!navigator.serviceWorker.controller`)
}

// atChores opens the thread and presses the chores door.
//
// The chores are a where, so a browser test that wants them goes there and waits
// for the cards — which is what a person does, and what makes these tests
// exercise the where's own draw as well as the cards.
//
// It pressed a menu form until 28 August 2026. A where is a link now, and going
// somewhere writes nothing.
func atChores(t *testing.T, srv *httptest.Server) *cdp {
	t.Helper()
	c := browserAt(t, srv, "/?bay=weekly")
	c.until(t, "the chores to arrive", `!!document.querySelector(".strip.h-weekly")`)
	return c
}

// openChores goes to the chores, which are a rack on the board since
// 2 September 2026 rather than a where you press a door for.
func openChores(t *testing.T, c *cdp, srv *httptest.Server) {
	t.Helper()
	c.navigate(t, srv.URL+"/?bay=weekly")
	c.until(t, "the chores to arrive", `!!document.querySelector(".strip.h-weekly")`)
}
