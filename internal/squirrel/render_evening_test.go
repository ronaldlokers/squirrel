package squirrel_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestEveningMessageShowsBothSections(t *testing.T) {
	m := squirrel.EveningMessage([]string{"bin day"}, []string{"buy milk"}, nil)
	require.Contains(t, m.Text, "Today")
	require.Contains(t, m.Text, "bin day")
	require.Contains(t, m.Text, "Since yesterday")
	require.Contains(t, m.Text, "buy milk")
	require.Empty(t, m.Actions)
}

// An empty list is a scoreboard reading nil. An absent section says nothing
// about you. This is the difference, and it is the whole reason "what you did"
// is safe to add at all.
func TestEveningMessageOmitsTodayWhenNothingWasDone(t *testing.T) {
	m := squirrel.EveningMessage(nil, []string{"buy milk"}, nil)
	require.NotContains(t, m.Text, "Today")
	require.NotContains(t, m.Text, "0")
	require.Contains(t, m.Text, "buy milk")
}

func TestEveningMessageIsEmptyWhenThereIsNothingToSay(t *testing.T) {
	m := squirrel.EveningMessage(nil, nil, nil)
	require.Empty(t, m.Text)
	require.Empty(t, m.Actions)
}

// On a day no trigger fired, the fallback nudge joins this message rather than
// arriving as a second notification a second later.
func TestEveningMessageCarriesTheFallbackNudge(t *testing.T) {
	c := overdue(1, "bin day", 19, 14)
	m := squirrel.EveningMessage(nil, []string{"buy milk"}, &c)

	require.Contains(t, m.Text, "bin day")
	require.Contains(t, m.Text, "buy milk")
	require.Len(t, m.Actions, 1, "the nudge keeps its button")
	require.Equal(t, "done:1", m.Actions[0].Value)
}
