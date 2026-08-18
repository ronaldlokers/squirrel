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

func TestNotesListsOnlyOpenNotes(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	chat, got := chatRecorder("m-1")

	kept := insertItem(t, store, p, "boiler serial is 44Q")
	require.NoError(t, store.SetItemState(ctx, kept, squirrel.ItemKept, time.Now()))
	insertItem(t, store, p, "buy milk")

	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(ctx, itemOf("!notes"), &p))

	require.Len(t, *got, 1)
	require.Contains(t, (*got)[0].message.Text, "1. buy milk")
	require.NotContains(t, (*got)[0].message.Text, "44Q", "kept has left the pile")
}

func TestNotesOnAnEmptyPileSaysSoWithoutAZero(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	chat, got := chatRecorder("m-1")

	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(ctx, itemOf("!notes"), &p))

	require.Len(t, *got, 1)
	require.NotContains(t, (*got)[0].message.Text, "0")
}

// An empty list must record no numbered surface at all.
//
// The harm is not that line 1 of an empty prompt resolves wrongly — it has no
// lines, so it resolves to nothing either way, and a test asserting that passes
// whether or not the prompt was recorded. The harm is that the empty prompt
// becomes the *newest* numbered surface and shadows the live one, so a chore
// whose button is still on screen stops answering to `done 1`. That is exactly
// the shape phase 4 spent a round removing when 'evening' was briefly numbered.
func TestNotesOnAnEmptyPileDoesNotShadowALiveSurface(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	c, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)
	promptID, err := store.RecordPrompt(ctx, p, "9", "nudge", time.Now(), nil, []squirrel.Chore{c})
	require.NoError(t, err)
	require.NoError(t, store.MarkPromptSent(ctx, promptID, "m-0", time.Now()))

	chat, _ := chatRecorder("m-1")
	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(ctx, itemOf("!notes"), &p))

	line, ok, err := store.LineAtPosition(ctx, p, 1)
	require.NoError(t, err)
	require.True(t, ok, "the nudge's own line must still be the one a typed 1 resolves against")
	require.NotNil(t, line.Chore)
	require.Equal(t, "vacuum", line.Chore.Name)
}

// The cap says there is more. It never says how much more.
func TestNotesCapsWithoutRevealingTheTotal(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	chat, got := chatRecorder("m-1")

	for i := range 25 {
		insertItem(t, store, p, "thought "+string(rune('a'+i)))
	}

	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(ctx, itemOf("!notes"), &p))

	text := (*got)[0].message.Text
	require.Contains(t, text, "more")
	require.NotContains(t, text, "25")
	require.NotContains(t, text, "15")
	require.Equal(t, 10, strings.Count(text, ". thought"), "ten lines, and no eleventh")
}

func TestFindSearchesAcrossStates(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	chat, got := chatRecorder("m-1")

	kept := insertItem(t, store, p, "boiler serial is 44Q")
	require.NoError(t, store.SetItemState(ctx, kept, squirrel.ItemKept, time.Now()))
	insertItem(t, store, p, "the boiler thing")
	insertItem(t, store, p, "buy milk")

	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(ctx, itemOf("!find boiler"), &p))

	text := (*got)[0].message.Text
	require.Contains(t, text, "44Q", "kept exists precisely to be found later")
	require.Contains(t, text, "the boiler thing")
	require.NotContains(t, text, "buy milk")
}

// An empty search reads as a mistake, not as a request for the whole pile —
// and answering it with everything would be the counting behaviour by another
// route.
func TestFindWithNoArgumentDoesNotListEverything(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	chat, got := chatRecorder("m-1")

	insertItem(t, store, p, "buy milk")

	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(ctx, itemOf("!find"), &p))

	require.Len(t, *got, 1)
	require.NotContains(t, (*got)[0].message.Text, "buy milk")

	_, ok, err := store.LineAtPosition(ctx, p, 1)
	require.NoError(t, err)
	require.False(t, ok, "a refusal must not record a numbered surface")
}

func TestFindWithNoMatchesSaysSo(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	chat, got := chatRecorder("m-1")

	insertItem(t, store, p, "buy milk")

	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(ctx, itemOf("!find boiler"), &p))

	require.Len(t, *got, 1)
	require.Contains(t, (*got)[0].message.Text, "boiler")

	_, ok, err := store.LineAtPosition(ctx, p, 1)
	require.NoError(t, err)
	require.False(t, ok, "an empty result must not become the newest numbered surface")
}

func TestAnUnknownCommandRepliesAndStoresNothing(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	chat, got := chatRecorder("m-1")

	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(ctx, itemOf("!fnid boiler"), &p))

	require.Len(t, *got, 1)
	require.Contains(t, (*got)[0].message.Text, "!fnid")
	require.Contains(t, (*got)[0].message.Text, "!find", "say what does exist")
}

func TestHelpNamesTheVocabulary(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	chat, got := chatRecorder("m-1")

	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(ctx, itemOf("!help"), &p))

	text := (*got)[0].message.Text
	for _, want := range []string{"!notes", "!find", "!chores", "!chore", "done", "keep", "drop"} {
		require.Contains(t, text, want)
	}
	require.Contains(t, strings.ToLower(text), "note",
		"the capture rule comes first: if you remember nothing else, typing a thought stores it")
}

// A note listed by !notes must be resolvable by number. This is the whole
// reason these lists are recorded as prompts.
func TestNotesRecordsAResolvableNumberedSurface(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	chat, _ := chatRecorder("m-1")

	insertItem(t, store, p, "buy milk")

	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(ctx, itemOf("!notes"), &p))

	line, ok, err := store.LineAtPosition(ctx, p, 1)
	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, line.Item)
	require.Equal(t, "buy milk", line.Item.RawText)
}

func TestChoresCommandMatchesQuery(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	_, err := store.UpsertChore(ctx, p, "vacuum", twoWeeks, oneWeek)
	require.NoError(t, err)

	chatA, gotA := chatRecorder("m-1")
	require.NoError(t, squirrel.NewApplier(store, nil, chatA, nil).
		Apply(ctx, itemOf("?"), &p))

	chatB, gotB := chatRecorder("m-2")
	require.NoError(t, squirrel.NewApplier(store, nil, chatB, nil).
		Apply(ctx, itemOf("!chores"), &p))

	require.Len(t, *gotA, 1)
	require.Len(t, *gotB, 1)
	require.Equal(t, (*gotA)[0].message.Text, (*gotB)[0].message.Text,
		"!chores is an alias, not a second implementation")
}

// A command must never be filed as a note, or the pile fills with the things
// you typed to look at the pile.
func TestACommandIsNotStoredAsANote(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	chat, _ := chatRecorder("m-1")

	insertItem(t, store, p, "buy milk")
	require.NoError(t, squirrel.NewApplier(store, nil, chat, nil).
		Apply(ctx, itemOf("!notes"), &p))

	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "buy milk", items[0].RawText)
}
