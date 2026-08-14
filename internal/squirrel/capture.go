package squirrel

import (
	"context"
	"encoding/json"
	"time"
)

// Capture is what every transport must produce. Nothing here is
// Campfire-shaped.
//
// The pointer fields are pointers for one reason: a transport that cannot
// parse an envelope must still be able to hand the message over, and "" is a
// real value that is not the same as unknown. See the fail-open rule in
// policy.go.
type Capture struct {
	Transport      string  `json:"transport"`
	ExternalID     *string `json:"externalId"`
	ConversationID *string `json:"conversationId"`
	SenderID       *string `json:"senderId"`
	// Text is verbatim. Never trimmed, lowercased, or otherwise interpreted.
	Text string `json:"text"`
	// ReceivedAt is our clock. Campfire sends no timestamp and the next
	// transport may not either.
	ReceivedAt time.Time `json:"receivedAt"`
	// Payload is the original message bytes, untouched and never
	// re-marshalled, for anything not worth a column.
	Payload json.RawMessage `json:"payload"`
}

type Outcome string

const (
	Stored  Outcome = "stored"
	Ignored Outcome = "ignored"
	Failed  Outcome = "failed"
)

// Sink returns an Outcome and deliberately no error: every failure is already
// an Outcome, and an error return would tempt a caller to handle what is
// handled.
type Sink interface {
	// Accept returns only once the capture is durable.
	Accept(ctx context.Context, c Capture) Outcome
}

// Ptr is a convenience for the optional identifier fields.
func Ptr[T any](v T) *T { return &v }
