package coach_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
)

var vet = coach.Decision{Kind: "task", RefID: 7, Text: "ring the vet", Because: "it is two minutes"}

func TestAnUnchangedDayServesTheSameAnswer(t *testing.T) {
	o := coach.NewOffers()
	o.Put(1, "task:7", vet, august)

	got, ok := o.Get(1, "task:7", august.Add(5*time.Minute))
	require.True(t, ok)
	require.Equal(t, vet, got)
}

// The basis is the picker's own answer, which already moves when anything the
// design listed as an invalidator happens — a check-in, a timer, a completion,
// a refusal, a moment entering its leave-by window.
func TestAMovedDayThrowsTheAnswerAway(t *testing.T) {
	o := coach.NewOffers()
	o.Put(1, "task:7", vet, august)

	_, ok := o.Get(1, "chore:3", august.Add(time.Minute))
	require.False(t, ok)
}

// A floor, so nothing is stale forever however unchanged the day looks.
func TestAnAnswerGoesStaleOnItsOwn(t *testing.T) {
	o := coach.NewOffers()
	o.Put(1, "task:7", vet, august)

	_, ok := o.Get(1, "task:7", august.Add(coach.StaleAfter))
	require.False(t, ok)

	_, ok = o.Get(1, "task:7", august.Add(coach.StaleAfter-time.Second))
	require.True(t, ok)
}

func TestAnAnswerIsNotSharedBetweenPeople(t *testing.T) {
	o := coach.NewOffers()
	o.Put(1, "task:7", vet, august)

	_, ok := o.Get(2, "task:7", august)
	require.False(t, ok)
}

// The nil receiver is the no-coach build, and it must not panic on a path that
// only exists because a coach might have been there.
func TestNilOffersAreSafe(t *testing.T) {
	var o *coach.Offers
	require.NotPanics(t, func() {
		o.Put(1, "task:7", vet, august)
		_, ok := o.Get(1, "task:7", august)
		require.False(t, ok)
	})
}

// Answering an offer throws the decision away, whatever the picker says next.
//
// The basis was the only invalidator, and it invalidates by the picker's
// answer moving. That holds when the model agreed with the picker, and fails
// when it did not: the card then carries the model's row, the answer is
// recorded against that row, and the picker — which was pointing at a
// different one — goes on saying exactly what it said before. Basis unchanged,
// entry alive, same card. See the comment in cache.go.
func TestAnsweringAnOfferThrowsTheAnswerAway(t *testing.T) {
	o := coach.NewOffers()
	o.Put(1, "chore:3", vet, august)

	o.Forget(1)

	_, ok := o.Get(1, "chore:3", august.Add(time.Minute))
	require.False(t, ok, "the picker still says chore:3, and that must no longer be enough")
}

func TestForgettingIsPerPerson(t *testing.T) {
	o := coach.NewOffers()
	o.Put(1, "task:7", vet, august)
	o.Put(2, "task:7", vet, august)

	o.Forget(1)

	_, ok := o.Get(2, "task:7", august.Add(time.Minute))
	require.True(t, ok)
}

func TestForgettingNothingIsSafe(t *testing.T) {
	var nilOffers *coach.Offers
	require.NotPanics(t, func() { nilOffers.Forget(1) })
	require.NotPanics(t, func() { coach.NewOffers().Forget(99) })
}
