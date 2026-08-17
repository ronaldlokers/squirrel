package squirrel_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestParseActionAccepts(t *testing.T) {
	cases := []struct {
		in        string
		messageID string
		kind      string
		position  int
		selected  bool
	}{
		{"!action 451 done:2 true", "451", "done", 2, true},
		{"!action 451 done:2 false", "451", "done", 2, false},
		{"!action 12 undefine:1 true", "12", "undefine", 1, true},
	}
	for _, c := range cases {
		got, ok := squirrel.ParseAction(c.in)
		require.True(t, ok, c.in)
		require.Equal(t, c.messageID, got.MessageID, c.in)
		require.Equal(t, c.kind, got.Kind, c.in)
		require.Equal(t, c.position, got.Position, c.in)
		require.Equal(t, c.selected, got.Selected, c.in)
	}
}

// Anything else is a thought. A person typing "!action" at the bot is writing a
// note about actions, and a note is never rejected.
func TestParseActionRejects(t *testing.T) {
	for _, in := range []string{
		"",
		"!action",
		"!action 451",
		"!action 451 done:2",
		"!action 451 done:2 maybe",
		"!action 451 done:x true",
		"!action 451 explode:2 true",
		"i keep meaning to !action 451 done:2 true",
		"buy milk",
	} {
		_, ok := squirrel.ParseAction(in)
		require.False(t, ok, in)
	}
}
