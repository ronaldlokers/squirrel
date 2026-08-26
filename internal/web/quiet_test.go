package web

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func quietPile() *fakeStore {
	return &fakeStore{
		checkin: fresh(),
		quiet: squirrel.HeldItem{
			ID: 9, Text: "the referral", State: squirrel.ItemWaiting,
			Because: "the surgery", Kind: squirrel.ItemNote, Since: 22 * 24 * time.Hour,
		},
		hasQuiet: true,
		items:    []squirrel.Item{note(1, "the boiler", squirrel.ItemOpen)},
		waiting:  squirrel.Waiting{Pile: 1},
	}
}

// Something parked and forgotten comes back, with what it is waiting on and
// how long it has been.
func TestSomethingGoneQuietComesBack(t *testing.T) {
	body := thread(t, quietPile())

	require.Contains(t, body, "gone quiet")
	require.Contains(t, body, "the referral")
	require.Contains(t, body, "the surgery")
	require.Contains(t, body, "about 3 weeks")
}

// Three answers, and saying "still" is not harder than the other two. If it
// were, this would be a screen that pushes you to close things — the opposite
// of what parking something is for.
func TestTheThreeAnswersAreEqualInStanding(t *testing.T) {
	body := thread(t, quietPile())

	require.Contains(t, body, "still waiting")
	require.Contains(t, body, "chase it")
	require.Contains(t, body, "let it go")
}

// It never asks. It says the fact and stops.
func TestItDoesNotAskYouToDoAnything(t *testing.T) {
	f := quietPile()
	thread(t, f)

	said := f.appended[0].Words
	require.NotContains(t, said, "?")
	for _, nag := range []string{"should", "still need", "have you", "don't forget", "remember to"} {
		require.NotContains(t, strings.ToLower(said), nag,
			"the line about something you parked told you to do something")
	}
}

// Where you got to wins. A run in progress is a thing you were doing a minute
// ago; this is a thing nobody has touched for three weeks.
func TestWhereYouGotToWinsOverWhatHasGoneQuiet(t *testing.T) {
	f := quietPile()
	f.run, f.hasRun = squirrel.Run{Place: squirrel.RunPile, Since: 10 * time.Minute}, true

	body := thread(t, f)
	require.Contains(t, body, "part way through")
	require.NotContains(t, body, "gone quiet")
}

// Nothing quiet, nothing said. This is the ordinary case.
func TestWithNothingQuietNothingIsSaid(t *testing.T) {
	f := quietPile()
	f.hasQuiet = false

	require.NotContains(t, thread(t, f), "gone quiet")
}

// A read that fails costs the sentence and nothing else.
func TestAQuietReadThatFailsIsNotAnErrorPage(t *testing.T) {
	f := quietPile()
	f.hasQuiet, f.quietErr = false, errors.New("the database is unwell")

	body := thread(t, f)
	require.NotContains(t, body, "gone quiet")
	require.Contains(t, body, `id="thread"`)
	require.NotContains(t, body, "cannot reach its memory")
}

// "still waiting" moves the clock and does not move the note.
func TestStillWaitingMovesTheClockOnly(t *testing.T) {
	f := quietPile()
	f.items = []squirrel.Item{note(9, "the referral", squirrel.ItemWaiting)}
	m := routed(t, f)

	m.call(t, "POST", "/held/act", strings.NewReader("id=9&act=still"))

	require.Equal(t, []int64{9}, f.stilled, "still waiting did not move the clock")
	require.Equal(t, squirrel.ItemWaiting, f.items[0].State,
		"still waiting moved the note as well as the clock")
}

// How long is said the way somebody would say it. "23 days" is a measurement;
// "about 3 weeks" is a remark, and the difference matters on a line about a
// thing you have not done.
func TestHowLongIsSaidAsARemarkNotAMeasurement(t *testing.T) {
	for _, c := range []struct {
		since time.Duration
		want  string
	}{
		{3 * 24 * time.Hour, "3 days"},
		{9 * 24 * time.Hour, "over a week"},
		{22 * 24 * time.Hour, "about 3 weeks"},
		{95 * 24 * time.Hour, "about 3 months"},
	} {
		require.Equal(t, c.want, waitedInWords(c.since))
	}
}
