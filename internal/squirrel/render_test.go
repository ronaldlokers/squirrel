package squirrel_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestRenderDefined(t *testing.T) {
	got := squirrel.RenderDefined(squirrel.Chore{Name: "vacuum", EveryDays: 14})
	require.Contains(t, got, "vacuum, every 14 days")
	require.Contains(t, got, "14 days")
	require.Contains(t, got, "nvm")
}

// A chore defined with a one-day interval reads "every 1 day", not "every 1
// days" — the plural helper both call sites in RenderDefined route through.
func TestRenderDefinedUsesSingularDayForAOneDayInterval(t *testing.T) {
	got := squirrel.RenderDefined(squirrel.Chore{Name: "meds", EveryDays: 1})
	require.Contains(t, got, "every 1 day.")
	require.NotContains(t, got, "1 days")
}

func TestRenderListEmpty(t *testing.T) {
	require.Contains(t, squirrel.RenderList(nil), "No chores yet")
}
