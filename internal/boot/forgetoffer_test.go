package boot

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
)

// countingCoach answers Decide with a fixed decision and counts the asks, so a
// test can show whether the cache served an answer or the model was paid for
// one. Everything else is the shipping no-coach behaviour, because nothing here
// touches it — and it must not be a NoCoach, which decider() answers with nil.
type countingCoach struct {
	coach.NoCoach
	d     coach.Decision
	asked int
}

func (c *countingCoach) Decide(context.Context, int64) (coach.Decision, error) {
	c.asked++
	return c.d, nil
}

// The decider and the forgetter are the same cache, and answering drops it.
//
// This is the bug in one test. The model answers with a *different row* than
// the picker chose — a task, where the picker said a chore — so the card
// carries the task and the refusal is written against the task, while the
// picker goes on saying the chore. The basis is what the cache watches, and it
// has not moved: without the forget, the same decision is handed back and the
// screen redraws the identical card.
func TestForgettingAnOfferMakesTheNextAskReal(t *testing.T) {
	c := &countingCoach{d: coach.Decision{
		Kind: "task", RefID: 7, Text: "ring the vet", Because: "you decided this",
	}}
	offers := coach.NewOffers()

	// One call, so the test cannot accidentally prove something about two
	// caches that a caller could get wrong.
	decide, forget := deciding(c, offers)

	// The picker says chore:3. The model replaces it with task:7.
	kind, refID, _, _, ok := decide(context.Background(), 1, "chore", 3, true)
	require.True(t, ok)
	require.Equal(t, "task", kind)
	require.Equal(t, int64(7), refID)
	require.Equal(t, 1, c.asked)

	// The picker still says chore:3, because a refusal written against task:7
	// is not in the set it reads. Held, that is the same card again.
	_, _, _, _, ok = decide(context.Background(), 1, "chore", 3, true)
	require.True(t, ok)
	require.Equal(t, 1, c.asked, "the cache answered, which is what it is for")

	// Answering the offer says so.
	forget(1)

	_, _, _, _, ok = decide(context.Background(), 1, "chore", 3, true)
	require.True(t, ok)
	require.Equal(t, 2, c.asked,
		"with the decision dropped, the model is asked again — and its own read tools "+
			"filter today's refusals, so it cannot hand back what was just turned down")
}

// No coach, nothing to decide and nothing to forget, and the screen's nil
// checks are what take it back to the behaviour that shipped before either
// existed. Both nil or neither: a decider with no way to drop what it cached is
// the bug above, wired in on purpose.
func TestNoCoachDecidesNothingAndHasNothingToForget(t *testing.T) {
	decide, forget := deciding(coach.NoCoach{}, coach.NewOffers())
	require.Nil(t, decide)
	require.Nil(t, forget)
}
