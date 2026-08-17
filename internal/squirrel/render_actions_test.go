package squirrel_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// dueChore builds a Chore carrying an id.
func dueChore(id int64, name string, sinceDays, everyDays int) squirrel.Chore {
	return squirrel.Chore{
		ID: id, Name: name, SinceDays: sinceDays, EveryDays: everyDays,
		Every: time.Duration(everyDays) * 24 * time.Hour,
	}
}

func TestDefinedMessageOffersTheCorrection(t *testing.T) {
	c := dueChore(1, "i have a headache", 0, 14)
	m := squirrel.DefinedMessage(c)

	require.Equal(t, squirrel.RenderDefined(c), m.Text)
	require.Len(t, m.Actions, 1, "no confirm button — doing nothing already means right")
	require.Equal(t, "undefine:1", m.Actions[0].Value)
	require.Equal(t, "📝", m.Actions[0].Emoji)
	require.Equal(t, "single", m.SelectionMode)
}

func TestMoreThanTwelveChoresKeepsEveryNumber(t *testing.T) {
	var chores []squirrel.Chore
	for i := range 15 {
		chores = append(chores, dueChore(int64(i+1), "chore", 3, 2))
	}
	m := squirrel.ListMessage(chores)

	require.Len(t, m.Actions, squirrel.MaxActions)
	require.Contains(t, m.Text, "13.", "every chore keeps its number in the text")
	require.NotContains(t, m.Text, "more", "nothing is said about the cut-off")
}
