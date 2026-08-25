package web

import (
	"bytes"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A photograph the row expects and the disk does not have leaves a trace.
//
// The comment above the branch has always said "that is worth a log and a 404
// rather than a 500". Only the 404 was there. On a single node with the
// photographs on a volume beside the pod, a remount or a restore that did not
// line up turns every affected photograph into a quiet 404 — and there was
// nothing afterwards to say it had happened, or to how many.
func TestAPhotographTheDiskHasLostSaysSo(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{{
		ID: 7, RawText: "the meter", State: squirrel.ItemOpen, Kind: squirrel.ItemNote,
		PhotoName: "gone.jpg", PhotoType: "image/jpeg",
	}}}

	said := &bytes.Buffer{}
	was := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(said, nil)))
	t.Cleanup(func() { slog.SetDefault(was) })

	// The handler directly: the test mux matches by prefix and cannot resolve
	// a `{id}` wildcard, which is why every other photo test asserts the route
	// exists rather than calling it. fakePhotos.Open has always answered
	// ErrNotExist, which is exactly the disagreement this is about.
	opts := Options{
		IdentityHeader: "X-Authentik-Username", Identity: "ronald",
		Owner: func() int64 { return 1 }, Spool: &fakeSpool{}, Photos: &fakePhotos{},
	}
	r := httptest.NewRequest("GET", "/photo/7", nil)
	r.SetPathValue("id", "7")
	res := httptest.NewRecorder()
	photoHandler(f, opts)(res, asking(r))

	require.Equal(t, 404, res.Code, "it is still a 404 and not a 500")
	require.Contains(t, said.String(), "a photograph the row expects is not on disk",
		"the disk disagreed with the row and nothing was written down")
	require.Contains(t, said.String(), "gone.jpg",
		"and the log does not say which one")
}
