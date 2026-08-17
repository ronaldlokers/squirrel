package squirrel

import "context"

// Action is one button on a message.
type Action struct {
	Label string
	// Value is what comes back when it is tapped. It carries a position and
	// never a chore id: a value is client-supplied, and a position only means
	// anything against a prompt we look up scoped by person.
	Value string
	Emoji string
}

// Message is what the system says when it speaks first. A transport renders it;
// nothing here is Campfire-shaped.
type Message struct {
	Text    string
	Actions []Action
	// SelectionMode is "", "single" or "multiple". Empty means the transport's
	// default, which is no retained selection.
	SelectionMode string
}

// MaxActions is Campfire's limit. A message carrying more is rejected outright,
// so the digest would not be sent at all.
const MaxActions = 12

// Capped drops the surplus buttons and leaves the text alone. Every chore keeps
// its number in the rendered list, so the ones past twelve are still reachable
// by typing — losing a button is a degraded surface, losing the whole digest is
// silence. Nothing is said about the cut-off: a line about what could not be
// rendered is noise.
func (m Message) Capped() Message {
	if len(m.Actions) <= MaxActions {
		return m
	}
	m.Actions = m.Actions[:MaxActions]
	return m
}

// Chat is everything the system needs to speak. Func fields rather than an
// interface so internal/boot can assemble it from a transport while this
// package still imports nothing from internal/transport.
//
// Send returns the transport's own id for the message it created, which is what
// lets a later prompt disable this one's buttons.
type Chat struct {
	Send   func(ctx context.Context, conversationID string, m Message) (messageID string, err error)
	Update func(ctx context.Context, conversationID, messageID string, m Message) error
	Boost  func(ctx context.Context, conversationID, messageID, content string) error
}
