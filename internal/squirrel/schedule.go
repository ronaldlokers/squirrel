package squirrel

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type SchedulerOptions struct {
	Store *Store
	// Send is the phase 2 plain-text surface. Once() no longer calls it — the
	// digest is a Message now, sent through Chat — but the field stays so
	// boot.go (rewired in a later task) and phase 2 callers still compile.
	Send           Sender
	Chat           Chat
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

	m := DigestMessage(due, captures)
	if m.Text == "" {
		return nil
	}

	forDate := midnight
	promptID, err := s.opts.Store.RecordPrompt(ctx, s.opts.PersonID, s.opts.ConversationID,
		"digest", now, &forDate, due)
	if err != nil {
		if errors.Is(err, ErrDigestAlreadySent) {
			// Some other process already recorded today's digest — most
			// likely this same one, on an earlier tick. Either way, today is
			// spoken for, so remember it and stop asking.
			s.sentDate = dateKey
			return nil
		}
		return err
	}

	messageID, err := s.sendDigest(ctx, m)
	if err != nil {
		// The prompt row is already committed, so the numbering stands and the
		// digest will not be retried today. Reported rather than retried:
		// re-sending risks two messages, and the next day's is minutes away in
		// the scheme of things. delivered_at stays null, so LastDigestSentAt
		// will skip straight past this row rather than anchoring the next
		// digest's capture window to a message that never arrived.
		return fmt.Errorf("sending digest: %w", err)
	}

	if messageID == "" {
		// The transport reported success but returned no id to hang the
		// buttons off — see chatVia's messageIDFrom. The digest still went
		// out, so it is still marked delivered below; it just can never have
		// its buttons disabled and no tap can ever resolve back to it. That is
		// worth a log line, not a lie stored as if it were addressable.
		s.opts.OnError(fmt.Errorf("digest prompt %d delivered with no addressable message id", promptID))
	}

	if err := s.opts.Store.MarkPromptSent(ctx, promptID, messageID, now); err != nil {
		// The message is already out, so this is reported rather than
		// retried. s.sentDate is not set on this path — the return below
		// skips the assignment further down — but the in-memory guard is
		// only ever an optimisation: the next tick retries once(), finds
		// RecordPrompt already satisfied by today's row and fails it with
		// ErrDigestAlreadySent, which is what actually arms sentDate. Worst
		// case the row is never marked delivered and LastDigestSentAt anchors
		// to whichever earlier digest it last saw as delivered instead — a
		// capture window that overlaps and re-lists something already seen,
		// never one that drops something unseen.
		return err
	}

	closePrevious(ctx, s.opts.Store, s.opts.Chat, s.opts.OnError, s.opts.PersonID, promptID)
	s.sentDate = dateKey
	return nil
}

// sendDigest sends through Chat when the transport supports it, and falls
// back to the phase 2 plain-text Send otherwise — Boost and Update are
// already guarded the same way, and Send was the one field this package
// still called unconditionally. That makes "degrade to phase 2 behaviour
// against a transport with no Chat" true by construction rather than by
// deployment discipline, and gives the Send field a reason to still exist.
func (s *Scheduler) sendDigest(ctx context.Context, m Message) (string, error) {
	if s.opts.Chat.Send == nil {
		return "", s.opts.Send(ctx, s.opts.ConversationID, m.Text)
	}
	return s.opts.Chat.Send(ctx, s.opts.ConversationID, m)
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

// closePrevious disables the buttons on the numbered prompt before current, so
// there is exactly one live surface. That bound is what makes undo safe
// without any date arithmetic — there is nothing old left to un-tap.
//
// The update rebuilds the exact action values the previous prompt was
// originally sent with — done:1, done:2, … with the same chore names and
// emoji — rather than sending a synthetic replacement. Two reasons: the
// transport forces disabled on every action regardless of what the values
// say, so reusing the real values is free; and because the values match,
// Campfire's per-user retained selection on the old message survives the
// update instead of being wiped by a button it does not recognise. Text is
// left empty, which chatVia's omitempty then leaves off the request
// entirely — the fork's controller only touches keys actually present, so an
// update carrying no body leaves the room's existing text alone.
//
// Shared by the scheduler and the applier, the only two places that ever open
// a new numbered surface. A failure here is reported and swallowed: the old
// buttons staying live is a degraded surface, but failing to speak in the
// present because closing the past went wrong is silence, and silence is the
// failure this whole phase exists to remove.
func closePrevious(ctx context.Context, store *Store, chat Chat, onError func(error), personID, current int64) {
	if chat.Update == nil {
		return
	}
	prev, ok, err := store.PreviousNumberedPrompt(ctx, personID, current)
	if err != nil || !ok {
		if err != nil {
			onError(fmt.Errorf("finding the previous prompt: %w", err))
		}
		return
	}

	chores, err := store.ChoresOnPrompt(ctx, prev.ID)
	if err != nil {
		onError(fmt.Errorf("loading prompt %d's chores: %w", prev.ID, err))
		return
	}
	// Capped: RecordPrompt stores a prompt_line for every due chore regardless
	// of the button cap the original send applied, so rebuilding straight from
	// prompt_lines can carry more than Campfire's limit of twelve. Above that,
	// Campfire rejects the update outright — and since a failed close is
	// reported and swallowed rather than retried, the old surface would then
	// stay live indefinitely.
	msg := Message{Actions: actionsForChores(chores, "done", "✅")}.Capped()
	if len(msg.Actions) == 0 {
		// The prompt never carried a button to begin with — a query prompt
		// that offered nothing, say. There is nothing to disable, and sending
		// an update with zero actions would fall back to a plain-text body
		// (chatVia only encodes JSON when there is at least one action),
		// which would overwrite the old message with an empty string: the
		// exact bug this rebuild exists to fix, for a different reason.
		return
	}

	if err := chat.Update(ctx, prev.ConversationID, prev.ExternalMessageID, msg); err != nil {
		onError(fmt.Errorf("closing prompt %d: %w", prev.ID, err))
	}
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
