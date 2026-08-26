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

// The way out is still one press from anywhere. It moved; it did not go.
func TestTheWayOutIsStillOnePress(t *testing.T) {
	body := thread(t, &fakeStore{})

	require.Contains(t, body, `href="/enough"`)
	require.Contains(t, body, `class="leaving"`)
}

// The menu opens with the script off, which every other disclosure here does.
func TestTheMenuOpensWithoutScript(t *testing.T) {
	body := thread(t, &fakeStore{})

	require.Contains(t, body, `<details class="menu">`)
	require.Contains(t, body, "<summary")
}

// Everywhere is reachable from every screen, not only the conversation. A menu
// that emptied itself on the screen you navigated to would be a one-way door.
func TestTheMenuIsOnEveryScreen(t *testing.T) {
	m := mounted(t, &fakeStore{checkin: fresh()})

	for _, screen := range []string{"/", "/moods", "/enough"} {
		body := m.call(t, "GET", screen, nil).Body.String()
		require.Contains(t, body, `class="menupanel"`, "%s has no menu", screen)
		require.Contains(t, body, "the pile", "%s cannot reach the pile", screen)
	}
}
