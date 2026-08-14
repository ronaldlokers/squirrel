package squirrel

type Allow struct {
	Transport      string
	ConversationID string
	SenderID       string
}

type Verdict string

const (
	Accept Verdict = "accept"
	Ignore Verdict = "ignore"
)

// Decide is the guard, and it fails in two directions on purpose.
//
// Understood the envelope and it is the wrong room or the wrong person: fail
// closed, because the system genuinely was not addressed.
//
// Could not understand the envelope at all: fail open. A payload shape change
// upstream would otherwise drop every capture silently, with the bot still
// answering cheerfully. Junk rows in a table nobody reads yet is the cheaper
// mistake by a wide margin.
func Decide(c Capture, allows []Allow) Verdict {
	if c.ConversationID == nil || c.SenderID == nil {
		return Accept
	}
	for _, a := range allows {
		if a.Transport == c.Transport &&
			a.ConversationID == *c.ConversationID &&
			a.SenderID == *c.SenderID {
			return Accept
		}
	}
	return Ignore
}
