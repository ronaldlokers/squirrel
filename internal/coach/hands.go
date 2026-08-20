package coach

import "context"

// What the coach may do, and what it may only ask for.
//
// The policy is not invented here. It is read off rules the product already
// has, and the test is one question: **is this already a button the person
// could have pressed, and does one press undo it?**
//
// If both are true, the model may do it. Nothing is being given away — the
// action existed, the undo existed, and the only thing that changed is which
// finger pressed it. If either is false, the model may only propose, and the
// proposal renders as the control that already exists rather than as a dialog.

// Hands is the write surface. Six things, all of them reversible in one press.
//
// Declared here and implemented in internal/boot, like Facts and for the same
// reason. Every method may fail, and a failure is reported to the model as
// such — unlike a read, where a failure is silence: a model told nothing
// happened must not tell someone it did.
type Hands interface {
	// Complete marks a task done. Reverses through the `open` transition,
	// which is one press on the card.
	Complete(ctx context.Context, personID, itemID int64) (string, error)
	// CompleteChore records a completion. Reverses through retraction —
	// events are never deleted in this product.
	CompleteChore(ctx context.Context, personID, choreID int64) (string, error)
	// StartTimer starts the body double. Reverses through the stop control,
	// which is in the lid on every screen.
	StartTimer(ctx context.Context, personID int64, label string, minutes int) error
	// Refuse turns something down for today. Reverses through UnrefuseToday.
	Refuse(ctx context.Context, personID int64, kind string, refID int64) error
	// SnoozeChore stops a chore being raised for a while. Reverses by saying
	// it again; the chore's own baseline never moved.
	SnoozeChore(ctx context.Context, personID, choreID int64, hours int) (string, error)
	// CreateTask records something decided outright. Reverses through untask,
	// then drop — two presses, but the first one is the whole of the undo and
	// the second is ordinary triage.
	CreateTask(ctx context.Context, personID int64, text string) error
}

// Proposal is a thing the coach wants to do and may not do on its own.
//
// Four kinds, and each one fails the test above in a way worth being explicit
// about: a moment will interrupt you later, a chore comes back forever, a
// retirement stops something recurring, and a drop is disposal. None of them
// is undone by one press on a control that is already on screen.
//
// It carries no state and is stored nowhere. The caller renders it and it
// travels in the form that renders it, exactly as a split does — so an
// unanswered proposal lasts as long as the page it is on, and there is nothing
// pending anywhere to expire.
type Proposal struct {
	// Do is which of the four: "moment", "chore", "retire", "drop".
	Do string
	// Said is the proposal in the coach's own words, one sentence, already
	// through the guard. It is what the person reads before pressing.
	Said string
	// Text is the label or name for a moment or a chore.
	Text string
	// At is when a moment starts, in the words ParseMoment already reads —
	// "14:30", "tomorrow 09:00". Not parsed here: extracting a time is
	// deterministic work the core already does, and a guessed time is a missed
	// appointment.
	At string
	// Every is a chore's rhythm, in the words the core already parses.
	Every string
	// RefID is the row a retirement or a drop names.
	RefID int64
}

// proposalKinds is the vocabulary, as a map rather than a switch so an unknown
// kind is a lookup miss rather than a default branch someone later fills in
// with something destructive. The same device the pile's actions use.
var proposalKinds = map[string]bool{
	"moment": true,
	"chore":  true,
	"retire": true,
	"drop":   true,
}

// KnownProposal reports whether this is one of the four. Callers check it
// before rendering and again before applying, because a proposal arrives back
// through a form and a form is a place a stranger can type.
func KnownProposal(do string) bool { return proposalKinds[do] }
