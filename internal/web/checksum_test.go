package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func askForChecksum(t *testing.T, f *fakeStore, ph *fakePhotos, id string) *httptest.ResponseRecorder {
	t.Helper()
	opts := Options{
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
		Photos:   ph,
	}
	r := httptest.NewRequest("GET", "/photo/"+id+"/checksum", nil)
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "a-token"})
	r.SetPathValue("id", id)
	w := httptest.NewRecorder()
	photoChecksumHandler(f, opts)(w, asking(r))
	return w
}

func TestTheChecksumRouteReportsSizeAndSHA256(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "photo-1.jpg"), []byte("jpegbytes"), 0o600))
	ph := &fakePhotos{dir: dir}

	w := askForChecksum(t, storeWithPhoto(7), ph, "7")

	require.Equal(t, 200, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var got photoChecksum
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, int64(9), got.Bytes)
	require.Equal(t, "bbf6124fd0287f049f9bd572995c6d04874b173eeeb32b070aef2504d686b87f", got.SHA256)
}

func TestTheChecksumRouteDoesNotServeTheBytesThemselves(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "photo-1.jpg"), []byte("jpegbytes"), 0o600))
	ph := &fakePhotos{dir: dir}

	w := askForChecksum(t, storeWithPhoto(7), ph, "7")

	require.NotContains(t, w.Body.String(), "jpegbytes")
}

func TestAChecksumOfANoteThatIsNotYoursIsNotServed(t *testing.T) {
	require.Equal(t, 404, askForChecksum(t, &fakeStore{}, photosOnDisk(t), "99").Code)
}

func TestAChecksumTheDiskHasLostIsNotFound(t *testing.T) {
	w := askForChecksum(t, storeWithPhoto(7), &fakePhotos{}, "7")

	require.Equal(t, 404, w.Code)
}

func TestTheChecksumRouteIsMounted(t *testing.T) {
	m := mountedWithCamera(t, storeWithPhoto(7), &fakeSpool{}, photosOnDisk(t))

	require.Contains(t, m.routes, "GET /photo/{id}/checksum")
}

func TestWithNowhereToKeepPhotographsThereIsNoChecksumRoute(t *testing.T) {
	m := mounted(t, &fakeStore{})

	require.NotContains(t, m.routes, "GET /photo/{id}/checksum")
}
