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

// Timer is what is running, if anything is — or, once it has ended, what you
// were on. See migration 0017 for why the row outlives its own ending, and why
// that is a breadcrumb rather than a history.
type Timer struct {
	Label   string
	Started time.Time
	Ends    time.Time
	// Ended is when it finished or was stopped, and the zero value means it is
	// still running.
	Ended time.Time
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
	// ended_at is cleared as well as the rest: starting a timer replaces
	// whatever was there, running or finished, and a new one must not inherit
	// the last one's ending.
	if _, err := s.pool.Exec(ctx, `
		insert into timers (person_id, label, started_at, ends_at, said_at, ended_at)
		values ($1, $2, $3, $4, null, null)
		on conflict (person_id) do update
		  set label = excluded.label,
		      started_at = excluded.started_at,
		      ends_at = excluded.ends_at,
		      said_at = null,
		      ended_at = null`,
		personID, label, t.Started, t.Ends); err != nil {
		return Timer{}, fmt.Errorf("starting timer: %w", err)
	}
	return t, nil
}

// CurrentTimer is what is running, or nothing. A row that has ended is not a
// running timer, however recently it stopped — that one is read by LastFocus.
func (s *Store) CurrentTimer(ctx context.Context, personID int64) (Timer, bool, error) {
	var t Timer
	err := s.pool.QueryRow(ctx,
		`select label, started_at, ends_at from timers
		  where person_id = $1 and ended_at is null`, personID).
		Scan(&t.Label, &t.Started, &t.Ends)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Timer{}, false, nil
		}
		return Timer{}, false, fmt.Errorf("reading timer: %w", err)
	}
	return t, true, nil
}

// StopTimer ends one early. Stopping something that is not running is not an
// error — it is the state you asked for.
//
// It marks rather than deletes, so that what you were on survives the stopping
// for an hour. Stopping is not failing: you may have stopped to answer the
// door, and the point of the breadcrumb is exactly that case.
func (s *Store) StopTimer(ctx context.Context, personID int64) error {
	if _, err := s.pool.Exec(ctx,
		`update timers set ended_at = now() where person_id = $1 and ended_at is null`,
		personID); err != nil {
		return fmt.Errorf("stopping timer: %w", err)
	}
	return nil
}

// ClaimFinishedTimer hands back a timer whose time is up, exactly once.
//
// The claim and the read are one statement, so a scheduler tick that overlaps
// another cannot say "time" twice — ended_at is what makes the second tick
// find nothing, where it used to be the row's absence.
//
// It marks rather than deletes now. What is kept is one row saying what you
// were on, for an hour, and nothing else: see migration 0017 for why that is
// still not a history of timers started and abandoned.
func (s *Store) ClaimFinishedTimer(ctx context.Context, personID int64, now time.Time) (Timer, bool, error) {
	var t Timer
	err := s.pool.QueryRow(ctx, `
		update timers set ended_at = $2
		 where person_id = $1 and ends_at <= $2 and ended_at is null
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

// breadcrumb is how long what you were on stays worth mentioning.
//
// An hour, because the thing being fixed is coming back to the desk and not
// remembering — and after an hour "you were on this" is a fact about earlier
// rather than an offer. It lapses rather than being deleted: nothing sweeps,
// and the next timer overwrites the row anyway.
const breadcrumb = time.Hour

// LastFocus is what you were on, if it was recent enough to still be the
// answer to "where was I".
//
// It reports the label and nothing else. Not how long you ran it for, not
// whether you finished, not whether you stopped early — none of which is the
// timer's business, and all of which would turn a way back in into a report
// card.
func (s *Store) LastFocus(ctx context.Context, personID int64, now time.Time) (Timer, bool, error) {
	var t Timer
	err := s.pool.QueryRow(ctx, `
		select label, started_at, ends_at, ended_at from timers
		 where person_id = $1 and ended_at is not null and ended_at >= $2`,
		personID, now.Add(-breadcrumb)).
		Scan(&t.Label, &t.Started, &t.Ends, &t.Ended)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Timer{}, false, nil
		}
		return Timer{}, false, fmt.Errorf("reading what you were on: %w", err)
	}
	return t, true, nil
}
