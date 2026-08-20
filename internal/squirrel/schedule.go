package squirrel

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"
)

type SchedulerOptions struct {
	Store *Store
	// Send is the phase 2 plain-text surface. sendMessage falls back to it
	// only when Chat.Send is nil; Once() and Nudge() otherwise send a Message
	// through Chat directly. The field stays so boot.go (rewired in a later
	// task) and phase 2 callers still compile.
	Send           Sender
	Chat           Chat
	PersonID       int64
	ConversationID string
	// At is the time since local midnight, so 08:00 is 8h.
	At       time.Duration
	Location *time.Location
	OnError  func(error)
	// Push reaches a browser rather than the room, and only the leave-by
	// warning uses it. Nil means no pushing, which is a supported state: the
	// room is the channel that always works and this is the one that is fast.
	Push Pusher
	// Interrupt is a veto on a nudge the rules already allowed, and optional
	// wording for it. Nil means every candidate goes out exactly as it did
	// before this existed.
	//
	// A veto and never a trigger. It is only ever called with something the
	// rules already chose to raise — due, inside its asking window, and inside
	// the day's budget — so it can make Squirrel quieter than the rules permit
	// and has no way to make it louder.
	//
	// The two floors below it — quiet hours and capacity — were added after
	// this field and are what make the list of rules it cannot lift true
	// rather than aspirational.
	Interrupt Interrupter
}

// resurfaceOdds is how often a kept note rides along with the evening message.
//
// One in three. Every evening would make the shelf a stream, and a stream is a
// second inbox — the thing this product exists not to have. Never is what it
// was before, and the shelf was then a place notes went to be forgotten
// properly rather than kept.
const resurfaceOdds = 0.34

// quietFrom and quietUntil are the hours nothing arrives unasked in, in the
// scheduler's own location.
//
// A chore with no stated preference is open at every hour, so before this a
// presence ping at four in the morning nudged. Fixed here rather than in
// Asking because it is a rule about interrupting, not about a chore's
// preference: a chore may perfectly well *want* raising at any hour and still
// not be worth waking someone for.
//
// Applied to the unasked path only — see Nudge. One of the rules the
// interrupter below cannot lift; it only ever quietens.
const (
	quietFrom  = 22
	quietUntil = 6
)

// quiet reports whether now is inside the hours nothing is raised in, read in
// the location the rest of the scheduler's day is read in.
func quiet(now time.Time, loc *time.Location) bool {
	h := now.In(loc).Hour()
	return h >= quietFrom || h < quietUntil
}

// Interrupter decides whether to say something now, and what.
//
// A func of primitives, like the coach's other seams. It returns the wording
// to use — empty means the deterministic message stands — and whether to speak
// at all.
//
// It **fails open** by contract: an implementation that cannot answer must
// return ("", true), because a coach that is down must not silently turn off a
// feature that worked without one for months.
type Interrupter func(ctx context.Context, personID int64, about string, at time.Time) (string, bool)

type Scheduler struct {
	opts SchedulerOptions

	// sentDate is the local calendar date (YYYY-MM-DD) the evening message has
	// already gone out for, in this process's lifetime. It is purely an
	// optimisation — the unique index on (person_id, kind, sent_for_date) is
	// what actually guarantees at most one evening message a day, and stays
	// authoritative across a restart that clears this field. Once set it lets
	// a stray tick between the send and midnight skip the database entirely
	// instead of running DueChores, CapturesSince and a doomed insert every
	// minute. It says nothing about whether a nudge has gone out — that budget
	// is enforced by the same index, keyed on a different kind, and is never
	// held in memory at all.
	sentDate string

	// saying is the wording the interrupter supplied for the nudge about to be
	// sent, or empty for the deterministic message. It lives for the few lines
	// between nudgeFor deciding and Nudge sending, the same way the applier's
	// pending prompt id does, and is cleared on every pass.
	saying string
}

// allowed is the veto, and it defaults to yes.
//
// Nil interrupter, or one that cannot answer, means the nudge goes out exactly
// as it did before this existed. That is the opposite of how the budget treats
// a database it cannot read, and the exception is deliberate: a failure there
// costs money, and a failure here would silently stop a feature that never
// needed a model.
func (s *Scheduler) allowed(ctx context.Context, c Chore, now time.Time) (string, bool) {
	if s.opts.Interrupt == nil {
		return "", true
	}
	say, ok := s.opts.Interrupt(ctx, s.opts.PersonID, c.Name, now)
	if !ok {
		slog.Info("nudge: held back", "person_id", s.opts.PersonID, "chore", c.Name)
		return "", false
	}
	return say, true
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

// Once sends today's evening message if it is past the hour and today's has
// not been sent. Idempotency comes from the unique index on
// (person_id, kind, sent_for_date), not from anything held in memory — a
// restart inside the window cannot produce a second message.
//
// It also makes the last attempt at today's nudge, via nudgeFor: if nothing
// has claimed the nudge slot yet, the chosen chore rides along in this same
// message rather than arriving as a second notification a second apart.
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

	midnight := s.localMidnight(now)

	// A pre-flight read, before nudgeFor gets anywhere near claiming a nudge
	// slot: if today's evening message is already delivered — most likely
	// this same process, on an earlier run, before a restart cleared
	// sentDate — the RecordPrompt("evening", ...) below is doomed to collide
	// regardless of what nudgeFor does first. Without this check, nudgeFor
	// would still commit a nudge row on the way to that doomed collision,
	// spending today's nudge slot on a chore nobody is ever shown. See
	// EveningDeliveredFor for why this is a plain read rather than a lock.
	alreadySent, err := s.opts.Store.EveningDeliveredFor(ctx, s.opts.PersonID, midnight)
	if err != nil {
		return err
	}
	if alreadySent {
		s.sentDate = dateKey
		return nil
	}

	// nudgeFor is tried first. A non-nil chore means nothing has claimed
	// today's nudge slot yet, so it joins this message instead of the evening
	// message arriving as a second notification a second apart from an
	// earlier nudge. A nil chore means either nothing is due, or a nudge
	// already went out today — either way the evening message carries
	// captures alone.
	nudge, nudgePromptID, err := s.nudgeFor(ctx, now)
	if err != nil {
		return err
	}

	// Everything worth saying back, not only the chores. A day with four
	// tasks finished and no chores used to produce a message that mentioned
	// none of it.
	handled, err := s.opts.Store.HandledSince(ctx, s.opts.PersonID, midnight)
	if err != nil {
		return err
	}

	// The capture window is anchored to the last dated message that actually
	// sent, not to a fixed "yesterday midnight" offset: a fixed offset either
	// double-counts (every normal day's window overlaps the previous day's
	// between local midnight and the send) or drops captures entirely (a
	// missed day leaves a gap between where the last window ended and the
	// next one begins). Anchoring to the real last send closes both gaps.
	// Before any dated message has ever gone out there is nothing to anchor
	// to, so the window falls back to the last 24 hours.
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

	// A kept note, sometimes, riding along. Roughly one evening in three,
	// because every evening would make the shelf a stream and a stream is a
	// second inbox. A failure to read one costs a nicety and never the
	// message, so it is swallowed rather than returned.
	kept := ""
	if rand.Float64() < resurfaceOdds {
		if text, found, err := s.opts.Store.AKeptItem(ctx, s.opts.PersonID); err != nil {
			s.opts.OnError(err)
		} else if found {
			kept = text
		}
	}

	m := EveningMessage(handled, captures, nudge, kept)
	if m.Text == "" {
		return nil
	}

	// The evening prompt never carries its own lines. Its kind is
	// deliberately not in numberedKinds (see prompts.go), so anything that
	// would read prompt_lines off it — a typed position, closePrevious — never
	// looks: the nudge row is the sole owner of the chosen chore's line,
	// whether it was delivered standalone earlier or is riding along in this
	// same message.
	forDate := midnight
	promptID, err := s.opts.Store.RecordPrompt(ctx, s.opts.PersonID, s.opts.ConversationID,
		"evening", now, &forDate, nil)
	if err != nil {
		if errors.Is(err, ErrDigestAlreadySent) {
			// Some other process already recorded today's evening message —
			// most likely this same one, on an earlier tick. Either way,
			// today is spoken for, so remember it and stop asking.
			s.sentDate = dateKey
			return nil
		}
		return err
	}

	messageID, err := s.sendMessage(ctx, m)
	if err != nil {
		// The prompt row is already committed, so the numbering stands and
		// the evening message will not be retried today. Reported rather
		// than retried: re-sending risks two messages, and tomorrow's is
		// hours away in the scheme of things. delivered_at stays null, so
		// LastDigestSentAt will skip straight past this row rather than
		// anchoring the next dated message's capture window to a message
		// that never arrived.
		//
		// When a nudge rode along, nudgeFor already committed its own row,
		// claiming today's nudge slot in the unique index before it was known
		// whether this send would even succeed — the same shape as Nudge()'s
		// own cleanup below, and for the same reason: left in place, that
		// claim survives the failure and every later trigger today —
		// including a second 19:00 attempt — is refused by a message the
		// room never received. Deleting it here gives the next trigger a
		// real chance instead.
		if nudge != nil {
			deleteUndeliveredNudge(ctx, s.opts.Store, s.opts.OnError, nudgePromptID)
		}
		return fmt.Errorf("sending evening message: %w", err)
	}

	if messageID == "" {
		// The transport reported success but returned no id to hang the
		// button off — see chatVia's messageIDFrom. The message still went
		// out, so it is still marked delivered below; it just can never have
		// its button disabled and no tap can ever resolve back to it. That is
		// worth a log line, not a lie stored as if it were addressable.
		s.opts.OnError(fmt.Errorf("evening prompt %d delivered with no addressable message id", promptID))
	}

	// A crash between here and either MarkPromptSent call below is a known,
	// unfixed gap: the message has already reached the room with a real
	// button on it, but no row in the database yet carries its
	// external_message_id. PreviousNumberedPrompt requires that column to be
	// non-null, so that button can never be found and disabled by any future
	// closePrevious call — it stays live indefinitely, and whatever opens
	// tomorrow becomes a second live surface alongside it, which is exactly
	// the "exactly one live numbered surface" bound this whole mechanism
	// exists to hold. The window is small and pre-existing — phase 2 and 3's
	// digest had the identical shape — and closing it needs a reconciliation
	// sweep (find delivered-looking sends with no recorded id and either
	// confirm or retract them against Campfire) that is its own piece of
	// work, not a fix that belongs inlined here.
	//
	// The message id belongs on whichever row owns the button: a tap
	// resolves through PromptByMessageID, so external_message_id must point
	// at the prompt whose prompt_lines the tap should land on. When a nudge
	// rode along, that is the nudge row — the button is its button, named
	// after its chore — not the evening row it happens to be printed inside.
	// external_message_id is also unique per prompt
	// (prompts_external_message_id_key), so only one of the two rows can
	// hold it regardless.
	//
	// The evening row still gets delivered_at, with no message id of its
	// own: on a quiet day it carries no button of its own, and delivered_at
	// alone is what LastDigestSentAt needs to anchor the next capture
	// window.
	if nudge != nil {
		if err := s.opts.Store.MarkPromptSent(ctx, nudgePromptID, messageID, now); err != nil {
			// The message is already out, so this is reported rather than
			// retried — same reasoning as the evening branch below.
			return err
		}
		if err := s.opts.Store.MarkPromptSent(ctx, promptID, "", now); err != nil {
			return err
		}
	} else if err := s.opts.Store.MarkPromptSent(ctx, promptID, messageID, now); err != nil {
		// The message is already out, so this is reported rather than
		// retried. s.sentDate is not set on this path — the return below
		// skips the assignment further down — but the in-memory guard is
		// only ever an optimisation: the next tick retries once(), finds
		// RecordPrompt already satisfied by today's row and fails it with
		// ErrDigestAlreadySent, which is what actually arms sentDate. Worst
		// case the row is never marked delivered and LastDigestSentAt anchors
		// to whichever earlier dated message it last saw as delivered
		// instead — a capture window that overlaps and re-lists something
		// already seen, never one that drops something unseen.
		return err
	}

	// closePrevious only runs when this send actually opened a button —
	// closing the past is only justified by opening a present. On a quiet
	// evening with nothing new claimed (nudge is nil: nothing is due, or a
	// nudge already went out earlier today), the evening message carries no
	// button of its own, so there is nothing here to justify disabling
	// whatever numbered surface is still live — most likely today's own
	// earlier nudge, mid-way through being actionable. Calling closePrevious
	// anyway would disable that live button while opening no replacement:
	// zero live surfaces from 19:00 until whatever opens tomorrow, for a
	// chore that is still genuinely due.
	//
	// When nudge is non-nil, current is the nudge row specifically, not the
	// evening row: PreviousNumberedPrompt's exclusion must land on whichever
	// row actually owns this send's real external_message_id. Excluding the
	// evening row instead would leave the nudge row just created — same
	// sent_at, same real id — eligible to be found as its own "previous" and
	// closePrevious would disable the button it just sent.
	if nudge != nil {
		closePrevious(ctx, s.opts.Store, s.opts.Chat, s.opts.OnError, s.opts.PersonID, nudgePromptID)
	}
	s.sentDate = dateKey
	return nil
}

// localMidnight is the start of now's day in the scheduler's location. It is
// the date a dated prompt is recorded against, and the boundary CompletedToday
// counts from.
func (s *Scheduler) localMidnight(now time.Time) time.Time {
	l := now.In(s.opts.Location)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, s.opts.Location)
}

// nudgeFor picks a chore and claims today's nudge slot for it. It returns nil
// when there is nothing due, or when a nudge has already gone out today — a
// refusal from the unique index is the budget working, not a failure.
//
// The prompt is recorded before the message is sent, the same ordering the
// evening message uses and for the same reason: the row is what makes the
// index the guarantee.
func (s *Scheduler) nudgeFor(ctx context.Context, now time.Time) (*Chore, int64, error) {
	// Cleared here rather than at either caller, so both passes start from the
	// deterministic message and neither can inherit the other's wording.
	s.saying = ""

	due, err := s.opts.Store.DueChores(ctx, s.opts.PersonID, now)
	if err != nil {
		return nil, 0, err
	}
	// Being due and being worth interrupting for are two questions, and this
	// is where they part company. A chore that says "tuesday evenings" is due
	// on the Sunday exactly as much as on the Tuesday — it is the asking that
	// waits, so the chores screen still shows it and nothing here makes it
	// late.
	//
	// Filtered here rather than in the query on purpose: the window is defined
	// once, in Go, and a second definition in SQL is a second definition to
	// drift.
	worth := due[:0:0]
	for _, c := range due {
		if c.Ask.Open(now) {
			worth = append(worth, c)
		}
	}
	if s.opts.Store.Capacity(ctx, s.opts.PersonID, now) == CapacityLow {
		// The same reading the picker already acts on, applied to the one
		// thing in this product that speaks first. A chore raised on a wiped
		// day is the product asking for something on the day you have least
		// to give — and the chore's own clock keeps running, so nothing is
		// lost by waiting for tomorrow.
		//
		// Unlike the picker there is no "anyway" here, because there is
		// nobody asking: this is Squirrel deciding to speak, and the answer
		// on a low day is not today.
		slog.Info("nudge: a low day", "person_id", s.opts.PersonID)
		return nil, 0, nil
	}

	c, ok := PickChore(worth, rand.Float64())
	if !ok {
		// The case that motivated this whole line: a presence ping that
		// legitimately produces no message is otherwise indistinguishable
		// from a ping that never arrived at all — event 1's "ping accepted"
		// only says a nudge attempt follows, not that it found anything.
		// This fires from both callers of nudgeFor — once()'s evening pass
		// and Nudge()'s presence-triggered one — because the decision being
		// logged ("nothing was due") is the same regardless of who asked.
		slog.Info("nudge: nothing due", "person_id", s.opts.PersonID)
		return nil, 0, nil
	}

	// Asked before the slot is claimed, and that ordering is the whole of it.
	// RecordPrompt below spends the day's one nudge in a unique index; a
	// decision to stay quiet taken after that would spend the day on a message
	// nobody ever received, and the next trigger — including the evening
	// fallback — would be refused by it. Declining here costs nothing and
	// leaves the day open.
	say, ok := s.allowed(ctx, c, now)
	if !ok {
		return nil, 0, nil
	}
	s.saying = say

	forDate := s.localMidnight(now)
	id, err := s.opts.Store.RecordPrompt(ctx, s.opts.PersonID, s.opts.ConversationID,
		"nudge", now, &forDate, []Chore{c})
	if errors.Is(err, ErrDigestAlreadySent) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	return &c, id, nil
}

// Nudge names one chore, at whichever moment reached you first today.
// ErrDigestAlreadySent from the claim below is not surfaced as an error: a
// refused nudge means today's slot is already spoken for, which is the budget
// working exactly as designed, not a failure to report or retry.
//
// Nudge is the only place that ever opens a new numbered surface without
// once() right behind it: once() skips closePrevious on a day it claims
// nothing new (see its comment), so if Nudge did not close the previous
// surface itself, nothing would — the button before this one would stay live
// forever, growing by one every day a nudge fires. That is not just a stray
// button: RetractCompletion is bounded only below, by
// e.occurred_at >= p.sent_at, so a day-1 button still live on day 10 would
// retract every completion of that chore since day 1, not only the one it
// was originally sent for. Closing it here is what keeps "exactly one live
// numbered surface" true, the same bound once()'s own comment on the crash
// window above invokes.
func (s *Scheduler) Nudge(ctx context.Context, now time.Time, why NudgeReason) error {
	// Quiet hours live here rather than in nudgeFor, and the distinction is
	// the point: this is the path that arrives unasked — a message, an
	// arrival — and nothing here is worth waking up for. The evening message
	// calls nudgeFor too, and it is not an interruption: it is a once-a-day
	// thing at an hour that was chosen. Setting that hour to 22:30 should not
	// quietly cost it its chore line.
	if quiet(now, s.opts.Location) {
		slog.Info("nudge: quiet hours", "person_id", s.opts.PersonID)
		return nil
	}

	c, promptID, err := s.nudgeFor(ctx, now)
	if err != nil || c == nil {
		return err
	}

	// Same crash window as once() has below its own sendMessage call: a
	// process that dies between the line above returning and MarkPromptSent
	// committing below leaves a live button in the room that no
	// external_message_id ever gets recorded for, and so nothing can ever
	// close. See once()'s comment for why this is a known, unfixed gap
	// rather than something patched inline here.
	// The coach's wording when it gave one, and the fixed message otherwise.
	// The buttons are the same either way: what may be said is a matter of
	// words, and what may be done is not.
	m := NudgeMessage(*c, why)
	if s.saying != "" {
		m.Text = s.saying
	}
	messageID, err := s.sendMessage(ctx, m)
	if err != nil {
		// nudgeFor already committed promptID, claiming today's nudge slot
		// in the unique index — before it was known whether the send would
		// even succeed. Left in place, that claim survives the failure: it
		// is undelivered, so nothing depends on it, but the index cannot
		// tell "undelivered" from "already sent" and every later trigger
		// today — including the 19:00 fallback the spec relies on to catch
		// exactly this — is refused by a message the room never received. A
		// transient Campfire error would silently spend the whole day's
		// nudge. Deleting it here gives the next trigger a real chance
		// instead. The pre-existing crash-window gap above already accepts
		// worse.
		deleteUndeliveredNudge(ctx, s.opts.Store, s.opts.OnError, promptID)
		return fmt.Errorf("sending nudge: %w", err)
	}
	if err := s.opts.Store.MarkPromptSent(ctx, promptID, messageID, now); err != nil {
		return err
	}

	closePrevious(ctx, s.opts.Store, s.opts.Chat, s.opts.OnError, s.opts.PersonID, promptID)
	return nil
}

// deleteUndeliveredNudge is the best-effort cleanup shared by once() and
// Nudge() for a nudge row that claimed today's slot but whose send failed —
// see the comments at both call sites for why the row must not survive the
// failure. If the delete itself fails, the row is reported rather than
// retried, same as every other failure on either path.
//
// The delete runs on a context derived from ctx with WithoutCancel, not ctx
// itself, and under its own short timeout. A slow send that fails because the
// caller's context was cancelled — a rollout tearing down the scheduler loop
// mid-send, say — would otherwise hand the cleanup an already-cancelled
// context: DeletePrompt would fail for the same reason the send just did,
// and the claimed row would survive anyway, spending the day exactly as if
// this cleanup did not exist. WithoutCancel detaches from that cancellation
// while keeping any values ctx carries (deadlines are dropped too, which is
// why an explicit timeout is added back here); the timeout then bounds how
// long a genuinely stuck delete can block the caller.
func deleteUndeliveredNudge(ctx context.Context, store *Store, onError func(error), promptID int64) {
	delCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := store.DeletePrompt(delCtx, promptID); err != nil {
		onError(fmt.Errorf("deleting undelivered nudge prompt %d: %w", promptID, err))
	}
}

// sendMessage sends any Message through Chat when the transport supports it,
// and falls back to the phase 2 plain-text Send otherwise — Boost and Update
// are already guarded the same way, and Send was the one field this package
// still called unconditionally. That makes "degrade to phase 2 behaviour
// against a transport with no Chat" true by construction rather than by
// deployment discipline, and gives the Send field a reason to still exist.
// Shared by the nudge and evening paths — neither is digest-specific.
func (s *Scheduler) sendMessage(ctx context.Context, m Message) (string, error) {
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
		// Separate from Once on purpose. Once returns early for the rest of
		// the day as soon as the evening message has gone, and a timer started
		// at eight in the evening has to be answered anyway.
		if err := s.TimerTick(ctx, time.Now()); err != nil {
			s.opts.OnError(err)
		}
		// Separate for the same reason, and more sharply: a fixed point at
		// nine in the evening is the one message in this product that must not
		// be skipped because something earlier in the day already ran.
		if err := s.MomentTick(ctx, time.Now()); err != nil {
			s.opts.OnError(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// MomentTick says when to leave, once per fixed point.
//
// It is marked said before the send rather than after, the same ordering every
// other outbound in this file uses and for the same reason: the row is what
// makes "once" a guarantee rather than a hope. The cost is the same too — a
// failed send spends the warning — and it is accepted here for a reason the
// others do not have. The alternative is a leave-by message that repeats every
// minute until it succeeds, at exactly the moment someone is trying to get out
// of the door.
//
// A moment whose warning was missed entirely — the process was down, the
// window passed — is not sent late. "Leave about 14:10" at 14:25 is worse than
// silence: it is wrong, and it is wrong in the direction that makes you trust
// the next one less.
func (s *Scheduler) MomentTick(ctx context.Context, now time.Time) error {
	m, found, err := s.opts.Store.DueMoment(ctx, s.opts.PersonID, now)
	if err != nil || !found {
		return err
	}
	if err := s.opts.Store.MarkMomentSaid(ctx, m.ID, now); err != nil {
		return err
	}

	// The room first, always. Push is the improvement and never the only
	// channel: a browser that has revoked its subscription, a phone with
	// notifications off, a laptop that is closed — all of those still leave the
	// message somewhere it can be found.
	_, err = s.sendMessage(ctx, LeaveMessage(m))

	// And then the fast one, whose failure is nobody's problem: the message
	// has already arrived somewhere by the time this runs.
	if s.opts.Push != nil {
		body := LeaveWords(m)
		if m.Bring != "" {
			body += " · take " + m.Bring
		}
		if pushErr := s.opts.Push(ctx, s.opts.PersonID, Push{
			Title: m.Label,
			Body:  body,
		}); pushErr != nil {
			s.opts.OnError(fmt.Errorf("pushing a leave-by warning: %w", pushErr))
		}
	}
	return err
}

// TimerTick says "time" when a timer's time is up, once.
//
// A minute's granularity, which is the tick's own, and that is fine precision
// for a body double: the point was the going, and a message that arrives
// within a minute of the end says the same thing as one that arrives on the
// second.
//
// The claim deletes the row as it reads it, so two overlapping ticks cannot
// both announce the same timer — and nothing is left behind afterwards,
// because a finished timer is not a thing this product keeps.
func (s *Scheduler) TimerTick(ctx context.Context, now time.Time) error {
	t, found, err := s.opts.Store.ClaimFinishedTimer(ctx, s.opts.PersonID, now)
	if err != nil || !found {
		return err
	}
	_, err = s.sendMessage(ctx, TimerUpMessage(t))
	return err
}
