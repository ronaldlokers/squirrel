package squirrel

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

type SchedulerOptions struct {
	Store *Store
	// Send is the plain-text surface. sendMessage falls back to it only when
	// Chat.Send is nil.
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
	// wording for it. Nil means every candidate goes out.
	//
	// A veto and never a trigger: it is only called with something the rules already
	// chose to raise, so it can make Squirrel quieter and has no way to make it
	// louder.
	Interrupt Interrupter
	// Notice reads the board and writes at most two lines about what is on it.
	// Nil is a working state: the strips carry no marginalia, which is what
	// shipped before this existed.
	Notice Noticer
	// Learn reads the record back and says what it shows about how this person
	// works. Nil means Squirrel never learns anything, which is a working state.
	Learn Learner
}

// Learner is the weekly read-back. Declared here rather than imported for the
// reason every other seam in this package is: internal/squirrel must not have
// to know that internal/coach exists.
type Learner func(ctx context.Context, personID int64, record []string) ([]string, error)

// Noticer reads what is on the board and writes at most a couple of lines about
// it, each naming the thing it is about.
type Noticer func(ctx context.Context, personID int64, things []NoticeThing, refused []string) ([]NoticeNote, error)

// NoticeThing is one row the noticer may say something about, and NoticeNote is
// one line it wrote. Both are plain structs rather than the coach's own types,
// because this package must not know internal/coach exists.
type NoticeThing struct {
	Kind  string
	RefID int64
	Words string
}

type NoticeNote struct {
	Kind  string
	RefID int64
	Words string
}

// resurfaceOdds is how often a kept note rides along with the evening message.
// One in three: every evening would make the shelf a stream, and a stream is a
// second inbox.
const resurfaceOdds = 0.34

// quietFrom and quietUntil are the hours nothing arrives unasked in, in the
// scheduler's own location.
//
// Here rather than in Asking because it is a rule about interrupting, not about
// a chore's preference: a chore may want raising at any hour and still not be
// worth waking someone for. Applied to the unasked path only — see Nudge.
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

// Interrupter decides whether to say something now, and what. It returns the
// wording to use — empty means the deterministic message stands — and whether to
// speak at all.
//
// It fails open by contract: an implementation that cannot answer must return
// ("", true).
type Interrupter func(ctx context.Context, personID int64, about string, at time.Time) (string, bool)

type Scheduler struct {
	opts SchedulerOptions

	// sentDate is the local calendar date the evening message has already gone out
	// for, in this process's lifetime.
	//
	// Purely an optimisation. The unique index on (person_id, kind, sent_for_date)
	// is the actual guarantee and stays authoritative across a restart. It says
	// nothing about the nudge, whose budget is the same index on a different kind.
	sentDate string

	// saying is the wording the interrupter supplied for the nudge about to be sent,
	// or empty for the deterministic message. Cleared on every pass.
	saying string
}

// allowed is the veto, and it defaults to yes: a nil interrupter, or one that
// cannot answer, means the nudge goes out. The opposite of how the budget treats
// an unreadable database, deliberately — a failure there costs money.
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

// Once sends today's evening message if it is past the hour and today's has not
// been sent. Idempotency comes from the unique index, not from memory.
//
// It also makes the last attempt at today's nudge via nudgeFor, so a chosen
// chore rides along rather than arriving as a second notification.
//
// A day slept through is skipped rather than sent late.
//
// A panic below is recovered: CapturesSince runs Match over every stored row, so
// one row that panics must not crash the scheduler on every tick from then on.
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

	// nudgeFor is tried first. A non-nil chore means nothing has claimed today's
	// slot, so it joins this message. Nil means nothing is due or a nudge already
	// went out; either way the evening message carries captures alone.
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

	// The capture window is anchored to the last dated message that actually sent,
	// not to a fixed offset: a fixed offset either double-counts between local
	// midnight and the send, or drops captures across a missed day. Before any dated
	// message has gone out it falls back to the last 24 hours.
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

	// A kept note, sometimes, riding along. A failure to read one costs a nicety and
	// never the message, so it is swallowed.
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

	// The evening prompt never carries its own lines: its kind is deliberately not
	// in numberedKinds, so nothing reads prompt_lines off it. The nudge row is the
	// sole owner of the chosen chore's line.
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
		// The prompt row is committed, so the numbering stands and the evening
		// message will not be retried today: re-sending risks two messages.
		// delivered_at stays null, so LastDigestSentAt skips this row rather
		// than anchoring the next capture window to a message nobody received.
		//
		// A nudge that rode along already claimed today's slot in the unique
		// index before this send was known to succeed. Left in place, that
		// claim would refuse every later trigger today over a message the room
		// never got.
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
	// unfixed gap: the message is in the room with a real button on it, but no
	// row yet carries its external_message_id. PreviousNumberedPrompt requires
	// that column, so the button can never be found and disabled — it stays
	// live, and tomorrow's surface becomes a second live one, breaking the
	// "exactly one live numbered surface" bound. Closing it needs a
	// reconciliation sweep against Campfire, which is its own piece of work.
	//
	// The message id belongs on whichever row owns the button, because a tap
	// resolves through PromptByMessageID. When a nudge rode along that is the
	// nudge row, not the evening row it printed inside. external_message_id is
	// unique per prompt, so only one row can hold it anyway.
	//
	// The evening row gets delivered_at with no message id: on a quiet day it
	// carries no button, and delivered_at is what LastDigestSentAt needs to
	// anchor the next capture window.
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
	// Being due and being worth interrupting for are two questions. A chore that
	// says "tuesday evenings" is due on the Sunday exactly as much; it is the asking
	// that waits.
	//
	// Filtered here rather than in the query so the window has one definition.
	worth := due[:0:0]
	for _, c := range due {
		if c.Ask.Open(now) {
			worth = append(worth, c)
		}
	}
	if s.opts.Store.Capacity(ctx, s.opts.PersonID, now) == CapacityLow {
		// The same capacity reading the picker acts on. No "anyway" here, because
		// nobody is asking: this is Squirrel deciding to speak.
		slog.Info("nudge: a low day", "person_id", s.opts.PersonID)
		return nil, 0, nil
	}

	c, ok := PickChore(worth, rand.Float64())
	if !ok {
		// A presence ping that legitimately produces no message is otherwise
		// indistinguishable from one that never arrived.
		slog.Info("nudge: nothing due", "person_id", s.opts.PersonID)
		return nil, 0, nil
	}

	// Asked before the slot is claimed, and that ordering is the whole of it.
	// RecordPrompt spends the day's one nudge in a unique index; deciding to stay
	// quiet after that would spend the day on a message nobody received.
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
// ErrDigestAlreadySent from the claim is the budget working, not a failure.
//
// Nudge is the only place that opens a new numbered surface without once()
// behind it, so it must close the previous one itself. RetractCompletion is
// bounded only below by e.occurred_at >= p.sent_at, so a day-1 button still live
// on day 10 would retract every completion of that chore since day 1.
func (s *Scheduler) Nudge(ctx context.Context, now time.Time, why NudgeReason) error {
	// Quiet hours live here rather than in nudgeFor: this is the path that arrives
	// unasked. The evening message calls nudgeFor too and is not an interruption, so
	// setting that hour to 22:30 must not cost it its chore line.
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

// sendMessage sends any Message through Chat when the transport supports it, and
// falls back to the plain-text Send otherwise, making "degrade against a
// transport with no Chat" true by construction. Shared by both paths.
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
// there is exactly one live surface. That bound is what makes undo safe without
// date arithmetic — there is nothing old left to un-tap.
//
// It rebuilds the exact action values the previous prompt was sent with rather
// than sending a synthetic replacement. The transport forces disabled
// regardless of the values, so reusing the real ones is free, and matching
// values let Campfire's per-user retained selection survive the update. Text is
// left empty, which chatVia's omitempty drops from the request entirely — the
// fork's controller only touches keys present, so the room's text is untouched.
//
// Shared by the scheduler and the applier, the only two places that open a new
// numbered surface. A failure is reported and swallowed: old buttons staying
// live is degraded, but failing to speak now because closing the past went
// wrong is silence.
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
		// Separate for the mildest version of the same reason: it runs once a
		// week and its own clock is the rows it writes, so it has to be
		// reachable on a day that has already spoken.
		if err := s.KnowingTick(ctx, time.Now()); err != nil {
			s.opts.OnError(err)
		}
		// And the board, once a day, for the same reason again: its clock is
		// the rows it writes.
		if err := s.NoticeTick(ctx, time.Now()); err != nil {
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
// Marked said before the send, so "once" is a guarantee rather than a hope. The
// cost is that a failed send spends the warning, accepted because the
// alternative repeats every minute at exactly the moment someone is leaving.
//
// A warning missed entirely is not sent late: "leave about 14:10" at 14:25 is
// worse than silence.
func (s *Scheduler) MomentTick(ctx context.Context, now time.Time) error {
	m, found, err := s.opts.Store.DueMoment(ctx, s.opts.PersonID, now)
	if err != nil || !found {
		return err
	}
	if err := s.opts.Store.MarkMomentSaid(ctx, m.ID, now); err != nil {
		return err
	}

	// The room first, always. Push is the improvement and never the only channel.
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
			// The fixed point's own screen, which is where what to take and
			// anything pointing at it lives. The front door was right while
			// there was nowhere better to land.
			URL: "/at/" + strconv.FormatInt(m.ID, 10),
		}); pushErr != nil {
			s.opts.OnError(fmt.Errorf("pushing a leave-by warning: %w", pushErr))
		}
	}
	return err
}

// TimerTick says "time" when a timer's time is up, once. A minute's granularity,
// which is the tick's own.
//
// The claim deletes the row as it reads it, so two overlapping ticks cannot both
// announce the same timer.
func (s *Scheduler) TimerTick(ctx context.Context, now time.Time) error {
	t, found, err := s.opts.Store.ClaimFinishedTimer(ctx, s.opts.PersonID, now)
	if err != nil || !found {
		return err
	}
	_, err = s.sendMessage(ctx, TimerUpMessage(t))
	return err
}

// learnEvery is how often Squirrel reads the record back. A week: an observation
// worth keeping is one that survived a week of evidence, and how somebody works
// moves on the scale of weeks.
const learnEvery = 7 * 24 * time.Hour

// learnFrom is how much of the record one pass reads. Four hundred turns —
// bounded by what a model can reason over, not by tokens.
const learnFrom = 400

// KnowingTick reads the record back, at most once a week.
//
// Its own tick rather than a branch inside Once, which returns early for the
// rest of the day once the evening message has gone.
//
// Every failure is silent: a week where the read-back could not run is a week
// where Buddy is what he was in July.
func (s *Scheduler) KnowingTick(ctx context.Context, now time.Time) error {
	if s.opts.Learn == nil {
		return nil
	}
	last, err := s.opts.Store.LearnedAt(ctx, s.opts.PersonID)
	if err != nil {
		return fmt.Errorf("reading when it last learned: %w", err)
	}
	if now.Sub(last) < learnEvery {
		return nil
	}

	turns, _, err := s.opts.Store.EverythingSaid(ctx, s.opts.PersonID, learnFrom)
	if err != nil {
		return fmt.Errorf("reading the record back: %w", err)
	}
	record := asRecord(turns)
	if len(record) == 0 {
		// Nothing has been said yet. Not a failure and not an empty
		// conclusion: there is no record to read, so nothing is written and
		// the next tick tries again rather than waiting a week from now.
		return nil
	}

	said, err := s.opts.Learn(ctx, s.opts.PersonID, record)
	if err != nil {
		// The model was unreachable or declined. Nothing is written, so
		// LearnedAt does not move and this is retried on the next tick — which
		// is the right direction for a job that costs one call a week.
		slog.Info("the record was not read back", "error", err)
		return nil
	}

	// Written even when empty, which is what makes the record replaced rather
	// than accumulated: a pass that concluded nothing clears what was there,
	// because keeping last week's would make the set older than it claims.
	if err := s.opts.Store.ReplaceKnowing(ctx, s.opts.PersonID, said, now); err != nil {
		return fmt.Errorf("keeping what was learned: %w", err)
	}
	slog.Info("read the record back", "observations", len(said), "turns", len(record))
	return nil
}

// asRecord is the transcript as lines a model can read: who said it and what was
// said. Not the cards, chips or ids — a serialised button is noise a model will
// dutifully find a pattern in.
func asRecord(turns []Turn) []string {
	out := make([]string, 0, len(turns))
	for _, t := range turns {
		words := strings.TrimSpace(t.Words)
		if words == "" {
			continue
		}
		who := "Buddy"
		if t.Who == SpeakerYou {
			who = "Them"
		}
		out = append(out, who+": "+words)
	}
	return out
}

// noticeEvery is how often the board is read. A day: a note worth having is one
// about what is on the board today, and a model asked every hour will find
// something every hour.
const noticeEvery = 24 * time.Hour

// noticeAtMost is how much of the board one pass is shown. Forty rows — enough
// that the connections worth finding are in it, and few enough that the model
// is not reading a filing cabinet.
const noticeAtMost = 40

// noticeRefusals is how many refused lines the pass is shown. Ten, newest
// first: the point is the shape of what was refused, and a hundred of them
// would be most of the prompt.
const noticeRefusals = 10

// NoticeTick reads the board and writes what it noticed, at most once a day.
//
// Every failure is silent, the same as KnowingTick: a day where this could not
// run is a day where the strips carry whatever they carried, which is what they
// carried for the product's whole life until now.
func (s *Scheduler) NoticeTick(ctx context.Context, now time.Time) error {
	if s.opts.Notice == nil {
		return nil
	}
	last, err := s.opts.Store.NoticedAt(ctx, s.opts.PersonID)
	if err != nil {
		return fmt.Errorf("reading when it last noticed: %w", err)
	}
	if !last.IsZero() && now.Sub(last) < noticeEvery {
		return nil
	}

	items, _, err := s.opts.Store.OpenItems(ctx, s.opts.PersonID, noticeAtMost)
	if err != nil {
		return fmt.Errorf("reading the board: %w", err)
	}
	things := make([]NoticeThing, 0, len(items))
	for _, it := range items {
		things = append(things, NoticeThing{Kind: "note", RefID: it.ID, Words: it.RawText})
	}
	if len(things) < 2 {
		// Nothing to connect. The whole value of this is what one row says
		// about another, and one row says nothing about anything.
		return nil
	}

	refused, err := s.opts.Store.WhatWasRefused(ctx, s.opts.PersonID, noticeRefusals)
	if err != nil {
		return fmt.Errorf("reading what was refused: %w", err)
	}

	notes, err := s.opts.Notice(ctx, s.opts.PersonID, things, refused)
	if err != nil {
		// Unreachable, out of budget, or nothing worth saying. Nothing is
		// written, so the clock does not move and the next tick tries again.
		slog.Info("the board was not read", "error", err)
		return nil
	}
	for _, one := range notes {
		if err := s.opts.Store.Notice(ctx, s.opts.PersonID, one.Kind, one.RefID, one.Words, now); err != nil {
			return fmt.Errorf("keeping what was noticed: %w", err)
		}
	}
	return nil
}
