package squirrel_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestCappedLeavesASmallMessageAlone(t *testing.T) {
	m := squirrel.Message{Text: "Due", Actions: []squirrel.Action{
		{Label: "bin day", Value: "done:1", Emoji: "✅"},
	}}
	require.Equal(t, m, m.Capped())
}

// Campfire takes at most twelve actions. Beyond that the message is rejected
// outright, so the digest would not be sent at all — every number still works,
// which is why dropping the surplus buttons is the safe direction.
func TestCappedTrimsToTwelve(t *testing.T) {
	var actions []squirrel.Action
	for i := range 20 {
		actions = append(actions, squirrel.Action{
			Label: fmt.Sprintf("chore %d", i+1),
			Value: fmt.Sprintf("done:%d", i+1),
		})
	}
	got := squirrel.Message{Text: "Due", Actions: actions}.Capped()
	require.Len(t, got.Actions, squirrel.MaxActions)
	require.Equal(t, "done:1", got.Actions[0].Value, "the first twelve, in order")
	require.Equal(t, "done:12", got.Actions[11].Value)
	require.Equal(t, "Due", got.Text, "the text is untouched — every chore keeps its number")
}
