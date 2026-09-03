package coach

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Noticing one thing about one thing.

var aBoard = []Thing{
	{Kind: "note", RefID: 4, Words: "the MOT is due"},
	{Kind: "chore", RefID: 7, Words: "book the garage"},
}

// The four refusals the shape of this depends on. Without the last one a model
// asked to read a board will always find something, and a note that restates
// the strip it hangs under is worse than an empty margin.
func TestThePreambleRefusesWhatANoteMustNeverBe(t *testing.T) {
	for _, refused := range []string{
		"Never count anything", "Never say anything about the person",
		"never ask them a question", "Nothing is better than something",
	} {
		require.Contains(t, noticingPreamble, refused,
			"the preamble stopped refusing %q", refused)
	}
}

// A note is kept against the thing it names, and it takes that thing's kind
// rather than one the model chose: the kind decides which strip it is drawn
// under.
func TestANoteTakesTheKindOfTheThingItNames(t *testing.T) {
	calls := noticedCall("notes", `{"notes":[{"ref":7,"words":"The date for this is on the other one."}]}`)

	got := noticedIn(calls, aBoard)

	require.Len(t, got, 1)
	require.Equal(t, "chore", got[0].Kind)
	require.Equal(t, int64(7), got[0].RefID)
	require.Equal(t, "The date for this is on the other one.", got[0].Words)
}

// An id that was not on the board belongs to nothing. Kept, it would hang the
// line on whichever row happens to carry that number.
func TestANoteAboutSomethingNotOnTheBoardIsDropped(t *testing.T) {
	calls := noticedCall("notes", `{"notes":[{"ref":999,"words":"about nothing here"},{"ref":4,"words":"about this one"}]}`)

	got := noticedIn(calls, aBoard)

	require.Len(t, got, 1, "the invented id survived")
	require.Equal(t, int64(4), got[0].RefID)
}

func TestAnEmptyNoteIsNoNote(t *testing.T) {
	calls := noticedCall("notes", `{"notes":[{"ref":4,"words":"   "}]}`)

	require.Empty(t, noticedIn(calls, aBoard))
}

// Two. Asked for more, a model writes more, and the extra ones restate.
func TestNoMoreThanTwoAreKept(t *testing.T) {
	calls := noticedCall("notes", `{"notes":[{"ref":4,"words":"one"},{"ref":7,"words":"two"},{"ref":4,"words":"three"}]}`)

	require.Len(t, noticedIn(calls, aBoard), mostNoticed)
}

func TestAnotherToolIsNotANote(t *testing.T) {
	calls := noticedCall("steps", `{"steps":["open the letter"]}`)

	require.Nil(t, noticedIn(calls, aBoard))
}

func TestUnreadableArgumentsAreNoNotes(t *testing.T) {
	require.Nil(t, noticedIn(noticedCall("notes", `{`), aBoard))
}

// The refused lines go in as themselves. A rule derived from them would be a
// guess about why they were refused.
func TestWhatWasRefusedGoesInAsItself(t *testing.T) {
	said := refusedBefore([]string{"You have written this down three times."})

	require.Contains(t, said, "You have written this down three times.")
	require.Contains(t, said, "Do not write anything like them again")
}

func TestNothingRefusedAddsNothing(t *testing.T) {
	require.Empty(t, refusedBefore(nil))
	require.Empty(t, refusedBefore([]string{}))
}
