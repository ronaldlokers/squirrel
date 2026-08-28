package web

import (
	"html"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// renderRoomDock renders a room and returns its dock.
func renderRoomDock(t *testing.T, r room) string {
	t.Helper()
	f := &fakeStore{}
	m := newTestMux()
	opts := signedInOptions()
	// A camera exists here, so the test about which rooms draw one is
	// measuring the room rather than the deployment.
	opts.Photos = &fakePhotos{}
	require.NoError(t, Mount(m, f, opts))
	body := m.call(t, "GET", "/r/"+r.Key, nil).Body.String()
	i := strings.Index(body, `<div class="dock">`)
	require.GreaterOrEqual(t, i, 0, "no dock on %s", r.Key)
	return body[i:]
}

// Every room's dock posts where its button says it will. A button that names a
// consequence and a form that does something else is worse than the grey
// placeholder it replaced.
func TestEveryRoomsDockPostsWhereItsButtonSays(t *testing.T) {
	for _, r := range rooms {
		t.Run(r.Key, func(t *testing.T) {
			dock := renderRoomDock(t, r)
			require.Contains(t, dock, `action="`+r.Action+`"`)
			require.Contains(t, dock, `name="`+r.Field+`"`)
			require.Contains(t, dock, ">"+r.Button+"<")
			// Escaped, because that is what a template writes: the
			// apostrophe in "what's going on?" is &#39; on the page.
			require.Contains(t, dock, `placeholder="`+html.EscapeString(r.Placeholder)+`"`)
		})
	}
}

// The one that would be invisible: a dock carrying no room files into whatever
// the handler defaults to, which is Buddy's room, and nothing on screen says
// so.
func TestTheDockSaysWhichRoomItIsIn(t *testing.T) {
	for _, r := range rooms {
		t.Run(r.Key, func(t *testing.T) {
			require.Contains(t, renderRoomDock(t, r),
				`<input type="hidden" name="room" value="`+r.Key+`">`)
		})
	}
}

// And what is typed there lands in that room's conversation, not in Buddy's.
func TestWhatYouTypeInARoomIsSaidInThatRoom(t *testing.T) {
	f := &fakeStore{}
	m := newTestMux()
	require.NoError(t, Mount(m, f, signedInOptions()))

	f.appended = nil
	m.call(t, "POST", "/tasks/new", strings.NewReader("text=ring+the+vet&room=tasks"))

	require.NotEmpty(t, f.appended, "nothing was said back")
	for _, turn := range f.appended {
		require.Equal(t, "tasks", turn.Room,
			"a turn typed in the tasks landed in %q", turn.Room)
	}
}

// A camera only where a photograph makes sense. A chore is a name and a
// rhythm; a control that cannot work is worse than one never drawn.
func TestOnlyTheRoomsThatKeepAPhotographDrawACamera(t *testing.T) {
	for _, r := range rooms {
		t.Run(r.Key, func(t *testing.T) {
			dock := renderRoomDock(t, r)
			if r.Action == "/capture" {
				require.Contains(t, dock, `type="file"`)
				return
			}
			require.NotContains(t, dock, `type="file"`,
				"%s draws a camera it cannot use", r.Key)
		})
	}
}

// The worker holds a capture when there is no network, and it has to know
// every route a dock posts to.
//
// A room missing from its table posts straight to the network and loses the
// words when there is none — and a field name missing from it holds the wrong
// half of the form. Neither shows up in Go: sw.js is a static file, and this
// is the only thing that reads it against the rooms.
func TestTheWorkerHoldsEveryRoomsDock(t *testing.T) {
	worker, err := staticFS.ReadFile("static/sw.js")
	require.NoError(t, err)
	js := string(worker)

	for _, r := range rooms {
		require.Contains(t, js, `"`+r.Action+`": "`+r.Field+`"`,
			"sw.js does not hold %s's dock — %s posting %q",
			r.Key, r.Action, r.Field)
	}
}
