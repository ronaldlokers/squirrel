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
	"encoding/base64"
	"fmt"
	"image/png"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
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

	image := c.eval(t, `return getComputedStyle(document.body, "::before").backgroundImage`)
	require.Contains(t, image, fmt.Sprintf("at %v", light),
		"the field's highlight is not where the day put it")
}

// atChores opens the thread and presses the chores door.
//
// The chores are a room, so a browser test that wants them goes there and waits
// for the cards — which is what a person does, and what makes these tests
// exercise the room's own draw as well as the cards.
//
// It pressed a menu form until 28 August 2026. A room is a link now, and going
// somewhere writes nothing.
func atChores(t *testing.T, srv *httptest.Server) *cdp {
	t.Helper()
	c := browserAt(t, srv, "/r/chores")
	c.until(t, "the chores to arrive", `!!document.querySelector("article.chore")`)
	return c
}

// openChores presses the chores door on a browser already open.
func openChores(t *testing.T, c *cdp, srv *httptest.Server) {
	t.Helper()
	// Straight to the room. This pressed a menu form until 28 August 2026,
	// when rooms became links — a click on the rail is what a person does now.
	c.navigate(t, srv.URL+"/r/chores")
	c.until(t, "the chores to arrive", `!!document.querySelector("article.chore")`)
}

// The lid's field, on the thread. It posts and the answer arrives as a turn —
// the deck's search-as-you-type would fetch a page and paste it over the
// conversation, so it stands aside here.
func TestBrowserSearchingOnTheThreadAnswersInIt(t *testing.T) {
	f := aPile()
	f.checkin = &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: time.Now()}
	c, srv := open(t, f)
	c.navigate(t, srv.URL+"/")

	// A chip on the live edge rather than a field in the lid: it asks for
	// words, and /find answers them.
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

// The deck's keys came with a machine for stamping a card and holding it still;
// none of that crosses, because the answer here is a new turn. The letters do.
func TestBrowserAKeyActsOnTheNoteBuddyIsHoldingOut(t *testing.T) {
	f := &fakeStore{
		items:   []squirrel.Item{note(9, "the boiler", squirrel.ItemOpen)},
		checkin: &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: time.Now()},
	}
	c, srv := open(t, f)
	c.navigate(t, srv.URL+"/r/pile")
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
	c.navigate(t, srv.URL+"/r/pile")
	c.until(t, "the note to arrive", `!!document.querySelector("#thread .turncard")`)

	c.eval(t, `document.querySelector(".dock textarea").focus()`)
	c.key(t, "k")
	c.eval(t, `return new Promise(r => setTimeout(r, 300))`)

	require.Empty(t, f.states, "a letter typed in the dock decided something")
}

// visible is whether an element is actually shown, rather than merely present:
// checkVisibility, because a hidden element keeps its geometry and a bounding
// rect says every one of them is on screen.
const visible = `(sel) => {
	const el = document.querySelector(sel);
	return !!el && el.checkVisibility({
		checkOpacity: true, checkVisibilityCSS: true, contentVisibilityAuto: true,
	});
}`

// A conversation long enough to scroll, which is what the tests below need to
// be able to reproduce anything: the first version of them had no turns at all,
// so the page did not scroll, nothing could overlap, and they passed with the
// defect deliberately put back.
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
	// The page itself does not scroll any more — the body is one viewport-high
	// grid and the transcript is the only thing with an overflow. Measuring
	// document.body here reported zero and the fixture looked broken when it
	// was the measurement that had moved.
	c.eval(t, `const s = document.querySelector(".scroll"); s.scrollTop = s.scrollHeight; return 1`)
	c.eval(t, `return new Promise(r => setTimeout(r, 300))`)
	require.Greater(t, c.eval(t, `
		const s = document.querySelector(".scroll");
		return s.scrollHeight - s.clientHeight`),
		float64(0), "the fixture does not scroll, so nothing can be hidden")
	return c
}

// The last thing on the screen must not be underneath the box you type into.
// Reported from a phone.
//
// It cannot be, now: they are two rows of one grid rather than a column with a
// fixed box over it. The test stays because that is a claim about the layout,
// and a claim is worth a check that would notice it being untrue.
func TestBrowserTheEndOfThePageClearsTheDock(t *testing.T) {
	c := atTheBottomOfAPhone(t)

	gap := c.eval(t, `
		const it = document.querySelector("#thread .turn:last-child").getBoundingClientRect();
		const dock = document.querySelector(".dock").getBoundingClientRect();
		return Math.round(dock.top - it.bottom);`)
	require.GreaterOrEqual(t, gap, float64(0),
		"the last turn sits %v pixels under the dock", gap)
}

// And the conversation gives way as the slot grows. A slot at four lines
// shortens the scroll region by its own growth; this used to be a measured
// reserve maintained by a ResizeObserver, and is now what a grid row does.
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

	// Scrolled to the end after the growth, because that is the claim: the end
	// of the conversation stays reachable when the slot takes more room.
	//
	// Not measured before scrolling. A turn inside a scrolling box that is
	// currently out of view legitimately has a rect below the fold — clipping
	// is not layout — so measuring without scrolling asks whether the end
	// happens to be on screen, which is a different and uninteresting question.
	c.eval(t, `const s = document.querySelector(".scroll"); s.scrollTop = s.scrollHeight; return 1`)
	c.eval(t, `return new Promise(r => setTimeout(r, 250))`)

	gap := c.eval(t, `
		const it = document.querySelector("#thread .turn:last-child").getBoundingClientRect();
		const dock = document.querySelector(".dock").getBoundingClientRect();
		return Math.round(dock.top - it.bottom);`)
	require.GreaterOrEqual(t, gap, float64(0),
		"the end of the conversation cannot be scrolled clear of the grown slot: %v pixels", gap)
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

// Nothing paints over the open room sheet.
//
// The sheet is an overlay and the way out is its last row, so the question is
// whether anything is on top of it — which is how this failed before: the
// sheet lived inside the lid, its z-index was scoped to the lid's stacking
// context, and the dock painted straight over it. On a landscape phone that
// covered the last three rooms, search and the way out, which is the failure
// the sheet exists to fix.
//
// Asserted by hit-testing rather than by comparing z-indexes. The dock is in
// flow now and carries no z-index at all, so the numbers no longer answer the
// question; what the person can actually press does.
func TestBrowserNothingPaintsOverTheOpenRoomSheet(t *testing.T) {
	c, srv := open(t, &fakeStore{})
	c.navigate(t, srv.URL+"/r/chores")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 844, "height": 390, "deviceScaleFactor": 0, "mobile": true,
	})
	c.eval(t, `document.querySelector(".roomsheet").open = true; return 1`)
	c.eval(t, `return new Promise(r => setTimeout(r, 200))`)

	hit := c.eval(t, `
		const out = document.querySelector(".rail .leaving");
		out.scrollIntoView({block: "center"});
		const b = out.getBoundingClientRect();
		const top = document.elementFromPoint(b.left + b.width / 2, b.top + b.height / 2);
		return top === out || out.contains(top) ? "the way out" : (top ? top.className || top.tagName : "nothing");`)

	require.Equal(t, "the way out", hit,
		"something is painted over the way out in the open room sheet")
}

func layer(t *testing.T, v any) int {
	t.Helper()
	s, ok := v.(string)
	require.True(t, ok, "z-index came back as %T", v)
	n, err := strconv.Atoi(s)
	require.NoError(t, err, "z-index %q is not a number", s)
	return n
}

// On a phone the dock's button sits on its own row, under the field.
//
// Reported from a phone, in the agenda: "put it in the agenda" is twenty
// characters and took over half the width, so the placeholder wrapped to two
// lines inside a field too narrow to type in. The button naming the
// consequence and the one-row dock cannot both hold at 390px, and the field is
// the control that has to work.
//
// Nothing else here can see it. The appearance snapshot visits one desktop
// viewport, where they legitimately share a row, and it samples no property
// that would change — min-width is not in its list.
func TestBrowserTheDockGivesTheFieldItsOwnRowOnAPhone(t *testing.T) {
	c, srv := open(t, &fakeStore{})
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 0, "mobile": true,
	})

	for _, room := range []string{"at", "buddy", "pile"} {
		c.navigate(t, srv.URL+"/r/"+room)
		gap := c.eval(t, `
			const box = document.querySelector(".dock textarea").getBoundingClientRect();
			const post = document.querySelector(".dock .post").getBoundingClientRect();
			return Math.round(post.top - box.bottom);`)
		require.GreaterOrEqual(t, gap, float64(0),
			"in %s the button shares the field's row on a phone, by %v pixels", room, gap)

		lines := c.eval(t, `
			const t = document.querySelector(".dock textarea");
			const cs = getComputedStyle(t);
			return Math.round(t.getBoundingClientRect().height / parseFloat(cs.lineHeight));`)
		require.LessOrEqual(t, lines, float64(1),
			"in %s the placeholder wraps to %v lines in an empty field", room, lines)
	}
}

// The worked example is laid out like the conversation it is a picture of.
//
// It carries its own card and act classes on purpose — a picture of a card is
// not a card — but it sat outside .thread with no width, no padding and no gap
// and inherited none of them. On a phone that ran its label off the left edge
// of the screen and let every card overlap the bubble beneath it.
//
// It shipped that way on 26 August and nobody saw it for two weeks, because it
// draws only when the record is empty and this record has never been empty
// since. The appearance snapshot cannot see it either: its fixture has turns,
// so the example is not on the page it samples.
func TestBrowserTheWorkedExampleIsInsideTheScreen(t *testing.T) {
	c, srv := open(t, &fakeStore{})
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 0, "mobile": true,
	})
	c.navigate(t, srv.URL+"/")
	c.until(t, "the worked example", `!!document.querySelector(".worked")`)

	left := c.eval(t, `return Math.round(document.querySelector(".workedsays").getBoundingClientRect().left)`)
	require.Greater(t, left, float64(0),
		"the example's first line starts at or past the left edge of the screen")

	// And its turns do not sit on top of one another, which is what having no
	// gap looked like: a card over the bubble under it.
	overlap := c.eval(t, `
		const turns = [...document.querySelectorAll(".worked .turn")];
		let worst = 0;
		for (let i = 1; i < turns.length; i++) {
			const above = turns[i - 1].getBoundingClientRect();
			const below = turns[i].getBoundingClientRect();
			worst = Math.min(worst, Math.round(below.top - above.bottom));
		}
		return worst;`)
	require.GreaterOrEqual(t, overlap, float64(0),
		"two turns of the worked example overlap by %v pixels", overlap)
}

func TestBrowserTheTranscriptClearsTheLidAndNoMore(t *testing.T) {
	srv := screen(t, aScrollingThread())
	c := browserAt(t, srv, "/")

	for _, size := range []struct {
		what          string
		width, height int
		mobile        bool
	}{
		{"a phone", 390, 844, true},
		{"a desktop", 1280, 900, false},
	} {
		c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
			"width": size.width, "height": size.height,
			"deviceScaleFactor": 0, "mobile": size.mobile,
		})
		c.navigate(t, srv.URL+"/")

		require.Equal(t, float64(8), c.eval(t, `
			const lid = document.querySelector(".lid").getBoundingClientRect();
			const s = document.querySelector(".scroll");
			const pad = parseFloat(getComputedStyle(s).paddingTop);
			return Math.round(s.getBoundingClientRect().top + pad - lid.bottom)`),
			"on %s the transcript does not start 8px under the lid", size.what)
	}
}

func TestBrowserTheRailClearsTheLidToo(t *testing.T) {
	srv := screen(t, aScrollingThread())
	c := browserAt(t, srv, "/")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 1280, "height": 900, "deviceScaleFactor": 0, "mobile": false,
	})
	c.navigate(t, srv.URL+"/")

	require.Equal(t, float64(10), c.eval(t, `
		const lid = document.querySelector(".lid").getBoundingClientRect();
		const rail = document.querySelector(".rail");
		const pad = parseFloat(getComputedStyle(rail).paddingTop);
		return Math.round(rail.getBoundingClientRect().top + pad - lid.bottom)`),
		"the first room does not clear the lid by 10")
}

func TestBrowserThePhoneLidReservesAnyTopInset(t *testing.T) {
	srv := screen(t, aScrollingThread())
	c := browserAt(t, srv, "/")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 0, "mobile": true,
	})
	c.send(t, "Emulation.setSafeAreaInsetsOverride", map[string]any{
		"insets": map[string]any{"top": 59, "left": 0, "right": 0, "bottom": 0},
	})
	c.navigate(t, srv.URL+"/")

	require.Equal(t, float64(108), c.eval(t, `
		return Math.round(document.querySelector(".lid").getBoundingClientRect().height)`),
		"the lid does not reserve a top inset when there is one")

	require.Equal(t, true, c.eval(t, `
		const lid = document.querySelector(".lid").getBoundingClientRect();
		return document.querySelector(".roomsheet > summary").getBoundingClientRect().bottom <= lid.bottom`),
		"the room control hangs below the rule")

	require.Equal(t, float64(8), c.eval(t, `
		const lid = document.querySelector(".lid").getBoundingClientRect();
		const s = document.querySelector(".scroll");
		const pad = parseFloat(getComputedStyle(s).paddingTop);
		return Math.round(s.getBoundingClientRect().top + pad - lid.bottom)`),
		"the transcript does not clear the taller lid by 8")

	require.Equal(t, true, c.eval(t, `
		return document.querySelector(".roomsheet > summary").getBoundingClientRect().top >= 59`),
		"the room control sits inside the top inset")

	c.eval(t, `document.querySelector(".roomsheet").open = true; return 1`)
	require.Equal(t, float64(0), c.eval(t, `
		const lid = document.querySelector(".lid").getBoundingClientRect();
		const rail = document.querySelector(".rail").getBoundingClientRect();
		return Math.round(rail.top - lid.bottom)`),
		"the open sheet does not start at the foot of the lid")
}

func TestBrowserTheDockDoesNotStackItsOwnPaddingOnTheInset(t *testing.T) {
	srv := screen(t, aScrollingThread())
	c := browserAt(t, srv, "/")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 0, "mobile": true,
	})
	c.send(t, "Emulation.setSafeAreaInsetsOverride", map[string]any{
		"insets": map[string]any{"top": 59, "left": 0, "right": 0, "bottom": 34},
	})
	c.navigate(t, srv.URL+"/")

	require.Equal(t, "34px", c.eval(t, `
		return getComputedStyle(document.querySelector(".dock")).paddingBottom`),
		"the dock adds its own padding on top of the home indicator's band")

	c.send(t, "Emulation.setSafeAreaInsetsOverride", map[string]any{
		"insets": map[string]any{"top": 0, "left": 0, "right": 0, "bottom": 0},
	})
	c.navigate(t, srv.URL+"/")

	require.Equal(t, "10px", c.eval(t, `
		return getComputedStyle(document.querySelector(".dock")).paddingBottom`),
		"with no inset the dock keeps no floor of its own")
}

func TestBrowserTheShellFillsTheViewport(t *testing.T) {
	srv := screen(t, aScrollingThread())
	c := browserAt(t, srv, "/")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 0, "mobile": true,
	})
	c.send(t, "Emulation.setSafeAreaInsetsOverride", map[string]any{
		"insets": map[string]any{"top": 59, "left": 0, "right": 0, "bottom": 34},
	})
	c.navigate(t, srv.URL+"/")

	require.Equal(t, float64(0), c.eval(t, `
		return Math.round(window.innerHeight - document.body.getBoundingClientRect().bottom)`),
		"the shell stops short of the bottom of the screen")

	require.Equal(t, float64(0), c.eval(t, `
		return Math.round(window.innerHeight - document.querySelector(".dock").getBoundingClientRect().bottom)`),
		"the dock stops short of the bottom of the screen")
}

func TestBrowserTheTranscriptPassesUnderTheDock(t *testing.T) {
	srv := screen(t, aScrollingThread())
	c := browserAt(t, srv, "/")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 0, "mobile": true,
	})
	c.navigate(t, srv.URL+"/")

	c.eval(t, `const s = document.querySelector(".scroll"); s.scrollTop = s.scrollHeight; return 1`)
	c.eval(t, `return new Promise(r => setTimeout(r, 300))`)

	require.Equal(t, float64(14), c.eval(t, `
		const dock = document.querySelector(".dock").getBoundingClientRect();
		const turns = [...document.querySelectorAll(".thread .turn")];
		return Math.round(dock.top - turns[turns.length - 1].getBoundingClientRect().bottom)`),
		"at the foot of the conversation the last thing said is not clear of the dock")

	require.Equal(t, float64(0), c.eval(t, `
		return Math.round(window.innerHeight - document.querySelector(".dock").getBoundingClientRect().bottom)`),
		"the dock is not at the bottom of the screen")

	c.eval(t, `document.querySelector(".scroll").scrollTop = 0; return 1`)
	c.eval(t, `return new Promise(r => setTimeout(r, 300))`)

	require.Equal(t, true, c.eval(t, `
		const dock = document.querySelector(".dock").getBoundingClientRect();
		return [...document.querySelectorAll(".thread .turn")].some(el => {
			const r = el.getBoundingClientRect();
			return r.top < dock.bottom && r.bottom > dock.top });`),
		"nothing passes behind the dock, so its blur has no backdrop")

	require.Equal(t, float64(0), c.eval(t, `
		return Math.round(window.innerHeight - document.querySelector(".dock").getBoundingClientRect().bottom)`),
		"the dock scrolled away from the bottom")
}

func lidTopBand(t *testing.T, c *cdp) []string {
	t.Helper()
	shot := c.send(t, "Page.captureScreenshot", map[string]any{"format": "png"})
	raw, err := base64.StdEncoding.DecodeString(shot["data"].(string))
	require.NoError(t, err)
	img, err := png.Decode(bytes.NewReader(raw))
	require.NoError(t, err)

	var out []string
	for _, x := range []int{5, 100, 195, 300, 385} {
		r, g, b, _ := img.At(x, 1).RGBA()
		out = append(out, fmt.Sprintf("%d,%d,%d", r>>8, g>>8, b>>8))
	}
	return out
}

func TestBrowserTheLidsTopBandHoldsStill(t *testing.T) {
	srv := screen(t, aScrollingThread())
	c := browserAt(t, srv, "/")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 1, "mobile": true,
	})
	c.navigate(t, srv.URL+"/")

	c.eval(t, `document.querySelector(".scroll").scrollTop = 0; return 1`)
	c.eval(t, `return new Promise(r => setTimeout(r, 300))`)
	atTheTop := lidTopBand(t, c)

	c.eval(t, `const s = document.querySelector(".scroll"); s.scrollTop = s.scrollHeight; return 1`)
	c.eval(t, `return new Promise(r => setTimeout(r, 300))`)
	atTheBottom := lidTopBand(t, c)

	for _, got := range [][]string{atTheTop, atTheBottom} {
		for _, c := range got {
			require.Equal(t, got[0], c, "the lid's top band is not one colour across the screen: %v", got)
		}
	}
	require.Equal(t, atTheTop, atTheBottom,
		"the lid's top band changes with what is under it, so the status bar strip beside it cannot match")
	require.Equal(t, "71,46,112", atTheTop[0],
		"the lid's top band is not --purple-bar, which is what the strip beside it takes")
}

// A press the server answers with a redirect takes you there, instead of the
// page it points at being pasted into the room you are standing in.
//
// fetch follows a redirect without telling anybody, so what comes back is a
// whole document. This is the guard that turns that into a navigation, and it
// is the reason "a new appointment" showed a room, and its navigation, inside
// the room.
func TestBrowserARedirectedPressGoesThere(t *testing.T) {
	f := aPile()
	f.items = []squirrel.Item{note(1, "the boiler code", squirrel.ItemKept)}
	srv := screen(t, f)
	c := browserAt(t, srv, "/r/kept")
	c.navigate(t, srv.URL+"/r/kept")
	c.until(t, "the kept note", `!!document.querySelector('input[name="act"][value="open"]')`)

	// A mark that only survives if the page never went anywhere. The redirect
	// lands back in the room the press was made in — see backToTheRoom — so
	// the URL alone cannot tell a navigation from standing still.
	c.eval(t, `window.__stillHere = true; return 1`)
	c.eval(t, `document.querySelector('input[name="act"][value="open"]').form
		.querySelector("button").click(); return 1`)
	c.until(t, "the browser to go", `!window.__stillHere && !!document.querySelector("#thread")`)

	require.Equal(t, "/r/kept", c.eval(t, `return location.pathname`),
		"the press left the room it was made in")
	require.Equal(t, float64(0), c.eval(t, `
		return document.querySelectorAll("#thread nav, #thread .rail").length`),
		"a whole page was pasted into the conversation instead of being followed")
}
