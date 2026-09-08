package web

import (
	"strings"
	"testing"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/stretchr/testify/require"
)

func aNoteAskedAboutTwice() *fakeStore {
	f := aRackWithoutAgenda()
	f.noticed = []squirrel.Noticed{
		{ID: 1, Kind: "ask:note", RefID: 1, Words: "The boiler code you wanted is on the note from August.",
			At: time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)},
		{ID: 2, Kind: "ask:note", RefID: 1, Words: "This is the third note about that boiler.",
			At: time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)},
	}
	return f
}

func TestAnOpenedStripSaysWhatWasNoticedAboutItBefore(t *testing.T) {
	body := mounted(t, aNoteAskedAboutTwice()).call(t, "GET", "/?open=1", nil).Body.String()

	require.Contains(t, body, "what Squirrel said about this before",
		"the older lines are kept and nothing reads them back")
	require.Contains(t, body, "The boiler code you wanted is on the note from August.")
}

func TestAnOpenedStripDoesNotRepeatTheLineTheRackAlreadyShows(t *testing.T) {
	body := mounted(t, aNoteAskedAboutTwice()).call(t, "GET", "/?open=1", nil).Body.String()

	require.Equal(t, 1, strings.Count(body, "This is the third note about that boiler."),
		"the newest line is drawn twice on the same screen")
}

func TestOneLineOnlyIsWhatTheRackDraws(t *testing.T) {
	body := mounted(t, aNoteAskedAboutTwice()).call(t, "GET", "/?bay=notes", nil).Body.String()
	rack := theRackIn(t, body, "bay=notes")

	require.Contains(t, rack, "This is the third note about that boiler.")
	require.NotContains(t, rack, "The boiler code you wanted is on the note from August.",
		"two lines under one strip is a conversation, and a rack is not one")
	require.NotContains(t, rack, "what Squirrel said about this before")
}

func TestAStripAskedAboutOnceHasNoBefore(t *testing.T) {
	f := aRackWithoutAgenda()
	f.noticed = []squirrel.Noticed{
		{ID: 2, Kind: "ask:note", RefID: 1, Words: "This is the third note about that boiler.",
			At: time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)},
	}

	body := mounted(t, f).call(t, "GET", "/?open=1", nil).Body.String()

	require.NotContains(t, body, "what Squirrel said about this before",
		"a heading stands over nothing")
}

func TestAnOpenedStripStillShowsTheLineTheRackWasShowing(t *testing.T) {
	body := mounted(t, aNoteAskedAboutTwice()).call(t, "GET", "/?open=1", nil).Body.String()

	require.Contains(t, body, "This is the third note about that boiler.",
		"opening a strip loses the line the rack was showing")
	require.Contains(t, body, `action="/board/notuseful"`,
		"the newest line can be refused in the rack and not on the strip it is about")
}
