package squirrel_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// dueChore builds a Chore carrying an id. render_test.go already defines a
// same-package chore() with a different signature, so this one gets its own
// name rather than colliding with it.
func dueChore(id int64, name string, sinceDays, everyDays int) squirrel.Chore {
	return squirrel.Chore{
		ID: id, Name: name, SinceDays: sinceDays, EveryDays: everyDays,
		Every: time.Duration(everyDays) * 24 * time.Hour,
	}
}

func TestDigestMessageCarriesOneButtonPerDueChore(t *testing.T) {
	m := squirrel.DigestMessage(
		[]squirrel.Chore{dueChore(1, "bin day", 2, 7), dueChore(2, "water plants", 5, 4)},
		[]string{"buy milk"},
	)

	require.Equal(t, squirrel.RenderDigest(
		[]squirrel.Chore{dueChore(1, "bin day", 2, 7), dueChore(2, "water plants", 5, 4)},
		[]string{"buy milk"}), m.Text, "the text is unchanged from phase 2")
	require.Equal(t, "multiple", m.SelectionMode)
	require.Len(t, m.Actions, 2)
	require.Equal(t, "bin day", m.Actions[0].Label)
	require.Equal(t, "done:1", m.Actions[0].Value)
	require.Equal(t, "✅", m.Actions[0].Emoji)
	require.Equal(t, "done:2", m.Actions[1].Value)
}

func TestDigestWithNoDueChoresHasNoButtons(t *testing.T) {
	m := squirrel.DigestMessage(nil, []string{"buy milk"})
	require.Empty(t, m.Actions)
	require.NotEmpty(t, m.Text)
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

func TestMoreThanTwelveDueChoresKeepsEveryNumber(t *testing.T) {
	var due []squirrel.Chore
	for i := range 15 {
		due = append(due, dueChore(int64(i+1), "chore", 3, 2))
	}
	m := squirrel.DigestMessage(due, nil)

	require.Len(t, m.Actions, squirrel.MaxActions)
	require.Contains(t, m.Text, "13.", "every chore keeps its number in the text")
	require.NotContains(t, m.Text, "more", "nothing is said about the cut-off")
}
