package web

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// photographed builds the multipart body a phone's camera actually sends.
func photographed(t *testing.T, words, kind string, bytesIn []byte) (string, *bytes.Buffer) {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	require.NoError(t, w.WriteField("text", words))

	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="photo"; filename="IMG_0042.jpg"`}
	h["Content-Type"] = []string{kind}
	part, err := w.CreatePart(h)
	require.NoError(t, err)
	_, err = part.Write(bytesIn)
	require.NoError(t, err)

	require.NoError(t, w.Close())
	return w.FormDataContentType(), &body
}

func postPhoto(t *testing.T, m *testMux, contentType string, body *bytes.Buffer) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/capture", body)
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "a-token"})
	r.Header.Set("Content-Type", contentType)
	r.Header.Set("Origin", "http://"+r.Host)
	w := httptest.NewRecorder()
	m.routes["POST /capture"](w, r)
	return w
}

func TestAPhotographIsKeptAndReferenced(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	m := mountedWithCamera(t, &fakeStore{}, sp, ph)

	kind, body := photographed(t, "the tax letter", "image/jpeg", []byte("jpegbytes"))
	w := postPhoto(t, m, kind, body)

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/", w.Header().Get("Location"))

	// The bytes went to the volume, and the capture references them rather
	// than carrying them.
	require.Equal(t, []string{"jpegbytes"}, ph.kept)
	require.Len(t, sp.written, 1)
	require.Equal(t, "photo-1.jpg", sp.written[0].PhotoName)
	require.Equal(t, "image/jpeg", sp.written[0].PhotoType)
	require.Equal(t, "the tax letter", sp.written[0].Text)
}

// A photograph on its own is a capture — that is most of the point of having
// a camera.
func TestAPhotographWithNoWordsIsStillACapture(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	m := mountedWithCamera(t, &fakeStore{}, sp, ph)

	kind, body := photographed(t, "", "image/jpeg", []byte("jpegbytes"))
	w := postPhoto(t, m, kind, body)

	require.Equal(t, "/", w.Header().Get("Location"))
	require.Len(t, sp.written, 1)
	require.Empty(t, sp.written[0].Text)
	require.Equal(t, "photo-1.jpg", sp.written[0].PhotoName)
}

// Neither words nor a photograph is nothing to keep.
func TestNothingAtAllIsStillNothing(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	m := mountedWithCamera(t, &fakeStore{}, sp, ph)

	kind, body := photographed(t, "   ", "image/jpeg", nil)
	postPhoto(t, m, kind, body)

	// An empty part is not a photograph; the fake keeps it, so this asserts
	// the pair rather than the store's own emptiness check.
	require.Len(t, sp.written, 1)
}

// What the browser claimed is checked against the handful this keeps, and the
// words come back rather than the capture quietly losing the picture.
func TestAKindThisDoesNotKeepIsRefusedAndTheWordsComeBack(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	f := &fakeStore{}
	m := mountedWithCamera(t, f, sp, ph)

	kind, body := photographed(t, "the tax letter", "application/pdf", []byte("%PDF"))
	w := postPhoto(t, m, kind, body)

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Len(t, f.appended, 2)
	require.Contains(t, f.appended[1].Words, "That photograph was not kept")
	require.NotContains(t, f.appended[1].Words, "cannot reach its memory",
		"a refused photograph was reported as a machine that had failed")
	require.Equal(t, "the tax letter", f.appended[0].Words)
	require.Empty(t, ph.kept, "it kept something it does not keep")
	require.Empty(t, sp.written, "it captured a note referencing nothing")
}

// A volume that will not take it is the same shape of failure: loud, and the
// words survive.
func TestAVolumeThatRefusesKeepsTheWords(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{err: errTest}
	f := &fakeStore{}
	m := mountedWithCamera(t, f, sp, ph)

	kind, body := photographed(t, "the tax letter", "image/jpeg", []byte("jpegbytes"))
	postPhoto(t, m, kind, body)

	require.Len(t, f.appended, 2)
	require.Contains(t, f.appended[1].Words, "not kept")
	require.Equal(t, "the tax letter", f.appended[0].Words)
	require.Empty(t, sp.written)
}

func TestWordsAloneStillPostWithACameraPresent(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	m := mountedWithCamera(t, &fakeStore{}, sp, ph)

	w := post(t, m, "/capture", map[string][]string{"text": {"a thought"}})

	require.Equal(t, "/", w.Header().Get("Location"))
	require.Len(t, sp.written, 1)
	require.Equal(t, "a thought", sp.written[0].Text)
	require.Empty(t, sp.written[0].PhotoName)
	require.Empty(t, ph.kept)
}

// Nowhere to put one is a supported state, and the camera is simply not drawn.
func TestNoVolumeMeansNoCamera(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/", nil).Body.String()
	require.NotContains(t, body, `name="photo"`)
	require.NotContains(t, body, "Add a photograph")
}

func TestAVolumeMeansACamera(t *testing.T) {
	m := mountedWithCamera(t, &fakeStore{}, &fakeSpool{}, &fakePhotos{})
	body := m.call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `name="photo"`)
	require.Contains(t, body, `accept="image/*"`)
	require.Contains(t, body, `enctype="multipart/form-data"`)
}

// And it does not forbid the gallery. `capture` reads like "prefer the camera"
// and does not mean that: on a phone it removes every other source, so a
// photograph you already have becomes unreachable. It was there for one
// release and that is exactly what it did.
func TestTheCameraDoesNotForbidTheGallery(t *testing.T) {
	m := mountedWithCamera(t, &fakeStore{}, &fakeSpool{}, &fakePhotos{})
	body := m.call(t, "GET", "/", nil).Body.String()

	require.NotContains(t, body, "capture=")
}

// And with no volume the route does not exist at all, rather than answering
// 404 to something it could have served.
func TestNoVolumeMeansNoRoute(t *testing.T) {
	require.NotContains(t, mounted(t, &fakeStore{}).routes, "GET /photo/{id}")
	require.Contains(t,
		mountedWithCamera(t, &fakeStore{}, &fakeSpool{}, &fakePhotos{}).routes,
		"GET /photo/{id}")
}

// The pile shows it by the note's id, never by the file's name: the name is
// the only string here that becomes a path.
func TestThePileShowsAPhotographByTheNotesID(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{{
		ID: 7, RawText: "", State: squirrel.ItemOpen, Kind: squirrel.ItemNote,
		PhotoName: "photo-1.jpg", PhotoType: "image/jpeg",
	}}}
	body := opened(t, f, "notes")
	require.Contains(t, body, `src="/photo/7/thumb"`)
	require.Contains(t, body, `href="/photo/7"`)
	require.NotContains(t, body, "photo-1.jpg")
}

// The one that only production could tell us about, until now.
//
// ParseMultipartForm holds the first megabyte in memory and spills the rest to
// a temporary file. This pod runs with a read-only root filesystem and no
// writable /tmp, because everything it writes has a volume of its own — so
// every photograph over a megabyte, which is every photograph a phone takes,
// failed in the parser before the handler saw it. And it failed with the one
// message that was not true: that Squirrel could not reach its memory.
//
// It survived a whole release of tests because the natural way to test an
// upload is with a small file, and a small file is exactly the one that never
// touches the disk. So this one is deliberately over the bound, and TMPDIR is
// pointed somewhere that does not exist, which is what the container is.
func TestAPhotographTooBigForMemoryNeedsNoTemporaryFile(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "there-is-no-such-place"))

	sp, ph := &fakeSpool{}, &fakePhotos{}
	m := mountedWithCamera(t, &fakeStore{}, sp, ph)

	// Over the megabyte ParseMultipartForm would have kept in memory.
	big := bytes.Repeat([]byte("x"), (1<<20)+4096)
	kind, body := photographed(t, "the tax letter", "image/jpeg", big)
	w := postPhoto(t, m, kind, body)

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/", w.Header().Get("Location"),
		"a photograph too big to hold in memory was refused")
	require.Len(t, ph.kept, 1)
	require.Len(t, ph.kept[0], len(big), "the photograph arrived truncated")
	require.Len(t, sp.written, 1)
	require.Equal(t, "the tax letter", sp.written[0].Text)
}

// And the words still get through on their own with nowhere to spill to, which
// is the same request without the part that needed a disk.
func TestWordsAloneNeedNoTemporaryFileEither(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "there-is-no-such-place"))

	sp, ph := &fakeSpool{}, &fakePhotos{}
	m := mountedWithCamera(t, &fakeStore{}, sp, ph)

	kind, body := photographed(t, "just the words", "image/jpeg", nil)
	w := postPhoto(t, m, kind, body)

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Len(t, sp.written, 1)
	require.Equal(t, "just the words", sp.written[0].Text)
}

// The parts are read by name, so moving the camera above the box in the markup
// is a change to the markup and not to what gets kept.
func TestTheCameraCanComeBeforeTheWords(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	m := mountedWithCamera(t, &fakeStore{}, sp, ph)

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	h := map[string][]string{
		"Content-Disposition": {`form-data; name="photo"; filename="IMG_0042.jpg"`},
		"Content-Type":        {"image/jpeg"},
	}
	part, err := w.CreatePart(h)
	require.NoError(t, err)
	_, err = part.Write([]byte("jpegbytes"))
	require.NoError(t, err)
	require.NoError(t, w.WriteField("text", "the tax letter"))
	require.NoError(t, w.Close())

	res := postPhoto(t, m, w.FormDataContentType(), &body)

	require.Equal(t, "/", res.Header().Get("Location"))
	require.Len(t, sp.written, 1)
	require.Equal(t, "the tax letter", sp.written[0].Text)
	require.NotEmpty(t, sp.written[0].PhotoName)
}

// A form with the camera on it and nothing chosen sends an empty file part.
// That is most captures, and it is not a refusal.
func TestAnEmptyFilePartIsJustWords(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	m := mountedWithCamera(t, &fakeStore{}, sp, ph)

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	require.NoError(t, w.WriteField("text", "no picture today"))
	_, err := w.CreateFormFile("photo", "")
	require.NoError(t, err)
	require.NoError(t, w.Close())

	res := postPhoto(t, m, w.FormDataContentType(), &body)

	require.Equal(t, "/", res.Header().Get("Location"))
	require.Empty(t, ph.kept, "an empty file part was kept as a photograph")
	require.Len(t, sp.written, 1)
	require.Empty(t, sp.written[0].PhotoName)
}

// `toView` called `.Local()` on it — the *process* clock, and the pods run in
// UTC on purpose (#148), so anything captured after ten in the evening wore the
// previous day's date. The store hands rows back in the person's clock; converting
// again here would be the same bug with an extra step.
func TestACardPrintsTheDateTheNoteCarries(t *testing.T) {
	ams, err := time.LoadLocation("Europe/Amsterdam")
	require.NoError(t, err)
	// Half past midnight on the 14th where the person is, which is half past
	// ten on the 13th in UTC. The whole test is that those are different days.
	at := time.Date(2026, 3, 14, 0, 30, 0, 0, ams)
	require.Equal(t, 13, at.UTC().Day(), "the fixture does not straddle a day boundary")

	v := toView(squirrel.Item{
		ID: 1, RawText: "the boiler makes a noise", ReceivedAt: at,
		State: squirrel.ItemOpen, Kind: squirrel.ItemNote,
	})

	require.Equal(t, "14 MARCH", v.When)
}
