package web

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A chore usually starts as a note, and that path is unchanged. This is the
// other case: you are standing in the kitchen having just descaled the kettle,
// and the thing you want is for that to come back — not a note about wanting
// it to.
func TestMakingAChoreFromNothing(t *testing.T) {
	f := &fakeStore{}

	w := post(t, mounted(t, f), "/chores/new", url.Values{
		"name": {"descale the kettle"}, "every": {"every month"}, "part": {"morning"},
	})

	require.Equal(t, 303, w.Code)
	require.Len(t, f.chores, 1)
	require.Equal(t, "descale the kettle", f.chores[0].Name)
	require.Equal(t, 30*24*time.Hour, f.chores[0].Every)
	require.Equal(t, squirrel.Morning, f.chores[0].Ask.Part)
}

// Any time is the default, and it is a real answer rather than the absence of
// one — a chore with no preference is the common case.
func TestAChoreMadeWithNoPreferenceHasNone(t *testing.T) {
	f := &fakeStore{}

	post(t, mounted(t, f), "/chores/new", url.Values{
		"name": {"water the ferns"}, "every": {"every week"}, "part": {""},
	})

	require.Len(t, f.chores, 1)
	require.Equal(t, squirrel.AnyPart, f.chores[0].Ask.Part)
	require.Equal(t, "", f.chores[0].Ask.Words(), "nothing to say about when")
}

// An empty form submitted by accident is not a mistake worth a sentence.
func TestAnEmptyNewChoreSaysNothing(t *testing.T) {
	f := &fakeStore{}

	w := post(t, mounted(t, f), "/chores/new", url.Values{"name": {"  "}, "every": {"every week"}})

	require.Equal(t, 303, w.Code)
	require.Empty(t, f.chores)
}

// The interval comes from the chips, so it is one of four phrases the core
// already parses. Anything else is not an interval this screen offers.
func TestANewChoreNeedsAnIntervalItWasOffered(t *testing.T) {
	f := &fakeStore{}

	post(t, mounted(t, f), "/chores/new", url.Values{
		"name": {"something"}, "every": {"every fortnight or so"},
	})

	require.Empty(t, f.chores)
}

// The form is last on the screen, after what you already have — the opposite
// of an app that opens on an empty form.
func TestAnEmptyListSaysHowToMakeOne(t *testing.T) {
	full := opened(t, &fakeStore{chores: []squirrel.Chore{
		{ID: 1, Name: "bins out", Active: true, Every: 14 * 24 * time.Hour, EveryDays: 14},
	}}, "chores")
	empty := opened(t, &fakeStore{}, "chores")

	// The form went with the screen. What replaced it is the sentence — the
	// dock already understands "every 2 weeks: descale the kettle" — and it is
	// said when there is nothing there, which is when it is worth knowing.
	// Saying it over a list you already keep is nagging.
	require.Contains(t, empty, "every 2 weeks: descale the kettle")
	require.NotContains(t, full, "descale the kettle")
	require.Contains(t, full, "bins out")
}

func indexOf(body, sub string) int {
	for i := 0; i+len(sub) <= len(body); i++ {
		if body[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
