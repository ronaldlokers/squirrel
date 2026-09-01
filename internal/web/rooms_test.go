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

	rec := m.call(t, "GET", "/r/notes", nil)
	require.Equal(t, 200, rec.Code)
	require.Empty(t, f.appended,
		"entering a room appended to a conversation that already ended open")
}

// The room's own turn goes in the room, not in the record Buddy's room holds.
// Entering a room writes nothing at all, and draws what is true now.
//
// It wrote its list into the conversation until 31 August 2026, which is why a
// chore you had done was still on the screen asking: the room refused to write
// the list a second time while the first one was still the last thing said.
// See view.Edge.
func TestEnteringARoomWritesNothingAndDrawsWhatIsThere(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 1, Name: "the bins", EveryDays: 7, SinceDays: 8, Active: true},
	}}
	m := newTestMux()
	require.NoError(t, Mount(m, f, signedInOptions()))

	rec := m.call(t, "GET", "/r/chores", nil)

	require.Equal(t, 200, rec.Code)
	require.Empty(t, f.appended, "entering a room wrote to the record")
	require.Contains(t, rec.Body.String(), "the bins", "the chores drew nothing")
}

// And what it draws is current, however many times you come back.
//
// The conversation holds a list from an earlier visit, which is what used to
// stop a room drawing anything at all: endsOpen saw cards on the last turn and
// said nothing more. Read on the edge alone and never on the whole page — the
// old list is still in the scrollback, so a page-wide assertion would pass on
// the very thing this is about.
func TestARoomIsCurrentEveryTimeYouComeBack(t *testing.T) {
	f := &fakeStore{
		chores: []squirrel.Chore{
			{ID: 2, Name: "water the plants", EveryDays: 3, SinceDays: 4, Active: true},
		},
		turns: []squirrel.Turn{{
			ID: 1, Who: squirrel.SpeakerBuddy, Words: "1 comes back.",
			Shown: []byte(`{"cards":[{"kind":"chore","title":"the bins"}]}`),
		}},
	}
	m := newTestMux()
	require.NoError(t, Mount(m, f, signedInOptions()))

	body := m.call(t, "GET", "/r/chores", nil).Body.String()
	edge := body[strings.Index(body, `id="edge"`):]

	require.Contains(t, edge, "water the plants",
		"the room drew nothing, because the conversation already ended open")
	require.NotContains(t, edge, "the bins",
		"the edge is showing a chore that is only in the scrollback")
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
	rail := roomsFor(context.Background(), f, 1, "everything")

	by := map[string]railView{}
	for _, r := range rail {
		by[r.Key] = r
	}
	require.Len(t, by, len(rooms))
	require.Equal(t, 2, by["notes"].Count)
	require.Equal(t, 1, by["chores"].Count)
	require.Zero(t, by["tasks"].Count, "an empty room carries a number")
	require.Zero(t, by["everything"].Count, "the room you are standing in carries a number")
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

	for _, where := range []string{"/r/everything", "/r/chores", "/r/notes"} {
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
		if r.Key == "everything" {
			// Everything is deliberately absent there: it is not a room he is
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

	require.NotContains(t, m.call(t, "GET", "/r/everything", nil).Body.String(), `<a class="brand" href="/"`)
	require.Contains(t, m.call(t, "GET", "/r/chores", nil).Body.String(), `<a class="brand" href="/"`)
}
