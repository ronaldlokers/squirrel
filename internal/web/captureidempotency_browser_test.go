//go:build browser

package web

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBrowserAHeldCaptureCarriesAKey(t *testing.T) {
	c, srv := open(t, aPile())
	c.navigate(t, srv.URL+"/r/everything")
	waitForTheWorker(t, c, srv.URL+"/r/everything")

	srv.Close()

	require.Equal(t, true, c.eval(t, `
		const res = await fetch("/capture", {
			method: "POST",
			headers: { "Content-Type": "application/x-www-form-urlencoded" },
			body: new URLSearchParams({ text: "ask the garage about the rattle" }),
		});
		return new URL(res.url).search.includes("held");`),
		"the worker answered, and said so")

	require.Equal(t, true, c.eval(t, `
		return await new Promise(resolve => {
			const open = indexedDB.open("squirrel-held", 1);
			open.onerror = () => resolve(false);
			open.onsuccess = () => {
				const db = open.result;
				if (!db.objectStoreNames.contains("notes")) return resolve(false);
				const req = db.transaction("notes").objectStore("notes").getAll();
				req.onsuccess = () => {
					const held = req.result.find(n => n.text.includes("the rattle"));
					resolve(!!held && typeof held.captureKey === "string" && held.captureKey.length > 0);
				};
				req.onerror = () => resolve(false);
			};
		});`), "the note held for a later retry carries no key to repeat on it")
}
