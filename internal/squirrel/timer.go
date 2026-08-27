package squirrel

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// A body double, not a stopwatch. You say how long and what on; Squirrel says go,
// stays quiet, and checks once at the end.
//
// Nothing is kept about it afterwards: a record of timers started and abandoned
// is a streak with a different name.
//
// One per person, replaced each time. Starting a second is starting a second, not
// an error.

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
	// Ramp says the exit ramp was opted in on when this timer was started.
	// Only ever true for a timer somebody started on the screen and ticked the
	// box for — never for one the chat, the coach or a nudge began.
	Ramp bool
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
	// ended_at is cleared too: a new timer must not inherit the last one's ending.
	//
	// `ramp` is cleared for a stronger version of the same reason. It is an opt-in
	// made about one timer, and six things start timers — none of which can have been
	// opted in on. Left set, a chat-started timer would inherit the last
	// screen-started one's box and interrupt somebody who never asked.
	//
	// ramp_hushed_until is deliberately not cleared: "leave me alone" is about today.
	if _, err := s.pool.Exec(ctx, `
		insert into timers (person_id, label, started_at, ends_at, said_at, ended_at)
		values ($1, $2, $3, $4, null, null)
		on conflict (person_id) do update
		  set label = excluded.label,
		      started_at = excluded.started_at,
		      ends_at = excluded.ends_at,
		      said_at = null,
		      ended_at = null,
		      ramp = false,
		      ramp_said_at = null`,
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

// StopTimer ends one early. Stopping something that is not running is the state
// you asked for, not an error.
//
// It marks rather than deletes, so what you were on survives for an hour: you may
// have stopped to answer the door.
func (s *Store) StopTimer(ctx context.Context, personID int64) error {
	if _, err := s.pool.Exec(ctx,
		`update timers set ended_at = now() where person_id = $1 and ended_at is null`,
		personID); err != nil {
		return fmt.Errorf("stopping timer: %w", err)
	}
	return nil
}

// ClaimFinishedTimer hands back a timer whose time is up, exactly once. The claim
// and the read are one statement, so overlapping ticks cannot say "time" twice.
//
// It marks rather than deletes: one row saying what you were on, for an hour, and
// nothing else — see migration 0017.
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

	// The one place a run is recorded, because this is the only path that means "it
	// reached its end". StopTimer writes nothing: a timer stopped early is not a
	// measurement, and a table holding both would be a record of what you do not
	// finish. See migration 0022.
	//
	// Best effort: a run that fails to record costs one sample.
	if minutes := int(t.Ends.Sub(t.Started).Minutes()); minutes > 0 {
		if _, err := s.pool.Exec(ctx,
			`insert into timer_runs (person_id, label, minutes, ended_at) values ($1, $2, $3, $4)`,
			personID, t.Label, minutes, now); err != nil {
			slog.Error("recording a finished timer", "error", err)
		}
	}
	return t, true, nil
}

// fewestRuns is how many finished runs it takes before the median is worth
// having. Three, because a median of one is not a median — it is the last time
// you did it, wearing a word that implies more.
const fewestRuns = 3

// TypicalMinutes is how long something usually takes, from runs that finished, or
// false when there are too few to say. Its only reader is the coach's typically()
// tool, and it exists to replace a model's guess with an observation.
//
// Matched on the label, case-insensitively and trimmed.
func (s *Store) TypicalMinutes(ctx context.Context, personID int64, label string) (int, bool, error) {
	label = strings.ToLower(strings.TrimSpace(label))
	if label == "" {
		return 0, false, nil
	}

	var median float64
	var runs int
	if err := s.pool.QueryRow(ctx, `
		select coalesce(percentile_cont(0.5) within group (order by minutes), 0), count(*)
		  from timer_runs
		 where person_id = $1 and lower(btrim(label)) = $2`, personID, label).
		Scan(&median, &runs); err != nil {
		return 0, false, fmt.Errorf("reading how long that usually takes: %w", err)
	}
	if runs < fewestRuns {
		return 0, false, nil
	}
	return int(median + 0.5), true, nil
}

// breadcrumb is how long what you were on stays worth mentioning. An hour: after
// that "you were on this" is a fact about earlier rather than an offer. It lapses
// rather than being deleted.
const breadcrumb = time.Hour

// LastFocus is what you were on, if it is still the answer to "where was I". It
// reports the label and nothing else — not how long, not whether you finished,
// all of which would turn a way back in into a report card.
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

// RampAfter is how long past a timer's end counts as "long past". Half an hour,
// flat, rather than a multiple: a multiple lets a two-hour session run four hours
// and interrupts a ten-minute one at twenty.
const RampAfter = 30 * time.Minute

// ArmRamp turns the exit ramp on or off for the running timer. Separate from
// StartTimer because six callers start timers and only the screen, where the
// checkbox is, can know the answer.
func (s *Store) ArmRamp(ctx context.Context, personID int64, on bool) error {
	_, err := s.pool.Exec(ctx, `
		update timers set ramp = $2, ramp_said_at = null where person_id = $1`,
		personID, on)
	if err != nil {
		return fmt.Errorf("arming the exit ramp: %w", err)
	}
	return nil
}

// RampDue is a timer that ran out a while ago and was opted in, if it has not
// been spoken about and today has not been hushed. All four conditions are in the
// query so no caller can forget one.
func (s *Store) RampDue(ctx context.Context, personID int64, at time.Time) (Timer, bool, error) {
	var t Timer
	err := s.pool.QueryRow(ctx, `
		select label, started_at, ends_at from timers
		 where person_id = $1
		   and ramp
		   and ended_at is null
		   and ramp_said_at is null
		   and (ramp_hushed_until is null or $2::timestamptz >= ramp_hushed_until)
		   and $2::timestamptz >= ends_at + make_interval(secs => $3::double precision)`,
		personID, at, int64(RampAfter/time.Second)).
		Scan(&t.Label, &t.Started, &t.Ends)

	if errors.Is(err, pgx.ErrNoRows) {
		return Timer{}, false, nil
	}
	if err != nil {
		return Timer{}, false, fmt.Errorf("reading the exit ramp: %w", err)
	}
	t.Started, t.Ends = s.here(t.Started), s.here(t.Ends)
	return t, true, nil
}

// RampSaid marks it spoken, so it says it once and not once per page draw.
func (s *Store) RampSaid(ctx context.Context, personID int64, at time.Time) error {
	if _, err := s.pool.Exec(ctx,
		`update timers set ramp_said_at = $2 where person_id = $1`, personID, at); err != nil {
		return fmt.Errorf("marking the exit ramp said: %w", err)
	}
	return nil
}

// HushRamp is "leave me alone", meaning today rather than this timer: until the
// end of the day in the person's own zone, because a rolling day would silence
// tomorrow morning as well.
func (s *Store) HushRamp(ctx context.Context, personID int64, at time.Time) error {
	until := s.today(at).AddDate(0, 0, 1)
	if _, err := s.pool.Exec(ctx,
		`update timers set ramp_hushed_until = $2, ramp_said_at = $3 where person_id = $1`,
		personID, until, at); err != nil {
		return fmt.Errorf("hushing the exit ramp: %w", err)
	}
	return nil
}
