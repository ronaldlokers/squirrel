package squirrel_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestMatchCommands(t *testing.T) {
	cases := []struct {
		in       string
		kind     squirrel.IntentKind
		position int
	}{
		{"done", squirrel.IntentComplete, 0},
		{"Done", squirrel.IntentComplete, 0},
		{"did it", squirrel.IntentComplete, 0},
		{"✅", squirrel.IntentComplete, 0},
		{"done 2", squirrel.IntentComplete, 2},
		{"2", squirrel.IntentComplete, 2},
		{"  2  ", squirrel.IntentComplete, 2},
		{"stop 3", squirrel.IntentStop, 3},
		{"?", squirrel.IntentQuery, 0},
		{"nvm", squirrel.IntentDrop, 0},
		{"forget it", squirrel.IntentDrop, 0},
	}
	for _, c := range cases {
		got := squirrel.Match(c.in)
		require.Equal(t, c.kind, got.Kind, c.in)
		require.Equal(t, c.position, got.Position, c.in)
	}
}

// The governing example from the original kickoff, and its siblings. Every one
// of these is a thought, and losing one is the failure this system exists to
// prevent.
func TestMatchTreatsTheseAsCaptures(t *testing.T) {
	for _, in := range []string{
		". done with the flux migration",
		"done with the flux migration",
		"done and dusted",
		"did it work?",
		"stop the world",
		"stop 3 of them",
		"2 things to remember",
		"?? what was that",
		"nvm i figured it out",
		"buy milk",
		"i vacuum every 2 weeks",
		"every day i think about leaving",
	} {
		require.Equal(t, squirrel.IntentCapture, squirrel.Match(in).Kind, in)
	}
}

// The dot forces capture even when the rest is an exact command.
func TestMatchDotForcesCapture(t *testing.T) {
	got := squirrel.Match(". done")
	require.Equal(t, squirrel.IntentCapture, got.Kind)
	require.Equal(t, "done", got.Text)

	got = squirrel.Match(".every 2 weeks: vacuum")
	require.Equal(t, squirrel.IntentCapture, got.Kind)
	require.Equal(t, "every 2 weeks: vacuum", got.Text)
}

func TestMatchDefine(t *testing.T) {
	got := squirrel.Match("every 2 weeks: vacuum")
	require.Equal(t, squirrel.IntentDefine, got.Kind)
	require.Equal(t, "vacuum", got.Name)
	require.Equal(t, 14*24*time.Hour, got.Every)
}

func TestMatchCaptureKeepsTextVerbatim(t *testing.T) {
	got := squirrel.Match("  Buy MILK  ")
	require.Equal(t, squirrel.IntentCapture, got.Kind)
	require.Equal(t, "  Buy MILK  ", got.Text)
}
