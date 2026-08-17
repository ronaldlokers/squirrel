package squirrel

import (
	"regexp"
	"strconv"
)

// ActionIntent is a tap, resolved as far as text alone can take it. The
// position still has to be looked up against a person-scoped prompt before it
// means a chore.
type ActionIntent struct {
	MessageID string
	// Kind is "done" or "undefine".
	Kind     string
	Position int
	// Selected is the resulting state Campfire reports, not a delta. false on a
	// "done" button means the completion is being taken back.
	Selected bool
}

// Anchored at both ends, like every other matcher in this system: a message
// that merely contains this shape is a thought about it, not a tap.
var actionPattern = regexp.MustCompile(`^!action (\d+) (done|undefine):(\d{1,3}) (true|false)$`)

// ParseAction recognises the encoding CaptureFrom writes for an action webhook.
// It is deliberately not part of Match: an action is not something a person can
// type into the room, and putting it in the intent matcher would make it one.
func ParseAction(text string) (ActionIntent, bool) {
	m := actionPattern.FindStringSubmatch(text)
	if m == nil {
		return ActionIntent{}, false
	}
	position, err := strconv.Atoi(m[3])
	if err != nil || position < 1 {
		return ActionIntent{}, false
	}
	return ActionIntent{
		MessageID: m[1],
		Kind:      m[2],
		Position:  position,
		Selected:  m[4] == "true",
	}, true
}
