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
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
		Spool:    &fakeSpool{},
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
	return browserAt(t, srv, "/"), srv
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

// Escape, which the platform gives us and which nothing here implements.

// The acorn opens a sheet over the page rather than going anywhere. This is
// the half of the bug above that was invisible: it did navigate, and the page
// it landed on worked, so nothing looked broken until you tried to get out.

// The same rule in the coach's own box, which is reachable from every screen
// and so meets the deck's keys as well as the chores'.

// Sending from the sheet. Reported: you cannot send a message to the coach
// from the app. Enter is the send affordance — it is how the slot works and
// how the room this product lives in works — and the sheet's box never had it,
// because the slot's handler is bound once at load to the page's own box and
// the sheet arrives later.

// Shift+Enter is the newline, for a thought with two parts. Same rule as the
// slot, because it is the same interaction.

// The chips send too, and they carry which one was pressed. Same path as the
// box, so the same bug hit both: a chip press reached the server with no
// answer in it and bounced straight back.

// What the sheet puts on the wire, pinned. It is not a detail: a FormData body
// goes out as multipart, Go's ParseForm reads only urlencoded, and the
// mismatch is silent at both ends — the server finds no words and answers as
// though nothing was said.

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

// And the overlay keeps the card it is supposed to be.
//
// The same markup, so this is the half that a fix to the page could quietly
// take away. It has taken three releases to notice a sheet regression before.

// atChores opens the thread and presses the chores door.
//
// The chores stopped being a page on 24 August 2026, so a browser test that
// wants them presses the door and waits for the cards — which is what a person
// does, and what makes these tests exercise the swap as well as the cards.
func atChores(t *testing.T, srv *httptest.Server) *cdp {
	t.Helper()
	c := browserAt(t, srv, "/")
	c.eval(t, `document.querySelector('.menupanel input[value="chores"]').form
		.querySelector("button").click()`)
	c.until(t, "the chores to arrive", `!!document.querySelector("article.chore")`)
	return c
}

// openChores presses the chores door on a browser already open.
func openChores(t *testing.T, c *cdp, srv *httptest.Server) {
	t.Helper()
	c.navigate(t, srv.URL+"/")
	c.eval(t, `document.querySelector('.menupanel input[value="chores"]').form
		.querySelector("button").click()`)
	c.until(t, "the chores to arrive", `!!document.querySelector("article.chore")`)
}

// TestBrowserTypingAChoreNameIsNotAnAction was retired on 24 August 2026 with
// the new-chore form it covered: making a chore from nothing is a sentence in
// the dock now, and the dock's own keys are covered by the slot's tests.

// The lid's field, on the thread. It posts and the answer arrives as a turn —
// the deck's search-as-you-type would fetch a page and paste it over the
// conversation, so it stands aside here.
func TestBrowserSearchingOnTheThreadAnswersInIt(t *testing.T) {
	f := aPile()
	f.checkin = &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: time.Now()}
	c, srv := open(t, f)
	c.navigate(t, srv.URL+"/")

	// A chip on the live edge since 25 August 2026, rather than a field in the
	// lid: it asks for words, and /find answers them.
	c.eval(t, `document.querySelector('form[action="/find/ask"] button').click()`)
	c.until(t, "the question", `!!document.querySelector(".wordbox")`)
	c.eval(t, `const f = document.querySelector(".wordbox textarea");
		f.value = "boiler"; f.form.requestSubmit(); return 1`)
	// A hit, not a card: a result is a thing you went looking for rather than
	// a thing you are deciding about, and it carries no verbs until you open
	// it. See DESIGN.md, Results.
	c.until(t, "the answer to arrive",
		`!!document.querySelector("#thread .turn:last-child .hit")`)

	require.Equal(t, "/", c.eval(t, `return location.pathname + location.search`),
		"searching navigated")

	// And tapping one turns it into the ordinary card, with the ordinary
	// verbs, in the next turn.
	c.eval(t, `document.querySelector("#thread .turn:last-child .hit button").click()`)
	c.until(t, "the card to arrive",
		`!!document.querySelector("#thread .turn:last-child .turncard .turnacts")`)
}

// Letters are actions, on the note Buddy is holding out.
//
// The deck's keys came with a machine for stamping a card and holding it still;
// none of that crosses, because the answer here is a new turn. The letters do.
func TestBrowserAKeyActsOnTheNoteBuddyIsHoldingOut(t *testing.T) {
	f := &fakeStore{
		items:   []squirrel.Item{note(9, "the boiler", squirrel.ItemOpen)},
		checkin: &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: time.Now()},
	}
	c, srv := open(t, f)
	c.navigate(t, srv.URL+"/")
	c.eval(t, `document.querySelector('.menupanel input[value="pile"]').form
		.querySelector("button").click()`)
	c.until(t, "the note to arrive", `!!document.querySelector("#thread .turncard")`)

	c.key(t, "k")
	c.until(t, "it to be kept", `document.querySelectorAll("#thread .turn").length >= 4`)

	require.Equal(t, squirrel.ItemKept, f.states[9], "the letter did not act")
}

// And a letter typed into a field is a letter, not an action.
func TestBrowserAKeyInTheDockIsJustALetter(t *testing.T) {
	f := &fakeStore{
		items:   []squirrel.Item{note(9, "the boiler", squirrel.ItemOpen)},
		checkin: &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: time.Now()},
	}
	c, srv := open(t, f)
	c.navigate(t, srv.URL+"/")
	c.eval(t, `document.querySelector('.menupanel input[value="pile"]').form
		.querySelector("button").click()`)
	c.until(t, "the note to arrive", `!!document.querySelector("#thread .turncard")`)

	c.eval(t, `document.querySelector(".dock textarea").focus()`)
	c.key(t, "k")
	c.eval(t, `return new Promise(r => setTimeout(r, 300))`)

	require.Empty(t, f.states, "a letter typed in the dock decided something")
}

// Retired with the deck on 25 August 2026.
//
// These pinned the card's own machinery: the tray of answers that opened
// without a press, the stamp that leaned at the day's angle, the hold that gave
// an undo somewhere to be, and the skip link a key pressed. None of it crosses
// to a conversation, where the answer is a new turn and the way back travels
// with it.
//
// What did cross is pinned elsewhere: the letters, by
// TestBrowserAKeyActsOnTheNoteBuddyIsHoldingOut; the interval question, by
// TestTheCurrentIntervalSaysSoAndNotOnlyInPurple; and skipping, by
// TestLaterHandsYouTheNextAndDecidesNothing.
//
// TestBrowserSearchAnswersAsYouType went with search-as-you-type, which stands
// aside on the thread: a search is a thing you asked, and the answer is a turn.

// visible is whether an element is actually shown, rather than merely present.
//
// It lived in answers_test.go, which was the deck's card and went with it. The
// helper outlived the file: the lid's panels and the sheet still ask this.
const visible = `(sel) => {
	const el = document.querySelector(sel);
	return !!el && el.checkVisibility({
		checkOpacity: true, checkVisibilityCSS: true, contentVisibilityAuto: true,
	});
}`

// Nothing at the foot of the conversation hides behind the dock.
//
// The reported bug, measured the way a phone meets it: a conversation long
// enough to scroll, scrolled to the very bottom, with the last two things on
// the page checked against the top of the box.
//
// The first version of this test had no turns in its fixture at all, so the
// page did not scroll and nothing could overlap — it passed with the defect
// deliberately put back. A test that cannot reproduce the bug it is named
// after is worse than none.
func aScrollingThread() *fakeStore {
	f := aPile()
	f.checkin = &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: time.Now()}
	for i := int64(1); i <= 24; i++ {
		who := squirrel.SpeakerBuddy
		if i%2 == 0 {
			who = squirrel.SpeakerYou
		}
		f.turns = append(f.turns, squirrel.Turn{
			ID: i, Who: who,
			Words: "a line of the conversation that is long enough to wrap on a phone",
		})
	}
	return f
}

func atTheBottomOfAPhone(t *testing.T) *cdp {
	t.Helper()
	srv := screen(t, aScrollingThread())
	c := browserAt(t, srv, "/")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 0, "mobile": true,
	})
	c.eval(t, `window.scrollTo(0, document.body.scrollHeight); return 1`)
	c.eval(t, `return new Promise(r => setTimeout(r, 300))`)
	require.Greater(t, c.eval(t, `return document.body.scrollHeight - window.innerHeight`),
		float64(0), "the fixture does not scroll, so nothing can be hidden")
	return c
}

func TestBrowserTheEndOfThePageClearsTheDock(t *testing.T) {
	c := atTheBottomOfAPhone(t)

	// The last turn in the conversation, since `.alsochips` and `.ends` both
	// left the thread on 26 August 2026. What is being measured is unchanged
	// and is the whole point: the last thing on the screen must not be
	// underneath the box you type into. Reported from a phone.
	for _, what := range []string{"#thread .turn:last-child"} {
		gap := c.eval(t, `
			const it = document.querySelector("`+what+`").getBoundingClientRect();
			const dock = document.querySelector(".dock").getBoundingClientRect();
			return Math.round(dock.top - it.bottom);`)
		require.GreaterOrEqual(t, gap, float64(0),
			"%s sits %v pixels under the dock", what, gap)
	}
}

// And the reserve follows the slot as it grows. At four lines a static reserve
// leaves the last thing you said underneath the box you said it in.
func TestBrowserTheReserveFollowsTheSlot(t *testing.T) {
	c := atTheBottomOfAPhone(t)

	before := c.eval(t, `
		const box = document.querySelector(".dock textarea");
		box.value = "one line";
		return Math.round(document.querySelector(".dock").getBoundingClientRect().height);`)

	c.eval(t, `
		const box = document.querySelector(".dock textarea");
		const nl = String.fromCharCode(10);
		box.value = ["a much longer thing to say", "that runs", "to four", "lines"].join(nl);
		box.style.height = "auto";
		box.style.height = box.scrollHeight + "px";
		return 1;`)
	c.eval(t, `return new Promise(r => setTimeout(r, 250))`)

	after := c.eval(t, `return Math.round(document.querySelector(".dock").getBoundingClientRect().height)`)
	require.Greater(t, after, before, "the slot did not grow, so this proves nothing")

	c.eval(t, `window.scrollTo(0, document.body.scrollHeight); return 1`)
	c.eval(t, `return new Promise(r => setTimeout(r, 250))`)

	gap := c.eval(t, `
		const it = document.querySelector("#thread .turn:last-child").getBoundingClientRect();
		const dock = document.querySelector(".dock").getBoundingClientRect();
		return Math.round(dock.top - it.bottom);`)
	require.GreaterOrEqual(t, gap, float64(0),
		"the grown slot covers the end of the page by %v pixels", gap)
}

// Buddy's face is the gutter wide, and nothing is drawn around it.
//
// `.face` is the check-in's mood button and carries a 44px tap target, so a
// face that took that class came out the wrong size — the stylesheet reads
// correctly either way, and only the rendered box says which class won.
//
// The second half is the design: the artwork brings its own outline, so a
// border or a fill here would stack two.
func TestBrowserBuddysFaceIsTheGutterAndNothingElse(t *testing.T) {
	f := aPile()
	f.checkin = &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: time.Now()}
	f.turns = []squirrel.Turn{{ID: 1, Who: squirrel.SpeakerBuddy, Words: "Kept."}}
	c := browserAt(t, screen(t, f), "/")

	require.Equal(t, float64(40), c.eval(t, `
		return Math.round(document.querySelector(".buddyface").getBoundingClientRect().width);`),
		"the face is not the gutter wide: something else owns this class")

	// Each read on its own. The first version of this joined them and asked
	// for substrings, which passed with a 3px purple disc put back: the
	// computed border width is "3px" rather than "px solid", and a background
	// *colour* leaves background-image reading "none".
	face := `getComputedStyle(document.querySelector(".buddyface"))`
	require.Equal(t, "0px", c.eval(t, `return `+face+`.borderTopWidth`),
		"there is a border around artwork that has its own outline")
	require.Equal(t, "none", c.eval(t, `return `+face+`.boxShadow`),
		"there is a shadow behind the artwork")
	require.Contains(t, []any{"rgba(0, 0, 0, 0)", "transparent"},
		c.eval(t, `return `+face+`.backgroundColor`),
		"there is a fill behind the artwork")
}

// And the image fills it rather than sitting in it.
func TestBrowserTheArtworkFillsTheGutter(t *testing.T) {
	f := aPile()
	f.checkin = &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: time.Now()}
	f.turns = []squirrel.Turn{{ID: 1, Who: squirrel.SpeakerBuddy, Words: "Kept."}}
	c := browserAt(t, screen(t, f), "/")

	require.Equal(t, float64(40), c.eval(t, `
		return Math.round(document.querySelector(".buddyface img").getBoundingClientRect().width);`))
}

// A full-width control spans the gutter. The check-in's five labels stopped
// fitting a 390px phone the moment the gutter took 44px off them.
func TestBrowserAControlStripSpansTheGutter(t *testing.T) {
	f := aPile()
	f.turns = []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerBuddy, Words: "how do you feel?", Shown: []byte(`{"faces":true}`)},
	}
	c := browserAt(t, screen(t, f), "/")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 0, "mobile": true,
	})
	c.eval(t, `return new Promise(r => setTimeout(r, 200))`)

	inset := c.eval(t, `
		const turn = document.querySelector(".turn.frombuddy").getBoundingClientRect();
		const faces = document.querySelector(".faces").getBoundingClientRect();
		return Math.round(faces.left - turn.left);`)

	require.Equal(t, float64(0), inset,
		"the mood row is indented past the gutter by %v pixels", inset)
}
