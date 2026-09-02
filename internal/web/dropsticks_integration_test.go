//go:build integration

package web

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The two presses this ran for went with the rooms on 2 September 2026: the
// board's stamp is one form, and the script animates it rather than posting a
// fragment of its own. What the pair protected — that a drop sticks whichever
// way it was pressed — is one path now, and this is it.
func TestDroppingANoteSurvivesLeavingTheBay(t *testing.T) {
	m, store, mine, _ := theSweep(t)
	ctx := context.Background()

	items, _, err := store.OpenItems(ctx, mine, 50)
	require.NoError(t, err)
	require.NotEmpty(t, items)
	note := items[0]

	shown := m.call(t, "GET", "/?bay=notes", nil).Body.String()
	require.Contains(t, shown, note.RawText, "the rack does not offer it to begin with")

	form := url.Values{
		"id":     {strconv.FormatInt(note.ID, 10)},
		"what":   {"note"},
		"answer": {"drop"},
		"bay":    {"notes"},
	}.Encode()

	code := m.call(t, "POST", "/board/act", strings.NewReader(form)).Code
	require.Less(t, code, 400, "the press was not reported as handled")

	after, found, err := store.ItemByID(ctx, mine, note.ID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, squirrel.ItemDropped, after.State,
		"the press said it was handled and the note is still %s", after.State)

	m.call(t, "GET", "/?bay=chores", nil)
	back := theRackIn(t, m.call(t, "GET", "/?bay=notes", nil).Body.String(), "bay=notes")
	require.NotContains(t, back, `name="id" value="`+strconv.FormatInt(note.ID, 10)+`"`,
		"the rack offers the dropped note again after leaving and coming back")
}
