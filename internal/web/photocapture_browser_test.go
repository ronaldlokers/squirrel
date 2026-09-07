//go:build browser

package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const putAPhotographOnTheInput = `
	const input = document.querySelector('form.blankstrip input[name="photo"]');
	const carrier = new DataTransfer();
	carrier.items.add(new File(["jpegbytes"], "IMG_0042.jpg", { type: "image/jpeg" }));
	input.files = carrier.files;
	input.dispatchEvent(new Event("change"));
	return true;`

func TestBrowserAPhotographChosenOnTheBoardReachesTheStore(t *testing.T) {
	f, ph := &fakeStore{}, &fakePhotos{}
	srv := screenWithACamera(t, f, ph)
	c := browserAt(t, srv, "/?bay=notes")
	c.until(t, "the camera", `!!document.querySelector('form.blankstrip input[name="photo"]')`)

	require.Equal(t, true, c.eval(t, putAPhotographOnTheInput))
	c.until(t, "the photograph to be drawn", `
		(() => {
			const shown = document.querySelector(".gotphoto");
			return !!shown && !shown.hidden;
		})()`)

	c.eval(t, `
		document.querySelector("form.blankstrip .words").value = "the tax letter";
		document.querySelector("form.blankstrip").requestSubmit();
		return true;`)
	c.until(t, "the board to come back", `location.search.includes("kept=1")`)

	require.Eventually(t, func() bool { return len(ph.kept) == 1 },
		4*time.Second, 50*time.Millisecond,
		"the photograph never reached the volume")
	require.Equal(t, "jpegbytes", ph.kept[0])

	require.Eventually(t, func() bool { return len(f.items) == 1 }, 4*time.Second, 50*time.Millisecond)
	require.Equal(t, "the tax letter", f.items[0].RawText)
	require.Equal(t, "photo-1.jpg", f.items[0].PhotoName,
		"the note was kept without the photograph it was captured with")
}

func TestBrowserTakingThePhotographOffSendsTheWordsAlone(t *testing.T) {
	f, ph := &fakeStore{}, &fakePhotos{}
	srv := screenWithACamera(t, f, ph)
	c := browserAt(t, srv, "/?bay=notes")
	c.until(t, "the camera", `!!document.querySelector('form.blankstrip input[name="photo"]')`)

	require.Equal(t, true, c.eval(t, putAPhotographOnTheInput))
	c.until(t, "the photograph to be drawn", `
		(() => {
			const shown = document.querySelector(".gotphoto");
			return !!shown && !shown.hidden;
		})()`)

	c.eval(t, `document.querySelector(".unphoto").click(); return true;`)
	c.until(t, "the photograph to be gone", `
		(() => {
			const shown = document.querySelector(".gotphoto");
			const input = document.querySelector('form.blankstrip input[name="photo"]');
			return !!shown && shown.hidden && input.files.length === 0;
		})()`)

	require.Equal(t, "application/x-www-form-urlencoded", c.eval(t, `
		return document.querySelector("form.blankstrip").enctype;`),
		"the form still claims multipart with nothing to carry, so the worker cannot hold it offline")

	c.eval(t, `
		document.querySelector("form.blankstrip .words").value = "the tax letter";
		document.querySelector("form.blankstrip").requestSubmit();
		return true;`)

	require.Eventually(t, func() bool { return len(f.items) == 1 }, 4*time.Second, 50*time.Millisecond)
	require.Equal(t, "the tax letter", f.items[0].RawText)
	require.Empty(t, f.items[0].PhotoName,
		"a photograph that was taken off was kept anyway")
	require.Empty(t, ph.kept, "a photograph that was taken off reached the volume")
}
