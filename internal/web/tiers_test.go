package web

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Four buttons end the card, and nothing else is on it.
//
// Seven equally-shaped buttons is six too many on a screen whose premise is
// that deciding is the expensive part, and "quieter" was not enough to say
// that four of them end the note and three do not.
func TestTheCardCarriesFourVerbsAndNoQuestions(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "the boiler", squirrel.ItemOpen)}}
	deck := opened(t, f, "notes")

	for _, verb := range []string{"done", "keep", "drop", "make a chore"} {
		require.Contains(t, deck, verb)
	}
	require.Equal(t, 4, strings.Count(deck, `name="answer" value=`)+strings.Count(deck, `name="chore" value=`),
		"the strip carries something other than the four verbs")
	for _, question := range []string{"make it a chore", "say it another way", "i can't act on this"} {
		require.NotContains(t, deck, question, "%q is still on the card", question)
	}
}

// The three questions behind one press went with the card on 2 September 2026.
// A strip carries its four answers and nothing else: correcting the words,
// asking why, and saying you cannot act on it are Buddy's, and Buddy is one
// press away in his room. What this protected — that a question is never a
// verb on the object — is the strip's own shape now: four answers, all of them
// things you do to it.

// The press answers as a turn. The card above keeps its place in the record,
// so you can see which note is being discussed — and the press itself is a
// true thing about the afternoon.
func TestSomethingElseAnswersAsATurn(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(7, "the boiler", squirrel.ItemOpen)}}
	m := routed(t, f)

	m.call(t, "POST", "/pile/more", strings.NewReader("id=7"))

	require.Len(t, f.appended, 2)
	require.Equal(t, squirrel.SpeakerYou, f.appended[0].Who)
	require.Equal(t, "something else?", f.appended[0].Words)
	require.Equal(t, squirrel.SpeakerBuddy, f.appended[1].Who)

	shown := string(f.appended[1].Shown)
	for _, question := range []string{"make it a chore", "say it another way", "i can't act on this"} {
		require.Contains(t, shown, question)
	}
}

// Room appears for a fourth. On the card it could only be offered when a free
// check guessed it was worth offering; behind a press it can always be there,
// because the press is the person saying they want more.
func TestBreakItUpIsOfferedBehindThePress(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(7, "milk and the vet and the bins", squirrel.ItemOpen)}}
	m := newTestMux()
	opts := signedInOptions()
	opts.Splittable = func(string) bool { return true }
	opts.Split = func(_ context.Context, _ int64, _ string) ([]string, bool) { return nil, false }
	require.NoError(t, Mount(m, f, opts))

	m.call(t, "POST", "/pile/more", strings.NewReader("id=7"))

	require.Contains(t, string(f.appended[1].Shown), "break it up")
}

// Somebody else's note is not yours to ask about.
func TestAskingAboutANoteThatIsNotThereIsNotFound(t *testing.T) {
	f := &fakeStore{}
	m := routed(t, f)

	require.Equal(t, 404, m.call(t, "POST", "/pile/more", strings.NewReader("id=99")).Code)
	require.Empty(t, f.appended, "it said something about a note that is not there")
}
