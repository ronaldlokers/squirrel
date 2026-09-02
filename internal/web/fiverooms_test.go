package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A shelf is a press inside the notes, not a door on the rail.
func TestAShelfIsAPressInsideTheNotes(t *testing.T) {
	// The ledge at the foot of the notes rack since 2 September 2026, where it
	// was a chip inside the notes room before. Still inside the notes, still
	// not a door of its own.
	body := opened(t, aShelf(), "notes")

	require.Contains(t, body, `href="/?shelf=kept"`, "the notes do not offer what you kept")
	require.Contains(t, body, `href="/?shelf=held"`, "the notes do not offer what you set aside")
	require.NotContains(t, body, `href="/r/kept"`, "a shelf is still a door on the rail")
	require.NotContains(t, body, `href="/r/held"`, "a shelf is still a door on the rail")
}

func TestPressingAShelfDrawsIt(t *testing.T) {
	f := aShelf()
	res := pressedShelf(t, f, "kept")

	require.Equal(t, 200, res.Code)
	require.Contains(t, res.Body.String(), "the things you kept")
	require.Len(t, f.appended, 2, "asking to see a shelf is a thing you said and a thing you were shown")
}

func TestAShelfNobodyNamedDrawsNothing(t *testing.T) {
	for _, shelf := range []string{"", "pile", "chores", "everything", "nowhere"} {
		f := aShelf()
		res := routed(t, f).call(t, "POST", "/notes/shelf",
			strings.NewReader(url.Values{"room": {"notes"}, "shelf": {shelf}}.Encode()))

		require.Equal(t, 303, res.Code, "%q drew something", shelf)
		require.Empty(t, f.appended, "%q was written into the conversation", shelf)
	}
}

// A room was a URL you could put on a home screen. All four still land.
func TestTheRoomsThatStoppedBeingRoomsStillLandSomewhere(t *testing.T) {
	for from, to := range map[string]string{
		"/r/buddy": "/r/everything", "/r/pile": "/?bay=notes", "/r/held": "/?shelf=held", "/r/kept": "/?shelf=kept",
		"/r/notes": "/?bay=notes", "/r/chores": "/?bay=chores", "/r/at": "/?bay=agenda", "/r/tasks": "/?bay=tasks",
	} {
		res := mounted(t, &fakeStore{}).call(t, "GET", from, nil)

		require.Equal(t, 301, res.Code, "%s answers %d", from, res.Code)
		require.Equal(t, to, res.Header().Get("Location"), "%s lands in the wrong place", from)
	}
}

// The notes read one conversation, and that test went with the room on
// 2 September 2026: the notes are a bay, and a bay reads the pile rather than a
// conversation. What it protected — that three rooms' worth of record became
// one — is Buddy's room's job now, and TestBuddysRoomReadsTheWholeRecord is
// where it is proved.
