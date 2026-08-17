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

func TestRenderListEmpty(t *testing.T) {
	require.Contains(t, squirrel.RenderList(nil), "No chores yet")
}
