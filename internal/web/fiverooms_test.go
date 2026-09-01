package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A shelf is a press inside the notes, not a door on the rail.
func TestAShelfIsAPressInsideTheNotes(t *testing.T) {
	body := opened(t, aShelf(), "notes")

	require.Contains(t, body, `value="kept"`, "the notes do not offer what you kept")
	require.Contains(t, body, `value="held"`, "the notes do not offer what you set aside")
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
		"/r/buddy": "/r/everything", "/r/pile": "/r/notes", "/r/held": "/r/notes", "/r/kept": "/r/notes",
	} {
		res := mounted(t, &fakeStore{}).call(t, "GET", from, nil)

		require.Equal(t, 301, res.Code, "%s answers %d", from, res.Code)
		require.Equal(t, to, res.Header().Get("Location"), "%s lands in the wrong place", from)
	}
}

// The notes hold what three rooms held, so what was said in any of them is one
// conversation now. The migration is what moves the record; this is the half
// the screen is responsible for.
func TestTheNotesReadOneConversation(t *testing.T) {
	f := &fakeStore{turns: []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerYou, Words: "the boiler code"},
		{ID: 2, Who: squirrel.SpeakerBuddy, Words: "Down it goes."},
	}}
	body := opened(t, f, "notes")

	require.Contains(t, body, "the boiler code")
	require.Equal(t, "notes", f.roomRead, "the notes read somebody else's conversation")
}
