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
		IdentityHeader: "X-Authentik-Username", Identity: "ronald",
		Owner: func() int64 { return 1 }, Spool: sp, Photos: ph,
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

// openCamera points a browser at home, where the slot and its camera are.
func openCamera(t *testing.T, sp *fakeSpool, ph *fakePhotos) (*cdp, *httptest.Server) {
	t.Helper()
	srv := cameraScreen(t, aPile(), sp, ph, nil)
	c := browserAt(t, srv, "/")
	return c, srv
}

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
	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.until(t, "the capture to land", `location.search.includes("kept")`)

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
	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.until(t, "the capture to land", `location.search.includes("kept")`)

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
	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.until(t, "the capture to land", `location.search.includes("kept")`)

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

	before := c.eval(t, `return !!document.querySelector(".slot .gotphoto:not([hidden])")`)
	require.Equal(t, false, before, "the slot claimed a photograph before there was one")

	c.attach(t, ".slot input[name=photo]", aPhotograph(t))
	c.until(t, "the slot to show the photograph",
		`!!document.querySelector(".slot .gotphoto:not([hidden])")`)
}

// And taking it off again, because a photograph attached by accident must not
// be a capture you have to undo afterwards.
func TestBrowserAPhotographCanBeTakenOffAgain(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	c, _ := openCamera(t, sp, ph)

	c.attach(t, ".slot input[name=photo]", aPhotograph(t))
	c.until(t, "the slot to show the photograph",
		`!!document.querySelector(".slot .gotphoto:not([hidden])")`)

	c.eval(t, `document.querySelector(".slot .unphoto").click()`)
	c.until(t, "the photograph to go",
		`!document.querySelector(".slot .gotphoto:not([hidden])")`)

	c.eval(t, `document.querySelector(".slot textarea").value = "words only"`)
	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.until(t, "the capture to land", `location.search.includes("kept")`)

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
		`!!document.querySelector(".slot .gotphoto:not([hidden])")`)

	c.navigate(t, srv.URL+"/")
	c.until(t, "the photograph to come back",
		`!!document.querySelector(".slot .gotphoto:not([hidden])")`)
	require.Equal(t, float64(1), c.eval(t,
		`return document.querySelector(".slot input[name=photo]").files.length`),
		"the photograph was shown but not put back on the input")

	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.until(t, "the capture to land", `location.search.includes("kept")`)

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
		`!!document.querySelector(".slot .gotphoto:not([hidden])")`)
	c.eval(t, `document.querySelector(".slot .post").click()`)
	// The whole page, not just the URL: the stash is dropped by the script on
	// the page that confirms the capture, the same way the worker deletes a
	// held note the moment it lands. Leaving before that page has run is
	// leaving before the acknowledgement, which is a thing a person can do and
	// which costs an offer of the photograph back — visibly, and refusable.
	c.until(t, "the confirmation to have loaded",
		`location.search.includes("kept") && document.readyState === "complete"`)
	require.Len(t, sp.written, 1)

	c.navigate(t, srv.URL+"/")
	// Long enough for a restore to have happened if one were going to.
	c.until(t, "the slot", `!!document.querySelector(".slot input[name=photo]")`)
	require.Equal(t, false, c.eval(t,
		`return !!document.querySelector(".slot .gotphoto:not([hidden])")`),
		"a photograph already kept was offered back")

	c.eval(t, `document.querySelector(".slot textarea").value = "a later thought"`)
	c.eval(t, `document.querySelector(".slot .post").click()`)
	c.until(t, "the second capture to land", `location.search.includes("kept")`)

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
func TestBrowserClosingTheCoachWithACoachBehindIt(t *testing.T) {
	coach := &fakeCoach{
		spent: "€0.61", ceiling: "€10",
		talk: []Exchange{{Said: "I am stuck", Replied: "Start with the envelope."}},
	}
	srv := cameraScreen(t, aPile(), &fakeSpool{}, &fakePhotos{}, coach)
	c := browserAt(t, srv, "/pile")

	c.eval(t, `document.querySelector(".askacorn").click()`)
	c.until(t, "the sheet to open", `!!document.querySelector("dialog.coachsheet[open]")`)

	c.eval(t, `document.querySelector("dialog.coachsheet .shut").click()`)
	c.until(t, "the sheet to close", `!document.querySelector("dialog.coachsheet[open]")`)
	require.Equal(t, "/pile", c.eval(t, `return location.pathname`),
		"closing the sheet navigated instead of closing")
}

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
func TestBrowserTheCloseButtonStaysReachableOnAPhone(t *testing.T) {
	srv := cameraScreen(t, aPile(), &fakeSpool{}, &fakePhotos{}, aLongConversation())
	c := browserAt(t, srv, "/pile")

	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 0, "mobile": true,
	})
	c.eval(t, `document.querySelector(".askacorn").click()`)
	c.until(t, "the sheet to open", `!!document.querySelector("dialog.coachsheet[open]")`)

	// Read to the bottom, the way you do when the answer is down there.
	c.eval(t, `
		const sheet = document.querySelector("dialog.coachsheet .sheet");
		sheet.scrollTop = sheet.scrollHeight;`)

	require.Equal(t, true, c.eval(t, `
		const shut = document.querySelector("dialog.coachsheet .shut");
		const r = shut.getBoundingClientRect();
		return r.top >= 0 && r.bottom <= innerHeight && r.height > 0;`),
		"the close button scrolled off the sheet")

	require.Equal(t, true, c.eval(t, `
		const shut = document.querySelector("dialog.coachsheet .shut");
		const r = shut.getBoundingClientRect();
		const hit = document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2);
		return shut.contains(hit);`),
		"the close button is covered by the conversation scrolled under it")
}

// The close button has to be somewhere a thumb can reach it, which is a
// different question from whether the handler fires. A control pushed out of
// its own lid by the spend beside it is a control that cannot be pressed.
func TestBrowserTheCloseButtonIsInsideTheSheet(t *testing.T) {
	coach := &fakeCoach{
		spent: "€0.61", ceiling: "€10",
		talk: []Exchange{{Said: "I am stuck", Replied: "Start with the envelope."}},
	}
	srv := cameraScreen(t, aPile(), &fakeSpool{}, &fakePhotos{}, coach)
	c := browserAt(t, srv, "/pile")

	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 844, "deviceScaleFactor": 0, "mobile": true,
	})
	c.eval(t, `document.querySelector(".askacorn").click()`)
	c.until(t, "the sheet to open", `!!document.querySelector("dialog.coachsheet[open]")`)

	inside := c.eval(t, `
		const shut = document.querySelector("dialog.coachsheet .shut").getBoundingClientRect();
		const lid = document.querySelector("dialog.coachsheet .sheetlid").getBoundingClientRect();
		return shut.right <= lid.right + 1 && shut.left >= lid.left - 1
			&& shut.width > 0 && shut.height > 0;`)
	require.Equal(t, true, inside, "the close button is not inside the lid")

	// And that a press at its own centre reaches it, rather than something
	// drawn over the top.
	onTop := c.eval(t, `
		const shut = document.querySelector("dialog.coachsheet .shut");
		const r = shut.getBoundingClientRect();
		const hit = document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2);
		return shut.contains(hit);`)
	require.Equal(t, true, onTop, "something else is drawn over the close button")
}

// The close button, in a short window.
//
// Read what this does not prove. An emulated viewport has no browser chrome,
// so vh and dvh resolve to the same number here and this test passes against
// the code that shipped the bug. It is a guard on the sticky lid, not evidence
// about a phone; the unit is guarded separately, in the stylesheet, below.
func TestBrowserTheCloseButtonSurvivesAShortWindow(t *testing.T) {
	srv := cameraScreen(t, aPile(), &fakeSpool{}, &fakePhotos{}, aLongConversation())
	c := browserAt(t, srv, "/pile")

	// 390 wide, and short: what is left of an iPhone once the address bar and
	// the home indicator have taken their share.
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 600, "deviceScaleFactor": 0, "mobile": true,
	})
	c.eval(t, `document.querySelector(".askacorn").click()`)
	c.until(t, "the sheet to open", `!!document.querySelector("dialog.coachsheet[open]")`)
	c.eval(t, `
		const s = document.querySelector("dialog.coachsheet .sheet");
		s.scrollTop = s.scrollHeight; return 1;`)

	require.Equal(t, true, c.eval(t, `
		const r = document.querySelector("dialog.coachsheet .shut").getBoundingClientRect();
		return r.top >= 0 && r.bottom <= innerHeight && r.left >= 0 && r.right <= innerWidth;`),
		"the close button is off the visible window")

	// And it is the thing at its own centre, not something drawn over it.
	require.Equal(t, true, c.eval(t, `
		const shut = document.querySelector("dialog.coachsheet .shut");
		const r = shut.getBoundingClientRect();
		return shut.contains(document.elementFromPoint(r.left + r.width/2, r.top + r.height/2));`),
		"something covers the close button")

	// Pressing it closes.
	c.eval(t, `document.querySelector("dialog.coachsheet .shut").click()`)
	c.until(t, "the sheet to close", `!document.querySelector("dialog.coachsheet[open]")`)
}

// The strip of backdrop above the sheet is the other way out, and it has to
// stay a strip: a sheet that fills the window leaves nothing to press.
func TestBrowserTheBackdropIsStillReachableOnAShortPhone(t *testing.T) {
	srv := cameraScreen(t, aPile(), &fakeSpool{}, &fakePhotos{}, aLongConversation())
	c := browserAt(t, srv, "/pile")

	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 390, "height": 600, "deviceScaleFactor": 0, "mobile": true,
	})
	c.eval(t, `document.querySelector(".askacorn").click()`)
	c.until(t, "the sheet to open", `!!document.querySelector("dialog.coachsheet[open]")`)

	gap := c.eval(t, `
		const r = document.querySelector("dialog.coachsheet .sheet").getBoundingClientRect();
		return Math.round(r.top);`)
	require.GreaterOrEqual(t, gap, float64(44),
		"no room above the sheet to press the backdrop")
}

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
