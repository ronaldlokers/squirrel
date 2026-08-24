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
	return screenWith(t, f, nil)
}

// screenWith is screen plus a coach, for the tests that need one behind the
// sheet. Without it the sheet already shows "which of these is it", which
// makes a test looking for an answer pass before anything has been sent.
func screenWith(t *testing.T, f *fakeStore, c *fakeCoach) *httptest.Server {
	t.Helper()

	opts := Options{
		IdentityHeader: "X-Authentik-Username", Identity: "ronald",
		Owner: func() int64 { return 1 }, Spool: &fakeSpool{},
	}
	if c != nil {
		opts = c.options(opts)
	}

	m := &serveMux{mux: http.NewServeMux()}
	require.NoError(t, Mount(m, f, opts))

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
	return openWith(t, f, nil)
}

func openWith(t *testing.T, f *fakeStore, coach *fakeCoach) (*cdp, *httptest.Server) {
	t.Helper()

	srv := screenWith(t, f, coach)
	return browserAt(t, srv, "/pile"), srv
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
	waitForTheWorker(t, c, srv.URL+"/")
	// Wait for the asset itself, not merely for a cache to exist.
	//
	// A cache appears the moment the worker opens one, which is before any
	// response has been put in it — so waiting on `caches.keys()` and then
	// asserting on the contents was a race the fast machine always won and a
	// loaded CI runner sometimes lost. It failed once on a green branch, which
	// is the worst way for a test to be wrong: it says the change broke
	// something it never touched.
	// `until` wraps what it is given in `await (...)`, so this is an expression
	// rather than a body.
	c.until(t, "an asset to be cached", `(async () => {
		for (const name of await caches.keys()) {
			const held = await (await caches.open(name)).keys();
			if (held.some(r => r.url.includes("/static/"))) return true;
		}
		return false;
	})()`)

	require.Equal(t, true, c.eval(t, `
		const cache = await caches.open((await caches.keys())[0]);
		const held = await cache.keys();
		return held.some(r => r.url.includes("/static/"));`),
		"what it keeps is assets")
}

// The chores screen is a list, so a key needs to know which chore it means.
// Rather than invent a selection model it uses the platform's own — the chore
// you are focused in — and DESIGN.md's rule decides the rest: letters are
// actions, movement is the arrow keys.
func TestBrowserTheChoresKeysFollowFocus(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 1, PersonID: 1, Name: "bins out", Active: true, EverDone: true,
			Every: 14 * 24 * time.Hour, EveryDays: 14, SinceDays: 3},
		{ID: 2, PersonID: 1, Name: "water the ferns", Active: true,
			Every: 7 * 24 * time.Hour, EveryDays: 7},
	}}
	c, srv := open(t, f)
	openChores(t, c, srv)

	// Nothing focused: the first key press says where you are rather than
	// acting on something you did not choose.
	c.key(t, "d")
	c.until(t, "the first chore to take focus",
		`document.activeElement.closest("article.chore")?.querySelector(".name").textContent === "bins out"`)

	c.key(t, "ArrowDown")
	c.until(t, "the second chore to take focus",
		`document.activeElement.closest("article.chore")?.querySelector(".name").textContent === "water the ferns"`)

	// O asks that chore's own question, not the first one's. It used to open a
	// disclosure inside the card; it posts now, and the question arrives as a
	// turn of its own — so what this checks is which chore the question is
	// about, which is the thing that was ever worth checking.
	c.key(t, "o")
	c.until(t, "the interval question to arrive", `!!document.querySelector(".pick")`)

	require.Equal(t, "2", c.eval(t, `
		return document.querySelector('.pick input[name="id"]').value`),
		"the question is about the chore that was focused")
}

// Five answers, five equal drawings. They were cut from one sheet at one
// scale, and each carries a different amount of decoration outside its tile —
// sparks, leaves, a scribble, a zzz — so sizing them by width would make the
// busiest one the smallest. Sized by height, they read as five answers rather
// than as one louder than the others.
func TestBrowserTheFacesAreAllOneSize(t *testing.T) {
	c, srv := open(t, aPile())
	c.navigate(t, srv.URL+"/")

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
	c.navigate(t, srv.URL+"/")

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

// The screen writes straight to the pile and there is no spool behind that, so
// a capture typed with no network would simply be lost. The worker holding it
// is the nearest honest substitute — and this is the test that it actually
// holds, rather than that the code reads as though it would.
//
// The server is closed rather than the network emulated: CDP's offline
// emulation applies to the page's network stack and not to the worker's own,
// so the first version of this test passed while the POST reached the server
// and came back "kept". A closed socket is offline for both.
// waitForTheWorker waits for a worker that is actually driving the page.
//
// Registration, install, activate and `clients.claim()` are four steps, and a
// page that loaded before the last of them is not controlled — so waiting on
// `controller` alone is waiting on a race, which this machine wins every time
// and a loaded runner does not. It failed CI twice on branches that had not
// touched the worker, which is the way a flake does its real damage: it
// teaches you to re-run the job instead of reading it.
//
// So this waits for the registration to be ready first, and then, if the page
// still is not controlled, navigates once more. A worker that has activated
// controls the next navigation by definition, so the second visit is a
// guarantee rather than another roll.
//
// What that costs, and it is worth naming rather than discovering later: these
// tests no longer notice `clients.claim()` going missing. Claiming is what
// makes the *first* visit controlled without a reload, and the only way to
// assert it is to race activation — which is the race that was flaking. The
// property is real and remains untested here on purpose; a test of it would be
// a test that fails on a busy machine for a reason unrelated to the change.
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

func TestBrowserACaptureSurvivesNoNetwork(t *testing.T) {
	c, srv := open(t, aPile())
	c.navigate(t, srv.URL+"/")
	waitForTheWorker(t, c, srv.URL+"/")

	srv.Close()

	require.Equal(t, true, c.eval(t, `
		const res = await fetch("/capture", {
			method: "POST",
			headers: { "Content-Type": "application/x-www-form-urlencoded" },
			body: new URLSearchParams({ text: "ask the garage about the rattle" }),
		});
		return new URL(res.url).search.includes("held");`),
		"the worker answered, and said so")

	// On disk, not merely in a promise somewhere.
	require.Equal(t, true, c.eval(t, `
		return await new Promise(resolve => {
			const open = indexedDB.open("squirrel-held", 1);
			open.onerror = () => resolve(false);
			open.onsuccess = () => {
				const db = open.result;
				if (!db.objectStoreNames.contains("notes")) return resolve(false);
				const req = db.transaction("notes").objectStore("notes").getAll();
				req.onsuccess = () => resolve(req.result.some(n => n.text.includes("the rattle")));
				req.onerror = () => resolve(false);
			};
		});`), "the words are held")
}

// Closing the coach, reported broken from production. The sheet never opened
// on the pile at all: `.acorn` was the card's drawn badge as well as the
// button, and pile.js wired the badge — so the acorn navigated away instead,
// and what looked like "closing does not work" was "this was never a sheet".
func TestBrowserClosingTheCoach(t *testing.T) {
	c, _ := open(t, aPile())

	c.eval(t, `document.querySelector(".tobuddy").click()`)
	c.until(t, "the sheet to open", `!!document.querySelector("dialog.coachsheet[open]")`)

	c.eval(t, `document.querySelector("dialog.coachsheet .shut").click()`)
	c.until(t, "the sheet to close", `!document.querySelector("dialog.coachsheet[open]")`)
	require.Equal(t, "/pile", c.eval(t, `return location.pathname`),
		"closing the sheet navigated instead of closing")
}

// Escape, which the platform gives us and which nothing here implements.
func TestBrowserEscapeClosesTheCoach(t *testing.T) {
	c, _ := open(t, aPile())

	c.eval(t, `document.querySelector(".tobuddy").click()`)
	c.until(t, "the sheet to open", `!!document.querySelector("dialog.coachsheet[open]")`)

	c.key(t, "Escape")
	c.until(t, "the sheet to close", `!document.querySelector("dialog.coachsheet[open]")`)
}

// The acorn opens a sheet over the page rather than going anywhere. This is
// the half of the bug above that was invisible: it did navigate, and the page
// it landed on worked, so nothing looked broken until you tried to get out.
func TestBrowserTheAcornDoesNotNavigate(t *testing.T) {
	c, _ := open(t, aPile())

	c.eval(t, `document.querySelector(".tobuddy").click()`)
	c.until(t, "the sheet to open", `!!document.querySelector("dialog.coachsheet[open]")`)
	require.Equal(t, "/pile", c.eval(t, `return location.pathname`))
}

// The card's badge is a 16px drawing in a title bar, not a button stuck to the
// corner of the screen. It was the latter for one release, when the button
// took `.acorn` for itself and every note's badge became a 62px circle.
//
// Nothing is stuck to the corner any more: the way to Buddy is in the lid.
// That closes the same hole from the other side — a fixed button over the
// content is a button that sits on top of whatever is underneath it, which is
// what it did to a chore's timer row and to the sheet's own send button.
func TestBrowserTheCardsBadgeIsStillInItsTitleBar(t *testing.T) {
	c, _ := open(t, aPile())

	require.Equal(t, "static", c.eval(t,
		`return getComputedStyle(document.querySelector(".titlebar .acorn")).position`))
	require.NotEqual(t, "fixed", c.eval(t,
		`return getComputedStyle(document.querySelector(".tobuddy")).position`),
		"the way to Buddy is floating over the content again")
	require.Equal(t, true, c.eval(t,
		`return !!document.querySelector(".lid .tobuddy")`),
		"the way to Buddy left the lid")
}

// The same rule in the coach's own box, which is reachable from every screen
// and so meets the deck's keys as well as the chores'.
func TestBrowserTypingToTheCoachIsNotAnAction(t *testing.T) {
	c, _ := open(t, aPile())

	c.eval(t, `document.querySelector(".tobuddy").click()`)
	c.until(t, "the sheet to open", `!!document.querySelector("dialog.coachsheet[open]")`)

	c.eval(t, `document.querySelector('dialog.coachsheet textarea[name=said]').focus()`)
	for _, k := range []string{"d", "k", "x"} {
		c.key(t, k)
	}

	require.Equal(t, "dkx", c.eval(t,
		`return document.querySelector('dialog.coachsheet textarea[name=said]').value`))
	require.Equal(t, false, c.eval(t, `return document.getElementById("card").classList.contains("stamped")`),
		"typing to the coach triaged the note behind it")
}

// Sending from the sheet. Reported: you cannot send a message to the coach
// from the app. Enter is the send affordance — it is how the slot works and
// how the room this product lives in works — and the sheet's box never had it,
// because the slot's handler is bound once at load to the page's own box and
// the sheet arrives later.
func TestBrowserEnterSendsToTheCoach(t *testing.T) {
	c, _ := openWith(t, aPile(), &fakeCoach{reply: "Start with the envelope."})

	c.eval(t, `document.querySelector(".tobuddy").click()`)
	c.until(t, "the sheet to open", `!!document.querySelector("dialog.coachsheet[open]")`)

	c.eval(t, `
		const t = document.querySelector('dialog.coachsheet textarea[name=said]');
		t.focus(); t.value = "everything at once";
	`)
	c.key(t, "Enter")

	c.until(t, "the coach to answer",
		`document.querySelector("dialog.coachsheet")?.textContent.includes("Start with the envelope.")`)
}

// Shift+Enter is the newline, for a thought with two parts. Same rule as the
// slot, because it is the same interaction.
func TestBrowserShiftEnterDoesNotSendToTheCoach(t *testing.T) {
	c, _ := openWith(t, aPile(), &fakeCoach{reply: "Start with the envelope."})

	c.eval(t, `document.querySelector(".tobuddy").click()`)
	c.until(t, "the sheet to open", `!!document.querySelector("dialog.coachsheet[open]")`)

	c.eval(t, `
		const t = document.querySelector('dialog.coachsheet textarea[name=said]');
		t.focus(); t.value = "one thing";
	`)
	c.eval(t, `
		document.querySelector('dialog.coachsheet textarea[name=said]')
			.dispatchEvent(new KeyboardEvent("keydown", {key: "Enter", shiftKey: true, bubbles: true, cancelable: true}));
	`)

	require.Equal(t, false, c.eval(t,
		`return document.querySelector("dialog.coachsheet").textContent.includes("Start with the envelope.")`),
		"shift+enter sent instead of making a newline")
}

// The chips send too, and they carry which one was pressed. Same path as the
// box, so the same bug hit both: a chip press reached the server with no
// answer in it and bounced straight back.
func TestBrowserAChipSendsToTheCoach(t *testing.T) {
	f := withOffer(&squirrel.Offer{
		Kind: squirrel.OfferTask, RefID: 7, Text: "the tax thing",
	})
	f.items = aPile().items
	c, _ := openWith(t, f, breaksInto(&fakeCoach{}, "open the letter", "ring the number"))

	c.eval(t, `document.querySelector(".tobuddy").click()`)
	c.until(t, "the sheet to open", `!!document.querySelector("dialog.coachsheet[open]")`)

	c.eval(t, `document.querySelector('dialog.coachsheet .why[value="big"]').click()`)
	c.until(t, "the ladder to answer",
		`document.querySelector("dialog.coachsheet")?.textContent.includes("open the letter")`)
}

// What the sheet puts on the wire, pinned. It is not a detail: a FormData body
// goes out as multipart, Go's ParseForm reads only urlencoded, and the
// mismatch is silent at both ends — the server finds no words and answers as
// though nothing was said.
func TestBrowserTheSheetPostsTheWayAFormPosts(t *testing.T) {
	c, _ := openWith(t, aPile(), &fakeCoach{reply: "Start with the envelope."})

	c.eval(t, `document.querySelector(".tobuddy").click()`)
	c.until(t, "the sheet to open", `!!document.querySelector("dialog.coachsheet[open]")`)

	c.eval(t, `
		window.__sent = [];
		const real = window.fetch;
		window.fetch = (url, opts) => {
			if (opts?.method === "POST") {
				window.__sent.push(String(url) + " " + (opts.headers?.["Content-Type"] || "none"));
			}
			return real(url, opts);
		};
		const t = document.querySelector('dialog.coachsheet textarea[name=said]');
		t.focus(); t.value = "everything at once";
	`)
	c.key(t, "Enter")
	c.until(t, "the coach to answer",
		`document.querySelector("dialog.coachsheet")?.textContent.includes("Start with the envelope.")`)

	require.Contains(t, c.eval(t, `return JSON.stringify(window.__sent)`),
		"/buddy/say application/x-www-form-urlencoded")
}

// The stylesheet actually reads the day.
//
// The Go tests prove the numbers reach the body as custom properties. Nothing
// there proves anything *uses* them: a typo in a variable name leaves the
// stamp at the fallback angle forever, looking exactly like it did before, and
// no test in this repository would notice. Only a browser resolves a `var()`.
func TestBrowserTheStampLeansAtTheDaysAngle(t *testing.T) {
	c, _ := open(t, aPile())

	// The body's own property, whatever the server sent today.
	tilt := c.eval(t, `return getComputedStyle(document.body).getPropertyValue("--tilt").trim()`)
	require.NotEmpty(t, tilt, "the body carries no angle")

	// The angle it rests at, read before anything animates.
	//
	// This is the assertion that catches a mistyped variable name, and it has
	// to come first: the slap fills forwards, so once it has run its own
	// keyframe supplies the transform and a broken base rule is invisible
	// underneath it. Checked in the first version of this test, which a
	// deliberately misspelt `--tiltt` walked straight through.
	//
	// Shown for the measurement, because a `display: none` element reports no
	// transform at all rather than the one it would have.
	require.Equal(t, degreesOf(t, c, tilt), c.eval(t, `
		const el = document.querySelector(".stamp");
		el.style.display = "inline-flex";
		const m = new DOMMatrix(getComputedStyle(el).transform);
		el.style.display = "";
		return Math.round(Math.atan2(m.b, m.a) * 180 / Math.PI);
	`), "the stamp rests at a different angle than the day's")

	c.key(t, "d")
	c.until(t, "the card to be stamped", `document.getElementById("card").classList.contains("stamped")`)
	// The slap has to land before the angle means anything — measured mid-flight
	// it reads whatever degree the animation was passing through, which is how
	// this first read -4 for a -3 day.
	c.until(t, "the slap to land", `
		document.querySelector(".stamp").getAnimations().every(a => a.playState === "finished")`)

	// A resolved matrix rather than the rotate() that was written, so this
	// compares angles rather than strings. It also pins the keyframe: the slap
	// ends at the same angle the stamp rests at, or it snaps on landing.
	got := c.eval(t, `
		const m = new DOMMatrix(getComputedStyle(document.querySelector(".stamp")).transform);
		return Math.round(Math.atan2(m.b, m.a) * 180 / Math.PI);
	`)
	require.Equal(t, degreesOf(t, c, tilt), got,
		"the slap landed at %v rather than at the day's %v", got, tilt)
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

// And the field is lit from where the day says.
//
// Same reason, other property. This one is behind every screen, so a var()
// that silently fell back would be the whole product missing the change.
func TestBrowserTheFieldIsLitFromTheDaysPlace(t *testing.T) {
	c, _ := open(t, aPile())

	light := c.eval(t, `return getComputedStyle(document.body).getPropertyValue("--light").trim()`)
	require.NotEmpty(t, light, "the body carries no light")

	image := c.eval(t, `return getComputedStyle(document.body).backgroundImage`)
	require.Contains(t, image, fmt.Sprintf("at %v", light),
		"the field's highlight is not where the day put it")
}

// Buddy, as a screen, is legible on the field.
//
// The page route and the overlay are the same markup, and until this the same
// stylesheet gave both the card's dark-on-cream inks. That is right inside the
// overlay and wrong on the page, where there is no cream — the sheet now
// stands directly on the field like every other screen's content.
//
// The failure mode is not subtle and it is not visible to Go: a `--brown`
// label on a purple field is dark ink on a dark ground. So this walks every
// piece of text that ends up standing on the field itself — anything inside an
// object with its own fill is that object's problem, and already tested — and
// checks it is light enough to read.
func TestBrowserBuddyAsAPageIsLegibleOnTheField(t *testing.T) {
	f := aPile()
	f.offer = &squirrel.Offer{
		Kind: squirrel.OfferChore, RefID: 1,
		Text: "put the bins out", Because: "it is bin day tomorrow",
	}
	srv := screenWith(t, f, &fakeCoach{reply: "start with the bins."})
	c := browserAt(t, srv, "/buddy?from=/pile")

	// Nothing between the conversation and the field: no card behind it.
	require.Equal(t, "rgba(0, 0, 0, 0)",
		c.eval(t, `return getComputedStyle(document.querySelector(".sheet")).backgroundColor`),
		"the page still draws a card behind Buddy")

	// The title is set like every other screen's: the wordmark's face on paper.
	// Without this only its legibility is pinned, and it would inherit the
	// sheet's quiet cream — readable, and the wrong one of the two inks.
	require.Equal(t, "rgb(255, 251, 243)",
		c.eval(t, `return getComputedStyle(document.querySelector(".sheetname")).color`),
		"Buddy's title is not written in the ink every other screen title uses")
	require.Equal(t, "21px",
		c.eval(t, `return getComputedStyle(document.querySelector(".sheetname")).fontSize`))

	dark := c.eval(t, `
		const lum = c => {
			const [r, g, b] = c.match(/\d+(\.\d+)?/g).slice(0, 3).map(Number).map(v => {
				v /= 255;
				return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
			});
			return 0.2126 * r + 0.7152 * g + 0.0722 * b;
		};
		// Does anything between this and the body paint its own ground?
		const onTheField = el => {
			for (let n = el; n && n !== document.body; n = n.parentElement) {
				const bg = getComputedStyle(n).backgroundColor;
				if (bg !== "rgba(0, 0, 0, 0)" && bg !== "transparent") return false;
			}
			return true;
		};
		const bad = [];
		for (const el of document.querySelectorAll(".sheet, .sheet *")) {
			// Its own words, not its children's.
			const own = [...el.childNodes]
				.filter(n => n.nodeType === 3).map(n => n.textContent.trim()).join("");
			if (!own || !onTheField(el)) continue;
			if (lum(getComputedStyle(el).color) < 0.5) {
				bad.push(el.className + ": " + own.slice(0, 24) + " " + getComputedStyle(el).color);
			}
		}
		return bad.join(" | ");
	`)
	require.Equal(t, "", dark, "dark ink standing on the purple field")
}

// And the overlay keeps the card it is supposed to be.
//
// The same markup, so this is the half that a fix to the page could quietly
// take away. It has taken three releases to notice a sheet regression before.
func TestBrowserBuddyAsAnOverlayIsStillACard(t *testing.T) {
	c, _ := open(t, aPile())

	c.eval(t, `document.querySelector(".tobuddy").click(); return 1`)
	c.until(t, "the sheet to open", `!!document.querySelector("dialog.coachsheet[open]")`)

	style := func(sel, prop string) any {
		return c.eval(t, fmt.Sprintf(
			`return getComputedStyle(document.querySelector("dialog.coachsheet %s")).%s`, sel, prop))
	}
	require.Equal(t, "rgb(253, 236, 212)", style(".sheet", "backgroundColor"),
		"the overlay lost its card stock")
	require.Equal(t, "rgb(28, 17, 11)", style(".sheet", "color"),
		"the overlay is no longer written in the card's ink")
	// Its own small lid, not a screen title: the overlay is an object.
	require.Equal(t, "17px", style(".sheetname", "fontSize"))
}

// atChores opens the thread and presses the chores door.
//
// The chores stopped being a page on 24 August 2026, so a browser test that
// wants them presses the door and waits for the cards — which is what a person
// does, and what makes these tests exercise the swap as well as the cards.
func atChores(t *testing.T, srv *httptest.Server) *cdp {
	t.Helper()
	c := browserAt(t, srv, "/")
	c.eval(t, `document.querySelector('.doorpress input[value="chores"]').form
		.querySelector("button").click()`)
	c.until(t, "the chores to arrive", `!!document.querySelector("article.chore")`)
	return c
}

// openChores presses the chores door on a browser already open.
func openChores(t *testing.T, c *cdp, srv *httptest.Server) {
	t.Helper()
	c.navigate(t, srv.URL+"/")
	c.eval(t, `document.querySelector('.doorpress input[value="chores"]').form
		.querySelector("button").click()`)
	c.until(t, "the chores to arrive", `!!document.querySelector("article.chore")`)
}

// TestBrowserTypingAChoreNameIsNotAnAction was retired on 24 August 2026 with
// the new-chore form it covered: making a chore from nothing is a sentence in
// the dock now, and the dock's own keys are covered by the slot's tests.
