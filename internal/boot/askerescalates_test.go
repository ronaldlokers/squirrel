package boot

import (
	"context"
	"testing"

	"github.com/ronaldlokers/squirrel/internal/coach"
	"github.com/stretchr/testify/require"
)

// seenCoach keeps the turn it was handed, so a test can read what the one place
// overwhelm is recognised actually decided.
type seenCoach struct {
	coach.NoCoach
	turn coach.Turn
}

func (c *seenCoach) Answer(_ context.Context, t coach.Turn) (coach.Reply, error) {
	c.turn = t
	return coach.Reply{Text: "start with the school"}, nil
}

// The escalation reads what was said, and the screen's press is the surface
// that has to put the strip there. Both halves, in one test each, because
// either alone passes while the path is broken: this one shows the rule fires
// on words handed over as said, and TestPressingAskSaysTheStripSoAPileCanBeSeenAsOne
// shows the board hands them over.
func TestAPileHandedOverAsWhatWasSaidEscalates(t *testing.T) {
	c := &seenCoach{}
	turn := asker(c, nil, coach.NewConversations(), true)
	require.NotNil(t, turn)

	_, err := turn(context.Background(), 1, "strip", "notes",
		"the tax thing, the vet, the bins and ring the school", "")

	require.NoError(t, err)
	require.True(t, c.turn.Deep, "a pile did not reach the deep model")
	require.Equal(t, coach.KindOverwhelm, c.turn.Kind)
}

// And a pile that only ever sits in subject does not, which is what the board
// used to do: a fixed phrase as said, the strip's words on screen. The rule
// never saw them.
func TestAPileLeftInSubjectDoesNotEscalate(t *testing.T) {
	c := &seenCoach{}
	turn := asker(c, nil, coach.NewConversations(), true)

	_, err := turn(context.Background(), 1, "strip", "notes",
		"What is going on with this?",
		"the tax thing, the vet, the bins and ring the school")

	require.NoError(t, err)
	require.False(t, c.turn.Deep)
	require.Equal(t, "strip", c.turn.Kind)
}
