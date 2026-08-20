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
	// Owner answers whose pile this is. There is one person and SeedOwner
	// reconciles them once, so this is not a per-request lookup — it is a
	// function only because of when it can be answered. Routes must be
	// registered before the server listens, and the owner's id is only known
	// once Postgres has answered, which may be a while after boot or never;
	// internal/boot's nudgeRelay exists for exactly that window. A zero id
	// means "not yet", and the screen says the same thing it says for any
	// other unreachable database.
	Owner func() int64
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
}
