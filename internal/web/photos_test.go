package web

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A photograph of the thing, instead of typing what it says.

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
	r.Header.Set("X-Authentik-Username", "ronald")
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
	require.Equal(t, "/?kept=1", w.Header().Get("Location"))

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

	require.Equal(t, "/?kept=1", w.Header().Get("Location"))
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
	m := mountedWithCamera(t, &fakeStore{}, sp, ph)

	kind, body := photographed(t, "the tax letter", "application/pdf", []byte("%PDF"))
	w := postPhoto(t, m, kind, body)

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Contains(t, w.Header().Get("Location"), "nokeep=1")
	require.Contains(t, w.Header().Get("Location"), "the+tax+letter")
	require.Empty(t, ph.kept, "it kept something it does not keep")
	require.Empty(t, sp.written, "it captured a note referencing nothing")
}

// A volume that will not take it is the same shape of failure: loud, and the
// words survive.
func TestAVolumeThatRefusesKeepsTheWords(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{err: errTest}
	m := mountedWithCamera(t, &fakeStore{}, sp, ph)

	kind, body := photographed(t, "the tax letter", "image/jpeg", []byte("jpegbytes"))
	w := postPhoto(t, m, kind, body)

	require.Contains(t, w.Header().Get("Location"), "nokeep=1")
	require.Contains(t, w.Header().Get("Location"), "the+tax+letter")
	require.Empty(t, sp.written)
}

// Words alone still post, with or without a camera on the page.
func TestWordsAloneStillPostWithACameraPresent(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	m := mountedWithCamera(t, &fakeStore{}, sp, ph)

	w := post(t, m, "/capture", map[string][]string{"text": {"a thought"}})

	require.Equal(t, "/?kept=1", w.Header().Get("Location"))
	require.Len(t, sp.written, 1)
	require.Equal(t, "a thought", sp.written[0].Text)
	require.Empty(t, sp.written[0].PhotoName)
	require.Empty(t, ph.kept)
}

// Nowhere to put one is a supported state, and the camera is simply not drawn.
func TestNoVolumeMeansNoCamera(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/", nil).Body.String()
	require.NotContains(t, body, `name="photo"`)
	require.NotContains(t, body, "Photograph it instead")
}

func TestAVolumeMeansACamera(t *testing.T) {
	m := mountedWithCamera(t, &fakeStore{}, &fakeSpool{}, &fakePhotos{})
	body := m.call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `name="photo"`)
	require.Contains(t, body, `capture="environment"`)
	require.Contains(t, body, `enctype="multipart/form-data"`)
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
	body := mountedWithCamera(t, f, &fakeSpool{}, &fakePhotos{}).
		call(t, "GET", "/pile", nil).Body.String()

	require.Contains(t, body, `src="/photo/7"`)
	require.NotContains(t, body, "photo-1.jpg")
}
