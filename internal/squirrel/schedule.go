package squirrel

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type SchedulerOptions struct {
	Store          *Store
	Send           Sender
	PersonID       int64
	ConversationID string
	// At is the time since local midnight, so 08:00 is 8h.
	At       time.Duration
	Location *time.Location
	OnError  func(error)
}

type Scheduler struct {
	opts SchedulerOptions

	// sentDate is the local calendar date (YYYY-MM-DD) a digest has already
	// gone out for, in this process's lifetime. It is purely an optimisation —
	// the unique index on (person_id, sent_for_date) is what actually
	// guarantees at most one digest a day, and stays authoritative across a
	// restart that clears this field. Once set it lets a stray tick between
	// the send and midnight skip the database entirely instead of running
	// DueChores, CapturesSince and a doomed insert every minute.
	sentDate string
}

func NewScheduler(o SchedulerOptions) *Scheduler {
	if o.Location == nil {
		o.Location = time.UTC
	}
	if o.OnError == nil {
		o.OnError = func(error) {}
	}
	return &Scheduler{opts: o}
}

// Once sends today's digest if it is past the hour and today's has not been
// sent. Idempotency comes from the unique index on (person_id, sent_for_date),
// not from anything held in memory — a restart inside the window cannot produce
// a second message.
//
// A day slept through is skipped rather than sent late: a message about
// yesterday's chores at three in the morning is noise, and the same chores
// reappear in a few hours anyway.
//
// A panic anywhere below is recovered and reported as an error rather than
// left to propagate: CapturesSince runs Match, in Go, over every stored row
// on every attempt, so a single row that panics Match must not be able to
// fatally crash the scheduler on every tick from then on.
func (s *Scheduler) Once(ctx context.Context, now time.Time) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("scheduler panicked: %v", r)
		}
	}()
	return s.once(ctx, now)
}

func (s *Scheduler) once(ctx context.Context, now time.Time) error {
	local := now.In(s.opts.Location)
	dateKey := local.Format("2006-01-02")
	if dateKey == s.sentDate {
		return nil
	}

	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, s.opts.Location)

	// The threshold is built as a wall-clock time in the target location, not
	// by adding a Duration to the instant "midnight". Add moves an absolute
	// instant, so across a DST transition midnight+8h lands an hour off
	// 08:00 local — late in spring, early in fall. time.Date asks the zone
	// database for "08:00 on this calendar date" directly, which is what the
	// config actually promises.
	hour, min, sec := clockParts(s.opts.At)
	threshold := time.Date(local.Year(), local.Month(), local.Day(), hour, min, sec, 0, s.opts.Location)
	if local.Before(threshold) {
		return nil
	}

	due, err := s.opts.Store.DueChores(ctx, s.opts.PersonID, now)
	if err != nil {
		return err
	}

	// The capture window is anchored to the last digest that actually sent,
	// not to a fixed "yesterday midnight" offset: a fixed offset either
	// double-counts (every normal day's window overlaps the previous day's
	// between local midnight and the send) or drops captures entirely (a
	// missed day leaves a gap between where the last window ended and the
	// next one begins). Anchoring to the real last send closes both gaps.
	// Before any digest has ever gone out there is nothing to anchor to, so
	// the window falls back to the last 24 hours.
	since := midnight.AddDate(0, 0, -1)
	if lastDigest, ok, err := s.opts.Store.LastDigestSentAt(ctx, s.opts.PersonID); err != nil {
		return err
	} else if ok {
		since = lastDigest
	}
	captures, err := s.opts.Store.CapturesSince(ctx, s.opts.PersonID, since)
	if err != nil {
		return err
	}

	text := RenderDigest(due, captures)
	if text == "" {
		return nil
	}

	forDate := midnight
	if _, err := s.opts.Store.RecordPrompt(ctx, s.opts.PersonID, s.opts.ConversationID,
		"digest", now, &forDate, due); err != nil {
		if errors.Is(err, ErrDigestAlreadySent) {
			// Some other process already recorded today's digest — most
			// likely this same one, on an earlier tick. Either way, today is
			// spoken for, so remember it and stop asking.
			s.sentDate = dateKey
			return nil
		}
		return err
	}

	if err := s.opts.Send(ctx, s.opts.ConversationID, text); err != nil {
		// The prompt row is already committed, so the numbering stands and the
		// digest will not be retried today. Reported rather than retried:
		// re-sending risks two messages, and the next day's is minutes away in
		// the scheme of things.
		return fmt.Errorf("sending digest: %w", err)
	}
	s.sentDate = dateKey
	return nil
}

// clockParts decomposes a Duration since midnight into hour, minute and
// second components suitable for time.Date, so a threshold can be built as a
// wall-clock time rather than by adding the Duration to an instant.
func clockParts(d time.Duration) (hour, min, sec int) {
	total := int(d / time.Second)
	hour = total / 3600
	min = (total % 3600) / 60
	sec = total % 60
	return
}

// Run ticks once a minute until the context is cancelled. A minute is fine
// precision for a message whose whole point is that it arrives some time in the
// morning.
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		if err := s.Once(ctx, time.Now()); err != nil {
			s.opts.OnError(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
