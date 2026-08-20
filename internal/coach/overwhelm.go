package coach

import "strings"

// The overwhelm turn.
//
// Someone lists five things at once. The listing *is* the overwhelm — it is
// what being unable to choose looks like when it is typed out — so the answer
// must not reflect it back. One thing, one reason, and the rest is kept.
//
// This is the turn the whole project exists for, and it is the only routine
// one that escalates to the expensive model. That is a judgement about where
// judgement is worth paying for: on an ordinary "what now" the difference
// between the two tiers is a nicer sentence, and here it is the difference
// between choosing well and reading a list back.

// Detected by rules rather than by a model, and the reason is arithmetic. A
// model asked "is this overwhelm?" before every turn doubles the calls to
// answer a question that costs nothing to answer badly: a false positive
// spends the deep model on a turn that did not need it, and a false negative
// gives an ordinary answer to something that wanted a better one. Neither is
// a failure worth a round trip.
//
// The rules are deliberately conservative — three things before it counts —
// because escalating too eagerly is the failure that costs money, and
// escalating too rarely is the failure that costs a nicer sentence.

// overwhelmParts is how many separate things have to be named. Three, not two:
// "the tax thing and the vet" is a sentence people write when they are fine.
const overwhelmParts = 3

// Overwhelmed reports whether what was typed is a pile rather than a question.
//
// It lives here rather than in either caller so the sheet and the chat cannot
// disagree about what overwhelm is — the same reason Trim lives here.
func Overwhelmed(said string) bool {
	return len(listedIn(said)) >= overwhelmParts
}

// listedIn splits on the three ways people actually write a list: separate
// lines, commas, and "and".
//
// Order matters. Lines are checked first and alone: someone who has put each
// thing on its own line has already done the separating, and splitting those
// lines further on their commas would turn one long sentence into four things
// that were never four things.
func listedIn(said string) []string {
	t := strings.TrimSpace(said)
	if t == "" {
		return nil
	}

	if lines := meaningful(strings.Split(t, "\n")); len(lines) >= overwhelmParts {
		return lines
	}

	// Commas next, then "and" within what is left. Both in one pass, because
	// "the bins, the vet and the tax thing" uses both and is one list.
	parts := make([]string, 0, 6)
	for _, chunk := range strings.Split(t, ",") {
		parts = append(parts, splitOnAnd(chunk)...)
	}
	return meaningful(parts)
}

// splitOnAnd separates on the word, never on the letters. "and" inside
// "standing" is not a list, and " and " with spaces on both sides is what
// makes the difference.
func splitOnAnd(chunk string) []string {
	out := []string{}
	rest := chunk
	for {
		lower := strings.ToLower(rest)
		i := strings.Index(lower, " and ")
		if i < 0 {
			return append(out, rest)
		}
		out = append(out, rest[:i])
		rest = rest[i+len(" and "):]
	}
}

// listItemShortest and listItemLongest are what a listed thing looks like.
//
// The floor is three characters, which keeps "vet" and drops the fragment a
// trailing comma leaves behind.
//
// The ceiling is the one that earns its keep, and it was added because of a
// sentence that got this wrong: "I have been trying to start this one thing
// all morning and I keep opening the page and closing it again without reading
// any of it" splits into three on "and" and is the opposite of a list — it is
// one thought, told at length. Things people list are short, because listing
// them is what you do instead of thinking about each one.
const (
	listItemShortest = 3
	listItemLongest  = 40
)

// meaningful keeps the parts that look like listed things.
//
// Deliberately crude: a rule that tried to judge whether something is a real
// task would be a model in disguise, and a wrong answer here costs one tier of
// one call in either direction.
func meaningful(parts []string) []string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if n := len([]rune(t)); n >= listItemShortest && n <= listItemLongest {
			out = append(out, t)
		}
	}
	return out
}

// KindOverwhelm is what the log and the prompt call this turn.
const KindOverwhelm = "overwhelm"

// overwhelmVoice is added to the preamble for this turn only.
//
// "Do not reflect it back" is the load-bearing line. Every instinct a chat
// model has says to acknowledge what it was told, and acknowledging a list is
// reading the list out — which hands back exactly the thing that could not be
// held in the first place.
//
// The preference order is the product's, not the model's: something at a fixed
// time first because the world imposed it and it will pass; then something
// short, because starting is the hard part and a short thing is the cheapest
// way to have started; then something bodily when capacity is low, because on
// a bad day eating is worth more than any task on the list.
const overwhelmVoice = `

The person has listed several things at once. That listing is the
overwhelm — do not reflect it back.

Choose ONE. Prefer, in order: something at a fixed time; something under
five minutes; something bodily (eating, washing, sleeping) when capacity
is low.

Say what to do and one reason. Then say the rest is kept.
Do not list the rest.`
