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
	// ended_at is cleared as well as the rest: starting a timer replaces
	// whatever was there, running or finished, and a new one must not inherit
	// the last one's ending.
	//
	// `ramp` is cleared for a stronger version of the same reason. It is an
	// opt-in made about one timer, and six things start timers — the chat's
	// !timer, the coach's own hand, a nudge — none of which can have been
	// opted in on. Left set, a timer the chat started would inherit the last
	// screen-started one's box and interrupt somebody who never asked. The
	// screen arms it again immediately afterwards; nothing else does.
	//
	// ramp_hushed_until is deliberately *not* cleared: "leave me alone" is
	// about today, not about the timer that was running when it was said.
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

	// The one place a run is recorded, and it is this one because this is the
	// only path that means "it reached its end". StopTimer deliberately writes
	// nothing: a timer stopped early is not a measurement, and a table that
	// held both would be a record of what you do not finish. See migration
	// 0022 for the whole of that argument.
	//
	// Best effort. A run that fails to record costs the median one sample; a
	// timer that fails to finish because the recording failed would cost
	// someone the thing they were waiting for.
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

// TypicalMinutes is how long something usually takes, from runs that actually
// finished, or false when there are too few to say.
//
// The only reader is the coach's typically() tool, and nothing renders it.
// What it exists to do is replace a model's guess about duration with an
// observation — a guess is fine for a first run and should not survive a few
// real ones.
//
// Matched on the label the timer carried, case-insensitively and trimmed,
// because "the bins" and "The Bins " are the same thing to everyone except a
// string comparison.
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

// RampAfter is how long past a timer's end counts as "long past".
//
// Half an hour, flat, rather than a multiple of the timer. A multiple sounds
// tidier and is wrong in both directions: it lets a two-hour session run four
// hours before saying anything, and it interrupts a ten-minute one at twenty.
// The number that matters is how long you have been gone, not what you
// originally said.
const RampAfter = 30 * time.Minute

// ArmRamp turns the exit ramp on or off for the timer that is running.
//
// Separate from StartTimer rather than another parameter on it, because six
// callers start timers and only one of them — the screen, where the checkbox
// is — can possibly know the answer. The chat's `!timer`, the coach's own hand
// and the nudge all start timers nobody opted in on, and they should.
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
// been spoken about and today has not been hushed.
//
// Every one of those four conditions is doing work, and the query holds all of
// them so that no caller can forget one.
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

// HushRamp is "leave me alone", and it means today rather than this timer.
//
// Until the end of the day in the person's own zone, not twenty-four hours:
// somebody who says this at four in the afternoon is talking about this
// afternoon, and a rolling day would silence tomorrow morning as well.
func (s *Store) HushRamp(ctx context.Context, personID int64, at time.Time) error {
	until := s.today(at).AddDate(0, 0, 1)
	if _, err := s.pool.Exec(ctx,
		`update timers set ramp_hushed_until = $2, ramp_said_at = $3 where person_id = $1`,
		personID, until, at); err != nil {
		return fmt.Errorf("hushing the exit ramp: %w", err)
	}
	return nil
}
