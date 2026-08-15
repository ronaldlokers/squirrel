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
func (s *Scheduler) Once(ctx context.Context, now time.Time) error {
	local := now.In(s.opts.Location)
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, s.opts.Location)
	if local.Before(midnight.Add(s.opts.At)) {
		return nil
	}

	due, err := s.opts.Store.DueChores(ctx, s.opts.PersonID, now)
	if err != nil {
		return err
	}
	captures, err := s.opts.Store.CapturesSince(ctx, s.opts.PersonID, midnight.AddDate(0, 0, -1))
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
	return nil
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
