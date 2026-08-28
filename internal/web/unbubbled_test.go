package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Buddy's words are not in a bubble and yours are.
//
// The rule this enforces is not about Buddy: it is that a bubble means "this is
// a thing rather than a sentence". A screen where both speakers wear stock is a
// screen where a note and a remark look alike.
func TestBuddyHasNoBubbleAndYouDo(t *testing.T) {
	body := thread(t, &fakeStore{turns: []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerYou, Words: "ask the garage"},
		{ID: 2, Who: squirrel.SpeakerBuddy, Words: "kept."},
	}})

	require.Contains(t, body, `<p class="bub">ask the garage</p>`)
	require.Contains(t, body, `<p class="said">kept.</p>`)
	require.NotContains(t, body, `<p class="bub">kept.</p>`,
		"Buddy is still wearing a bubble")
}

// Count the outlined boxes. Seven became three, with the same information.
func TestTheConversationHasFewerObjectsInIt(t *testing.T) {
	body := thread(t, &fakeStore{
		waiting: squirrel.Waiting{Pile: 3, Agenda: 1},
		turns: []squirrel.Turn{
			{ID: 1, Who: squirrel.SpeakerBuddy, Words: "dentist at 14:30."},
			{ID: 2, Who: squirrel.SpeakerYou, Words: "ask the garage"},
			{ID: 3, Who: squirrel.SpeakerBuddy, Words: "kept."},
		},
	})

	// Every turn of Buddy's is words on the field, and every one of yours is a
	// bubble. Stated as a relationship rather than as two numbers, because
	// Buddy opens with a turn of his own and the count of those is not the
	// point.
	require.Equal(t, strings.Count(body, "frombuddy"), strings.Count(body, `class="said"`),
		"a turn of Buddy's rendered as something other than words on the field")
	require.Equal(t, strings.Count(body, "fromyou"), strings.Count(body, `class="bub"`),
		"a turn of yours rendered as something other than a bubble")
	require.Positive(t, strings.Count(body, "frombuddy"))
}

func TestTheWayOutIsStillOnePress(t *testing.T) {
	body := thread(t, &fakeStore{})

	require.Contains(t, body, `href="/enough"`)
	require.Contains(t, body, `class="leaving"`)
}

// The rooms need no script.
//
// A <details> on a phone and furniture on a desktop, from one markup — and a
// <details> opens with the script off, which is the grammar every other
// disclosure here already uses. What must never come back is a room that needs
// JavaScript to be reachable.
func TestTheRoomsNeedNoScript(t *testing.T) {
	body := thread(t, &fakeStore{})

	require.Contains(t, body, `<details class="roomsheet">`)
	require.Contains(t, body, `<nav class="rail"`)
	for _, r := range rooms {
		require.Contains(t, body, `href="/r/`+r.Key+`"`,
			"%s is not a plain link", r.Key)
	}
	require.NotContains(t, body, `onclick`)
}

// Everywhere is reachable from every screen, not only the conversation. A rail
// that emptied itself on the screen you navigated to would be a one-way door.
func TestTheRailIsOnEveryOtherScreenToo(t *testing.T) {
	m := mounted(t, &fakeStore{checkin: fresh()})

	for _, screen := range []string{"/", "/moods", "/enough"} {
		body := m.call(t, "GET", screen, nil).Body.String()
		require.Contains(t, body, `<nav class="rail"`, "%s has no rail", screen)
		require.Contains(t, body, `href="/r/pile"`, "%s cannot reach the pile", screen)
	}
}
