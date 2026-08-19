//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// backdate makes a chore due, the way the other chore tests do.
func backdate(t *testing.T, store *squirrel.Store, name string, days int) {
	t.Helper()
	_, err := store.Pool().Exec(context.Background(),
		`update chores set created_at = now() - make_interval(days => $2) where name = $1`,
		name, days)
	require.NoError(t, err)
}

func dueNames(t *testing.T, store *squirrel.Store, personID int64) []string {
	t.Helper()
	due, err := store.DueChores(context.Background(), personID, time.Now())
	require.NoError(t, err)
	names := []string{}
	for _, c := range due {
		names = append(names, c.Name)
	}
	return names
}

// A nudge arrives at the moment you are least able to decide anything about
// the chore. "Not today" is the answer that was missing: the chore is not
// finished and not retired, it just stops asking for a while.
func TestSnoozeStopsTheAsking(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	choresOf(t, store, p, "bins out")
	backdate(t, store, "bins out", 20)
	require.Equal(t, []string{"bins out"}, dueNames(t, store, p))

	reply := triage(t, store, p, "!snooze bins out for 2 days")

	require.Contains(t, reply, "bins out")
	require.Empty(t, dueNames(t, store, p), "it stops asking")
}

// Snoozed is not done. Nothing about when it is next due changes, so when the
// quiet ends the chore is exactly as due as it was.
func TestSnoozeDoesNotCountAsDoing(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	choresOf(t, store, p, "bins out")
	backdate(t, store, "bins out", 20)

	triage(t, store, p, "!snooze bins out for 2 days")

	// Wind the quiet back to the past: the same thing tomorrow does.
	_, err := store.Pool().Exec(ctx,
		`update chores set snoozed_until = now() - interval '1 minute'`)
	require.NoError(t, err)

	due, err := store.DueChores(ctx, p, time.Now())
	require.NoError(t, err)
	require.Len(t, due, 1)
	require.GreaterOrEqual(t, due[0].SinceDays, 20, "the clock kept running while it was quiet")
}

func TestSnoozeAcceptsHowLong(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	choresOf(t, store, p, "bins out")

	reply := triage(t, store, p, "!snooze bins out for 3 days")

	require.Contains(t, reply, "3 days")

	var until time.Time
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select snoozed_until from chores where name = 'bins out'`).Scan(&until))
	require.WithinDuration(t, time.Now().Add(3*24*time.Hour), until, time.Minute)
}

func TestSnoozeByLineNumber(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	choresOf(t, store, p, "bins out")
	backdate(t, store, "bins out", 20)
	triage(t, store, p, "?")

	triage(t, store, p, "!snooze 1 for a week")

	require.Empty(t, dueNames(t, store, p))
}

// The quiet can be ended early, and it is the same write with a time in the
// past rather than a second command. A snooze you could not take back would be
// a small permanent decision, and this product does not have those.
func TestSnoozeIsReversible(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	choresOf(t, store, p, "bins out")
	backdate(t, store, "bins out", 20)

	triage(t, store, p, "!snooze bins out for 5 days")
	require.Empty(t, dueNames(t, store, p))

	reply := triage(t, store, p, "!snooze bins out now")

	require.Contains(t, reply, "asking again")
	require.Equal(t, []string{"bins out"}, dueNames(t, store, p))
}

func TestSnoozeSaysWhatItCannotFind(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	choresOf(t, store, p, "bins out")

	reply := triage(t, store, p, "!snooze the boiler for 3 days")

	require.Contains(t, reply, "bins out", "say what there is")
}

// No default. A default is a decision made in advance for a moment nobody can
// see, and both obvious ones are wrong somewhere: a day is nothing for a
// fortnightly chore, and an interval is retiring by another name.
func TestSnoozeAsksHowLongRatherThanGuessing(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)
	choresOf(t, store, p, "bins out")
	backdate(t, store, "bins out", 20)

	reply := triage(t, store, p, "!snooze bins out")

	require.Contains(t, reply, "how long")
	require.Equal(t, []string{"bins out"}, dueNames(t, store, p), "nothing happened yet")
}

func TestHelpMentionsSnooze(t *testing.T) {
	require.Contains(t, squirrel.HelpMessage().Text, "!snooze")
}
