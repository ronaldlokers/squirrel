package web

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/stretchr/testify/require"
)

func postToTheBoard(t *testing.T, m *testMux, contentType string, body *bytes.Buffer) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/board/capture", body)
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "a-token"})
	r.Header.Set("Content-Type", contentType)
	r.Header.Set("Origin", "http://"+r.Host)
	w := httptest.NewRecorder()
	h, ok := m.routes["POST /board/capture"]
	if !ok {
		t.Fatal("no route for POST /board/capture")
	}
	h(w, r)
	return w
}

// The camera writes to disk before the capture that references it, the same as
// every other photograph in this product: the board is a second door onto one
// path rather than a second path.
func TestAPhotographFromTheBoardIsKeptBeforeItIsSpooled(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	m := mountedWithCamera(t, aBoardStore(), sp, ph)

	kind, body := photographed(t, "the meter, before the bill", "image/jpeg", []byte("bytes"))
	w := postToTheBoard(t, m, kind, body)

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Len(t, ph.kept, 1, "the photograph did not reach the volume")
	require.Len(t, sp.written, 1)
	require.Equal(t, "the meter, before the bill", sp.written[0].Text)
	require.NotEmpty(t, sp.written[0].PhotoName, "the capture does not point at the photograph")
}

// A photograph with no words is a note, which is most of the point of having a
// camera.
func TestAPhotographWithNoWordsIsStillACaptureOnTheBoard(t *testing.T) {
	sp, ph := &fakeSpool{}, &fakePhotos{}
	m := mountedWithCamera(t, aBoardStore(), sp, ph)

	kind, body := photographed(t, "", "image/jpeg", []byte("bytes"))
	postToTheBoard(t, m, kind, body)

	require.Len(t, sp.written, 1, "a photograph on its own was dropped")
	require.NotEmpty(t, sp.written[0].PhotoName)
}

// A strip never carries a thumbnail. It says it has one, and opening it is what
// shows it.
func TestAStripWithAPhotographSaysSoAndOpens(t *testing.T) {
	f := aBoardStore()
	f.items = []squirrel.Item{{
		ID: 5, RawText: "the meter", State: squirrel.ItemOpen, Kind: squirrel.ItemNote,
		ReceivedAt: time.Now(), PhotoName: "IMG_0042.jpg", PhotoType: "image/jpeg",
	}}
	m := mounted(t, f)

	body := m.call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `href="/?open=5"`)
	require.NotContains(t, body, `/photo/5/thumb`, "the strip is carrying the photograph itself")
}

func TestOpeningAStripShowsThePhotographAndItsAnswers(t *testing.T) {
	f := aBoardStore()
	f.items = []squirrel.Item{{
		ID: 5, RawText: "the meter", State: squirrel.ItemOpen, Kind: squirrel.ItemNote,
		ReceivedAt: time.Now(), PhotoName: "IMG_0042.jpg", PhotoType: "image/jpeg",
	}}
	m := mounted(t, f)

	body := m.call(t, "GET", "/?open=5", nil).Body.String()

	require.Contains(t, body, `src="/photo/5"`)
	require.Contains(t, body, "the meter")
	require.Contains(t, body, `value="keep"`, "an opened note cannot be answered")
	require.Contains(t, body, "back to the board")
}

// Opening something that is not yours shows nothing, and says nothing about
// whether it exists.
func TestOpeningWhatIsNotYoursShowsNothing(t *testing.T) {
	f := aBoardStore()
	f.items = []squirrel.Item{{ID: 5, RawText: "the meter", State: squirrel.ItemOpen, Kind: squirrel.ItemNote}}
	f.notMine = map[int64]bool{5: true}
	m := mounted(t, f)

	body := m.call(t, "GET", "/?open=5", nil).Body.String()

	require.NotContains(t, body, `src="/photo/5"`)
	require.NotContains(t, body, "back to the board")
}
