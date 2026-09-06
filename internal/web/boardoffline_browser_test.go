//go:build browser

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func screenWithACamera(t *testing.T, f *fakeStore, ph *fakePhotos) *httptest.Server {
	t.Helper()

	m := &serveMux{mux: http.NewServeMux()}
	require.NoError(t, Mount(m, f, Options{
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
		Photos:   ph,
	}))

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

func TestBrowserWordsTypedOnTheBoardWithNoNetworkAreHeld(t *testing.T) {
	srv := screenWith(t, aPile(), nil)
	c := browserAt(t, srv, "/?bay=notes")
	waitForTheWorker(t, c, srv.URL+"/?bay=notes")

	srv.Close()

	require.Equal(t, "offline=1", c.eval(t, `
		const form = document.querySelector("form.blankstrip");
		form.querySelector(".words").value = "ask the garage about the rattle";
		const body = new URLSearchParams(
			[...new FormData(form)].filter(([, v]) => typeof v === "string"));
		try {
			const res = await fetch(form.action, {
				method: "POST",
				headers: { "Content-Type": "application/x-www-form-urlencoded" },
				body,
			});
			return new URL(res.url).search.includes("offline=1") ? "offline=1" : new URL(res.url).search;
		} catch (e) {
			return "the network error reached the page: " + e.message;
		}`),
		"nothing intercepted the route the board actually posts to, so the words are gone")

	require.Equal(t, "notes", c.eval(t, `
		return await new Promise(resolve => {
			const open = indexedDB.open("squirrel-held", 1);
			open.onerror = () => resolve("");
			open.onsuccess = () => {
				const db = open.result;
				if (!db.objectStoreNames.contains("notes")) return resolve("");
				const req = db.transaction("notes").objectStore("notes").getAll();
				req.onsuccess = () => {
					const held = req.result.find(n =>
						(n.fields || []).some(([, v]) => String(v).includes("the rattle")));
					if (!held) return resolve("");
					if (!held.captureKey) return resolve("no key");
					const bay = (held.fields || []).find(([k]) => k === "bay");
					resolve(bay ? bay[1] : "no bay");
				};
				req.onerror = () => resolve("");
			};
		});`),
		"the rack the words were typed in did not survive the hold")
}

func TestBrowserAPhotographChosenOnTheBoardIsHeldOnTheDevice(t *testing.T) {
	srv := screenWithACamera(t, aPile(), &fakePhotos{})
	c := browserAt(t, srv, "/?bay=notes")

	c.until(t, "the camera to be there", `!!document.querySelector('form.blankstrip input[name="photo"]')`)

	require.Equal(t, "application/x-www-form-urlencoded", c.eval(t, `
		return document.querySelector("form.blankstrip").enctype;`),
		"a words-only capture goes out as multipart, which the worker cannot hold")

	require.Equal(t, true, c.eval(t, `
		const input = document.querySelector('form.blankstrip input[name="photo"]');
		const carrier = new DataTransfer();
		carrier.items.add(new File([new Uint8Array([1, 2, 3])], "IMG_0042.jpg", { type: "image/jpeg" }));
		input.files = carrier.files;
		input.dispatchEvent(new Event("change"));
		return true;`))

	c.until(t, "the photograph to be drawn", `
		(() => {
			const shown = document.querySelector(".gotphoto");
			return !!shown && !shown.hidden && !!shown.querySelector("img").getAttribute("src");
		})()`)

	require.Equal(t, "multipart/form-data", c.eval(t, `
		return document.querySelector("form.blankstrip").enctype;`),
		"a capture carrying a photograph must go out as multipart")

	c.until(t, "the photograph to be held", `
		(async () => await new Promise(resolve => {
			const open = indexedDB.open("squirrel-photo", 1);
			open.onerror = () => resolve(false);
			open.onsuccess = () => {
				const db = open.result;
				if (!db.objectStoreNames.contains("photo")) return resolve(false);
				const req = db.transaction("photo").objectStore("photo").get("pending");
				req.onsuccess = () => resolve(!!req.result);
				req.onerror = () => resolve(false);
			};
		}))()`)

	c.navigate(t, srv.URL+"/?bay=notes")

	c.until(t, "the photograph to come back after the app was reclaimed", `
		(() => {
			const input = document.querySelector('form.blankstrip input[name="photo"]');
			const shown = document.querySelector(".gotphoto");
			return !!input && input.files.length === 1 && !!shown && !shown.hidden;
		})()`)
}
