package boot

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The scheduler is given a way to push.
//
// It was not. `pusher` was written, tested, and never called: `SchedulerOptions`
// has a `Push` field, nothing in boot ever set it, and `MomentTick`'s
// `if s.opts.Push != nil` was therefore false on every tick since the feature
// shipped. The sending code was dead code, and no push has ever been attempted
// in production.
//
// Nothing caught it. The unit tests for `pusher` exercised the function
// directly, so they passed against something nobody called. Go does not warn
// about an unused package-level function. And the symptom — no notification —
// is indistinguishable from every other reason a notification might not arrive,
// which is what made it survive three releases aimed squarely at it.
//
// The check is that the wiring exists, so this fails if the field goes back to
// being unset. It is deliberately about the option and not about a delivery:
// what happens after the scheduler calls it is `pusher`'s business and is
// covered in pushsays_test.go.
func TestTheSchedulerIsGivenAWayToPush(t *testing.T) {
	opts := schedulerOptionsFor(schedulerWiring{
		config:         squirrel.Config{Push: testPushCfg(t)},
		personID:       1,
		conversationID: "9",
	})

	require.NotNil(t, opts.Push,
		"the scheduler cannot push without one, and its own nil check makes that silent")
}

// And nil stays meaningful: with no VAPID pair there is nothing to push with,
// and the room is still told. That is a supported state rather than a failure,
// so it must not be papered over with a non-nil function that cannot work.
func TestNoKeysMeansNoPusherRatherThanABrokenOne(t *testing.T) {
	opts := schedulerOptionsFor(schedulerWiring{
		config:         squirrel.Config{},
		personID:       1,
		conversationID: "9",
	})

	require.Nil(t, opts.Push)
}
