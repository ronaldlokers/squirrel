package squirrel_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func chore(name string, since, every int) squirrel.Chore {
	return squirrel.Chore{Name: name, SinceDays: since, EveryDays: every}
}

func TestRenderDigest(t *testing.T) {
	got := squirrel.RenderDigest(
		[]squirrel.Chore{chore("bin day", 2, 7), chore("vacuum", 19, 14)},
		[]string{"buy milk", "the thing with the boiler"},
	)

	require.Contains(t, got, " 1. bin day — 2 days, usually 7")
	require.Contains(t, got, " 2. vacuum — 19 days, usually 14")
	require.Contains(t, got, "· buy milk")
	require.Contains(t, got, "· the thing with the boiler")
}

// Nothing due and nothing captured means no message at all. A daily "nothing to
// report" is how you teach someone to stop reading.
func TestRenderDigestSaysNothingWhenThereIsNothing(t *testing.T) {
	require.Empty(t, squirrel.RenderDigest(nil, nil))
}

func TestRenderDigestOmitsEmptySections(t *testing.T) {
	onlyChores := squirrel.RenderDigest([]squirrel.Chore{chore("bin day", 2, 7)}, nil)
	require.NotContains(t, onlyChores, "Since yesterday")

	onlyCaptures := squirrel.RenderDigest(nil, []string{"buy milk"})
	require.NotContains(t, onlyCaptures, "Due")
}

// No streaks, no counts of misses, no escalation as the number grows.
func TestRenderDigestNeverScolds(t *testing.T) {
	got := strings.ToLower(squirrel.RenderDigest(
		[]squirrel.Chore{chore("vacuum", 200, 14)}, nil))

	for _, word := range []string{"overdue", "still", "again", "late", "!", "should"} {
		require.NotContains(t, got, word)
	}
}

func TestRenderDefined(t *testing.T) {
	got := squirrel.RenderDefined(squirrel.Chore{Name: "vacuum", EveryDays: 14})
	require.Contains(t, got, "vacuum, every 14 days")
	require.Contains(t, got, "14 days")
	require.Contains(t, got, "nvm")
}

func TestRenderListEmpty(t *testing.T) {
	require.Contains(t, squirrel.RenderList(nil), "No chores yet")
}
