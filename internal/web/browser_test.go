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
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// browser finds something to drive. GitHub's runners ship Chrome; this machine
// has chromium; both answer to the same flags.
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

// screen stands the whole thing up over a real socket, with the identity
// header added on the way in the way the forward-auth middleware adds it. No
// Postgres: what is under test is the script, and the fake store is the same
// one the rest of this package's tests use.
func screen(t *testing.T, f *fakeStore) *httptest.Server {
	t.Helper()

	m := &serveMux{mux: http.NewServeMux()}
	require.NoError(t, Mount(m, f, Options{
		IdentityHeader: "X-Authentik-Username", Identity: "ronald",
		Owner:          func() int64 { return 1 },
	}))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("X-Authentik-Username", "ronald")
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
	t.Helper()

	srv := screen(t, f)
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
	c.navigate(t, srv.URL+"/pile")
	return c, srv
}

func aPile() *fakeStore {
	return &fakeStore{items: []squirrel.Item{
		note(3, "the boiler makes a noise", squirrel.ItemOpen),
		note(2, "meter reading 48213", squirrel.ItemOpen),
		note(1, "ask about the bins", squirrel.ItemOpen),
	}}
}

func TestBrowserAKeyStampsTheCardAndHoldsIt(t *testing.T) {
	c, _ := open(t, aPile())

	c.key(t, "d")
	c.until(t, "the card to be stamped", `document.getElementById("card").classList.contains("stamped")`)

	require.Equal(t, "marked done", c.eval(t, `return document.getElementById("said").textContent`))
	require.Equal(t, false, c.eval(t, `return document.getElementById("undoRow").hidden`),
		"the undo is reachable while the card it undoes is still there")
	require.Equal(t, "undo", c.eval(t, `return document.activeElement.id`))
	require.Contains(t, c.eval(t, `return document.getElementById("say").textContent`), "marked done")
}

func TestBrowserTheIntervalQuestionTakesTheRowAndGivesItBack(t *testing.T) {
	c, _ := open(t, aPile())

	c.key(t, "c")
	c.until(t, "the question to open", `document.querySelector("details.everyFallback").open`)

	require.Equal(t, "none", c.eval(t, `return getComputedStyle(document.querySelector(".btn[data-act=done]")).display`),
		"choosing replaces the ways out rather than growing a second row")
	require.Equal(t, false, c.eval(t, `return document.querySelector("[data-close=chore]").hidden`))

	c.key(t, "Escape")
	c.until(t, "the question to close", `!document.querySelector("details.everyFallback").open`)
	require.Equal(t, "flex", c.eval(t, `return getComputedStyle(document.querySelector(".btn[data-act=done]")).display`))
}

func TestBrowserSearchAnswersAsYouType(t *testing.T) {
	c, _ := open(t, aPile())

	c.eval(t, `
		const find = document.querySelector(".find input");
		find.value = "boiler";
		find.dispatchEvent(new Event("input", { bubbles: true }));
		return true;`)
	c.until(t, "results to arrive", `!!document.querySelector(".rcard")`)

	require.Equal(t, []any{"the boiler makes a noise"},
		c.eval(t, `return [...document.querySelectorAll(".rcard p")].map(p => p.textContent.trim())`))
	require.Equal(t, "?q=boiler", c.eval(t, `return location.search`))
	require.Equal(t, float64(1), c.eval(t, `return performance.getEntriesByType("navigation").length`),
		"the page never reloaded")
	require.Contains(t, c.eval(t, `return document.getElementById("say").textContent`), "boiler")

	c.eval(t, `
		const find = document.querySelector(".find input");
		find.value = "";
		find.dispatchEvent(new Event("input", { bubbles: true }));
		return true;`)
	c.until(t, "the deck to come back", `!!document.getElementById("card")`)

	// The keys have to work on markup that arrived by fetch, which is the
	// whole reason the script's wiring is a function rather than a set of
	// listeners bound once at load.
	c.key(t, "d")
	c.until(t, "the key to work on fetched markup",
		`document.getElementById("card").classList.contains("stamped")`)
}

// Skipping is a link, which is what makes it work on a phone and with this
// file absent. The key presses that link rather than knowing where it points,
// so the two can never disagree about which note is next.
func TestBrowserSkipIsALinkTheKeyPresses(t *testing.T) {
	c, srv := open(t, aPile())

	require.Equal(t, "?after=3", c.eval(t, `return new URL(document.querySelector("a.later").href).search`))
	require.GreaterOrEqual(t,
		c.eval(t, `return Math.round(document.querySelector("a.later").getBoundingClientRect().height)`),
		float64(44), "a tap target on a phone")

	c.key(t, " ")
	deadline := time.Now().Add(10 * time.Second)
	for c.eval(t, `return location.search`) != "?after=3" {
		require.False(t, time.Now().After(deadline), "space never moved past the note")
		time.Sleep(50 * time.Millisecond)
	}
	require.Equal(t, "meter reading 48213", c.eval(t, `return document.getElementById("noteText").textContent`))
	require.Equal(t, srv.URL+"/pile?after=3", c.eval(t, `return location.href`))
}

func TestBrowserTheWorkerTakesTheScreen(t *testing.T) {
	c, srv := open(t, aPile())

	// Served from /sw.js, so it scopes to the root and controls every screen —
	// including home, which is the URL an installed app opens. This needed a
	// Service-Worker-Allowed header when the screen was mounted under a path,
	// and needs none now.
	require.Equal(t, "/", c.eval(t, `
		const reg = await navigator.serviceWorker.ready;
		return new URL(reg.scope).pathname;`),
		"a worker that does not scope to / controls everything except the screens")

	// The second visit is the one that matters. On the first, the page's assets
	// are already on their way before the worker takes control, so nothing goes
	// through it and its cache is legitimately empty — asserting on that load
	// was testing how quickly a worker installs rather than what it does.
	c.navigate(t, srv.URL+"/")
	c.until(t, "the worker to be controlling the page", `!!navigator.serviceWorker.controller`)
	c.until(t, "an asset to be cached", `(await caches.keys()).length > 0`)

	require.Equal(t, true, c.eval(t, `
		const cache = await caches.open((await caches.keys())[0]);
		const held = await cache.keys();
		return held.some(r => r.url.includes("/static/"));`),
		"what it keeps is assets")
}

// The chore picker replaces the action row with no script at all — it is a
// disclosure and a :has() rule. Go can see the markup but not the cascade, and
// the cascade is the whole mechanism.
func TestBrowserTheChorePickerReplacesTheRow(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 1, PersonID: 1, Name: "bins out", Active: true, EverDone: true,
			Every: 14 * 24 * time.Hour, EveryDays: 14, SinceDays: 3},
	}}
	c, srv := open(t, f)
	c.navigate(t, srv.URL+"/chores")

	require.Equal(t, "flex", c.eval(t, `return getComputedStyle(document.querySelector(".abtn.did")).display`))
	require.Equal(t, "every 2 weeks", c.eval(t, `return document.querySelector(".chip.current").textContent`),
		"the picker says where the interval sits now")

	c.eval(t, `document.querySelector("details.often").open = true; return true`)
	c.until(t, "the other actions to step aside",
		`getComputedStyle(document.querySelector(".abtn.did")).display === "none"`)

	// A chore at rest is not a creation, so nothing on it is orange.
	require.Equal(t, false, c.eval(t, `
		return [...document.querySelectorAll(".chore *")].some(el => {
			const bg = getComputedStyle(el).backgroundColor;
			return bg === "rgb(230, 109, 13)" || bg === "rgb(255, 138, 43)";
		});`))
}
