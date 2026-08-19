package squirrel

import "fmt"

// How a chore is spoken about, in one place, because there are two views of it
// and they must agree.
//
// The screen said "a while back" while the chat said "20 days, usually 14" —
// the same chore, the same moment, two different claims, and the harsher one
// was the one that arrives unasked. That is not a wording slip: the reason the
// screen stopped printing the number is a reason about the product, and a rule
// that only one surface follows is a rule that will drift back.
//
// So the words live here, below both transports. internal/web imports this
// package and the reverse would be an import cycle, which is what makes the
// agreement structural rather than a promise.

// SinceWords says roughly when a chore was last done, and never how long ago.
//
// A number attached to something undone goes up while you are not looking, and
// that accumulating shape is the one this product exists without. Precision
// buys nothing: nobody waters a plant differently for knowing it was nine days
// rather than eight.
//
// The buckets are deliberately coarse and stop at "a while back". There is no
// bucket for a long time, because that sentence is about the person.
func SinceWords(sinceDays int) string {
	switch {
	case sinceDays <= 0:
		return "today"
	case sinceDays == 1:
		return "yesterday"
	case sinceDays < 7:
		return "this week"
	case sinceDays < 14:
		return "last week"
	case sinceDays < 31:
		return "this month"
	default:
		return "a while back"
	}
}

// Cadence says the rhythm the way a person would. "every 14 days" is
// arithmetic the reader has to do; "every 2 weeks" is the thing they set.
//
// Lowercase, because this is the sentence. The screen upper-cases it for its
// own meta role, which is a matter of style rather than of vocabulary.
func Cadence(days int) string {
	switch days {
	case 1:
		return "every day"
	case 7:
		return "every week"
	case 14:
		return "every 2 weeks"
	case 30:
		return "every month"
	default:
		return "every " + plural(days, "day")
	}
}

// ChoreWords is a whole chore in a sentence: what it is, when it was last
// done, and how often it comes back. Never how late it is, because it is never
// late — it has a rhythm, not a deadline.
//
// A chore nobody has ever done says only its rhythm. Its baseline is its own
// birthday, and reporting that as "last done" would be a sentence about the
// person rather than about the chore.
func ChoreWords(c Chore) string {
	if !c.EverDone {
		return fmt.Sprintf("%s — %s", c.Name, Cadence(c.EveryDays))
	}
	return fmt.Sprintf("%s — %s, %s", c.Name, SinceWords(c.SinceDays), Cadence(c.EveryDays))
}
