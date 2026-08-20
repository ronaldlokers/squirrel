//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// A sequence, handed back one step at a time. What is being tested as much as
// the queries is what is missing: there is no way from this store to the list.

func TestStepsAreHandedBackOneAtATime(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)
	now := time.Now()

	require.NoError(t, store.SaveSteps(ctx, p, nil, "the tax thing",
		[]string{"open the letter", "find the reference", "ring the number"}))

	first, found, err := store.NextStep(ctx, p)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "open the letter", first.Body)
	require.Equal(t, "the tax thing", first.Label)
	require.False(t, first.Last)

	require.NoError(t, store.StepDone(ctx, p, first.ID, now))

	second, found, err := store.NextStep(ctx, p)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "find the reference", second.Body)
}

// Not a position out of a total. Only whether anything comes after — which is
// the difference between finishing and waiting for something that never comes.
func TestTheLastStepSaysSo(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)
	now := time.Now()

	require.NoError(t, store.SaveSteps(ctx, p, nil, "the tax thing",
		[]string{"open the letter", "ring the number"}))

	first, _, err := store.NextStep(ctx, p)
	require.NoError(t, err)
	require.NoError(t, store.StepDone(ctx, p, first.ID, now))

	last, found, err := store.NextStep(ctx, p)
	require.NoError(t, err)
	require.True(t, found)
	require.True(t, last.Last)

	require.NoError(t, store.StepDone(ctx, p, last.ID, now))
	_, found, err = store.NextStep(ctx, p)
	require.NoError(t, err)
	require.False(t, found, "a finished sequence is still offering steps")
}

// One sequence at a time, replaced wholesale. Two half-finished breakdowns is
// a list of things you did not finish wearing a different hat.
func TestASecondBreakdownReplacesTheFirst(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)

	require.NoError(t, store.SaveSteps(ctx, p, nil, "the tax thing",
		[]string{"open the letter", "ring the number"}))
	require.NoError(t, store.SaveSteps(ctx, p, nil, "the vet",
		[]string{"find the number", "ring it"}))

	st, found, err := store.NextStep(ctx, p)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "the vet", st.Label)
	require.Equal(t, "find the number", st.Body)
}

// One press, no consequence, nothing asked back — the same shape "not now"
// already has.
func TestClearingThrowsTheWholeSequenceAway(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)

	require.NoError(t, store.SaveSteps(ctx, p, nil, "the tax thing",
		[]string{"open the letter", "ring the number"}))
	require.NoError(t, store.ClearSteps(ctx, p))

	_, found, err := store.NextStep(ctx, p)
	require.NoError(t, err)
	require.False(t, found)
}

func TestStepsAreNotSharedBetweenPeople(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	mine := owner(t, store)

	require.NoError(t, store.SaveSteps(ctx, mine, nil, "the tax thing",
		[]string{"open the letter", "ring the number"}))

	_, found, err := store.NextStep(ctx, mine+1000)
	require.NoError(t, err)
	require.False(t, found)
}

// Marking the same step twice is not an error — it is the state you asked for,
// which is the rule every other transition in this product already keeps.
func TestFinishingAStepTwiceChangesNothing(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)
	now := time.Now()

	require.NoError(t, store.SaveSteps(ctx, p, nil, "the tax thing",
		[]string{"open the letter", "ring the number"}))

	first, _, err := store.NextStep(ctx, p)
	require.NoError(t, err)
	require.NoError(t, store.StepDone(ctx, p, first.ID, now))
	require.NoError(t, store.StepDone(ctx, p, first.ID, now))

	st, found, err := store.NextStep(ctx, p)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "ring the number", st.Body)
}

// Steps are not notes. Nothing a model wrote belongs in the pile, and the two
// must never end up in the same list.
func TestStepsNeverReachThePile(t *testing.T) {
	ctx := context.Background()
	store := withStore(t)
	p := owner(t, store)

	require.NoError(t, store.SaveSteps(ctx, p, nil, "the tax thing",
		[]string{"open the letter", "ring the number"}))

	items, _, err := store.OpenItems(ctx, p, 20)
	require.NoError(t, err)
	for _, it := range items {
		require.NotEqual(t, "open the letter", it.RawText)
	}

	found, _, err := store.SearchItems(ctx, p, "open the letter", 20)
	require.NoError(t, err)
	require.Empty(t, found)
}
