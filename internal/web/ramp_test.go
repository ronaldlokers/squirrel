package web

import (
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func rampingPile() *fakeStore {
	at := now()
	return &fakeStore{
		checkin: fresh(),
		ramp: squirrel.Timer{
			Label: "the tax return", Started: at.Add(-160 * time.Minute), Ends: at.Add(-135 * time.Minute),
		},
		hasRamp: true,
		items:   []squirrel.Item{note(1, "the boiler", squirrel.ItemOpen)},
		waiting: squirrel.Waiting{Pile: 1},
	}
}

// It says how long you have been at it, in a person's units.
func TestTheExitRampSaysHowLong(t *testing.T) {
	body := thread(t, rampingPile())

	require.Contains(t, body, "the tax return")
	require.Contains(t, body, "2h 40m")
	require.NotContains(t, body, "160 minutes")
}

func TestItOffersAPlaceToStopRatherThanTellingYouTo(t *testing.T) {
	f := rampingPile()
	body := thread(t, f)

	require.Contains(t, body, "a good place to stop is after this bit")
	require.Contains(t, body, "you asked me to say something")
	for _, order := range []string{"you should stop", "time to stop", "stop now", "!"} {
		require.NotContains(t, strings.ToLower(f.appended[0].Words), strings.ToLower(order),
			"the one interruption this product allows itself gave an order")
	}
}

func TestTheExitRampOffersThreeAnswers(t *testing.T) {
	body := thread(t, rampingPile())

	require.Contains(t, body, "STOPPING")
	require.Contains(t, body, "20 more minutes")
	require.Contains(t, body, "leave me alone")
}

// It is marked said in the same breath as being drawn. "Once" is the entire
// safety property, so drawing it without recording it would be the version of
// this that repeats.
func TestDrawingItMarksItSaid(t *testing.T) {
	f := rampingPile()
	thread(t, f)

	require.Equal(t, 1, f.rampSaid, "it was shown without being marked said")
}

func TestIfItCannotBeMarkedItIsNotSaid(t *testing.T) {
	f := rampingPile()
	f.rampSaidErr = errors.New("the database is unwell")

	require.NotContains(t, thread(t, f), "a good place to stop")
}

// It comes before everything else in the opening, because it is the only one
// about something happening now.
func TestTheExitRampComesFirst(t *testing.T) {
	f := rampingPile()
	f.run, f.hasRun = squirrel.Run{Place: squirrel.RunPile, Since: 10 * time.Minute}, true
	f.quiet, f.hasQuiet = squirrel.HeldItem{ID: 9, Text: "the referral",
		State: squirrel.ItemWaiting, Since: 30 * 24 * time.Hour}, true

	body := thread(t, f)
	require.Contains(t, body, "a good place to stop")
	require.NotContains(t, body, "part way through")
	require.NotContains(t, body, "gone quiet")
}

// Nothing armed, nothing said. This is every timer that existed before the
// feature and every timer the chat starts.
func TestWithNoRampNothingIsSaid(t *testing.T) {
	f := rampingPile()
	f.hasRamp = false

	require.NotContains(t, thread(t, f), "a good place to stop")
}

// A read that fails costs the sentence and nothing else.
func TestARampReadThatFailsIsNotAnErrorPage(t *testing.T) {
	f := rampingPile()
	f.hasRamp, f.rampErr = false, errors.New("the database is unwell")

	body := thread(t, f)
	require.NotContains(t, body, "a good place to stop")
	require.Contains(t, body, `id="thread"`)
}

func TestTickingTheBoxArmsTheRamp(t *testing.T) {
	f := &fakeStore{}
	m := routed(t, f)

	m.call(t, "POST", "/timer", strings.NewReader(
		url.Values{"minutes": {"25"}, "label": {"the tax return"}, "ramp": {"1"}}.Encode()))

	require.Equal(t, []bool{true}, f.armed)
}

// Not ticking it leaves the timer unwatched, which is what every timer was
// before this existed.
func TestNotTickingTheBoxLeavesItUnwatched(t *testing.T) {
	f := &fakeStore{}
	m := routed(t, f)

	m.call(t, "POST", "/timer", strings.NewReader(
		url.Values{"minutes": {"25"}, "label": {"the tax return"}}.Encode()))

	require.Empty(t, f.armed, "a timer nobody opted in on was armed anyway")
}

// "leave me alone" silences the day and leaves the timer running. You did not
// say you were stopping; you said you did not want to be asked.
func TestLeaveMeAloneSilencesTheDayAndKeepsTheTimer(t *testing.T) {
	f := &fakeStore{timer: &squirrel.Timer{Label: "the tax return", Ends: now().Add(time.Hour)}}
	m := routed(t, f)

	m.call(t, "POST", "/timer", strings.NewReader("hush=1&from=home"))

	require.Equal(t, 1, f.hushed)
	require.NotNil(t, f.timer, "leave me alone stopped the timer")
}

func TestHowLongYouHaveBeenAtItIsSaidInPersonUnits(t *testing.T) {
	for _, c := range []struct {
		d    time.Duration
		want string
	}{
		{40 * time.Minute, "40m"},
		{60 * time.Minute, "1h 0m"},
		{160 * time.Minute, "2h 40m"},
	} {
		require.Equal(t, c.want, onItInWords(c.d))
	}
}
