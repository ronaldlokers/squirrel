// Package transport holds adapters for chat systems. It imports
// internal/squirrel; the reverse is an import cycle and will not compile,
// which is what enforces the boundary between the core and any one chat
// system.
package transport

import (
	"context"
	"net/http"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Sink and Mount are declared here, not shared with the core. Go satisfies
// interfaces structurally, so squirrel's types fit these without either
// package importing the other's declaration.
type Sink interface {
	Accept(ctx context.Context, c squirrel.Capture) squirrel.Outcome
}

type Mount interface {
	Post(pattern string, h http.HandlerFunc)
}

// Transport is a struct of funcs rather than an interface, because Send has to
// be nil-able and Go cannot express a nil method. A transport that cannot
// initiate a conversation says so in its type rather than failing at the
// moment it is needed.
type Transport struct {
	Name string
	// Start begins receiving and returns a stop func. Mount may be ignored by
	// a transport that polls.
	Start func(ctx context.Context, sink Sink, mount Mount) (func(context.Context) error, error)
	// Send is nil when this transport cannot initiate.
	Send func(ctx context.Context, conversationID, text string) error
}
