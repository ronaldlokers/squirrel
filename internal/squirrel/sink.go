package squirrel

import (
	"context"
	"fmt"
)

// SpoolWriter is the only thing the sink needs from a spool.
type SpoolWriter interface {
	Write(Capture) (string, error)
}

type SinkHooks struct {
	OnError func(error)
	// OnIgnored fires for a capture that is silent in the room. It should not
	// be silent in the logs.
	OnIgnored func(Capture)
}

type sink struct {
	writer SpoolWriter
	allows []Allow
	hooks  SinkHooks
}

// NewSink is the only place a capture becomes durable.
//
// Note what is absent: Postgres. The request path reaches Accept and no
// further, which is what lets a database outage pass without a lost thought.
func NewSink(w SpoolWriter, allows []Allow, hooks SinkHooks) Sink {
	return &sink{writer: w, allows: allows, hooks: hooks}
}

func (s *sink) Accept(_ context.Context, c Capture) Outcome {
	if Decide(c, s.allows) == Ignore {
		safely(func() {
			if s.hooks.OnIgnored != nil {
				s.hooks.OnIgnored(c)
			}
		})
		return Ignored
	}

	if err := s.write(c); err != nil {
		safely(func() {
			if s.hooks.OnError != nil {
				s.hooks.OnError(err)
			}
		})
		return Failed
	}
	return Stored
}

// write converts a panicking writer into an error, so Accept always returns an
// Outcome and the HTTP handler above it can never emit a 500 — which Campfire
// would upload into the room as a file.
func (s *sink) write(c Capture) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("spool write panicked: %v", r)
		}
	}()
	_, err = s.writer.Write(c)
	return err
}

// safely runs an observability hook. A hook failure must never reach the
// caller: a throwing logger turning a stored capture into a failed reply would
// make the person resend, and the resend carries a new message id that the
// dedupe index cannot catch.
func safely(f func()) {
	defer func() { _ = recover() }()
	f()
}
