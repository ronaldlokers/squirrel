package web

import (
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// aScreenful is one of everything a room can draw, so the walk below has
// something to press on every screen.
func aScreenful() *fakeStore {
	f := &fakeStore{}
	f.items = append(f.items,
		note(1, "the bike rack", squirrel.ItemOpen),
		note(2, "ring the vet", squirrel.ItemOpen),
		note(3, "the boiler code", squirrel.ItemKept),
	)
	f.chores = []squirrel.Chore{{ID: 1, Name: "bins out", Every: 7 * 24 * time.Hour,
		EveryDays: 7, SinceDays: 6, Active: true, EverDone: true}}
	f.checkin = &squirrel.Checkin{Mood: squirrel.MoodCalm, SaidAt: time.Now()}
	f.readings = []squirrel.Checkin{{Mood: squirrel.MoodCalm, SaidAt: time.Now()}}
	f.aside = []squirrel.HeldItem{{
		ID: 9, Text: "chase the landlord", State: squirrel.ItemWaiting,
		Because: "waiting on him", Kind: squirrel.ItemNote,
	}}
	f.turns = []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerYou, Words: "the agenda"},
		{ID: 2, Who: squirrel.SpeakerBuddy, Words: "Nothing there yet."},
	}
	return f
}

// aNavigation is every press that legitimately takes you somewhere rather than
// answering in the thread. Short on purpose: the agenda's "a new appointment"
// looked exactly like one of these until it turned out to be a press the
// handler could not satisfy.
var aNavigation = map[string]bool{
	// Putting a note back, or letting go of something held, empties the room
	// you are standing in. There is nothing left to answer into.
	"/pile/act": true,
	"/held/act": true,
	// The board has no thread to answer into: every press on it re-draws the
	// board, which is a navigation by construction rather than by omission.
	"/board/act":  true,
	"/board/undo": true,
	"/board/new":  true,
}

var (
	formPattern   = regexp.MustCompile(`(?s)<form[^>]*method="post"[^>]*action="([^"]+)"[^>]*>(.*?)</form>`)
	hiddenPattern = regexp.MustCompile(`<input type="hidden" name="([^"]+)" value="([^"]*)"`)
	fullPage      = regexp.MustCompile(`(?i)<!doctype|<html|class="rail"|<nav`)
)

// Every button on every screen, pressed the way the script presses it.
//
// This exists because a bug nobody would have written a test for shipped: the
// agenda's "a new appointment" posted to the dock's route without the label
// that route requires, the handler answered with a redirect, and fetch follows
// a redirect without telling anybody — so the whole page came back and the
// script pasted a room, its navigation and all, inside the room.
//
// The invariant is one sentence: a press that says it wants a fragment must
// never be answered with a redirect or with a page. Per-button tests would not
// have caught it, because nobody writes the test for the press they did not
// think about.
func TestEveryButtonAnswersWithAFragment(t *testing.T) {
	m := routed(t, aScreenful())

	screens := map[string]string{}
	for _, screen := range []string{
		"/", "/board", "/r/everything", "/r/notes", "/r/chores", "/r/at", "/r/tasks",
		"/r/chores",
	} {
		page := m.call(t, "GET", screen, nil)
		require.Equal(t, 200, page.Code, "%s does not render", screen)
		screens[screen] = page.Body.String()
	}
	// The two shelves, which are drawn by a press rather than reached by a URL
	// since 31 August 2026. Their cards carry buttons like any other card, and
	// a walk that only walks GETs would stop covering them the day they stopped
	// being rooms — which is the exact way coverage goes quiet.
	for _, shelf := range []string{"kept", "held"} {
		press := m.callFragment(t, "/notes/shelf",
			url.Values{"room": {"notes"}, "shelf": {shelf}}.Encode())
		require.Equal(t, 200, press.Code, "the %s shelf does not draw", shelf)
		screens["the "+shelf+" shelf"] = press.Body.String()
	}

	for screen, drawn := range screens {
		for _, form := range formPattern.FindAllStringSubmatch(drawn, -1) {
			whole, action, fields := form[0], form[1], form[2]
			isTheDock := strings.Contains(whole, `class="slot"`)
			if strings.HasPrefix(action, "/auth/") {
				continue // Signing out is a navigation and says so.
			}
			values := url.Values{}
			for _, hidden := range hiddenPattern.FindAllStringSubmatch(fields, -1) {
				values.Set(hidden[1], hidden[2])
			}

			t.Run(screen+" → "+action, func(t *testing.T) {
				w := m.callFragment(t, action, values.Encode())

				if w.Code >= 300 && w.Code < 400 {
					// A redirect is legitimate when the press really is a
					// navigation — leaving a room by emptying it, say — and
					// thread.js now goes there instead of pasting what comes
					// back. It is listed rather than merely allowed: a new one
					// is a decision somebody has to make on purpose, because
					// this is the shape the agenda's bug wore.
					require.True(t, aNavigation[action],
						"answered %d and is not a listed navigation. Either it should answer in the thread, or add it to aNavigation and say why",
						w.Code)
					require.NotEmpty(t, w.Header().Get("Location"), "a redirect to nowhere")
					return
				}
				if w.Code == 204 {
					// An empty box, pressed: nothing back and nothing done,
					// which is what the script is written to expect. Only the
					// dock may say it. A chip always means something, so a
					// chip that answers nothing is a button that does nothing
					// — which is what the agenda's looked like once its
					// redirect was fixed and its aim was not.
					require.True(t, isTheDock,
						"a chip answered 204, so pressing it does nothing at all")
					return
				}
				require.Equal(t, 200, w.Code)
				require.NotRegexp(t, fullPage, w.Body.String(),
					"answered with a page rather than a fragment, which the script pastes into the room it is already in")
			})
		}
	}
}
