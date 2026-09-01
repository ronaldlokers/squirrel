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
	"testing"

	"github.com/stretchr/testify/require"
)

// cameraScreen is the screen with somewhere to put a photograph, which is what
// makes the camera appear at all.
func cameraScreen(t *testing.T, f *fakeStore, sp *fakeSpool, ph *fakePhotos) *httptest.Server {
	t.Helper()

	opts := Options{
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
		Spool:    sp, Photos: ph,
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
	srv := cameraScreen(t, aPile(), sp, ph)
	c := browserAt(t, srv, "/r/everything")
	return c, srv
}

// marking stamps how many turns are on the page before the press, and landed
// waits for one more.
//
// A capture has landed when Buddy has said so, and the page keeps in place — so
// the sign that it worked is a new turn rather than a change of address.
//
// It waits for a turn rather than for a phrasing: the acknowledgement varies by
// the day, and since the box became a conversation it may be a sentence Buddy
// wrote rather than an acknowledgement at all.
//
// The count has to be stamped first. A predicate that only asked "has Buddy said
// something" was satisfied by whatever was already on screen and returned
// instantly, which is how a spool assertion ran before the write it was about.
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
	c.navigate(t, srv.URL+"/r/everything")
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

	c.navigate(t, srv.URL+"/r/everything")
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

	c.navigate(t, srv.URL+"/r/everything")
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
