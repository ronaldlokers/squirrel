package web

import (
	"html"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
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
	m.call(t, "POST", "/tasks/new", strings.NewReader("text=ring+the+vet&room=everything"))

	require.NotEmpty(t, f.appended, "nothing was said back")
	for _, turn := range f.appended {
		require.Equal(t, "everything", turn.Room,
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

// Asking Buddy something in a room asks him as that room's Buddy.
//
// Without this the screen hands every question over as Buddy's own room, and
// the narrowing is code nothing reaches: the chores' Buddy would arrive with
// the whole toolset and the whole conversation.
func TestAskingInARoomAsksThatRoomsBuddy(t *testing.T) {
	f := &fakeStore{}
	c := &fakeCoach{reply: "Which bin."}
	m := mountedWith(t, f, c)

	m.call(t, "POST", "/buddy/say", strings.NewReader("said=the+bins&room=everything"))

	require.NotEmpty(t, c.asked, "nothing was asked")
	require.Equal(t, "everything", c.asked[len(c.asked)-1].room,
		"the question went to Buddy's room instead of the one it was asked in")
	require.Contains(t, c.remembered, "everything",
		"the exchange was remembered in the wrong room")
}

// The notice a door carried about its set went with the doors on 2 September
// 2026 — recorded in DESIGN.md as a capability the board does not have — and
// the test that pinned which room it was asked as went with it.

// And the page sends the room, which is the half a handler test cannot prove.
//
// A test that posts room=chores itself proves the handler reads the field. It
// says nothing about whether anything ever writes one — and a form that omits
// it answers as Buddy's room, which looks exactly like a room that had nothing
// to say.
func TestEveryFormATurnDrawsCarriesItsRoom(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 1, Name: "the bins", EveryDays: 7, SinceDays: 8, Active: true},
	}}
	m := newTestMux()
	require.NoError(t, Mount(m, f, signedInOptions()))

	body := m.call(t, "GET", "/r/everything", nil).Body.String()

	// Only the turns, not the rail or the dock, which are tested elsewhere.
	// The conversation and the edge below it — the room's list is drawn there
	// rather than written into the record since 31 August 2026, and it is the
	// list that carries most of the forms.
	turns := body[strings.Index(body, `<div class="thread"`):]
	turns = turns[:strings.Index(turns, `<div class="dock">`)]

	forms := strings.Count(turns, "<form")
	require.Positive(t, forms, "no forms drawn, so this measured nothing")
	require.Equal(t, forms, strings.Count(turns, `name="room" value="everything"`),
		"a form in Buddy's room does not say which room it is in")
}

// And the room it falls back to is a room. A held capture replayed into a room
// that no longer exists is a thought filed where nothing reads, which is the
// one failure the worker exists to prevent.
func TestTheWorkerFallsBackToARoomThatExists(t *testing.T) {
	worker, err := staticFS.ReadFile("static/sw.js")
	require.NoError(t, err)

	for _, fallback := range regexp.MustCompile(`room[^"\n]*\|\| "([a-z]+)"`).
		FindAllStringSubmatch(string(worker), -1) {
		_, ok := roomByKey(fallback[1])
		require.True(t, ok, "sw.js falls back to %q, which is not a room", fallback[1])
	}
}
