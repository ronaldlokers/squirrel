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
type Options struct {
	// Path is where the screen lives. Sub-routes hang off it.
	Path string
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
	ItemByID(ctx context.Context, personID, itemID int64) (squirrel.Item, bool, error)
	SetItemState(ctx context.Context, itemID int64, state squirrel.ItemState, at time.Time) error
	PromoteItem(ctx context.Context, personID, itemID int64, every time.Duration) (squirrel.Chore, bool, error)

	// The chores half. A chore is not a note and shares none of the note
	// functions, but it is the other thing this pile holds and the screen was
	// the only surface that could not see it.
	ActiveChores(ctx context.Context, personID int64) ([]squirrel.Chore, error)
	UpsertChore(ctx context.Context, personID int64, name string, every, tolerance time.Duration) (squirrel.Chore, error)
	DeactivateChore(ctx context.Context, choreID int64) error
	RecordCompletion(ctx context.Context, choreID, personID int64, source string, at time.Time) error
}
