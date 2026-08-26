// Package web is the screen, and it is a transport: it imports
// internal/squirrel and the reverse would be an import cycle, which is what
// keeps HTML out of the core.
//
// It is read-and-triage only. There is no route that creates an item and there
// never will be — two capture surfaces means two places to look for a thought,
// which is the problem this product exists to solve.
package web

import (
	"context"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Options is everything the screen needs to be mounted.
//
// Not where it lives: the screen is at the root, and the route table in Mount
// is the whole of it. A configurable mount path meant a prefix on every URL in
// every template, a header to widen the worker's scope by one character, and an
// ingress that had to agree with all of it — for a setting nothing ever set.
type Options struct {
	// Gate is the way in: the OIDC client that turns a code into a person.
	// Nil is refused at mount — a pile with no way in is not a working
	// Squirrel, and a pile with no way in and every route open is worse.
	Gate *Gate
	// Sessions is who is signed in, remembered for a minute. Nil is refused at
	// mount: guard would otherwise refuse every request forever.
	Sessions *sessions
	// RequiredGroup is the Authentik group an account must be in. Refused when
	// empty, and it is the only value in this struct that is. Everything else
	// missing degrades to less product — no coach, no camera, no push. This
	// would degrade to more access.
	RequiredGroup string
	// SessionLife is how long a session lasts without being used. Zero takes
	// the default.
	SessionLife time.Duration
	// Location is where the person is, and it is not where the process is.
	//
	// Threaded rather than read off the clock, the way the scheduler's quiet
	// hours and evening message already take one: a container's zone is an
	// accident of its deployment, and a fixed point booked in the wrong one
	// reads back exactly as typed. See issue #148.
	Location *time.Location
	// PushKey is the VAPID public key the browser needs to subscribe, or empty.
	// Empty means the screen never offers: a subscribe button with no key
	// behind it is a button that fails silently, which is worse than one that
	// was never drawn.
	PushKey string
	// Login turns an OIDC subject into a person, creating them the first
	// time. A func rather than a Store method because this package must not
	// have to know how a person is made — internal/boot supplies it.
	Login func(ctx context.Context, sub, handle string) (int64, error)
	// Photos is where a photograph is kept, or nil. Nil means the camera is
	// never offered — a control that cannot work is worse than one that was
	// never drawn.
	Photos Photos
	// Spool is where a capture is made durable before anything says it was
	// kept. Nil is refused at mount: a screen that captures without one is the
	// gap this exists to close.
	Spool Spool

	// The coach's three seams. Funcs rather than an interface, and funcs of
	// primitives rather than of coach types, because this package must not
	// have to know internal/coach exists — it is a transport, and a transport
	// that depends on a model being reachable is the thing this architecture
	// refuses. internal/boot supplies all three.
	//
	// Ask is nil when there is no coach, and the nil carries meaning: the
	// acorn is still drawn and the four chips still answer, but typing a
	// sentence gets the chips back rather than a reply.
	Ask func(ctx context.Context, personID int64, kind, said, subject string) (Answer, error)
	// Reads answers what was typed into the box and says whether it was a
	// thought worth keeping. Nil with no coach, and the nil means the box does
	// exactly what it always did: keeps the words and says "Kept."
	//
	// The boolean is advice, not an instruction. captureHandler keeps the
	// words first and drops them afterwards if this says to, so a wrong answer
	// costs a note in the pile rather than a note that is gone.
	// The fourth return is a place to draw underneath the reply, when the
	// words asked to see one and Buddy said which. Empty on every other turn,
	// which is nearly all of them.
	Reads func(ctx context.Context, personID int64, said string) (reply string, keep bool, open string, err error)
	// AskedAQuestion is the model in the house: a small one on the cluster,
	// asked whether the words are a question. The second return is whether it
	// answered at all — false falls back to squirrel.LooksLikeAQuestion, which
	// needs nothing running.
	//
	// Nil is a supported configuration and the one this shipped with. It costs
	// electricity in a cupboard rather than money abroad, which is why it may
	// run on everything typed and Reads may not.
	AskedAQuestion func(ctx context.Context, said string) (question bool, answered bool)
	// Recent is the conversation so far, oldest first, or nil.
	Recent func(personID int64) []Exchange
	// Remember adds one round to it. The screen calls this for the ladder's
	// deterministic answers as well as the model's, because what makes the
	// window worth having is that it is the conversation — not the part of it
	// a model happened to produce.
	Remember func(personID int64, said, replied string)
	// Forget drops it. Closing the sheet has to mean the conversation is over,
	// and it has to mean nothing else.
	Forget func(personID int64)
	// Decide lets a model choose among what the picker found, or is nil. The
	// screen never calls it when the picker found nothing: absent rather than
	// empty is a rule about this region, not about who chose.
	Decide squirrel.Decider
	// ForgetOffer drops the decision Decide made, or is nil where there is no
	// coach to have made one.
	//
	// It exists because Decide may answer with a *different row* than the
	// picker chose, and the card then carries that row rather than the
	// picker's. So answering the card writes against a row the picker was
	// never pointing at, the picker's answer does not move, and the cache
	// behind Decide — which invalidates by watching for exactly that movement
	// — serves the same decision back. The handler that answers an offer is
	// the one place that knows an answer happened at all.
	ForgetOffer func(personID int64)
	// Smaller breaks the thing being offered into steps, or is nil. Nil means
	// the ladder's own fixed line is the whole answer, which is what it was
	// before this existed.
	Smaller squirrel.Breaker
	// Spent is what the coach has cost this month and what it is allowed, both
	// already rendered as money, or empty. Nil when there is no coach.
	//
	// The only accruing number this product puts on a screen, and the
	// exception is narrow: it is money rather than a score, it is bounded by a
	// ceiling you set, and it is a fact about a machine rather than about you.
	Spent func(ctx context.Context, personID int64) (spent, ceiling string, ok bool)
	// Split proposes the separate things in one note, or is nil. Splittable is
	// the free check that decides whether asking is worth a call at all —
	// separate, because the card has to know whether to draw the press before
	// anything has been asked.
	Split      func(ctx context.Context, personID int64, text string) ([]string, bool)
	Splittable func(text string) bool
}

// Options.person() was here until 25 August 2026. It read Owner, a
// process-global atomic.Int64, and answered "whose pile this is" for every
// handler in the package. It is deleted rather than left in place because a
// global that still compiles is a global something will use, and the answer it
// gave was correct for exactly one person. personOf(r) replaced it.

// Spool is the durable half of capture, and the same one the room's captures
// go through.
//
// Declared here and satisfied structurally by *squirrel.Spool, like Store. The
// screen wrote straight to Postgres for its whole life, which meant a live
// network and an unhealthy database lost the words — accepted while the screen
// was secondary, wrong once it became the front door, and recorded as wrong at
// the time.
type Spool interface {
	// Write is durable when it returns: written, fsynced, renamed, and the
	// directory fsynced too.
	Write(c squirrel.Capture) (string, error)
	// Writable reports whether the directory can be written to at all, so the
	// slot can refuse loudly rather than accept a thought it cannot keep.
	Writable() bool
}

// Store is the narrow surface the screen consumes. Declared here rather than
// imported: Go satisfies interfaces structurally, so *squirrel.Store fits this
// without either package importing the other's declaration, the same way
// transport.Sink does.
type Store interface {
	OpenItems(ctx context.Context, personID int64, limit int) ([]squirrel.Item, bool, error)
	OpenItemsAfter(ctx context.Context, personID, afterID int64, limit int) ([]squirrel.Item, bool, error)
	SearchItems(ctx context.Context, personID int64, query string, limit int) ([]squirrel.Item, bool, error)
	KeptItems(ctx context.Context, personID int64, limit int) ([]squirrel.Item, bool, error)
	// The screen captures as of v0.12.0. See captureHandler for what that
	// overruled and what it cost.
	InsertItem(ctx context.Context, i squirrel.Item) (bool, error)
	InsertItemReturningID(ctx context.Context, i squirrel.Item) (int64, error)

	// Tasks: what you decided. A kind rather than a state, so a done task is
	// still a task and the archive can exist.
	Tasks(ctx context.Context, personID int64, limit int) ([]squirrel.Item, bool, error)
	ArchivedTasks(ctx context.Context, personID int64, limit int) ([]squirrel.Item, bool, error)
	// What a weekly read of the record concluded about how this person works,
	// and the way to throw it away. The screen only reads and deletes: what
	// writes is the scheduler, once a week.
	Knowing(ctx context.Context, personID int64) ([]string, error)
	ForgetKnowing(ctx context.Context, personID int64) error
	SetItemKind(ctx context.Context, personID, itemID int64, k squirrel.ItemKind) (bool, error)

	// How you are right now. One reading in, one reading out — there is
	// deliberately no way to ask this store for a series.
	RecordCheckin(ctx context.Context, personID int64, m squirrel.Mood, source string, at time.Time) error
	LatestCheckin(ctx context.Context, personID int64) (squirrel.Checkin, bool, error)
	// The readings, for the one page that asks. See internal/web/moods.go for
	// what giving up "unreadable by construction" was traded for.
	CheckinsSince(ctx context.Context, personID int64, since time.Time) ([]squirrel.Checkin, error)

	// Things you cannot act on. Note what is absent and stays absent: nothing
	// here counts them, so this screen could not render "4 waiting" even if a
	// later author wanted it to.
	HoldItem(ctx context.Context, personID, itemID int64, state squirrel.ItemState, because string, at time.Time) (bool, error)
	HeldItems(ctx context.Context, personID int64, limit int) ([]squirrel.HeldItem, bool, error)
	Unhold(ctx context.Context, personID, itemID int64, at time.Time) (bool, error)
	// Something you set aside that nobody has mentioned since. The three
	// states shipped as a one-way door: you park something precisely so you do
	// not have to hold it, and that only works if something else does.
	//
	// One, never a list — a screen that handed back everything you had ever
	// parked would be a second pile wearing a different word.
	GoneQuiet(ctx context.Context, personID int64, at time.Time) (squirrel.HeldItem, bool, error)
	StillHolding(ctx context.Context, personID, itemID int64, at time.Time) (bool, error)

	// The one thing. PickNow chooses it, and the other three are the only
	// answers it takes. There is deliberately no function here that returns
	// more than one offer, for the same reason there is none that returns more
	// than one check-in: a caller cannot render a list it cannot obtain.
	PickNow(ctx context.Context, personID int64, now time.Time, showAnyway bool) (squirrel.Offer, bool, error)
	Did(ctx context.Context, personID int64, o squirrel.Offer, at time.Time) error
	Refuse(ctx context.Context, personID int64, kind squirrel.OfferKind, refID int64, at time.Time) error
	RecordAnswer(ctx context.Context, personID int64, kind squirrel.OfferKind, refID int64, answer squirrel.OfferAnswer, at time.Time) error

	// Where to reach you when you are not looking at the room. Only the
	// leave-by warning ever uses it.
	SaveSubscription(ctx context.Context, personID int64, sub squirrel.Subscription) error

	// The body double. One per person, replaced each time, and nothing kept
	// once it is over.
	StartTimer(ctx context.Context, personID int64, label string, d time.Duration, now time.Time) (squirrel.Timer, error)
	CurrentTimer(ctx context.Context, personID int64) (squirrel.Timer, bool, error)
	StopTimer(ctx context.Context, personID int64) error
	// The hyperfocus exit ramp. Opted in on at the moment a timer is started,
	// on the screen and nowhere else — the chat, the coach and the nudge all
	// start timers nobody ticked a box for.
	ArmRamp(ctx context.Context, personID int64, on bool) error
	RampDue(ctx context.Context, personID int64, at time.Time) (squirrel.Timer, bool, error)
	RampSaid(ctx context.Context, personID int64, at time.Time) error
	HushRamp(ctx context.Context, personID int64, at time.Time) error
	ItemByID(ctx context.Context, personID, itemID int64) (squirrel.Item, bool, error)
	SetItemState(ctx context.Context, itemID int64, state squirrel.ItemState, at time.Time) error
	// MoveItemState is the same write for a caller that knows what the note
	// was when the decision was made. The deck's is deferred by the length of
	// the undo hold, so it is the one write here that can be stale.
	MoveItemState(ctx context.Context, itemID int64, from, to squirrel.ItemState, at time.Time) (bool, error)
	// LandedBadlyLatest is one press saying the last thing Buddy said did not
	// land. Principle 5 was opened knowing this could happen; this is the half
	// that makes it matter afterwards rather than only being recorded.
	LandedBadlyLatest(ctx context.Context, personID int64, at time.Time) (bool, error)
	Reword(ctx context.Context, personID, itemID int64, text string) (bool, error)
	PromoteItem(ctx context.Context, personID, itemID int64, every time.Duration) (squirrel.Chore, bool, error)

	// The chores half. A chore is not a note and shares none of the note
	// functions, but it is the other thing this pile holds and the screen was
	// the only surface that could not see it.
	ActiveChores(ctx context.Context, personID int64) ([]squirrel.Chore, error)
	SearchChores(ctx context.Context, personID int64, query string, limit int) ([]squirrel.Chore, error)
	UpsertChore(ctx context.Context, personID int64, name string, every, tolerance time.Duration) (squirrel.Chore, error)
	UpsertChoreAsking(ctx context.Context, personID int64, name string, every, tolerance time.Duration, ask squirrel.Asking) (squirrel.Chore, error)
	// A chore that comes back on a day rather than after an interval.
	// Separate from the upserts because almost every caller only ever asks how
	// often; only if a day was named does the screen then say which.
	SetChoreRhythm(ctx context.Context, personID, choreID int64, day time.Weekday, weeks int) error
	DeactivateChore(ctx context.Context, choreID int64) error
	RecordCompletion(ctx context.Context, choreID, personID int64, source string, at time.Time) error

	// A thing broken into steps. Note what is absent and stays absent: there
	// is no function here that returns the sequence, so this screen could not
	// render one if a later author wanted it to — the same device that keeps
	// it from rendering a count of the pile.
	// A fixed point the coach proposed and you kept. The same function `!at`
	// and the screen already call.
	CreateMoment(ctx context.Context, personID int64, m squirrel.Moment) (squirrel.Moment, error)
	// One fixed point, what is still coming, and the notes pointing at one.
	// The list was refused for this product's whole life; see at.go for what
	// the owner overturned on 24 August 2026 and what replaced it.
	MomentByID(ctx context.Context, personID, id int64) (squirrel.Moment, bool, error)
	Upcoming(ctx context.Context, personID int64, now time.Time, limit int) ([]squirrel.Moment, error)
	NotesFor(ctx context.Context, personID, momentID int64) ([]squirrel.Item, error)
	// Pointing a note at a fixed point, and putting it back. The pointer is the
	// disposition, so there is no state to move and the reversal is a null.
	AttachNote(ctx context.Context, personID, itemID, momentID int64) (bool, error)
	DetachNote(ctx context.Context, personID, itemID int64) (bool, error)

	// The conversation. The screen is one now — see
	// docs/superpowers/specs/2026-08-24-the-thread-design.md — and these are
	// the only three things done with it: add to it, read the end of it, and
	// walk back up it. There is deliberately nothing here that edits a turn
	// or removes one.
	AppendTurn(ctx context.Context, personID int64, t squirrel.Turn) (squirrel.Turn, error)
	RecentTurns(ctx context.Context, personID int64, limit int) ([]squirrel.Turn, bool, error)
	TurnsBefore(ctx context.Context, personID, beforeID int64, limit int) ([]squirrel.Turn, bool, error)
	// The four numbers on the doors. Computed at read time and stored
	// nowhere, which is what makes the decision that allowed them reversible.
	Waiting(ctx context.Context, personID int64, now time.Time) (squirrel.Waiting, error)

	// Where you got to, when something interrupted you.
	//
	// Losing your place is the failure this product is built around, and until
	// 26 August 2026 it kept no memory of a run in progress. There is
	// deliberately no function here that returns a history of runs: one row per
	// person, replaced, so this cannot become a record of your afternoons.
	MarkRun(ctx context.Context, personID int64, place string, at time.Time) error
	RunFor(ctx context.Context, personID int64, at time.Time) (squirrel.Run, bool, error)
	EndRun(ctx context.Context, personID int64) error

	SaveSteps(ctx context.Context, personID int64, itemID *int64, label string, steps []string) error
	NextStep(ctx context.Context, personID int64) (squirrel.Step, bool, error)
	StepDone(ctx context.Context, personID, stepID int64, at time.Time) error
	ClearSteps(ctx context.Context, personID int64) error
}
