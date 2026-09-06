package boot

import (
	"context"
	"testing"

	"github.com/ronaldlokers/squirrel/internal/coach"
	"github.com/stretchr/testify/require"
)

type seenCoach struct {
	coach.NoCoach
	turn coach.Turn
}

func (c *seenCoach) Answer(_ context.Context, t coach.Turn) (coach.Reply, error) {
	c.turn = t
	return coach.Reply{Text: "start with the school"}, nil
}

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
