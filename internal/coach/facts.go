package coach

import "context"

// What the model is allowed to look at.
//
// Six things, every one capped, and the caps live here rather than in the
// prompt — a cap the model is asked to respect is a cap it can ignore. There
// is no way from here to the pile, and no way to a mood history: Clock hands
// back the derived capacity and nothing behind it.
//
// Declared as an interface for the usual reason: internal/squirrel must not be
// imported here. internal/boot implements it over the store, which is the same
// job it already does for the budget's log and the screen's window.

// Work is one thing that could be done. It is deliberately not squirrel.Item:
// the model has no business knowing about states, arrival order, or which
// transport a thought came in on.
type Work struct {
	ID   int64  `json:"id"`
	Kind string `json:"kind"` // "task" or "chore"
	Text string `json:"text"`
}

// Fixed is a thing the world imposed. The only one that matters is the next
// one, which is why nothing here returns a list of them.
type Fixed struct {
	Label string `json:"label"`
	At    string `json:"at"`
	// LeaveIn is minutes until you would have to walk out. The arithmetic is
	// the product's and is done before it gets here — handing the model the
	// parts and letting it subtract would be two answers to one question.
	LeaveIn int `json:"leave_in"`
	// Guessed says the travel time was never given, so anything said about
	// leaving has to admit it.
	Guessed bool `json:"guessed,omitempty"`
}

// Happened is one thing that was already done today.
//
// Done only, never refused: something turned down is already absent from
// OpenWork, and a list of refusals would be a record of what you keep saying no
// to. Today only, and never a total.
type Happened struct {
	What string `json:"what"`
}

// Facts is the read surface. Every method may fail, and every failure is
// handed to the model as an empty result rather than an error: a tool that
// cannot answer is a fact the model does not have, and the deterministic floor
// is underneath the whole call anyway.
type Facts interface {
	Clock(ctx context.Context, personID int64) (Now, error)
	OpenWork(ctx context.Context, personID int64, limit int) ([]Work, error)
	NextFixed(ctx context.Context, personID int64) (Fixed, bool, error)
	Lately(ctx context.Context, personID int64, limit int) ([]Happened, error)
	Item(ctx context.Context, personID, id int64) (Work, bool, error)
	// Typically is how long something usually takes, measured from timers that
	// reached their end, or false when there are too few runs to say.
	//
	// Migration 0017 refused a timer history in writing; 0022 narrows that
	// refusal rather than reversing it. Only runs that finished are recorded, so
	// there is no failure rate in the table and the median is a fact about the
	// bins rather than about you.
	Typically(ctx context.Context, personID int64, label string) (int, bool, error)
}

// The caps. Ten is the number the pile screen already uses for a page of
// notes, and it is here for the same reason: past ten, a list stops being
// something you read and becomes something you skim.
const (
	workCap   = 10
	latelyCap = 10
)

func capped(asked, most int) int {
	if asked < 1 || asked > most {
		return most
	}
	return asked
}
