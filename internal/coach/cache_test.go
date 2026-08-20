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
