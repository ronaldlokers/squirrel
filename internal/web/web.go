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
	// IdentityHeader is filled by Traefik's forward-auth middleware.
	IdentityHeader string
	// Identity is the one value that may read this pile. Mount refuses to
	// register a single route when it is empty.
	Identity string
	// PushKey is the VAPID public key the browser needs to subscribe, or empty.
	// Empty means the screen never offers: a subscribe button with no key
	// behind it is a button that fails silently, which is worse than one that
	// was never drawn.
	PushKey string
	// Owner answers whose pile this is. There is one person and SeedOwner
	// reconciles them once, so this is not a per-request lookup — it is a
	// function only because of when it can be answered. Routes must be
	// registered before the server listens, and the owner's id is only known
	// once Postgres has answered, which may be a while after boot or never;
	// internal/boot's nudgeRelay exists for exactly that window. A zero id
	// means "not yet", and the screen says the same thing it says for any
	// other unreachable database.
	Owner func() int64

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

// person answers who the pile belongs to, and whether that is known yet.
func (o Options) person() (int64, bool) {
	if o.Owner == nil {
		return 0, false
	}
	id := o.Owner()
	return id, id != 0
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
	SetItemKind(ctx context.Context, personID, itemID int64, k squirrel.ItemKind) (bool, error)

	// How you are right now. One reading in, one reading out — there is
	// deliberately no way to ask this store for a series.
	RecordCheckin(ctx context.Context, personID int64, m squirrel.Mood, source string, at time.Time) error
	LatestCheckin(ctx context.Context, personID int64) (squirrel.Checkin, bool, error)

	// Things you cannot act on. Note what is absent and stays absent: nothing
	// here counts them, so this screen could not render "4 waiting" even if a
	// later author wanted it to.
	HoldItem(ctx context.Context, personID, itemID int64, state squirrel.ItemState, because string, at time.Time) (bool, error)
	HeldItems(ctx context.Context, personID int64, limit int) ([]squirrel.HeldItem, bool, error)
	Unhold(ctx context.Context, personID, itemID int64, at time.Time) (bool, error)

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
	ItemByID(ctx context.Context, personID, itemID int64) (squirrel.Item, bool, error)
	SetItemState(ctx context.Context, itemID int64, state squirrel.ItemState, at time.Time) error
	Reword(ctx context.Context, personID, itemID int64, text string) (bool, error)
	PromoteItem(ctx context.Context, personID, itemID int64, every time.Duration) (squirrel.Chore, bool, error)

	// The chores half. A chore is not a note and shares none of the note
	// functions, but it is the other thing this pile holds and the screen was
	// the only surface that could not see it.
	ActiveChores(ctx context.Context, personID int64) ([]squirrel.Chore, error)
	SearchChores(ctx context.Context, personID int64, query string, limit int) ([]squirrel.Chore, error)
	UpsertChore(ctx context.Context, personID int64, name string, every, tolerance time.Duration) (squirrel.Chore, error)
	UpsertChoreAsking(ctx context.Context, personID int64, name string, every, tolerance time.Duration, ask squirrel.Asking) (squirrel.Chore, error)
	DeactivateChore(ctx context.Context, choreID int64) error
	RecordCompletion(ctx context.Context, choreID, personID int64, source string, at time.Time) error

	// A thing broken into steps. Note what is absent and stays absent: there
	// is no function here that returns the sequence, so this screen could not
	// render one if a later author wanted it to — the same device that keeps
	// it from rendering a count of the pile.
	// A fixed point the coach proposed and you kept. The same function `!at`
	// and the screen already call.
	CreateMoment(ctx context.Context, personID int64, m squirrel.Moment) (squirrel.Moment, error)

	SaveSteps(ctx context.Context, personID int64, itemID *int64, label string, steps []string) error
	NextStep(ctx context.Context, personID int64) (squirrel.Step, bool, error)
	StepDone(ctx context.Context, personID, stepID int64, at time.Time) error
	ClearSteps(ctx context.Context, personID int64) error
}
