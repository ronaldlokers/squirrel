//go:build integration

package squirrel_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A fixed point that has already been warned about must not hold up the next
// one.
//
// DueMoment asked NextMoment for the single earliest upcoming fixed point and
// gave up if that one was already `said`. It never looked past it. So between a
// warning and the thing it was about — twenty-five minutes with the default
// travel and ready times — every later fixed point was invisible to the
// scheduler.
//
// That is not only a testing artefact. Two appointments less than half an hour
// apart means the second one is never warned about, and it is worse than
// silence: a moment blocked past its own warn point never fires at all, because
// MomentTick deliberately refuses to send a warning late. The one job this
// feature has is helping somebody leave on time.
func TestAWarnedFixedPointDoesNotBlockTheNextOne(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	now := time.Now()

	// Both are already inside their warning windows: with the defaults, a
	// warning is given twenty-five minutes before the thing starts.
	earlier, err := store.CreateMoment(ctx, p, squirrel.Moment{
		Label: "dentist", Starts: now.Add(5 * time.Minute),
		Travel: 15 * time.Minute, Ready: 10 * time.Minute,
	})
	require.NoError(t, err)
	later, err := store.CreateMoment(ctx, p, squirrel.Moment{
		Label: "school run", Starts: now.Add(10 * time.Minute),
		Travel: 15 * time.Minute, Ready: 10 * time.Minute,
	})
	require.NoError(t, err)

	require.True(t, now.After(earlier.WarnAt()), "the fixture only means something inside the window")
	require.True(t, now.After(later.WarnAt()))

	// The earlier one has been warned about. It has not started yet, so it is
	// still the next fixed point there is.
	require.NoError(t, store.MarkMomentSaid(ctx, earlier.ID, now))

	got, found, err := store.DueMoment(ctx, p, now)
	require.NoError(t, err)
	require.True(t, found, "the later one is due and nothing has been said about it")
	require.Equal(t, later.ID, got.ID, "got %q", got.Label)
}

// And the rule it must not break: one warning per fixed point, ever.
func TestEveryFixedPointIsStillWarnedAboutOnlyOnce(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	now := time.Now()

	only, err := store.CreateMoment(ctx, p, squirrel.Moment{
		Label: "dentist", Starts: now.Add(5 * time.Minute),
		Travel: 15 * time.Minute, Ready: 10 * time.Minute,
	})
	require.NoError(t, err)

	_, found, err := store.DueMoment(ctx, p, now)
	require.NoError(t, err)
	require.True(t, found)

	require.NoError(t, store.MarkMomentSaid(ctx, only.ID, now))

	_, found, err = store.DueMoment(ctx, p, now)
	require.NoError(t, err)
	require.False(t, found, "said once is said")
}

// Nothing is warned about before its time, however many are waiting.
func TestAFixedPointOutsideItsWindowIsNotDue(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	now := time.Now()

	_, err := store.CreateMoment(ctx, p, squirrel.Moment{
		Label: "dentist", Starts: now.Add(3 * time.Hour),
		Travel: 15 * time.Minute, Ready: 10 * time.Minute,
	})
	require.NoError(t, err)

	_, found, err := store.DueMoment(ctx, p, now)
	require.NoError(t, err)
	require.False(t, found, "hours out, there is nothing to say yet")
}

// The leave-by push names where to go, and where to go is the fixed point.
//
// Nothing tested this payload at all: removing the URL only broke the build,
// on an unused import, which is not proof of anything. A field nothing asserts
// is a field that can quietly go back to being empty — and `Push.URL` already
// spent its whole life written and read by nobody.
func TestTheLeaveByPushNamesTheFixedPoint(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	m := aFixedPoint(t, store, p, "dentist", 5*time.Minute)

	var sent []squirrel.Push
	s := squirrel.NewScheduler(squirrel.SchedulerOptions{
		Store: store, PersonID: p, ConversationID: "9",
		At: 8 * time.Hour, Location: time.UTC,
		Send:    func(context.Context, string, string) error { return nil },
		OnError: func(error) {},
		Push: func(_ context.Context, _ int64, push squirrel.Push) error {
			sent = append(sent, push)
			return nil
		},
	})

	require.NoError(t, s.MomentTick(ctx, time.Now()))

	require.Len(t, sent, 1, "the warning was due and a pusher was configured")
	require.Equal(t, "dentist", sent[0].Title)
	require.Equal(t, "/at/"+strconv.FormatInt(m.ID, 10), sent[0].URL,
		"the tap has to land on the thing it is about")
}
