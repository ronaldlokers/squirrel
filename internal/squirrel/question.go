package squirrel

import "strings"

// Does this read as a question asked of Buddy?
//
// The floor under the whole reading path: no model, no network, no cluster.
// It answers when there is no house model running and no key configured, which
// is the configuration this product shipped with and still supports.
//
// It is deliberately narrow. Match's own rule applies here word for word —
// "when in doubt the answer is always capture" — because the two failures are
// not the same size. A question read as a thought is a note in the pile you
// could have had answered, and there is a chip on the answer to fix it. A
// thought read as a question is a thought dropped out of the pile, which is
// the one failure this product does not have.
//
// So it says yes only when the sentence is doing nothing else.

// asking are the openings that make a sentence a question without a question
// mark. Every one of them is a thing you say to somebody rather than about
// something — which is the distinction, and it is why "what a day" is not on
// the list and "what should" is.
var asking = []string{
	"what should", "what do you", "what would you", "what can i", "what do i",
	"why do i", "why does", "why is it", "why can't i", "why cant i",
	"how do i", "how does", "how can i", "how would i", "how long",
	"when should", "where do i", "which should",
	"should i", "shall i", "can you", "could you", "will you", "would you",
	"do you know", "any idea", "any thoughts", "help me with",
	"remind me how", "tell me how",
}

// showing are the ways somebody asks to be shown one of their places without
// asking a question at all.
//
// "show chores" has no question mark and no asking opening, so it read as a
// thought and was filed as one — twice, in the same minute, by somebody trying
// to see their chores. The note that came back said "Kept.", which is the
// product answering a request to look at something by writing it down.
//
// Narrow on purpose, and narrower than the openings above: this fires only
// when a verb of showing is followed by the name of a place that exists. "show
// me the chores" is a request; "show mum the photos" is a thought, and stays
// one.
var showing = []string{"show", "open", "list", "see", "view"}

// somewhere are the words that name a place, as somebody would say them rather
// than as the routes spell them.
//
// It is deliberately not doorNames: this is a rule about English, and the
// screen's vocabulary is a rule about routes. They agree today and they are
// allowed to stop agreeing — a place renamed on screen should not silently
// change what sentence reaches Buddy.
var somewhere = []string{
	"pile", "task", "tasks", "chore", "chores", "agenda", "diary",
	"kept", "set aside", "aside", "held", "calendar",
}

// asksToBeShown reports a request to look at one of the places.
func asksToBeShown(t string) bool {
	verb := false
	for _, opening := range showing {
		if strings.HasPrefix(t, opening+" ") {
			verb = true
			break
		}
	}
	if !verb {
		return false
	}
	for _, place := range somewhere {
		// Bounded, so "taskmaster" is not "task": the place has to be a word
		// of its own or the end of the sentence.
		if strings.Contains(t, " "+place+" ") || strings.HasSuffix(t, " "+place) {
			return true
		}
	}
	return false
}

// LooksLikeAQuestion is the rule.
func LooksLikeAQuestion(text string) bool {
	t := strings.ToLower(strings.TrimSpace(text))
	if t == "" {
		return false
	}

	// The escape hatch wins over everything, including this.
	//
	// A leading dot means "keep exactly this", and Match honours it by
	// returning a capture — which is why asking Match alone is not enough
	// here: a dotted question is a capture *and* ends in a question mark, so
	// it fell through and was sent to be answered. Found by the test that says
	// so, which is the only reason this line exists.
	if strings.HasPrefix(t, ".") {
		return false
	}

	// A command is never a question. The grammar has already claimed these and
	// answering them with a model would be answering something the product
	// knows how to do itself.
	if Match(text).Kind != IntentCapture {
		return false
	}

	// A question mark at the end, which is how most people ask one.
	//
	// At the end and not anywhere: "ring the vet? no, the dentist" is somebody
	// thinking on the page, and the mark in the middle of it is punctuation
	// rather than a request.
	if strings.HasSuffix(t, "?") {
		return true
	}

	for _, opening := range asking {
		if strings.HasPrefix(t, opening) {
			return true
		}
	}

	// And a request to be shown one of the places, which is asked of Buddy
	// just as surely as a question is, and was being filed as a note.
	return asksToBeShown(t)
}
