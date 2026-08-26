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
// It was a capture slot that said "Kept.", which is what a filing cabinet says.
// The owner asked for typing to be talking, and chose the version where Buddy
// decides what the words were — knowing, and saying so at the time, that a model
// between you and the capture promise can be wrong.
//
// Everything below is about where that risk lands. The words are spooled and
// already a note before Buddy is asked anything, so the worst any failure can
// do is leave a note in the pile you did not want.

func aThought(reply string) func(string) (string, bool, string, error) {
	return func(string) (string, bool, string, error) { return reply, true, "", nil }
}

func aQuestion(reply string) func(string) (string, bool, string, error) {
	return func(string) (string, bool, string, error) { return reply, false, "", nil }
}

// This is the architecture the product states and briefly stopped following:
// rules narrow, and the model answers the few that survive. Almost everything
// typed into this box is a thought, so almost everything costs nothing.
func TestAThoughtNeverReachesAModel(t *testing.T) {
	f := &fakeStore{}
	m := mountedReading(t, f, aThought("should not be called"))

	m.call(t, "POST", "/capture", strings.NewReader("text=the+boiler+is+making+that+noise+again"))

	require.Empty(t, f.readAsked, "a thought was sent abroad to be read")
	require.Len(t, f.appended, 2)
	require.Equal(t, "the boiler is making that noise again", f.appended[0].Words)
	require.Contains(t, squirrel.Sayings(squirrel.SayingKept), f.appended[1].Words)
	require.Empty(t, f.states, "a thought was dropped")
}

// A question is answered, and the note it made is dropped rather than left in
// the pile. Dropped and not deleted: the words stay in the database, and the
// pile can put it back.
func TestAQuestionIsAnsweredAndNotLeftInThePile(t *testing.T) {
	f := &fakeStore{}
	f.items = []squirrel.Item{note(7, "what should I do about the tax thing?", squirrel.ItemOpen)}
	m := mountedReading(t, f, aQuestion("Open the envelope and read the first line."))

	m.call(t, "POST", "/capture",
		strings.NewReader("text=what+should+I+do+about+the+tax+thing%3F"))

	require.Equal(t, "Open the envelope and read the first line.", f.appended[1].Words)
	require.Equal(t, squirrel.ItemDropped, f.states[7], "the question stayed in the pile")
}

// The one that matters. Every failure leaves the words in the pile, because
// they were kept before Buddy was asked anything.

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
	f.items = []squirrel.Item{note(7, "what now?", squirrel.ItemOpen)}
	m := mountedReading(t, f, func(string) (string, bool, string, error) {
		return "", true, "", errors.New("no coach")
	})

	m.call(t, "POST", "/capture", strings.NewReader("text=what+now%3F"))

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
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
		Spool:    sp,
		Reads: func(_ context.Context, _ int64, said string) (string, bool, string, error) {
			spooledFirst = len(sp.written) == 1
			return "answered", false, "", nil
		},
	}))

	m.call(t, "POST", "/capture", strings.NewReader("text=what+should+I+do%3F"))

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

	m.call(t, "POST", "/capture", strings.NewReader("text=what+now%3F"))

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
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
		Spool:    &fakeSpool{}, Photos: ph,
		Reads: func(context.Context, int64, string) (string, bool, string, error) {
			asked = true
			return "", false, "", nil
		},
	}))

	// With words on it, so the photograph itself is the only thing that can
	// stop this being read — an empty box would stop it anyway.
	kind, body := photographed(t, "the meter", "image/jpeg", []byte("some jpeg bytes"))
	postPhoto(t, m, kind, body)

	require.False(t, asked, "a photograph was handed to a model to judge")
	require.Empty(t, f.states)
}

func TestAnEmptyBoxAsksNothing(t *testing.T) {
	f := &fakeStore{}
	m := mountedReading(t, f, aThought("should not be called"))

	m.call(t, "POST", "/capture", strings.NewReader("text=+++"))

	require.Empty(t, f.readAsked)
}

// The three tiers, and which one answers.

// The rule is the floor and it needs nothing running. No house, no key, and
// the box still tells a question from a thought.
func TestTheRuleAnswersWithNothingRunning(t *testing.T) {
	for _, thought := range []string{
		"the boiler is making that noise again",
		"bins out thursday",
		"meter reading 48213",
		"ring the vet? no, the dentist",
		"what a day",
	} {
		require.False(t, squirrel.LooksLikeAQuestion(thought), "%q read as a question", thought)
	}
	for _, question := range []string{
		"what should I do about the tax thing?",
		"what should I do about the tax thing",
		"should I ring them today",
		"can you break this down",
		"how do i even start this",
	} {
		require.True(t, squirrel.LooksLikeAQuestion(question), "%q read as a thought", question)
	}
}

// The house is asked about everything and overrules the rule, because it has
// read the sentence rather than matched its shape.
func TestTheHouseOverrulesTheRule(t *testing.T) {
	f := &fakeStore{}
	f.items = []squirrel.Item{note(7, "remind me what the boiler code was", squirrel.ItemOpen)}
	var housed []string
	m := newTestMux()
	require.NoError(t, Mount(m, f, Options{
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
		Spool:    &fakeSpool{},
		AskedAQuestion: func(_ context.Context, said string) (bool, bool) {
			housed = append(housed, said)
			return true, true
		},
		Reads: func(context.Context, int64, string) (string, bool, string, error) {
			return "4471.", false, "", nil
		},
	}))

	m.call(t, "POST", "/capture",
		strings.NewReader("text=remind+me+what+the+boiler+code+was"))

	require.Len(t, housed, 1, "the house was not asked")
	require.Equal(t, "4471.", f.appended[1].Words, "the rule won over the house")
	require.Equal(t, squirrel.ItemDropped, f.states[7])
}

func TestAHouseThatDoesNotAnswerFallsThroughToTheRule(t *testing.T) {
	f := &fakeStore{}
	m := newTestMux()
	require.NoError(t, Mount(m, f, Options{
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions: newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:    aTestLogin,
		Spool:    &fakeSpool{},
		// True and "did not answer". Believing the first return without
		// checking the second would send a thought abroad — which is what the
		// mutation that caught this test being weak actually did.
		AskedAQuestion: func(context.Context, string) (bool, bool) { return true, false },
		Reads: func(context.Context, int64, string) (string, bool, string, error) {
			return "should not be called", false, "", nil
		},
	}))

	m.call(t, "POST", "/capture", strings.NewReader("text=the+boiler+again"))

	require.Contains(t, squirrel.Sayings(squirrel.SayingKept), f.appended[1].Words)
}

// A question with nobody to answer it is kept, which is the honest outcome: a
// question nobody answered is a note you will want to see again.
func TestAQuestionWithNoCoachIsKept(t *testing.T) {
	f := &fakeStore{}
	f.items = []squirrel.Item{note(7, "what should I do?", squirrel.ItemOpen)}
	routed(t, f).call(t, "POST", "/capture", strings.NewReader("text=what+should+I+do%3F"))

	require.Contains(t, squirrel.Sayings(squirrel.SayingKept), f.appended[1].Words)
	require.Empty(t, f.states, "a question nobody could answer was thrown away")
}

// And the way out when the rule reads it wrong: one press hands the words to
// Buddy properly.
func TestAnAcknowledgementOffersToBeAnswered(t *testing.T) {
	f := &fakeStore{}
	m := mountedReading(t, f, aThought("should not be called"))

	m.call(t, "POST", "/capture", strings.NewReader("text=the+boiler+again"))

	drew := string(f.appended[1].Shown)
	require.Contains(t, drew, "answer this")
	require.Contains(t, drew, "/buddy/say")
	require.Contains(t, drew, "the boiler again", "the chip does not carry the words")
}

// Not on something Buddy actually answered. A chip offering to answer what has
// just been answered is furniture.
func TestAnAnswerDoesNotOfferToBeAnswered(t *testing.T) {
	f := &fakeStore{}
	f.items = []squirrel.Item{note(7, "what should I do?", squirrel.ItemOpen)}
	m := mountedReading(t, f, aQuestion("Open the envelope."))

	m.call(t, "POST", "/capture", strings.NewReader("text=what+should+I+do%3F"))

	require.NotContains(t, string(f.appended[1].Shown), "answer this")
}

func TestWithNoCoachThereIsNoWayToBeAnswered(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/capture", strings.NewReader("text=the+boiler+again"))

	require.NotContains(t, string(f.appended[1].Shown), "answer this")
}

// A command is never a question. The grammar has already claimed these, and
// answering them with a model would be answering something the product knows
// how to do itself.
func TestACommandIsNeverAQuestion(t *testing.T) {
	for _, command := range []string{"!find boiler", "!at 14:30 dentist", "!notes"} {
		require.False(t, squirrel.LooksLikeAQuestion(command), "%q read as a question", command)
	}
}

// The box can show you a place. This is the bug from the screenshot: asking to
// see the chores in the one box this product has got "I can't see your chores
// from here", because the dock's reading path carried no tools at all and the
// fix in v0.42.0 had landed on the menu's route instead.
func TestTheDockCanShowYouAPlace(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 1, Name: "bleed the boiler", Active: true, EveryDays: 30},
	}}
	m := mountedReading(t, f, func(string) (string, bool, string, error) {
		return "Here they are.", false, "chores", nil
	})
	m.call(t, "POST", "/capture", strings.NewReader("text=show+me+the+chores"))

	require.Len(t, f.appended, 3)
	require.Equal(t, "Here they are.", f.appended[1].Words)
	require.Equal(t, squirrel.SpeakerBuddy, f.appended[2].Who)
	require.Contains(t, string(f.appended[2].Shown), "bleed the boiler")
}

// A thought is a thought. Nothing is drawn under one, whatever the model says
// about a place — a note being filed is not a request to look at something.
//
// The words have to be ones the gate lets through, or this proves nothing: a
// thought that never reaches the reading path never had a place to draw. What
// this pins is the case that does happen — the rule sent it to be answered,
// and the model read the whole sentence and disagreed.
func TestAThoughtNeverDrawsAPlace(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 1, Name: "bleed the boiler", Active: true, EveryDays: 30},
	}}
	m := mountedReading(t, f, func(string) (string, bool, string, error) {
		return "Kept.", true, "chores", nil
	})
	m.call(t, "POST", "/capture", strings.NewReader("text=show+me+the+chores"))

	require.NotEmpty(t, f.readAsked, "the reading path was never reached")
	require.Len(t, f.appended, 2)
}
