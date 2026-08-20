package coach_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
)

// Four thoughts captured as one note, offered as four. Split returns strings
// and has no way to write anything, which is decision 8 as a property rather
// than an intention.

const dump = "ring the vet, put the bins out and do the tax thing"

func piecesTurn(pieces ...string) map[string]any {
	items := make([]any, 0, len(pieces))
	for _, p := range pieces {
		items = append(items, p)
	}
	return turnOf(call("a", "pieces", map[string]any{"pieces": items}))
}

func TestSplitFindsTheSeams(t *testing.T) {
	api := newToolAPI(t, piecesTurn("ring the vet", "put the bins out", "do the tax thing"))
	log := &fakeLog{}

	pieces, err := deciderFor(api, &fakeFacts{}, log).Split(context.Background(), 1, dump)
	require.NoError(t, err)
	require.Equal(t, []string{"ring the vet", "put the bins out", "do the tax thing"}, pieces)

	// Luna, not Terra. Finding the seams in a sentence someone already wrote
	// is not a judgement call — the judgement was theirs, when they typed it.
	require.Equal(t, "gpt-5.6-luna", api.sent[0]["model"])
	require.Len(t, log.recorded, 1)
	require.Equal(t, "split", log.recorded[0].Kind)
}

// The free rule first, so an ordinary note never costs anything. The same rule
// that recognises the overwhelm turn, because a brain dump and an overwhelm
// turn are the same shape.
func TestSplitDoesNotAskAboutAnOrdinaryNote(t *testing.T) {
	api := newToolAPI(t, piecesTurn("one", "two"))
	_, err := deciderFor(api, &fakeFacts{}, &fakeLog{}).
		Split(context.Background(), 1, "the boiler makes a noise on tuesdays")
	require.ErrorIs(t, err, coach.ErrUnavailable)
	require.Empty(t, api.sent, "an ordinary note reached the model")
}

// The check that matters. "Use their own words" is exactly the instruction a
// model is most willing to improve on, and something a model wrote must never
// be proposed back as a thing you said.
func TestSplitRefusesPiecesItInvented(t *testing.T) {
	api := newToolAPI(t, piecesTurn(
		"ring the vet", "put the bins out", "schedule a meeting with the accountant"))

	_, err := deciderFor(api, &fakeFacts{}, &fakeLog{}).Split(context.Background(), 1, dump)
	require.ErrorIs(t, err, coach.ErrUnavailable)
}

// Splitting drops the joins — "the bins and the vet" becomes two pieces and
// neither is a substring of the original once the "and" is gone. Dropping is
// allowed; inventing is not.
func TestSplitAllowsWordsToBeDroppedButNotAdded(t *testing.T) {
	api := newToolAPI(t, piecesTurn("ring the vet", "put the bins out"))
	pieces, err := deciderFor(api, &fakeFacts{}, &fakeLog{}).
		Split(context.Background(), 1, "ring the vet and put the bins out and do the tax")
	require.NoError(t, err)
	require.Len(t, pieces, 2)
}

// Someone typing in a hurry does not capitalise, and a model tidying a full
// stop onto the end is not the failure being guarded against.
func TestSplitForgivesCaseAndPunctuation(t *testing.T) {
	api := newToolAPI(t, piecesTurn("Ring the vet.", "Put the bins out.", "Do the tax thing."))
	pieces, err := deciderFor(api, &fakeFacts{}, &fakeLog{}).Split(context.Background(), 1, dump)
	require.NoError(t, err)
	require.Len(t, pieces, 3)
}

// One piece is not a split, it is the note with the punctuation moved.
func TestSplitRefusesASinglePiece(t *testing.T) {
	api := newToolAPI(t, piecesTurn("ring the vet"))
	_, err := deciderFor(api, &fakeFacts{}, &fakeLog{}).Split(context.Background(), 1, dump)
	require.ErrorIs(t, err, coach.ErrUnavailable)
}

// It decided the note is really one thought, which it was told it may do.
func TestSplitAcceptsThatItIsReallyOneThing(t *testing.T) {
	api := newToolAPI(t, piecesTurn())
	_, err := deciderFor(api, &fakeFacts{}, &fakeLog{}).Split(context.Background(), 1, dump)
	require.ErrorIs(t, err, coach.ErrUnavailable)
}

// A piece longer than the note it came out of is not a piece.
func TestSplitRefusesAPieceThatGrew(t *testing.T) {
	api := newToolAPI(t, piecesTurn("ring the vet", strings.Repeat("the tax thing ", 20)))
	_, err := deciderFor(api, &fakeFacts{}, &fakeLog{}).Split(context.Background(), 1, dump)
	require.ErrorIs(t, err, coach.ErrUnavailable)
}

// Four is the ceiling: a note that is really six things is a note where the
// splitting is the smaller problem.
func TestSplitKeepsAtMostFour(t *testing.T) {
	long := "the vet, the bins, the tax, the mot, the dentist and the shopping"
	api := newToolAPI(t, piecesTurn(
		"the vet", "the bins", "the tax", "the mot", "the dentist", "the shopping"))

	pieces, err := deciderFor(api, &fakeFacts{}, &fakeLog{}).Split(context.Background(), 1, long)
	require.NoError(t, err)
	require.Len(t, pieces, 4)
}

func TestSplitMakesNoCallWhenOverBudget(t *testing.T) {
	api := newToolAPI(t, piecesTurn("ring the vet", "put the bins out"))
	_, err := deciderFor(api, &fakeFacts{}, &fakeLog{spent: 10_000_000}).
		Split(context.Background(), 1, dump)
	require.ErrorIs(t, err, coach.ErrUnavailable)
	require.Empty(t, api.sent)
}

func TestNoCoachSplitsNothing(t *testing.T) {
	_, err := coach.NoCoach{}.Split(context.Background(), 1, dump)
	require.ErrorIs(t, err, coach.ErrUnavailable)
}
