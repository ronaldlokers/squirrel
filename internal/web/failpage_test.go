package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTheFailurePageSaysWhereNotesStillGo(t *testing.T) {
	w := httptest.NewRecorder()

	fail(w, errTest)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	require.Contains(t, w.Body.String(), "Campfire",
		"the failure page names no way on; the offline page has named one since it was written")
}

func TestTheFailurePageAndTheOfflinePageAgree(t *testing.T) {
	w := httptest.NewRecorder()
	fail(w, errTest)

	worker, err := staticFS.ReadFile("static/sw.js")
	require.NoError(t, err)

	const said = "Notes are still kept by talking to Squirrel in Campfire."
	require.Contains(t, w.Body.String(), said, "the failure page does not carry the sentence")
	require.Contains(t, string(worker), said, "the offline page no longer carries it")
}
