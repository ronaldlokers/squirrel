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
	want := map[string]string{
		"buddy":  "Buddy",
		"pile":   "the pile",
		"chores": "the chores",
		"at":     "the agenda",
		"tasks":  "the tasks",
		"held":   "what you set aside",
		"kept":   "the things you kept",
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

// Buddy's is the one button that names no room, because his room is where you
// talk rather than where a thing lands.
func TestOnlyBuddysButtonNamesNoPlace(t *testing.T) {
	for _, r := range rooms {
		if r.Key == "buddy" {
			require.Equal(t, "Tell it", r.Button)
			continue
		}
		require.True(t, strings.Contains(r.Button, "chore") ||
			strings.Contains(r.Button, "task") ||
			strings.Contains(r.Button, "pile") ||
			strings.Contains(r.Button, "agenda"),
			"%s's button %q does not say where the words go", r.Key, r.Button)
	}
}

func TestTheRoomIsBuddysWhenNobodySaidWhich(t *testing.T) {
	require.Equal(t, "buddy", roomOf(context.Background()))
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

	rec := m.call(t, "GET", "/r/pile", nil)
	require.Equal(t, 200, rec.Code)
	require.Empty(t, f.appended,
		"entering a room appended to a conversation that already ended open")
}

// The room's own turn goes in the room, not in the record Buddy's room holds.
func TestEnteringARoomPutsItsTurnInThatRoom(t *testing.T) {
	f := &fakeStore{}
	m := newTestMux()
	require.NoError(t, Mount(m, f, signedInOptions()))

	rec := m.call(t, "GET", "/r/chores", nil)
	require.Equal(t, 200, rec.Code)
	require.NotEmpty(t, f.appended, "the chores drew nothing")
	for _, turn := range f.appended {
		require.Equal(t, "chores", turn.Room,
			"a turn drawn in the chores landed in %q", turn.Room)
	}
}

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

func TestTheRailCountsWhatIsWaitingAndNothingElse(t *testing.T) {
	f := &fakeStore{waiting: squirrel.Waiting{Pile: 2, Chores: 1}}
	rail := roomsFor(context.Background(), f, 1, "buddy")

	by := map[string]railView{}
	for _, r := range rail {
		by[r.Key] = r
	}
	require.Len(t, by, len(rooms))
	require.Equal(t, 2, by["pile"].Count)
	require.Equal(t, 1, by["chores"].Count)
	require.Zero(t, by["tasks"].Count, "an empty room carries a number")
	require.Zero(t, by["kept"].Count, "a shelf carries a number")
	require.Zero(t, by["held"].Count, "a shelf carries a number")
	require.Zero(t, by["buddy"].Count, "Buddy's room carries a number")
}

func TestTheRailSaysWhichRoomYouAreIn(t *testing.T) {
	f := &fakeStore{}
	var current []string
	for _, r := range roomsFor(context.Background(), f, 1, "chores") {
		if r.Current {
			current = append(current, r.Key)
		}
	}
	require.Equal(t, []string{"chores"}, current)
}

// The rail is furniture: it is on every screen, not only on the one it was
// built for. A rail that vanished inside a room would be the menu again.
func TestTheRailIsOnEveryScreen(t *testing.T) {
	f := &fakeStore{waiting: squirrel.Waiting{Pile: 2}}
	m := newTestMux()
	require.NoError(t, Mount(m, f, signedInOptions()))

	for _, where := range []string{"/", "/r/chores", "/r/kept"} {
		t.Run(where, func(t *testing.T) {
			body := m.call(t, "GET", where, nil).Body.String()
			require.Contains(t, body, `<nav class="rail"`)
			for _, r := range rooms {
				require.Contains(t, body, `href="/r/`+r.Key+`"`,
					"the rail on %s cannot reach %s", where, r.Key)
			}
		})
	}
}

// The coach keeps its own copy of the room names, because internal/coach must
// not import internal/web. Two lists of the same six is one list that goes
// stale, and a room the coach has never heard of is a room it does not narrow
// in — which is Buddy's whole toolset, silently.
func TestTheRoomNamesAgreeWithTheCoach(t *testing.T) {
	for _, r := range rooms {
		if r.Key == "buddy" {
			// Buddy's own is deliberately absent there: it is not a room he is
			// confined to, it is where he is.
			require.Empty(t, coach.RoomName(r.Key))
			continue
		}
		require.Equal(t, r.Name, coach.RoomName(r.Key),
			"the coach calls %q something else", r.Key)
	}
	require.Len(t, coach.RoomKeys(), len(rooms)-1,
		"a room was added on one side only")
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

	require.NotContains(t, m.call(t, "GET", "/", nil).Body.String(), `<a class="brand" href="/"`)
	require.Contains(t, m.call(t, "GET", "/r/chores", nil).Body.String(), `<a class="brand" href="/"`)
}
