package web

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// The page the screen shows when it cannot reach the store has to say what to
// do, not only what happened.
//
// It said only what happened. The service worker's offline page — the same
// failure, met from the other side — has always carried the sentence that
// matters, because the room writes through the same spool the screen does and
// a database the screen cannot reach does not stop you keeping a thought. The
// server-rendered one, which fires on a mid-triage write as well as on a read,
// left you on a full page with no way forward at all.
func TestTheFailurePageSaysWhereNotesStillGo(t *testing.T) {
	f := &fakeStore{err: errTest}

	res := mounted(t, f).call(t, "GET", "/kept", nil)

	require.Equal(t, http.StatusServiceUnavailable, res.Code)
	require.Contains(t, res.Body.String(), "Campfire",
		"the failure page names no way on; the offline page has named one since it was written")
}

// And the two pages keep saying the same thing, because they are one event met
// from two sides. If the words drift, the one you meet depends on which half
// of the product noticed first — which is the defect the capture vocabulary
// had, fixed the day before this test was written.
func TestTheFailurePageAndTheOfflinePageAgree(t *testing.T) {
	f := &fakeStore{err: errTest}

	page := mounted(t, f).call(t, "GET", "/kept", nil).Body.String()

	worker, err := staticFS.ReadFile("static/sw.js")
	require.NoError(t, err)

	const said = "Notes are still kept by talking to Squirrel in Campfire."
	require.Contains(t, page, said, "the failure page does not carry the sentence")
	require.Contains(t, string(worker), said, "the offline page no longer carries it")
}
