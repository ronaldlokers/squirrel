package coach_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
)

func reading(api *toolAPI) *coach.Provider {
	p := coach.NewProvider(api.server.URL, "sk-test", "gpt-5.6-luna", "gpt-5.6-terra",
		coach.Budget{Log: &fakeLog{}, CeilingFor: coach.FlatCeiling(10_000_000)})
	p.Clock = func() time.Time { return august }
	return p
}

// The box can show you a place, which is the whole of this fix: asking to see
// the chores in the one box this product has got "I can't see your chores from
// here", because this path carried no tools beyond the answer itself.
func TestTheBoxNamesAPlaceToShow(t *testing.T) {
	api := newToolAPI(t, turnOf(call("a", "answer", map[string]any{
		"say": "Here they are.", "keep": false, "open": "chores",
	})))

	say, keep, open, err := reading(api).Reads(
		context.Background(), 1, "show me the chores", coach.Now{})
	require.NoError(t, err)
	require.Equal(t, "Here they are.", say)
	require.False(t, keep)
	require.Equal(t, "chores", open)
}

// A place that is not one of them is no place. The same lookup the acting path
// uses: a name nobody recognises is a miss here rather than an empty turn on
// the screen.
func TestTheBoxRefusesAPlaceThatDoesNotExist(t *testing.T) {
	api := newToolAPI(t, turnOf(call("a", "answer", map[string]any{
		"say": "You have no inbox.", "keep": false, "open": "inbox",
	})))

	say, _, open, err := reading(api).Reads(
		context.Background(), 1, "show me the inbox", coach.Now{})
	require.NoError(t, err)
	require.Equal(t, "You have no inbox.", say, "the answer is kept even when the place is not")
	require.Empty(t, open)
}

// And nearly every turn names none, which must stay the ordinary shape.
func TestAnOrdinaryThoughtNamesNoPlace(t *testing.T) {
	api := newToolAPI(t, turnOf(call("a", "answer", map[string]any{
		"say": "That is worth doing.", "keep": true,
	})))

	_, keep, open, err := reading(api).Reads(
		context.Background(), 1, "the boiler again", coach.Now{})
	require.NoError(t, err)
	require.True(t, keep)
	require.Empty(t, open)
}
