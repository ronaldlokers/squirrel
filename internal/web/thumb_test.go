package web

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A directory holding one photograph and its smaller copy, with different
// bytes in each so a test can tell which one came back.
func photosOnDisk(t *testing.T) *fakePhotos {
	t.Helper()
	dir := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 40, 30))
	for y := range 30 {
		for x := range 40 {
			img.Set(x, y, color.RGBA{R: uint8(x * 6), G: 40, B: 90, A: 255})
		}
	}
	var big bytes.Buffer
	require.NoError(t, jpeg.Encode(&big, img, &jpeg.Options{Quality: 95}))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "photo-1.jpg"), big.Bytes(), 0o600))
	// The copy is deliberately not the same bytes, so "which one did it serve"
	// is answerable.
	require.NoError(t, os.WriteFile(filepath.Join(dir, squirrel.ThumbName("photo-1.jpg")),
		[]byte("the smaller copy"), 0o600))
	return &fakePhotos{dir: dir}
}

// The handler directly, with the path value set: the test mux matches by
// prefix and cannot resolve a `{id}` wildcard. Same device photomissing_test.go
// uses, and for the same reason.
func askForThumb(t *testing.T, f *fakeStore, ph *fakePhotos, id string) *httptest.ResponseRecorder {
	t.Helper()
	opts := Options{
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
		Spool:    &fakeSpool{}, Photos: ph,
	}
	r := httptest.NewRequest("GET", "/photo/"+id+"/thumb", nil)
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "a-token"})
	r.SetPathValue("id", id)
	w := httptest.NewRecorder()
	thumbHandler(f, opts)(w, asking(r))
	return w
}

func storeWithPhoto(id int64) *fakeStore {
	return &fakeStore{items: []squirrel.Item{{
		ID: id, RawText: "the boiler letter", Kind: squirrel.ItemNote,
		State: squirrel.ItemOpen, PhotoName: "photo-1.jpg", PhotoType: "image/jpeg",
	}}}
}

// This is the bandwidth half of the fix: the card is 260 pixels wide and the
// original is a photograph from a phone.
func TestACardAsksForTheSmallerCopy(t *testing.T) {
	f := storeWithPhoto(7)
	fDrew := drewIn(t, f, "pile")

	shown := string(fDrew[len(fDrew)-1].Shown)
	require.Contains(t, shown, `"photo":"/photo/7"`)

	body := opened(t, storeWithPhoto(7), "pile")
	require.Contains(t, body, `src="/photo/7/thumb"`, "the card downloaded the original")
	require.Contains(t, body, `href="/photo/7"`, "there is no way to the whole picture")
}

func TestTheThumbRouteServesTheSmallerCopy(t *testing.T) {
	ph := photosOnDisk(t)
	w := askForThumb(t, storeWithPhoto(7), ph, "7")

	require.Equal(t, 200, w.Code)
	require.Equal(t, "the smaller copy", w.Body.String())
	require.Equal(t, "image/jpeg", w.Header().Get("Content-Type"))
}

// A kind Go cannot decode has no smaller copy and never will. The card must
// still draw something, so the original is served rather than a 404 — a broken
// image is a worse answer than a large one.
func TestNoSmallerCopyServesTheWholePhotograph(t *testing.T) {
	ph := photosOnDisk(t)
	ph.noThumb = true
	w := askForThumb(t, storeWithPhoto(7), ph, "7")

	require.Equal(t, 200, w.Code)
	require.Greater(t, w.Body.Len(), 100, "that is not a photograph")
	require.Equal(t, "image/jpeg", w.Header().Get("Content-Type"))
}

// The same guard as the photograph itself. A smaller copy of a private letter
// is a private letter.
//
// A boundary test rather than a mutation-proved one, said plainly: the handler
// refuses at three independent points — the row was not found, the row has no
// photograph, and the name is not a name — so no single deletion shows here.
// The person-scoping itself is the store's, and is proved against a real
// database in internal/squirrel.
func TestAThumbnailOfANoteThatIsNotYoursIsNotServed(t *testing.T) {
	require.Equal(t, 404, askForThumb(t, &fakeStore{}, photosOnDisk(t), "99").Code)
}

// And it is still cached hard: the name is random and never gets new bytes.
func TestASmallerCopyIsCachedLikeTheOriginal(t *testing.T) {
	ph := photosOnDisk(t)
	w := askForThumb(t, storeWithPhoto(7), ph, "7")

	require.Contains(t, w.Header().Get("Cache-Control"), "immutable")
	require.Contains(t, w.Header().Get("Cache-Control"), "private")
	require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
}

// And the route is actually mounted. The tests above call the handler
// directly, so without this one the whole set would pass with nothing wired
// up — which is the shape of test this project has been caught by before.
func TestTheThumbRouteIsMounted(t *testing.T) {
	m := mountedWithCamera(t, storeWithPhoto(7), &fakeSpool{}, photosOnDisk(t))

	require.Contains(t, m.routes, "GET /photo/{id}/thumb")
}

// Nowhere to keep a photograph means no route at all, the same as the
// photograph's own: a control that cannot work is worse than one never drawn.
func TestWithNowhereToKeepPhotographsThereIsNoThumbRoute(t *testing.T) {
	m := mounted(t, &fakeStore{})

	require.NotContains(t, m.routes, "GET /photo/{id}/thumb")
}
