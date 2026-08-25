package web

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Typing into the box is talking to Buddy.
//
// It was a capture slot that said "Kept.", which is what a filing cabinet
// says. Ronald asked on 25 August 2026 for typing to be talking, and chose the
// version where Buddy decides what the words were — knowing, because it was
// said at the time, that a model between you and the capture promise can be
// wrong.
//
// Everything below is about where that risk lands. The words are spooled and
// already a note before Buddy is asked anything, so the worst any failure can
// do is leave a note in the pile you did not want.

func aThought(reply string) func(string) (string, bool, error) {
	return func(string) (string, bool, error) { return reply, true, nil }
}

func aQuestion(reply string) func(string) (string, bool, error) {
	return func(string) (string, bool, error) { return reply, false, nil }
}

// A thought is kept, and answered with something real rather than "Kept."
func TestAThoughtIsKeptAndAnswered(t *testing.T) {
	f := &fakeStore{}
	m := mountedReading(t, f, aThought("That is the third time the boiler has come up."))

	m.call(t, "POST", "/capture", strings.NewReader("text=the+boiler+again"))

	require.Equal(t, []string{"the boiler again"}, f.readAsked)
	require.Len(t, f.appended, 2)
	require.Equal(t, "the boiler again", f.appended[0].Words)
	require.Equal(t, "That is the third time the boiler has come up.", f.appended[1].Words)
	require.Empty(t, f.states, "a thought was dropped")
}

// A question is answered, and the note it made is dropped rather than left in
// the pile. Dropped and not deleted: the words stay in the database, and the
// pile can put it back.
func TestAQuestionIsAnsweredAndNotLeftInThePile(t *testing.T) {
	f := &fakeStore{}
	f.items = []squirrel.Item{note(7, "what should I do about the tax thing", squirrel.ItemOpen)}
	m := mountedReading(t, f, aQuestion("Open the envelope and read the first line."))

	m.call(t, "POST", "/capture",
		strings.NewReader("text=what+should+I+do+about+the+tax+thing"))

	require.Equal(t, "Open the envelope and read the first line.", f.appended[1].Words)
	require.Equal(t, squirrel.ItemDropped, f.states[7], "the question stayed in the pile")
}

// The one that matters. Every failure leaves the words in the pile, because
// they were kept before Buddy was asked anything.

// No coach at all: the box does exactly what it always did.
func TestWithNoCoachTheBoxKeepsAndSaysSo(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/capture", strings.NewReader("text=the+boiler"))

	require.Len(t, f.appended, 2)
	require.Contains(t, squirrel.Sayings(squirrel.SayingKept), f.appended[1].Words)
	require.Empty(t, f.states)
}

// A model that cannot be reached, or a budget that is spent.
func TestAnUnreachableModelKeepsTheWords(t *testing.T) {
	f := &fakeStore{}
	f.items = []squirrel.Item{note(7, "the boiler", squirrel.ItemOpen)}
	m := mountedReading(t, f, func(string) (string, bool, error) {
		return "", true, errors.New("no coach")
	})

	m.call(t, "POST", "/capture", strings.NewReader("text=the+boiler"))

	require.Contains(t, squirrel.Sayings(squirrel.SayingKept), f.appended[1].Words)
	require.Empty(t, f.states, "an unreachable model threw a thought away")
}

// And the words are spooled before any of it, which is what makes all of the
// above true rather than merely arranged.
func TestTheWordsAreSpooledBeforeBuddyIsAsked(t *testing.T) {
	f := &fakeStore{}
	sp := &fakeSpool{}
	var spooledFirst bool
	m := newTestMux()
	require.NoError(t, Mount(m, f, Options{
		IdentityHeader: "X-Authentik-Username", Identity: "ronald",
		Owner: func() int64 { return 1 }, Spool: sp,
		Reads: func(_ context.Context, _ int64, said string) (string, bool, error) {
			spooledFirst = len(sp.written) == 1
			return "answered", false, nil
		},
	}))

	m.call(t, "POST", "/capture", strings.NewReader("text=what+now"))

	require.True(t, spooledFirst,
		"Buddy was asked before the words were durable, so a crash mid-call loses them")
}

// A note that does not match what was typed is not dropped. The drain may not
// have caught up, and dropping the wrong note is the one thing worse than
// leaving the right one.
func TestOnlyTheNoteThatMatchesIsDropped(t *testing.T) {
	f := &fakeStore{}
	f.items = []squirrel.Item{note(7, "something else entirely", squirrel.ItemOpen)}
	m := mountedReading(t, f, aQuestion("Open the envelope."))

	m.call(t, "POST", "/capture", strings.NewReader("text=what+now"))

	require.Empty(t, f.states, "it dropped a note it had not just made")
	require.Equal(t, "Open the envelope.", f.appended[1].Words, "the answer was lost too")
}

// A photograph is kept and never read. It is not words, there is nothing to
// answer, and it is the one capture that is hardest to make again.
func TestAPhotographIsNeverJudged(t *testing.T) {
	f := &fakeStore{}
	ph := &fakePhotos{}
	m := newTestMux()
	var asked bool
	require.NoError(t, Mount(m, f, Options{
		IdentityHeader: "X-Authentik-Username", Identity: "ronald",
		Owner: func() int64 { return 1 }, Spool: &fakeSpool{}, Photos: ph,
		Reads: func(context.Context, int64, string) (string, bool, error) {
			asked = true
			return "", false, nil
		},
	}))

	// With words on it, so the photograph itself is the only thing that can
	// stop this being read — an empty box would stop it anyway.
	kind, body := photographed(t, "the meter", "image/jpeg", []byte("some jpeg bytes"))
	postPhoto(t, m, kind, body)

	require.False(t, asked, "a photograph was handed to a model to judge")
	require.Empty(t, f.states)
}

// An empty box is still nothing at all, and costs nothing.
func TestAnEmptyBoxAsksNothing(t *testing.T) {
	f := &fakeStore{}
	m := mountedReading(t, f, aThought("should not be called"))

	m.call(t, "POST", "/capture", strings.NewReader("text=+++"))

	require.Empty(t, f.readAsked)
}
