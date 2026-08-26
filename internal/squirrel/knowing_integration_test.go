//go:build integration

package squirrel_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestWhatIsKnownIsReplacedRatherThanAccumulated(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	at := time.Now()

	require.NoError(t, store.ReplaceKnowing(ctx, p,
		[]string{"Phone calls get done.", "Forms get put off."}, at))
	require.NoError(t, store.ReplaceKnowing(ctx, p,
		[]string{"Things started before lunch get finished."}, at.Add(time.Hour)))

	known, err := store.Knowing(ctx, p)
	require.NoError(t, err)
	require.Equal(t, []string{"Things started before lunch get finished."}, known,
		"last week's survived this week's")
}

// An empty pass clears what was there. Keeping last week's because this week's
// was empty would make the record older than it claims to be.
func TestAnEmptyPassClearsWhatWasThere(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	require.NoError(t, store.ReplaceKnowing(ctx, p, []string{"Phone calls get done."}, time.Now()))
	require.NoError(t, store.ReplaceKnowing(ctx, p, nil, time.Now()))

	known, err := store.Knowing(ctx, p)
	require.NoError(t, err)
	require.Empty(t, known)
}

// The order a pass wrote them in is the order they are read in.
func TestWhatIsKnownKeepsItsOrder(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	said := []string{"first", "second", "third", "fourth"}

	require.NoError(t, store.ReplaceKnowing(ctx, p, said, time.Now()))

	known, err := store.Knowing(ctx, p)
	require.NoError(t, err)
	require.Equal(t, said, known)
}

// When the last pass ran comes from the rows themselves. A second place to
// store it is a second place for it to be wrong.
func TestWhenItLastLearnedComesFromTheRows(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	never, err := store.LearnedAt(ctx, p)
	require.NoError(t, err)
	require.True(t, never.Year() < 2000, "a person nothing is known about has a recent date")

	at := time.Now().Truncate(time.Second)
	require.NoError(t, store.ReplaceKnowing(ctx, p, []string{"Phone calls get done."}, at))

	got, err := store.LearnedAt(ctx, p)
	require.NoError(t, err)
	require.WithinDuration(t, at, got, time.Second)
}

// A pass that concluded nothing leaves no marker either, so the next tick
// tries again rather than waiting a week from a pass that wrote nothing.
func TestAnEmptyPassLeavesNoMarker(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	require.NoError(t, store.ReplaceKnowing(ctx, p, nil, time.Now()))

	at, err := store.LearnedAt(ctx, p)
	require.NoError(t, err)
	require.True(t, at.Year() < 2000)
}

func TestForgettingThrowsItAllAway(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	require.NoError(t, store.ReplaceKnowing(ctx, p,
		[]string{"Phone calls get done.", "Forms get put off."}, time.Now()))

	require.NoError(t, store.ForgetKnowing(ctx, p))

	known, err := store.Knowing(ctx, p)
	require.NoError(t, err)
	require.Empty(t, known)
}

// The bound is applied where the rows are written, because that is the only
// place every writer has to pass through.
func TestTheBoundIsAppliedAtTheDatabase(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	var many []string
	for i := range 20 {
		many = append(many, "an observation "+string(rune('a'+i)))
	}
	require.NoError(t, store.ReplaceKnowing(ctx, p, many, time.Now()))

	known, err := store.Knowing(ctx, p)
	require.NoError(t, err)
	require.Len(t, known, squirrel.KnowingMost)
}

// Anything countable never reaches the table. "You have done this four times"
// is a fact about a person, and rule 2 forbids one on any surface — including
// this one, which is mostly read by a model.
func TestNothingCountableIsKept(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	require.NoError(t, store.ReplaceKnowing(ctx, p, []string{
		"Put the bins off 4 times.",
		"Always finishes phone calls.",
		"Never opens forms.",
		"Puts the same thing off every time.",
		"Phone calls get done; forms get put off.",
	}, time.Now()))

	known, err := store.Knowing(ctx, p)
	require.NoError(t, err)
	require.Equal(t, []string{"Phone calls get done; forms get put off."}, known)
}

func TestAParagraphIsNotAnObservation(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	require.NoError(t, store.ReplaceKnowing(ctx, p, []string{
		strings.Repeat("a long thought about how somebody works ", 8),
		"Phone calls get done.",
	}, time.Now()))

	known, err := store.Knowing(ctx, p)
	require.NoError(t, err)
	require.Equal(t, []string{"Phone calls get done."}, known)
}

// A numbered list is a model ignoring the shape it was asked for. The number
// goes and the sentence stays.
func TestANumberedObservationLosesItsNumber(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	require.NoError(t, store.ReplaceKnowing(ctx, p,
		[]string{"- Phone calls get done."}, time.Now()))

	known, err := store.Knowing(ctx, p)
	require.NoError(t, err)
	require.Equal(t, []string{"Phone calls get done."}, known)
}

func TestTheSameObservationTwiceIsKeptOnce(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	require.NoError(t, store.ReplaceKnowing(ctx, p, []string{
		"Phone calls get done.", "phone calls get done.", "Forms get put off.",
	}, time.Now()))

	known, err := store.Knowing(ctx, p)
	require.NoError(t, err)
	require.Len(t, known, 2)
}
