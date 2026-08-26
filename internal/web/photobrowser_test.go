//go:build browser

// The camera, in a real browser, because everything reported broken about it
// was invisible to Go.
//
// The handler had tests and they passed. What none of them touched was the
// only part a person actually uses: a file input in a page, chosen with a
// finger, submitted by a form. Four things were wrong there at once and the
// Go tests could not have found any of them.
package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// cameraScreen is the screen with somewhere to put a photograph, which is what
// makes the camera appear at all.
func cameraScreen(t *testing.T, f *fakeStore, sp *fakeSpool, ph *fakePhotos, c *fakeCoach) *httptest.Server {
	t.Helper()

	opts := Options{
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
		Spool:    sp, Photos: ph,
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

// openCamera points a browser at home, where the slot and its camera are.
func openCamera(t *testing.T, sp *fakeSpool, ph *fakePhotos) (*cdp, *httptest.Server) {
	t.Helper()
	srv := cameraScreen(t, aPile(), sp, ph, nil)
	c := browserAt(t, srv, "/")
	return c, srv
}

// landed is the slot saying a capture went in. It used to be a navigation to
// /?kept=1 — the capture posted the form, the browser left, and the page came
// back at the top with a word on it. It keeps in place now, so the sign that
// it worked is the slot's own answer rather than a change of address.
// It landed when Buddy has said so. The answer used to be a word inside the
// box; it is a turn in the conversation now, and the box being empty is the
// other half of the same fact.
// The exact word varies by the day, the way the slot's own line does — and
// since the box became a conversation it may be a sentence Buddy wrote rather
// than an acknowledgement at all. So this waits for a turn that was not there
// before rather than for a phrasing.
//
// The count is stamped on the page before the press. A predicate that only
// asked "has Buddy said something" was satisfied by whatever was already on
// screen and returned instantly, which is how a spool assertion ran before the
// write it was about.
const marking = `window.__before = document.querySelectorAll("#thread .turn").length; return 1`

const landed = `document.querySelectorAll("#thread .turn").length > window.__before`

// heldPhoto asks the page's own database whether a photograph is being held.
//
// The preview and the hold are two moments, not one: the picture is drawn the
// instant it is chosen so the screen never lags behind the finger, and the
// write lands afterwards. Anything that means to test the hold has to wait for
// the hold.
const heldPhoto = `new Promise(done => {
	const open = indexedDB.open("squirrel-photo", 1);
	open.onerror = () => done(false);
	open.onsuccess = () => {
		try {
			const got = open.result.transaction("photo", "readonly").objectStore("photo").get("pending");
			got.onsuccess = () => done(!!got.result);
			got.onerror = () => done(false);
		} catch { done(false); }
	};
})`

// aPhotograph writes a real file for the browser to attach. The bytes are a
// one-pixel JPEG's worth of nothing: what is under test is the plumbing, and
// the store never looks inside.
func aPhotograph(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "letter.jpg")
	require.NoError(t, os.WriteFile(path, []byte("\xff\xd8\xff\xe0 not really a jpeg"), 0o600))
	return path
}

// attach puts a file on the input the way a person's finger does — through the
// browser, not by setting a value the page could never set itself.
func (c *cdp) attach(t *testing.T, selector, path string) {
	t.Helper()
	c.send(t, "DOM.enable", nil)
	doc := c.send(t, "DOM.getDocument", map[string]any{"depth": -1})
	root := doc["root"].(map[string]any)["nodeId"]

	found := c.send(t, "DOM.querySelector", map[string]any{
		"nodeId": root, "selector": selector,
	})
	node, ok := found["nodeId"].(float64)
	require.True(t, ok && node != 0, "no %s on the page", selector)

	c.send(t, "DOM.setFileInputFiles", map[string]any{
		"nodeId": found["nodeId"], "files": []string{path},
	})
}

// The whole point of the feature: photograph a letter, press keep, and have it
// arrive. This was reported as "saving it doesn't work".
func TestBrowserAPhotographIsKept(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	c, _ := openCamera(t, sp, ph)

	c.attach(t, ".slot input[name=photo]", aPhotograph(t))
	c.eval(t, marking)
	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.until(t, "the capture to land", landed)

	require.Len(t, sp.written, 1, "nothing was spooled")
	require.NotEmpty(t, sp.written[0].PhotoName, "the capture carried no photograph")
	require.Len(t, ph.kept, 1, "no bytes reached the store")
}

// Words and a photograph together, which is the ordinary case: you photograph
// the letter and say what it is about.
func TestBrowserAPhotographKeepsItsWords(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	c, _ := openCamera(t, sp, ph)

	c.eval(t, `document.querySelector(".slot textarea").value = "the tax letter"`)
	c.attach(t, ".slot input[name=photo]", aPhotograph(t))
	c.eval(t, marking)
	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.until(t, "the capture to land", landed)

	require.Len(t, sp.written, 1)
	require.Equal(t, "the tax letter", sp.written[0].Text)
	require.NotEmpty(t, sp.written[0].PhotoName)
}

// The same, with the worker in control — which is every visit after the first
// and therefore every visit the owner makes. The worker intercepts POST
// /capture to hold it when there is no network, so a photograph goes through
// its hands on the way out and nothing until now had ever put one there.
func TestBrowserAPhotographSurvivesTheWorker(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	c, srv := openCamera(t, sp, ph)

	// Registered first, then a fresh load: the worker claims its clients on
	// activate, but a page that started loading before it was active is not
	// one of them, and waiting for a controller on that page waits forever.
	c.eval(t, `await navigator.serviceWorker.ready`)
	c.navigate(t, srv.URL+"/")
	c.until(t, "the worker to be controlling the page", `!!navigator.serviceWorker.controller`)

	c.attach(t, ".slot input[name=photo]", aPhotograph(t))
	c.eval(t, marking)
	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.until(t, "the capture to land", landed)

	require.Len(t, sp.written, 1, "nothing was spooled")
	require.NotEmpty(t, sp.written[0].PhotoName, "the worker dropped the photograph")
	require.Len(t, ph.kept, 1, "no bytes reached the store")
}

// Choosing one has to be visible. The input is a pixel wide and invisible by
// design, so without this the camera looks exactly the same before and after —
// which is what "when I take a photo it does not show it" was.
func TestBrowserChoosingAPhotographSaysSo(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	c, _ := openCamera(t, sp, ph)

	before := c.eval(t, `return (`+visible+`)(".slot .gotphoto")`)
	require.Equal(t, false, before, "the slot claimed a photograph before there was one")

	c.attach(t, ".slot input[name=photo]", aPhotograph(t))
	c.until(t, "the slot to show the photograph",
		`(`+visible+`)(".slot .gotphoto")`)
}

// And taking it off again, because a photograph attached by accident must not
// be a capture you have to undo afterwards.
func TestBrowserAPhotographCanBeTakenOffAgain(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	c, _ := openCamera(t, sp, ph)

	c.attach(t, ".slot input[name=photo]", aPhotograph(t))
	c.until(t, "the slot to show the photograph",
		`(`+visible+`)(".slot .gotphoto")`)

	c.eval(t, `document.querySelector(".slot .unphoto").click()`)
	c.until(t, "the photograph to go",
		`!(`+visible+`)(".slot .gotphoto")`)

	c.eval(t, `document.querySelector(".slot textarea").value = "words only"`)
	c.eval(t, marking)
	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.until(t, "the capture to land", landed)

	require.Len(t, sp.written, 1)
	require.Empty(t, sp.written[0].PhotoName, "a removed photograph was kept anyway")
	require.Empty(t, ph.kept)
}

// The one that explains "saving it doesn't work".
//
// Choosing a photograph on a phone hands the screen to another app, and an
// installed app that is handed away can be reclaimed while it waits. It comes
// back reloaded: the input is empty, the page looks exactly as it did — it
// never looked any different — and the press that follows keeps the words with
// no photograph on them. A reload is what that is, from the page's side.
func TestBrowserAPhotographSurvivesTheAppBeingReclaimed(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	c, srv := openCamera(t, sp, ph)

	c.attach(t, ".slot input[name=photo]", aPhotograph(t))
	c.until(t, "the slot to show the photograph",
		`(`+visible+`)(".slot .gotphoto")`)
	// And then wait for it to actually be held, which is a different moment.
	// The picture appears the instant it is chosen and the write to IndexedDB
	// lands after that, so navigating in between tests the race rather than the
	// restore. CI is slow enough to lose that race and did.
	c.until(t, "the photograph to be held", heldPhoto)

	c.navigate(t, srv.URL+"/")
	c.until(t, "the photograph to come back",
		`(`+visible+`)(".slot .gotphoto")`)
	require.Equal(t, float64(1), c.eval(t,
		`return document.querySelector(".slot input[name=photo]").files.length`),
		"the photograph was shown but not put back on the input")

	c.eval(t, marking)
	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.until(t, "the capture to land", landed)

	require.Len(t, sp.written, 1)
	require.NotEmpty(t, sp.written[0].PhotoName, "the photograph did not survive the reload")
	require.Len(t, ph.kept, 1)
}

// And it is offered back exactly once. A photograph already kept must not come
// back on the next visit, or one press keeps it twice.
func TestBrowserAKeptPhotographIsNotOfferedAgain(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	c, srv := openCamera(t, sp, ph)

	c.attach(t, ".slot input[name=photo]", aPhotograph(t))
	c.until(t, "the slot to show the photograph",
		`(`+visible+`)(".slot .gotphoto")`)
	c.until(t, "the photograph to be held", heldPhoto)
	c.eval(t, marking)
	c.eval(t, `document.querySelector(".slot .post").click()`)
	// The hold is dropped by the same press that keeps it, rather than by the
	// page that used to load afterwards. There is no page afterwards now, and
	// a photograph still on hold after it has been kept is one press from
	// being kept twice.
	c.until(t, "the capture to land", landed)
	c.until(t, "the hold to be let go", `!(await (`+heldPhoto+`))`)
	require.Len(t, sp.written, 1)

	c.navigate(t, srv.URL+"/")
	c.until(t, "the slot", `!!document.querySelector(".slot input[name=photo]")`)
	require.Equal(t, false, c.eval(t,
		`return (`+visible+`)(".slot .gotphoto")`),
		"a photograph already kept was offered back")

	c.eval(t, `const t = document.querySelector(".slot textarea");
		t.value = "a later thought"; t.dispatchEvent(new Event("input")); return 1`)
	c.eval(t, marking)
	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.until(t, "the second capture to land", landed)

	require.Len(t, sp.written, 2)
	require.Empty(t, sp.written[1].PhotoName, "the photograph was kept a second time")
	require.Len(t, ph.kept, 1)
}

// A photograph you already have is the other half of the case — the letter you
// photographed this morning, the screenshot somebody sent you. `capture` on
// the input takes the gallery away on a phone: it does not prefer the camera,
// it forbids everything else.
func TestBrowserAnExistingPhotographCanBeChosen(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	c, _ := openCamera(t, sp, ph)

	require.Equal(t, false, c.eval(t,
		`return document.querySelector(".slot input[name=photo]").hasAttribute("capture")`),
		"the input forbids the gallery")
	require.Equal(t, "image/*", c.eval(t,
		`return document.querySelector(".slot input[name=photo]").getAttribute("accept")`))
}

// Closing the sheet with a coach actually behind it. The two tests that
// covered this ran with no coach configured, so the sheet they closed was the
// short one — no spend in the lid, no conversation in it.

// aLongConversation is what the sheet looks like after a few minutes of use,
// which is the only state the close button was ever reported broken in.
func aLongConversation() *fakeCoach {
	c := &fakeCoach{spent: "€0.61", ceiling: "€10"}
	for i := 0; i < 14; i++ {
		c.talk = append(c.talk, Exchange{
			Said:    "I still cannot make a start on the tax thing and I do not know why",
			Replied: "Open the envelope and read the first line. That is the whole of it.",
		})
	}
	return c
}

// Closing on a phone, after a conversation long enough to scroll.
//
// The sheet is a bottom sheet under 620px and it is what scrolls, and the lid
// holding the close button is its first child — so reading Buddy's answer
// carries the way out off the top of the screen. Escape needs a keyboard and
// the backdrop is a strip above an 88vh sheet, which leaves nothing to press.

// The close button has to be somewhere a thumb can reach it, which is a
// different question from whether the handler fires. A control pushed out of
// its own lid by the spend beside it is a control that cannot be pressed.

// The close button, in a short window.
//
// Read what this does not prove. An emulated viewport has no browser chrome,
// so vh and dvh resolve to the same number here and this test passes against
// the code that shipped the bug. It is a guard on the sticky lid, not evidence
// about a phone; the unit is guarded separately, in the stylesheet, below.

// The sheet is measured in dvh, and this is the only check on it that means
// anything.
//
// vh is the *large* viewport — the height the page would have if the browser's
// own chrome were hidden. With an address bar on screen, which is most of the
// time on a phone, a sheet anchored to the bottom at 88vh reaches higher than
// the window actually shows, and the first thing over the top edge is the lid,
// where the close button lives.
//
// It cannot be caught by driving a browser here: the emulator has no chrome to
// retract, so both units agree and every geometric test passes either way.
// That is exactly how this shipped twice. So the assertion is on the source —
// blunt, and honest about being blunt.
func TestTheSheetIsMeasuredInDynamicViewportHeight(t *testing.T) {
	css, err := staticFS.ReadFile("static/pile.css")
	require.NoError(t, err)

	// Comments stripped first. The prose above this rule explains dvh at
	// length, so a check on the raw bytes passes whatever the code does — which
	// is what the first version of this test did.
	naked := comments.ReplaceAllString(string(css), "")
	require.Regexp(t, `max-height:\s*[0-9.]+dvh`, naked,
		"the sheet went back to vh, which is not the height a phone shows")
	require.NotRegexp(t, `dialog\.coachsheet[^}]*max-height:\s*[0-9.]+vh\s*;\s*}`, naked,
		"a vh max-height is the last word on the sheet's height")
}

// CSS comments, for the check above. /* ... */ only; this stylesheet has no
// other kind.
var comments = regexp.MustCompile(`(?s)/\*.*?\*/`)
