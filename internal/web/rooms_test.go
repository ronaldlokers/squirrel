package web

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestTheRoomsAreTheProductsOwnNames(t *testing.T) {
	// One room, and its name is his. Four of the five became the board's bays
	// on 2 September 2026; what is left is not about rows, so it is not named
	// after any.
	require.Len(t, rooms, 1, "a room was added or removed without this test")
	require.Equal(t, "everything", rooms[0].Key)
	require.Equal(t, "Buddy", rooms[0].Name)

	// And the four that left are still sets Buddy can be asked to draw.
	for key, name := range theSets {
		drawn, ok := doorName(key)
		require.True(t, ok, "%s is not a set he can name", key)
		require.Equal(t, name, drawn)
	}
}

func theOldRoomNames(t *testing.T) {
	want := map[string]string{
		"everything": "everything",
		"notes":      "the notes",
		"chores":     "the chores",
		"at":         "the agenda",
		"tasks":      "the tasks",
	}
	require.Len(t, rooms, len(want), "a room was added or removed without this test")
	for _, r := range rooms {
		name, ok := want[r.Key]
		require.True(t, ok, "unknown room %q", r.Key)
		require.Equal(t, name, r.Name)
		require.NotContains(t, r.Name, "#", "hash-names carry Slack's register")
		require.NotEmpty(t, r.Button, "%s has no button", r.Key)
		require.NotEmpty(t, r.Action, "%s posts nowhere", r.Key)
	}
}

// A room's dock is one control carrying the whole of what typing will do, so
// the same button on two rooms that file differently is the one failure this
// design cannot afford.
func TestNoTwoRoomsShareAButtonUnlessTheyShareADestination(t *testing.T) {
	where := map[string]string{}
	for _, r := range rooms {
		if seen, ok := where[r.Button]; ok {
			require.Equal(t, seen, r.Action,
				"%q means two different things", r.Button)
			continue
		}
		where[r.Button] = r.Action
	}
}

// Everything's is the one button that names no room, because that room is where
// you talk rather than where a thing lands.
func TestOnlyBuddysButtonNamesNoPlace(t *testing.T) {
	for _, r := range rooms {
		if r.Key == "everything" {
			require.Equal(t, "Tell it", r.Button)
			continue
		}
		require.True(t, strings.Contains(r.Button, "chore") ||
			strings.Contains(r.Button, "task") ||
			strings.Contains(r.Button, "notes") ||
			strings.Contains(r.Button, "agenda"),
			"%s's button %q does not say where the words go", r.Key, r.Button)
	}
}

func TestTheRoomIsBuddysWhenNobodySaidWhich(t *testing.T) {
	require.Equal(t, "everything", roomOf(context.Background()))
}

func TestTheRoomOnTheRequestIsTheRoomThatIsRead(t *testing.T) {
	ctx := context.WithValue(context.Background(), roomKey{}, "chores")
	require.Equal(t, "chores", roomOf(ctx))
}

// Entering a room is navigation, and navigation must not write. The door it
// replaced appended "the pile" to the record on every press, which made a
// record of walking around rather than of anything said.
func TestEnteringARoomWritesNothingWhenItAlreadyEndsOpen(t *testing.T) {
	f := &fakeStore{turns: []squirrel.Turn{{
		ID: 1, Room: "pile", Who: squirrel.SpeakerBuddy, Words: "here is one",
		Shown: []byte(`{"cards":[{"title":"a note","acts":[{"label":"DONE"}]}]}`),
	}}}
	m := newTestMux()
	require.NoError(t, Mount(m, f, signedInOptions()))

	rec := m.call(t, "GET", "/r/everything", nil)
	require.Equal(t, 200, rec.Code)
	require.Empty(t, f.appended,
		"entering a room appended to a conversation that already ended open")
}

// Entering a room drew its list and wrote nothing. Four of the five rooms are
// the board's bays now and the fifth is Buddy's, which draws a conversation
// rather than a list — so what these two pinned is covered by the board's own
// tests and by TestEnteringARoomWritesNothingWhenItAlreadyEndsOpen above.

// A typo is not a page. Not a redirect to Buddy either: a URL that silently
// becomes a different room is a URL you cannot trust in a bookmark.
func TestAnUnknownRoomIsNotAPage(t *testing.T) {
	f := &fakeStore{}
	m := newTestMux()
	require.NoError(t, Mount(m, f, signedInOptions()))

	rec := m.call(t, "GET", "/r/nowhere", nil)
	require.Equal(t, 404, rec.Code)
	require.Empty(t, f.appended)
}

// The door, as it was until 28 August 2026. An installed home screen holds a
// cached page whose forms still post here, and it must send you to the room
// rather than write one more turn.
func TestTheOldDoorSendsYouToTheRoomAndWritesNothing(t *testing.T) {
	f := &fakeStore{}
	m := newTestMux()
	require.NoError(t, Mount(m, f, signedInOptions()))

	rec := m.call(t, "POST", "/open", strings.NewReader("where=chores"))
	require.Equal(t, 303, rec.Code)
	require.Equal(t, "/r/chores", rec.Header().Get("Location"))
	require.Empty(t, f.appended, "the old door still writes")
}

// The rail's three tests — what it counted, which room it said you were in, and
// that it was on every screen — went with the rail on 2 September 2026. Buddy's
// room has one link where the rail was, back to the board, and the board's own
// navigation is its four bay signs. What the counting rule protected is now the
// bay signs' own test: a sign says what is in the rack and no sign says nought.

// The coach keeps its own copy of the room names, because internal/coach must
// not import internal/web. Two lists of the same six is one list that goes
// stale, and a room the coach has never heard of is a room it does not narrow
// in — which is Buddy's whole toolset, silently.
func TestTheRoomNamesAgreeWithTheCoach(t *testing.T) {
	// The four stopped being rooms on 2 September 2026 and stayed sets he can
	// be asked to draw, which is exactly what the coach narrows in. So the
	// names that have to agree are the sets' — and Buddy's own is still
	// deliberately absent there: it is not a place he is confined to, it is
	// where he is.
	for key, name := range theSets {
		require.Equal(t, name, coach.RoomName(key),
			"the coach calls %q something else", key)
	}
	require.Empty(t, coach.RoomName("everything"))
	require.Len(t, coach.RoomKeys(), len(theSets),
		"a set was added on one side only")
}

// Every room is a conversation, so every room loads the conversation's script.
//
// Home carried both meanings until 28 August 2026 and every room rendered
// without thread.js — the fragment posting, the live edge and the chore keys
// all live there. Nothing looked broken, because the forms fall back to full
// navigations, which is why only a browser test found it.
func TestEveryRoomLoadsTheThreadsScript(t *testing.T) {
	f := &fakeStore{}
	m := newTestMux()
	require.NoError(t, Mount(m, f, signedInOptions()))

	for _, r := range rooms {
		t.Run(r.Key, func(t *testing.T) {
			body := m.call(t, "GET", "/r/"+r.Key, nil).Body.String()
			require.Contains(t, body, "/static/thread.js",
				"%s is a conversation with no conversation script", r.Key)
			require.Contains(t, body, "threadpage",
				"%s does not lay out as a conversation", r.Key)
		})
	}
}

// And only the front door keeps the mark unpressable. Everywhere else it is
// the way back, which is the convention every website has.
func TestOnlyTheFrontDoorHasNoWayBack(t *testing.T) {
	f := &fakeStore{}
	m := newTestMux()
	require.NoError(t, Mount(m, f, signedInOptions()))

	// The board is the front door and has no way back to itself; his room does,
	// and it is the mark in the lid.
	require.NotContains(t, m.call(t, "GET", "/", nil).Body.String(), `<a class="brand" href="/"`)
	require.Contains(t, m.call(t, "GET", "/r/everything", nil).Body.String(), `<a class="brand" href="/"`)
}
