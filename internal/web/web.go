// Package web is the screen, and it is a transport: it imports internal/squirrel
// and the reverse would be an import cycle, which keeps HTML out of the core.
//
// Read-and-triage only. No route creates an item and none ever will — two
// capture surfaces means two places to look for a thought.
package web

import (
	"context"
	"net/http"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Options is everything the screen needs to be mounted. Not where it lives: the
// screen is at the root and the route table in Mount is the whole of it.
type Options struct {
	// Gate is the way in: the OIDC client that turns a code into a person.
	// Nil is refused at mount — a pile with no way in is not a working
	// Squirrel, and a pile with no way in and every route open is worse.
	Gate *Gate
	// Sessions is who is signed in, remembered for a minute. Nil is refused at
	// mount: guard would otherwise refuse every request forever.
	Sessions *sessions
	// RequiredGroup is the Authentik group an account must be in, and the only value
	// here that is refused when empty. Everything else missing degrades to less
	// product; this would degrade to more access.
	RequiredGroup string
	// SessionLife is how long a session lasts without being used. Zero takes
	// the default.
	SessionLife time.Duration
	// Location is where the person is, not where the process is. Threaded rather
	// than read off the clock: a container's zone is an accident of its deployment.
	// See issue #148.
	Location *time.Location
	// PushKey is the VAPID public key the browser needs to subscribe, or empty.
	// Empty means the screen never offers, rather than drawing a button that fails
	// silently.
	PushKey string
	// Login turns an OIDC subject into a person, creating them the first
	// time. A func rather than a Store method because this package must not
	// have to know how a person is made — internal/boot supplies it.
	Login func(ctx context.Context, sub, handle string) (int64, error)
	// RememberWho keeps what the gate said that is not identity: a display name
	// and a picture. Separate from Login because Login resolves who you are
	// and this only decides what the screen calls you — widening Login would
	// have made presentation a parameter of identity.
	//
	// Optional: nil is a screen that shows a monogram and a handle.
	RememberWho func(ctx context.Context, personID int64, name string, face []byte, faceType string) error
	// Fetch is who goes and gets the picture, so a test can answer without a
	// network. nil is http.DefaultClient.
	Fetch interface {
		Do(*http.Request) (*http.Response, error)
	}
	// Photos is where a photograph is kept, or nil. Nil means the camera is
	// never offered — a control that cannot work is worse than one that was
	// never drawn.
	Photos Photos

	// The coach's three seams. Funcs of primitives rather than an interface, because
	// this package must not know internal/coach exists. internal/boot supplies them.
	//
	// Ask is nil when there is no coach, and the nil carries meaning: the four chips
	// still answer, but typing a sentence gets the chips back rather than a reply.
	Ask func(ctx context.Context, personID int64, kind, room, said, subject string) (Answer, error)
	// Reads answers what was typed into the box and says whether it was a thought
	// worth keeping. Nil with no coach, and then the box keeps the words and says
	// "Kept."
	//
	// The boolean is advice, not an instruction: captureHandler keeps the words first
	// and drops them afterwards, so a wrong answer costs a note rather than a
	// thought.
	//
	// The fourth return is a place to draw underneath the reply, when the words asked
	// to see one.
	Reads func(ctx context.Context, personID int64, said string) (reply string, keep bool, open string, err error)
	// AskedAQuestion is the model in the house, asked whether the words are a
	// question. The second return is whether it answered at all; false falls back to
	// squirrel.LooksLikeAQuestion, which needs nothing running.
	//
	// Nil is a supported configuration. It costs electricity rather than money, which
	// is why it may run on everything typed and Reads may not.
	AskedAQuestion func(ctx context.Context, said string) (question bool, answered bool)
	// Recent is the conversation so far, oldest first, or nil.
	Recent func(personID int64, room string) []Exchange
	// Remember adds one round to it, for the ladder's deterministic answers as well
	// as the model's: the window is the conversation, not the part a model produced.
	Remember func(personID int64, room, said, replied string)
	// Forget drops it. Ending a conversation has to mean it is over, and has to
	// mean nothing else.
	Forget func(personID int64, room string)
	// Smaller breaks the thing being offered into steps, or is nil. Nil means
	// the ladder's own fixed line is the whole answer, which is what it was
	// before this existed.
	Smaller squirrel.Breaker
	// Spent is what the coach has cost this month and what it is allowed, rendered as
	// money, or empty. The only accruing number on a screen here: money rather than a
	// score, bounded by a ceiling you set, and a fact about a machine.
	Spent func(ctx context.Context, personID int64) (spent, ceiling string, ok bool)
	// Split proposes the separate things in one note, or is nil. Splittable is the
	// free check that decides whether asking is worth a call, separate because the
	// card must know whether to draw the press before anything is asked.
	Split      func(ctx context.Context, personID int64, text string) ([]string, bool)
	Splittable func(text string) bool
}

// Store is the narrow surface the screen consumes. Declared here rather than
// imported: *squirrel.Store fits it structurally, like transport.Sink.
type Store interface {
	OpenItems(ctx context.Context, personID int64, limit int) ([]squirrel.Item, bool, error)
	OpenItemsAfter(ctx context.Context, personID, afterID int64, limit int) ([]squirrel.Item, bool, error)
	SearchItems(ctx context.Context, personID int64, query string, limit int) ([]squirrel.Item, bool, error)
	KeptItems(ctx context.Context, personID int64, limit int) ([]squirrel.Item, bool, error)
	TriagedSince(ctx context.Context, personID int64, since time.Time) ([]squirrel.Item, error)
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
	// Something you set aside that nobody has mentioned since. One, never a list: a
	// screen handing back everything you ever parked is a second pile.
	GoneQuiet(ctx context.Context, personID int64, at time.Time) (squirrel.HeldItem, bool, error)
	StillHolding(ctx context.Context, personID, itemID int64, at time.Time) (bool, error)

	// The one thing. There is deliberately no function here returning more than one
	// offer, for the same reason as the check-in: a caller cannot render a list it
	// cannot obtain.
	PickNow(ctx context.Context, personID int64, now time.Time, showAnyway bool) (squirrel.Offer, bool, error)
	Did(ctx context.Context, personID int64, o squirrel.Offer, at time.Time) error
	Refuse(ctx context.Context, personID int64, kind squirrel.OfferKind, refID int64, at time.Time) error
	RecordAnswer(ctx context.Context, personID int64, kind squirrel.OfferKind, refID int64, answer squirrel.OfferAnswer, at time.Time) error

	// Where to reach you when you are not looking at the room. Only the
	// leave-by warning ever uses it.
	SaveSubscription(ctx context.Context, personID int64, sub squirrel.Subscription) error
	// Notifying is whether anything would be sent to, and StopNotifying is the
	// way off. Both exist so that settings can say what the state is rather
	// than offer a control that cannot report one.
	Notifying(ctx context.Context, personID int64) (bool, error)
	StopNotifying(ctx context.Context, personID int64, at time.Time) error

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
	// MoveItemState is the same write for a caller that knows what the note was
	// when the decision was made. A card names the state it was drawn from, and
	// the room can move the row in between — so this is the one write here that
	// can arrive stale, and the only one that says so.
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

	// A thing broken into steps. There is no function here that returns the sequence,
	// so this screen cannot render one.
	CreateMoment(ctx context.Context, personID int64, m squirrel.Moment) (squirrel.Moment, error)
	// One fixed point, what is still coming, and the notes pointing at one.
	// The list was refused for this product's whole life; see at.go for what
	// the owner overturned on 24 August 2026 and what replaced it.
	MomentByID(ctx context.Context, personID, id int64) (squirrel.Moment, bool, error)
	// MomentDone closes one: you left, or it stopped mattering, and nothing
	// records which of the two it was.
	MomentDone(ctx context.Context, personID, id int64, at time.Time) error
	Upcoming(ctx context.Context, personID int64, now time.Time, limit int) ([]squirrel.Moment, error)
	NotesFor(ctx context.Context, personID, momentID int64) ([]squirrel.Item, error)
	// Pointing a note at a fixed point, and putting it back. The pointer is the
	// disposition, so there is no state to move and the reversal is a null.
	AttachNote(ctx context.Context, personID, itemID, momentID int64) (bool, error)
	DetachNote(ctx context.Context, personID, itemID int64) (bool, error)

	// The conversation: add to it, read the end of it, walk back up it. There is
	// deliberately nothing that edits a turn or removes one.
	AppendTurn(ctx context.Context, personID int64, room string, t squirrel.Turn) (squirrel.Turn, error)
	RecentTurns(ctx context.Context, personID int64, room string, limit int) ([]squirrel.Turn, bool, error)
	// EverythingSaid and EverythingBefore are the same read unscoped by room:
	// Buddy's room draws what was said rather than what was said in it.
	EverythingSaid(ctx context.Context, personID int64, limit int) ([]squirrel.Turn, bool, error)
	EverythingBefore(ctx context.Context, personID, beforeID int64, limit int) ([]squirrel.Turn, bool, error)
	WhatWasSaid(ctx context.Context, personID int64, limit int) ([]squirrel.Said, error)
	// What it noticed about the things on the board, and the way to say a line
	// was not worth having.
	WhatWasNoticed(ctx context.Context, personID int64) ([]squirrel.Noticed, error)
	NotUseful(ctx context.Context, personID, id int64, at time.Time) (bool, error)
	// Who the screen is talking to: a name to show, and whether there is a
	// picture to show beside it.
	WhoIs(ctx context.Context, personID int64) (squirrel.Whom, error)
	PersonFace(ctx context.Context, personID int64) ([]byte, string, bool, error)
	TurnsBefore(ctx context.Context, personID int64, room string, beforeID int64, limit int) ([]squirrel.Turn, bool, error)
	// The four numbers on the doors. Computed at read time and stored
	// nowhere, which is what makes the decision that allowed them reversible.
	Waiting(ctx context.Context, personID int64, now time.Time) (squirrel.Waiting, error)

	// Where you got to, when something interrupted you. One row per person,
	// replaced, so this cannot become a record of your afternoons.
	MarkRun(ctx context.Context, personID int64, place string, at time.Time) error
	RunFor(ctx context.Context, personID int64, at time.Time) (squirrel.Run, bool, error)
	EndRun(ctx context.Context, personID int64) error

	SaveSteps(ctx context.Context, personID int64, itemID *int64, label string, steps []string) error
	NextStep(ctx context.Context, personID int64) (squirrel.Step, bool, error)
	StepDone(ctx context.Context, personID, stepID int64, at time.Time) error
	ClearSteps(ctx context.Context, personID int64) error
}
