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

func TestDroppingANoteSurvivesLeavingTheRoom(t *testing.T) {
	for _, press := range []struct {
		what     string
		fragment bool
	}{
		{"the browser's own form", false},
		{"the script's fragment press", true},
	} {
		t.Run(press.what, func(t *testing.T) {
			m, store, mine, _ := theSweep(t)
			ctx := context.Background()

			items, _, err := store.OpenItems(ctx, mine, 50)
			require.NoError(t, err)
			require.NotEmpty(t, items)
			note := items[0]

			shown := m.call(t, "GET", "/r/pile", nil).Body.String()
			require.Contains(t, shown, note.RawText, "the room does not offer it to begin with")

			form := url.Values{
				"id":   {strconv.FormatInt(note.ID, 10)},
				"act":  {"drop"},
				"was":  {string(squirrel.ItemOpen)},
				"from": {"thread"},
				"room": {"pile"},
			}.Encode()

			var code int
			if press.fragment {
				code = m.callFragment(t, "/pile/act", form).Code
			} else {
				code = m.call(t, "POST", "/pile/act", strings.NewReader(form)).Code
			}
			require.Less(t, code, 400, "the press was not reported as handled")

			after, found, err := store.ItemByID(ctx, mine, note.ID)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, squirrel.ItemDropped, after.State,
				"the press said it was handled and the note is still %s", after.State)

			m.call(t, "GET", "/r/chores", nil)
			back := m.call(t, "GET", "/r/pile", nil).Body.String()
			require.NotContains(t, back, `name="id" value="`+strconv.FormatInt(note.ID, 10)+`"`,
				"the room offers the dropped note again after leaving and coming back")
		})
	}
}
