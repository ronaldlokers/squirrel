//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// deliver marks a prompt sent. Every resolver pins itself to the most recent
// *delivered* numbered prompt, because a prompt whose send failed must never
// become current for a typed position while the room's real buttons still point
// at the last list that actually went out.
func deliver(t *testing.T, store *squirrel.Store, promptID int64) {
	t.Helper()
	require.NoError(t, store.MarkPromptSent(context.Background(), promptID, "m-1", time.Now()))
}

func TestAPromptLineCanBeANote(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	id := insertItem(t, store, p, "the boiler thing")

	promptID, err := store.RecordPromptLines(ctx, p, "c1", "find", time.Now(), nil,
		[]squirrel.LineRef{{ItemID: &id}})
	require.NoError(t, err)
	deliver(t, store, promptID)

	line, ok, err := store.LineAtPosition(ctx, p, 1)
	require.NoError(t, err)
	require.True(t, ok)
	require.Nil(t, line.Chore)
	require.NotNil(t, line.Item)
	require.Equal(t, "the boiler thing", line.Item.RawText)
	require.Equal(t, id, line.Item.ID)
}

// The whole point of this task is that it changes a table three phases depend
// on without changing what they see.
func TestAPromptLineStillResolvesAChore(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	promptID, err := store.RecordPrompt(ctx, p, "c1", "query", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	deliver(t, store, promptID)

	line, ok, err := store.LineAtPosition(ctx, p, 1)
	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, line.Chore)
	require.Nil(t, line.Item)
	require.Equal(t, "vacuum", line.Chore.Name)
	require.Equal(t, 14, line.Chore.EveryDays, "the interval must survive the generalisation")
}

// RecordPrompt maps []Chore to []LineRef in a range loop, taking the address of
// the loop variable's field. Go 1.22 gave each iteration its own copy; on an
// older directive every line would alias the last chore, which is silent and
// looks like a numbering bug rather than a compiler-version bug.
func TestRecordPromptDoesNotAliasTheLastChore(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	first, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	second, err := store.UpsertChore(ctx, p, "bins", twoWeeks, oneWeek)
	require.NoError(t, err)

	promptID, err := store.RecordPrompt(ctx, p, "c1", "query", time.Now(), nil,
		[]squirrel.Chore{first, second})
	require.NoError(t, err)
	deliver(t, store, promptID)

	one, ok, err := store.LineAtPosition(ctx, p, 1)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "vacuum", one.Chore.Name)

	two, ok, err := store.LineAtPosition(ctx, p, 2)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "bins", two.Chore.Name)
}

func TestAPromptCanMixChoreAndNoteLines(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	id := insertItem(t, store, p, "the boiler thing")

	promptID, err := store.RecordPromptLines(ctx, p, "c1", "find", time.Now(), nil,
		[]squirrel.LineRef{{ChoreID: &c.ID}, {ItemID: &id}})
	require.NoError(t, err)
	deliver(t, store, promptID)

	one, _, err := store.LineAtPosition(ctx, p, 1)
	require.NoError(t, err)
	require.NotNil(t, one.Chore)

	two, _, err := store.LineAtPosition(ctx, p, 2)
	require.NoError(t, err)
	require.NotNil(t, two.Item)
}

func TestALineCannotTargetBoth(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	id := insertItem(t, store, p, "x")
	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	promptID, err := store.RecordPromptLines(ctx, p, "c1", "find", time.Now(), nil,
		[]squirrel.LineRef{{ItemID: &id}})
	require.NoError(t, err)

	_, err = store.Pool().Exec(ctx,
		`update prompt_lines set chore_id = $2 where prompt_id = $1`, promptID, c.ID)
	require.Error(t, err, "exactly one target, enforced by the database rather than by remembering")
}

func TestALineCannotTargetNeither(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	_, err := store.RecordPromptLines(ctx, p, "c1", "find", time.Now(), nil,
		[]squirrel.LineRef{{}})
	require.Error(t, err, "a line pointing at nothing would resolve to a blank reply")
}

// An undelivered prompt must not become the surface a typed number resolves
// against — the room is still showing the previous list.
func TestLineAtPositionIgnoresAnUndeliveredPrompt(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	delivered := insertItem(t, store, p, "the delivered one")
	promptID, err := store.RecordPromptLines(ctx, p, "c1", "notes", time.Now(), nil,
		[]squirrel.LineRef{{ItemID: &delivered}})
	require.NoError(t, err)
	deliver(t, store, promptID)

	phantom := insertItem(t, store, p, "the one whose send failed")
	_, err = store.RecordPromptLines(ctx, p, "c1", "find", time.Now().Add(time.Minute), nil,
		[]squirrel.LineRef{{ItemID: &phantom}})
	require.NoError(t, err)

	line, ok, err := store.LineAtPosition(ctx, p, 1)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "the delivered one", line.Item.RawText)
}

func TestLineAtPositionIsScopedToThePerson(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	other, err := store.SeedOwner(ctx, "someone-else", nil)
	require.NoError(t, err)
	theirs := insertItem(t, store, other, "their note")
	promptID, err := store.RecordPromptLines(ctx, other, "c2", "notes", time.Now(), nil,
		[]squirrel.LineRef{{ItemID: &theirs}})
	require.NoError(t, err)
	deliver(t, store, promptID)

	_, ok, err := store.LineAtPosition(ctx, p, 1)
	require.NoError(t, err)
	require.False(t, ok, "one person's number must never resolve to another person's note")
}

func TestLineAtPositionBeyondTheLastLine(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	id := insertItem(t, store, p, "the only one")
	promptID, err := store.RecordPromptLines(ctx, p, "c1", "notes", time.Now(), nil,
		[]squirrel.LineRef{{ItemID: &id}})
	require.NoError(t, err)
	deliver(t, store, promptID)

	_, ok, err := store.LineAtPosition(ctx, p, 4)
	require.NoError(t, err)
	require.False(t, ok)
}
