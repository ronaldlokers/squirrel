//go:build integration

package squirrel_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// insertItem stores a note and returns its id. InsertItem answers "was this a
// fresh row" rather than "which row", because the drain needs the first
// question and nothing until now needed the second. Reading the id back by its
// external id keeps this helper on the public API.
func insertItem(t *testing.T, store *squirrel.Store, personID int64, text string) int64 {
	t.Helper()
	ctx := context.Background()

	ext := fmt.Sprintf("ext-%s-%d", text, time.Now().UnixNano())
	_, err := store.InsertItem(ctx, squirrel.Item{
		Transport:      "test",
		ExternalID:     &ext,
		ConversationID: squirrel.Ptr("c1"),
		SenderID:       squirrel.Ptr("s1"),
		PersonID:       &personID,
		RawText:        text,
		Payload:        []byte(`{}`),
		ReceivedAt:     time.Now(),
	})
	require.NoError(t, err)

	var id int64
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select id from items where external_id = $1`, ext).Scan(&id))
	return id
}

// A tap is a state assertion, not a delta. A redelivered webhook is
// byte-identical to a second tap, so writing the state a note already holds
// must be a no-op rather than an error — phase 3 spent two review rounds
// learning this about chore completions.
func TestSetItemStateIsIdempotent(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	id := insertItem(t, store, p, "buy milk")

	at := time.Now()
	require.NoError(t, store.SetItemState(ctx, id, squirrel.ItemDone, at))
	require.NoError(t, store.SetItemState(ctx, id, squirrel.ItemDone, at),
		"a repeated transition is an assertion of the same state, not a failure")

	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Empty(t, items)
}

// Undo is a transition, not a special case.
func TestSetItemStateReverses(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	id := insertItem(t, store, p, "buy milk")

	require.NoError(t, store.SetItemState(ctx, id, squirrel.ItemDropped, time.Now()))
	require.NoError(t, store.SetItemState(ctx, id, squirrel.ItemOpen, time.Now()))

	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, items, 1, "every transition reverses, including back to open")
}

func TestSetItemStateRecordsWhen(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	id := insertItem(t, store, p, "buy milk")

	var before *time.Time
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select state_at from items where id = $1`, id).Scan(&before))
	require.Nil(t, before, "state_at is null until the first transition")

	at := time.Now()
	require.NoError(t, store.SetItemState(ctx, id, squirrel.ItemKept, at))

	var after *time.Time
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select state_at from items where id = $1`, id).Scan(&after))
	require.NotNil(t, after)
}

func TestOpenItemsIsNewestFirst(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	insertItem(t, store, p, "older")
	insertItem(t, store, p, "newer")

	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "newer", items[0].RawText,
		"newest first: oldest-first is a backlog you are behind on")
}

// The cap reports that there is more. It never reports how much more, because
// a total beside an implied zero is the accumulating counter this project
// bans, and the caller cannot render a number it was never given.
func TestOpenItemsReportsMoreWithoutACount(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	for i := range 5 {
		insertItem(t, store, p, fmt.Sprintf("thought %d", i))
	}

	items, more, err := store.OpenItems(ctx, p, 3)
	require.NoError(t, err)
	require.Len(t, items, 3, "the extra row fetched to answer `more` must not be returned")
	require.True(t, more)
}

func TestOpenItemsSaysThereIsNoMoreWhenThereIsNot(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	insertItem(t, store, p, "only one")

	items, more, err := store.OpenItems(ctx, p, 3)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.False(t, more)
}

func TestOpenItemsExcludesEveryOtherState(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	for _, s := range []squirrel.ItemState{squirrel.ItemDone, squirrel.ItemDropped, squirrel.ItemKept} {
		id := insertItem(t, store, p, "gone: "+string(s))
		require.NoError(t, store.SetItemState(ctx, id, s, time.Now()))
	}
	insertItem(t, store, p, "still here")

	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "still here", items[0].RawText)
}

// `kept` exists precisely so a reference note leaves triage and stays
// findable. Filtering search by state would defeat the state.
func TestSearchItemsSpansEveryState(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	kept := insertItem(t, store, p, "boiler serial is 44Q")
	require.NoError(t, store.SetItemState(ctx, kept, squirrel.ItemKept, time.Now()))
	insertItem(t, store, p, "the boiler thing")

	items, _, err := store.SearchItems(ctx, p, "boiler", 10)
	require.NoError(t, err)
	require.Len(t, items, 2)
}

func TestSearchItemsIsCaseInsensitive(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	insertItem(t, store, p, "The Boiler Thing")

	items, _, err := store.SearchItems(ctx, p, "boiler", 10)
	require.NoError(t, err)
	require.Len(t, items, 1)
}

// A search for "%" is a search for that character. Interpolating the term into
// a LIKE pattern would make it a wildcard and return the entire pile, which
// looks like a working search right up until you notice every note is in it.
func TestSearchItemsTreatsWildcardsAsText(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	insertItem(t, store, p, "the battery was at 50% again")
	insertItem(t, store, p, "unrelated thought")
	insertItem(t, store, p, "another unrelated thought")

	items, _, err := store.SearchItems(ctx, p, "%", 10)
	require.NoError(t, err)
	require.Len(t, items, 1, "a literal %% must not become a wildcard matching everything")
	require.Contains(t, items[0].RawText, "50%")

	under, _, err := store.SearchItems(ctx, p, "_", 10)
	require.NoError(t, err)
	require.Empty(t, under, "_ is LIKE's single-character wildcard and must be just as literal")
}

func TestSearchItemsIsScopedToThePerson(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	other, err := store.SeedOwner(ctx, "someone-else", nil)
	require.NoError(t, err)
	insertItem(t, store, other, "their boiler note")
	insertItem(t, store, p, "my boiler note")

	items, _, err := store.SearchItems(ctx, p, "boiler", 10)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "my boiler note", items[0].RawText)
}

// The drain stores every inbound message as an item — commands included. The
// pile is notes, so the things you typed to look at the pile must not be in it.
//
// This is the same test CapturesSince applies for the evening message, and the
// two have to agree: a row one shows and the other hides is a disagreement
// about what a thought is.
func TestOpenItemsExcludesCommands(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	for _, raw := range []string{"!notes", "!find boiler", "?", "done 2", "done", "nvm", "every 2 weeks: vacuum"} {
		insertItem(t, store, p, raw)
	}
	insertItem(t, store, p, "an actual thought")

	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "an actual thought", items[0].RawText)
}

func TestSearchItemsExcludesCommands(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	insertItem(t, store, p, "!find boiler")
	insertItem(t, store, p, "the boiler thing")

	items, _, err := store.SearchItems(ctx, p, "boiler", 10)
	require.NoError(t, err)
	require.Len(t, items, 1, "searching must not return your own searches")
	require.Equal(t, "the boiler thing", items[0].RawText)
}

// Text that merely looks like a tap, typed by a person, is a thought — the
// payload is the only thing that tells them apart. Phase 3 settled this and
// the pile has to settle it the same way.
func TestOpenItemsKeepsTypedActionText(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	insertItem(t, store, p, "!action 451 done:1 true")

	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, items, 1, "no action payload means a person typed it, and that is a thought")
}

// The cap counts notes, not rows. A run of commands between notes must not eat
// into the ten lines the pile is allowed to print.
func TestOpenItemsCapCountsNotesNotRows(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	for i := range 4 {
		insertItem(t, store, p, fmt.Sprintf("thought %d", i))
		insertItem(t, store, p, "!notes")
		insertItem(t, store, p, "?")
	}

	items, more, err := store.OpenItems(ctx, p, 3)
	require.NoError(t, err)
	require.Len(t, items, 3)
	require.True(t, more)
	for _, it := range items {
		require.Contains(t, it.RawText, "thought")
	}
}

// A search result has to say what a note became, or the screen cannot colour
// it. The capture path still knows nothing about state: the column default
// fills a fresh row.
func TestSearchItemsCarriesState(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	open := insertItem(t, store, p, "the boiler makes a noise")
	done := insertItem(t, store, p, "boiler service is booked")
	require.NoError(t, store.SetItemState(ctx, done, squirrel.ItemDone, time.Now()))

	items, more, err := store.SearchItems(ctx, p, "boiler", 10)
	require.NoError(t, err)
	require.False(t, more)
	require.Len(t, items, 2)

	states := map[int64]squirrel.ItemState{}
	for _, it := range items {
		states[it.ID] = it.State
	}
	require.Equal(t, squirrel.ItemOpen, states[open])
	require.Equal(t, squirrel.ItemDone, states[done])
}

// One promotion path, called by the chat command and by the screen. The chore
// carries the note's own text and the note becomes done — there is no chore
// state.
func TestPromoteItemCreatesChoreAndClosesNote(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	itemID := insertItem(t, store, p, "bins out")

	chore, ok, err := store.PromoteItem(ctx, p, itemID, 14*24*time.Hour)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "bins out", chore.Name)

	items, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Empty(t, items, "a promoted note leaves the pile")

	found, _, err := store.SearchItems(ctx, p, "bins", 10)
	require.NoError(t, err)
	require.Len(t, found, 1)
	require.Equal(t, squirrel.ItemDone, found[0].State,
		"a promoted note is recorded as done; there is no chore state")
}

// The person is part of the lookup rather than checked afterwards: a handler
// is handed an id by whoever is on the other end.
func TestPromoteItemRefusesAnotherPersonsNote(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	itemID := insertItem(t, store, p, "bins out")

	_, ok, err := store.PromoteItem(ctx, p+1, itemID, 24*time.Hour)
	require.NoError(t, err)
	require.False(t, ok)
}

// The two read paths have to agree about what a note is. itemsWhere filters
// the pile through isNote so that "!notes" and "done 2" never appear in it; a
// lookup by id that skipped the same test would let the other view act on a
// row the pile itself refuses to show.
func TestItemByIDUsesThePilesDefinitionOfANote(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	thought := insertItem(t, store, p, "buy milk")
	command := insertItem(t, store, p, "!notes")

	_, ok, err := store.ItemByID(ctx, p, thought)
	require.NoError(t, err)
	require.True(t, ok)

	_, ok, err = store.ItemByID(ctx, p, command)
	require.NoError(t, err)
	require.False(t, ok, "a command is not a note on either read path")
}

func TestPromoteItemRefusesACommand(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	command := insertItem(t, store, p, "!notes")

	_, ok, err := store.PromoteItem(ctx, p, command, 24*time.Hour)
	require.NoError(t, err)
	require.False(t, ok, "there is no chore called !notes")
}
