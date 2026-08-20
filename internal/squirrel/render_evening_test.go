package squirrel_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestEveningMessageShowsBothSections(t *testing.T) {
	m := squirrel.EveningMessage(squirrel.Handled{Chores: []string{"bin day"}}, []string{"buy milk"}, nil)
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
	m := squirrel.EveningMessage(squirrel.Handled{}, []string{"buy milk"}, nil)
	require.NotContains(t, m.Text, "Today")
	require.NotContains(t, m.Text, "0")
	require.Contains(t, m.Text, "buy milk")
}

func TestEveningMessageIsEmptyWhenThereIsNothingToSay(t *testing.T) {
	m := squirrel.EveningMessage(squirrel.Handled{}, nil, nil)
	require.Empty(t, m.Text)
	require.Empty(t, m.Actions)
}

// On a day no trigger fired, the fallback nudge joins this message rather than
// arriving as a second notification a second later.
func TestEveningMessageCarriesTheFallbackNudge(t *testing.T) {
	c := overdue(1, "bin day", 19, 14)
	m := squirrel.EveningMessage(squirrel.Handled{}, []string{"buy milk"}, &c)

	require.Contains(t, m.Text, "bin day")
	require.Contains(t, m.Text, "buy milk")
	// The same pair the nudge itself carries: a chore raised in two places
	// that could be answered two different ways is two views disagreeing.
	require.Len(t, m.Actions, 2, "the nudge keeps its buttons")
	require.Equal(t, "done:1", m.Actions[0].Value)
	require.Equal(t, "snooze:1", m.Actions[1].Value)
}

// A day with tasks finished and no chores used to say nothing at all, on the
// one surface positioned to correct "I did nothing today".
func TestEveningMessageNamesTasksAsWellAsChores(t *testing.T) {
	m := squirrel.EveningMessage(squirrel.Handled{
		Chores: []string{"bin day"},
		Tasks:  []string{"rang the vet"},
		Notes:  3,
	}, nil, nil)

	require.Contains(t, m.Text, "bin day")
	require.Contains(t, m.Text, "rang the vet")
	require.Contains(t, m.Text, "3 notes cleared")
}

// The count is of what happened, and it is the only one. Nothing here says how
// much is left, in any form.
func TestEveningMessageNeverCountsWhatRemains(t *testing.T) {
	m := squirrel.EveningMessage(squirrel.Handled{Tasks: []string{"rang the vet"}},
		[]string{"buy milk"}, nil)

	require.NotContains(t, m.Text, "left")
	require.NotContains(t, m.Text, "outstanding")
	require.NotContains(t, m.Text, "remaining")
	require.NotContains(t, m.Text, "0")
}

// One note is a note.
func TestEveningMessageSaysOneNoteSingular(t *testing.T) {
	m := squirrel.EveningMessage(squirrel.Handled{Notes: 1}, nil, nil)
	require.Contains(t, m.Text, "1 note cleared")
}
