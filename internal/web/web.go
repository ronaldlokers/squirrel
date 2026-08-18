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
	// PersonID is the owner. There is one person, resolved once at boot,
	// because SeedOwner already reconciles that identity every boot and a
	// second lookup per request would be a second source of truth.
	PersonID int64
}

// Store is the narrow surface the screen consumes. Declared here rather than
// imported: Go satisfies interfaces structurally, so *squirrel.Store fits this
// without either package importing the other's declaration, the same way
// transport.Sink does.
type Store interface {
	OpenItems(ctx context.Context, personID int64, limit int) ([]squirrel.Item, bool, error)
	SearchItems(ctx context.Context, personID int64, query string, limit int) ([]squirrel.Item, bool, error)
	ItemByID(ctx context.Context, personID, itemID int64) (squirrel.Item, bool, error)
	SetItemState(ctx context.Context, itemID int64, state squirrel.ItemState, at time.Time) error
	PromoteItem(ctx context.Context, personID, itemID int64, every time.Duration) (squirrel.Chore, bool, error)
}
