package squirrel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// A body double, not a stopwatch.
//
// You say how long and what on; Squirrel says go, stays quiet, and checks once
// at the end. The point is the going, so nothing here is built to be watched
// and nothing is kept about it afterwards — a record of timers started and
// abandoned is a streak with a different name, and abandoning one halfway is a
// normal ending here exactly as stopping partway through the pile is.
//
// One per person, replaced each time. Starting a second one is starting a
// second one, not an error: the answer to "actually, twenty minutes" is to say
// twenty minutes.

// Timer is what is running, if anything is.
type Timer struct {
	Label   string
	Started time.Time
	Ends    time.Time
}

// Left is how much of it remains, never negative.
func (t Timer) Left(now time.Time) time.Duration {
	if d := t.Ends.Sub(now); d > 0 {
		return d
	}
	return 0
}

// Done reports whether the time is up.
func (t Timer) Done(now time.Time) bool { return !t.Ends.After(now) }

// StartTimer begins one, replacing whatever was there.
func (s *Store) StartTimer(ctx context.Context, personID int64, label string, d time.Duration, now time.Time) (Timer, error) {
	t := Timer{Label: label, Started: now, Ends: now.Add(d)}
	if _, err := s.pool.Exec(ctx, `
		insert into timers (person_id, label, started_at, ends_at, said_at)
		values ($1, $2, $3, $4, null)
		on conflict (person_id) do update
		  set label = excluded.label,
		      started_at = excluded.started_at,
		      ends_at = excluded.ends_at,
		      said_at = null`,
		personID, label, t.Started, t.Ends); err != nil {
		return Timer{}, fmt.Errorf("starting timer: %w", err)
	}
	return t, nil
}

// CurrentTimer is what is running, or nothing.
func (s *Store) CurrentTimer(ctx context.Context, personID int64) (Timer, bool, error) {
	var t Timer
	err := s.pool.QueryRow(ctx,
		`select label, started_at, ends_at from timers where person_id = $1`, personID).
		Scan(&t.Label, &t.Started, &t.Ends)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Timer{}, false, nil
		}
		return Timer{}, false, fmt.Errorf("reading timer: %w", err)
	}
	return t, true, nil
}

// StopTimer ends one early, and leaves nothing behind. Stopping something that
// is not running is not an error — it is the state you asked for.
func (s *Store) StopTimer(ctx context.Context, personID int64) error {
	if _, err := s.pool.Exec(ctx, `delete from timers where person_id = $1`, personID); err != nil {
		return fmt.Errorf("stopping timer: %w", err)
	}
	return nil
}

// ClaimFinishedTimer hands back a timer whose time is up, exactly once.
//
// The claim and the read are one statement, so a scheduler tick that overlaps
// another cannot say "time" twice. Nothing is left afterwards: the row is
// deleted rather than marked, because a finished timer is not a thing this
// product keeps.
func (s *Store) ClaimFinishedTimer(ctx context.Context, personID int64, now time.Time) (Timer, bool, error) {
	var t Timer
	err := s.pool.QueryRow(ctx, `
		delete from timers
		 where person_id = $1 and ends_at <= $2
		returning label, started_at, ends_at`, personID, now).
		Scan(&t.Label, &t.Started, &t.Ends)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Timer{}, false, nil
		}
		return Timer{}, false, fmt.Errorf("claiming finished timer: %w", err)
	}
	return t, true, nil
}
