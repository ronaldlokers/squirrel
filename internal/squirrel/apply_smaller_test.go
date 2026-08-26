//go:build integration

package squirrel_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// `!stuck too big`, with something behind it. The fixed line is what shows
// when nothing is, which is what it did on its own before this existed.

type breakRecord struct {
	task    string
	blocker string
	steps   []string
	asked   int
}

func breaking(t *testing.T, store *squirrel.Store, personID int64, text string, b *breakRecord) string {
	t.Helper()
	chat, got := chatRecorder(strconv.FormatInt(replyIDs.Add(1), 10))
	a := squirrel.NewApplier(store, nil, chat, nil)
	if b != nil {
		a.SetBreaker(func(_ context.Context, _ int64, task, blocker string) ([]string, bool) {
			b.asked++
			b.task, b.blocker = task, blocker
			return b.steps, len(b.steps) > 0
		})
	}
	require.NoError(t, a.Apply(context.Background(), itemOf(text), &personID))
	require.Len(t, *got, 1)
	return (*got)[0].message.Text
}

func TestTooBigHandsBackOneStepAndNeverTheList(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	taskOf(t, store, p, "the tax thing")
	b := &breakRecord{steps: []string{
		"open the letter", "find the reference", "ring the number",
	}}
	reply := breaking(t, store, p, "!stuck too big", b)

	require.Equal(t, "the tax thing", b.task)
	require.Equal(t, "too big", b.blocker)

	require.Contains(t, reply, "open the letter")
	// The whole safety argument, asserted rather than assumed: a model may
	// produce a list, and nothing here may show one.
	require.NotContains(t, reply, "find the reference")
	require.NotContains(t, reply, "ring the number")
}

// The floor. A model that is slow, absent or wrong costs nothing anyone sees.
func TestTooBigFallsBackToTheFixedLine(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	taskOf(t, store, p, "the tax thing")
	reply := breaking(t, store, p, "!stuck too big", &breakRecord{})

	require.Contains(t, reply, squirrel.UnstuckFor(squirrel.BlockerBig).Line)
}

func TestTooBigWithNoBreakerIsTheLadderExactlyAsItWas(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	taskOf(t, store, p, "the tax thing")
	reply := breaking(t, store, p, "!stuck too big", nil)

	require.Contains(t, reply, squirrel.UnstuckFor(squirrel.BlockerBig).Line)
}

// The other three already have answers that are not a sequence. Handing any of
// them a list of steps would be answering a question nobody asked.
func TestOnlyTooBigEverAsksForABreakdown(t *testing.T) {
	for _, said := range []string{"!stuck don't know how", "!stuck boring", "!stuck not today"} {
		store := withStore(t)
		p := owner(t, store)
		taskOf(t, store, p, "the tax thing")

		b := &breakRecord{steps: []string{"one", "two"}}
		breaking(t, store, p, said, b)
		require.Zero(t, b.asked, "%q asked for a breakdown", said)
	}
}

// Nothing to break down is not a reason to invent something.
func TestTooBigWithNothingToHandAsksForNoBreakdown(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	b := &breakRecord{steps: []string{"one", "two"}}
	breaking(t, store, p, "!stuck too big", b)
	require.Zero(t, b.asked)
}

func TestNextWalksTheSequenceAndThenStops(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	taskOf(t, store, p, "the tax thing")
	breaking(t, store, p, "!stuck too big",
		&breakRecord{steps: []string{"open the letter", "ring the number"}})

	second := breaking(t, store, p, "!next", nil)
	require.Contains(t, second, "ring the number")
	require.Contains(t, second, "last one")

	end := breaking(t, store, p, "!next", nil)
	require.Contains(t, end, "the tax thing")
	require.NotContains(t, end, "2")
	require.NotContains(t, end, "well done")
}

func TestNextWithNothingBrokenDownSaysSo(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	require.Contains(t, breaking(t, store, p, "!next", nil), "!stuck too big")
}

// A step is never a count of what is left. "Step 2 of 5" is the accruing
// number this product refuses, said about the one thing you cannot start.
func TestAStepNeverSaysHowManyAreLeft(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	taskOf(t, store, p, "the tax thing")
	reply := breaking(t, store, p, "!stuck too big",
		&breakRecord{steps: []string{"one thing", "two thing", "three thing"}})

	require.NotContains(t, reply, "of 3")
	require.NotContains(t, reply, "1/3")
	require.NotContains(t, reply, "step 1")
}
