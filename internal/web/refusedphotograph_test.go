package web

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func postBoardPhoto(t *testing.T, m *testMux, contentType string, body *bytes.Buffer) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/board/capture", body)
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "a-token"})
	r.Header.Set("Content-Type", contentType)
	r.Header.Set("Origin", "http://"+r.Host)
	w := httptest.NewRecorder()
	m.routes["POST /board/capture"](w, r)
	return w
}

func TestABoardCaptureWhosePhotographIsRefusedKeepsTheWords(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	m := mountedWithCamera(t, &fakeStore{}, sp, ph)

	kind, body := photographed(t, "the tax letter", "application/pdf", []byte("%PDF"))
	w := postBoardPhoto(t, m, kind, body)

	require.Equal(t, http.StatusSeeOther, w.Code,
		"the failure page says nothing has been lost, and the words had been")
	require.Equal(t, "/?bay=notes&nophoto=the+tax+letter", w.Header().Get("Location"))
	require.Empty(t, ph.kept)
}

func TestTheRefusedPhotographsWordsComeBackIntoTheBoxWithTheReason(t *testing.T) {
	body := mounted(t, aBoardStore()).
		call(t, "GET", "/?bay=notes&nophoto=the+tax+letter", nil).Body.String()
	rack := theRackIn(t, body, "bay=notes")

	require.Contains(t, rack, `value="the tax letter"`, "the words were not carried back")
	require.Contains(t, rack, "that photograph was not kept")
	require.Contains(t, rack, "keep them without it, or try another picture")
}

func TestARackSaysNothingAboutAPhotographWhenNoneWasRefused(t *testing.T) {
	rack := theRackIn(t, mounted(t, aBoardStore()).call(t, "GET", "/", nil).Body.String(), "bay=notes")

	require.NotContains(t, rack, "that photograph was not kept")
}

func TestAVolumeThatRefusesIsNotDrawnAsAWrongKindOfPicture(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{err: errTest}
	m := mountedWithCamera(t, &fakeStore{}, sp, ph)

	kind, body := photographed(t, "the tax letter", "image/jpeg", []byte("jpegbytes"))
	w := postBoardPhoto(t, m, kind, body)

	require.Equal(t, http.StatusServiceUnavailable, w.Code,
		"a volume that is down was reported as a picture Squirrel does not take")
}
