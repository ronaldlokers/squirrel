//go:build browser

package web

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const heldNotes = `
	return await new Promise(resolve => {
		const open = indexedDB.open("squirrel-held", 1);
		open.onerror = () => resolve(-1);
		open.onsuccess = () => {
			const db = open.result;
			if (!db.objectStoreNames.contains("notes")) return resolve(0);
			const req = db.transaction("notes").objectStore("notes").getAll();
			req.onsuccess = () => resolve(req.result.length);
			req.onerror = () => resolve(-1);
		};
	});`

func signOut(t *testing.T, c *cdp, srv string) {
	t.Helper()
	c.navigate(t, srv+"/me")
	c.until(t, "the way out to be there", `!!document.querySelector('form[action="/auth/out"]')`)
	c.eval(t, `document.querySelector('form[action="/auth/out"]').requestSubmit(); return true;`)
	c.until(t, "the gate", `location.pathname.startsWith("/auth")`)
}

func TestBrowserAPhotographNeverKeptDoesNotOutliveTheSession(t *testing.T) {
	srv := screen(t, aPile())
	c := browserAt(t, srv, "/notes")

	require.Equal(t, true, c.eval(t, `
		await new Promise(resolve => {
			const open = indexedDB.open("squirrel-photo", 1);
			open.onupgradeneeded = () => open.result.createObjectStore("photo");
			open.onsuccess = () => {
				const put = open.result.transaction("photo", "readwrite").objectStore("photo")
					.put(new Blob(["a picture of the tax letter"]), "pending");
				put.onsuccess = () => resolve(true);
				put.onerror = () => resolve(false);
			};
		});
		return true;`))

	signOut(t, c, srv.URL)

	require.Equal(t, false, c.eval(t, `
		return (await indexedDB.databases()).some(d => d.name === "squirrel-photo");`),
		"a photograph chosen and never kept is still on this device after signing out")
}

func TestBrowserSigningOutDoesNotDropACaptureThatCouldNotBeDelivered(t *testing.T) {
	srv := screen(t, aPile())
	c := browserAt(t, srv, "/notes")
	waitForTheWorker(t, c, srv.URL+"/notes")

	require.Equal(t, true, c.eval(t, `
		await new Promise(resolve => {
			const open = indexedDB.open("squirrel-held", 1);
			open.onupgradeneeded = () => open.result.createObjectStore("notes", { autoIncrement: true });
			open.onsuccess = () => {
				const put = open.result.transaction("notes", "readwrite").objectStore("notes")
					.add({ fields: [["words", "ask the garage about the rattle"]], action: "/photo/1" });
				put.onsuccess = () => resolve(true);
				put.onerror = () => resolve(false);
			};
		});
		return true;`))
	require.Equal(t, float64(1), c.eval(t, heldNotes))

	signOut(t, c, srv.URL)

	require.Equal(t, float64(1), c.eval(t, heldNotes),
		"the only copy of something you said was deleted to tidy up the browser")
}
