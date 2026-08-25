package boot

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The scheduler is given a way to learn.
//
// The same check `Push` earned the hard way: `SchedulerOptions` has a `Learn`
// field, `KnowingTick` returns immediately when it is nil, and a field left
// unset in a struct literal is invisible — the symptom is Squirrel simply not
// knowing anything about you, which is indistinguishable from a quiet fortnight.
//
// Deliberately about the option and not about what a pass concludes. What
// happens after the scheduler calls it is the coach's business.
func TestTheSchedulerIsGivenAWayToLearn(t *testing.T) {
	opts := schedulerOptionsFor(schedulerWiring{
		personID:       1,
		conversationID: "9",
		learn: func(context.Context, int64, []string) ([]string, error) {
			return nil, nil
		},
	})

	require.NotNil(t, opts.Learn,
		"the scheduler cannot learn without one, and its own nil check makes that silent")
}

// And nil stays meaningful. With no key there is nothing to read the record
// back with, and a Buddy who knows nothing about you is the Buddy that shipped
// for a month — a supported state rather than a failure, so it must not be
// papered over with a function that cannot work.
func TestNoCoachMeansNoLearnerRatherThanABrokenOne(t *testing.T) {
	require.Nil(t, learner(coach.NoCoach{}),
		"a build with no key was given something to call")

	opts := schedulerOptionsFor(schedulerWiring{personID: 1, conversationID: "9"})
	require.Nil(t, opts.Learn)
}

// With a coach there is one, and it is the coach's own method rather than a
// wrapper that could quietly answer something else.
func TestWithACoachThereIsALearner(t *testing.T) {
	require.NotNil(t, learner(stubCoach{}))
}

// stubCoach is a coach that is not NoCoach. Every method answers
// ErrUnavailable, which is what the real one does when it cannot reach a model
// — what is being tested here is the type check, not a behaviour.
type stubCoach struct{ coach.NoCoach }

var _ coach.Coach = stubCoach{}

// And what the prompt does with it. The observations reach the model or they
// change nothing at all, and nowFor is the one place that decides.
func TestWhatIsKnownReachesTheModel(t *testing.T) {
	require.Contains(t, coach.System(coach.Now{
		Knowing: []string{"Phone calls get done; forms get put off."},
	}, "chat"), "Phone calls get done")
}

var _ = squirrel.Learner(nil)

// The screen is given a way to read the box.
//
// The third field to earn this check. `captureHandler` returns the old
// behaviour when `Reads` is nil, so a field left unset in the options literal
// is a box that silently goes back to being a filing cabinet — and the symptom
// is Buddy saying "Kept.", which is exactly what he said for a month.
func TestTheScreenIsGivenAWayToReadTheBox(t *testing.T) {
	require.Nil(t, reader(coach.NoCoach{}, nil),
		"a build with no key was given something to call")
	require.NotNil(t, reader(stubCoach{}, nil))
}
