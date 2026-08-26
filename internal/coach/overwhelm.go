package coach

import "strings"

// The overwhelm turn.
//
// Someone lists five things at once, and the listing is the overwhelm — so the
// answer must not reflect it back. One thing, one reason, and the rest is kept.
//
// The only routine turn that escalates to the expensive model: on an ordinary
// "what now" the difference between tiers is a nicer sentence, and here it is the
// difference between choosing well and reading a list back.

// Detected by rules rather than by a model: asking one before every turn doubles
// the calls to answer a question that costs nothing to answer badly.
//
// Conservative — three things before it counts — because escalating too eagerly
// costs money and escalating too rarely costs a nicer sentence.

// overwhelmParts is how many separate things have to be named. Three, not two:
// "the tax thing and the vet" is a sentence people write when they are fine.
const overwhelmParts = 3

// Overwhelmed reports whether what was typed is a pile rather than a question.
//
// It lives here rather than in either caller so the screen and the chat cannot
// disagree about what overwhelm is — the same reason Trim lives here.
func Overwhelmed(said string) bool {
	return len(listedIn(said)) >= overwhelmParts
}

// listedIn splits on separate lines, commas, and "and". Order matters: lines are
// checked first and alone, because someone who put each thing on its own line has
// already done the separating.
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

// listItemShortest and listItemLongest are what a listed thing looks like. The
// floor of three keeps "vet" and drops the fragment a trailing comma leaves.
//
// The ceiling earns its keep: "I have been trying to start this one thing all
// morning and I keep opening the page and closing it again" splits into three on
// "and" and is one thought told at length. Listed things are short, because
// listing them is what you do instead of thinking about each one.
const (
	listItemShortest = 3
	listItemLongest  = 40
)

// meaningful keeps the parts that look like listed things. Deliberately crude: a
// rule that judged whether something is a real task would be a model in disguise.
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

// overwhelmVoice is added to the preamble for this turn only. "Do not reflect it
// back" is the load-bearing line: acknowledging a list is reading it out, which
// hands back the thing that could not be held.
//
// The preference order is the product's: something at a fixed time first because
// the world imposed it, then something short because starting is the hard part,
// then something bodily when capacity is low.
const overwhelmVoice = `

The person has listed several things at once. That listing is the
overwhelm — do not reflect it back.

Choose ONE. Prefer, in order: something at a fixed time; something under
five minutes; something bodily (eating, washing, sleeping) when capacity
is low.

Say what to do and one reason. Then say the rest is kept.
Do not list the rest.`
