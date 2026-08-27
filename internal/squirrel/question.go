package squirrel

import "strings"

// Does this read as a question asked of Buddy?
//
// The floor under the whole reading path: no model, no network, no cluster.
//
// Deliberately narrow, because the two failures are not the same size. A question
// read as a thought is a note in the pile with a chip on the answer to fix it. A
// thought read as a question is a thought dropped out of the pile.

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

// showing are the ways somebody asks to be shown a place without asking a
// question. "show chores" has no question mark and no asking opening, so it was
// filed as a thought — twice in one minute, by somebody trying to see their
// chores.
//
// It fires only when a verb of showing is followed by a place that exists: "show
// mum the photos" is a thought and stays one.
var showing = []string{"show", "open", "list", "see", "view"}

// somewhere are the words that name a place as somebody would say them.
// Deliberately not doorNames: this is a rule about English and that is a rule
// about routes, and they are allowed to stop agreeing.
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

	// The escape hatch wins over everything. A leading dot means "keep exactly this",
	// and Match honours it by returning a capture — so asking Match alone is not
	// enough: a dotted question is a capture and ends in a question mark.
	if strings.HasPrefix(t, ".") {
		return false
	}

	// A command is never a question. The grammar has already claimed these and
	// answering them with a model would be answering something the product
	// knows how to do itself.
	if Match(text).Kind != IntentCapture {
		return false
	}

	// A question mark at the end and not anywhere: "ring the vet? no, the dentist" is
	// somebody thinking on the page.
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
