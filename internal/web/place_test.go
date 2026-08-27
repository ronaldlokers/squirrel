package web

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Losing your place is the failure this product is built around, and the
// conversation used to open as if nothing had been happening.
func TestBuddyOffersBackWhereYouGotTo(t *testing.T) {
	f := &fakeStore{
		checkin: fresh(),
		run:     squirrel.Run{Place: squirrel.RunPile, Since: 40 * time.Minute},
		hasRun:  true,
		items:   []squirrel.Item{note(1, "the boiler", squirrel.ItemOpen)},
		waiting: squirrel.Waiting{Pile: 1},
	}
	body := thread(t, f)

	require.Contains(t, body, "part way through the pile")
	require.Contains(t, body, "40 minutes ago")
	require.Contains(t, body, "carry on")
	require.Contains(t, body, "start fresh")
}

// And it wins over what is coming. The dentist will still be there after you
// have been told about it; the run is the thing you were doing.
func TestWhereYouGotToComesBeforeWhatIsComing(t *testing.T) {
	at := time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)
	was := now
	now = func() time.Time { return at }
	t.Cleanup(func() { now = was })

	f := &fakeStore{
		checkin:  fresh(),
		run:      squirrel.Run{Place: squirrel.RunPile, Since: 12 * time.Minute},
		hasRun:   true,
		items:    []squirrel.Item{note(1, "the boiler", squirrel.ItemOpen)},
		waiting:  squirrel.Waiting{Pile: 1, Agenda: 1},
		upcoming: []squirrel.Moment{{ID: 1, Label: "dentist", Starts: at.Add(3 * time.Hour)}},
	}
	body := thread(t, f)

	require.Contains(t, body, "part way through")
	require.NotContains(t, body, "dentist today",
		"it opened with the agenda while a run was waiting to be resumed")
}

// No run, no sentence. This is the ordinary case and it must be silent.
func TestWithNoRunNothingIsSaidAboutOne(t *testing.T) {
	f := &fakeStore{checkin: fresh(), items: []squirrel.Item{note(1, "the boiler", squirrel.ItemOpen)},
		waiting: squirrel.Waiting{Pile: 1}}
	body := thread(t, f)

	require.NotContains(t, body, "part way through")
	require.NotContains(t, body, "start fresh")
}

// A read that fails costs the sentence and nothing else. You came to talk, not
// to be told the database is unwell.
func TestARunThatCannotBeReadIsNotAnErrorPage(t *testing.T) {
	f := &fakeStore{checkin: fresh(), runErr: errors.New("the database is unwell"),
		items: []squirrel.Item{note(1, "the boiler", squirrel.ItemOpen)}, waiting: squirrel.Waiting{Pile: 1}}
	body := thread(t, f)

	require.NotContains(t, body, "part way through")
	require.Contains(t, body, `id="thread"`,
		"a failed run read took the whole screen with it")
	require.NotContains(t, body, "cannot reach its memory")
}

// Answering a card is what remembers your place. Marked on every answer, so the
// clock measures silence rather than how long the run has been going.
func TestAnsweringACardKeepsYourPlace(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(7, "the boiler", squirrel.ItemOpen)}}
	m := routed(t, f)

	m.call(t, "POST", "/pile/act", strings.NewReader("id=7&act=done&was=open"))

	require.Equal(t, []string{squirrel.RunPile}, f.marked,
		"deciding on a note did not remember where you got to")
}

func TestStartingFreshForgetsTheRun(t *testing.T) {
	f := &fakeStore{run: squirrel.Run{Place: squirrel.RunPile}, hasRun: true,
		items: []squirrel.Item{note(1, "the boiler", squirrel.ItemOpen)}}
	m := routed(t, f)

	m.call(t, "POST", "/place/fresh", nil)

	require.Equal(t, 1, f.ended, "start fresh did not forget the run")
	require.Equal(t, squirrel.ItemOpen, f.items[0].State,
		"start fresh changed a note; it is only meant to forget the run")
}

// Stopping ends it too. Being offered your place back after choosing to stop
// would make the screen that says stopping is normal argue with you about it.
func TestStoppingForgetsTheRun(t *testing.T) {
	f := &fakeStore{run: squirrel.Run{Place: squirrel.RunPile}, hasRun: true}
	m := routed(t, f)

	m.call(t, "GET", "/enough", nil)

	require.Equal(t, 1, f.ended, "stopping left a run to be offered back")
}

// The sentence never names a clock time. How long ago is a fact about the gap;
// "you stopped at 14:12" is a record of your afternoon.
func TestWhereYouGotToNamesNoClockTime(t *testing.T) {
	for _, since := range []time.Duration{30 * time.Second, 40 * time.Minute, 90 * time.Minute, 4 * time.Hour} {
		f := &fakeStore{checkin: fresh(), run: squirrel.Run{Place: squirrel.RunPile, Since: since},
			hasRun: true, items: []squirrel.Item{note(1, "the boiler", squirrel.ItemOpen)},
			waiting: squirrel.Waiting{Pile: 1}}
		said := thread(t, f)
		said = said[strings.Index(said, "part way through"):]
		said = said[:min(len(said), 120)]
		require.NotContains(t, said, ":", "the sentence put a clock time in the record")
	}
}
