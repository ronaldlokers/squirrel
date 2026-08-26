package coach

import (
	"fmt"
	"strings"
)

// What the model is told, and nothing else. Short because its job is narrow, and
// because a short prefix is a cheap cached prefix.
//
//   - "one manageable thing at a time" is the product in a sentence.
//   - The ban on plans is said here as well as enforced in the guard: a model
//     asked not to is cheaper than one whose answer gets thrown away.
//   - "should", "just" and "simply" turn help into a reprimand. Nothing enforces
//     this — a shape guard cannot judge tone.
//   - A model allowed to decline produces silence, and silence is the
//     deterministic answer taking over.
const preamble = `You are Buddy. You help one person with ADHD by handing them one
manageable thing at a time.

Never produce a plan, a checklist, or numbered steps in what you say.
One thing. Two sentences at most. Plain words.
Never say "should", "just", or "simply".
If you cannot answer usefully, say nothing rather than something generic.`

// lowVoice is appended when capacity is low. A plainer voice is itself legible:
// when Squirrel goes flat, that is a visible sign it noticed something. Allowed,
// but it is a signal rather than a neutral setting and will be read as one.
const lowVoice = `

When capacity is "low", drop warmth and character. Shorter sentences,
plainer words, no turns of phrase. Say the thing and stop.`

// System is the preamble plus what varies with the turn. Both additions are
// appended rather than substituted and they compose: an overwhelm turn on a low
// day gets both.
func System(n Now, kind string) string {
	var b strings.Builder
	b.WriteString(preamble)
	if kind == KindOverwhelm {
		b.WriteString(overwhelmVoice)
	}
	if n.Capacity == "low" {
		b.WriteString(lowVoice)
	}
	b.WriteString(badlyLanded(n.LandedBadly))
	b.WriteString(knowsYou(n.Knowing))
	return b.String()
}

// badlyLanded shows the model what has not landed here, in its own words. An
// instruction nobody can check is a wish; these are the actual sentences.
//
//   - Examples, never a count.
//   - What, not who: lines that did not land, not a record of bad nights.
//   - Silence when there is nothing, rather than "nothing has landed badly",
//     which would invite the model to congratulate itself.
func badlyLanded(said []string) string {
	if len(said) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\nThese answers of yours did not land well with this person. " +
		"Do not repeat their shape or their tone:\n")
	for _, one := range said {
		b.WriteString("- " + strings.TrimSpace(one) + "\n")
	}
	return b.String()
}

// Context is the state of now, as one line rather than a block: a block reads as
// a document being handed over. Roughly thirty tokens.
//
// Deliberately absent: any mood word, any history, any count. Capacity is derived
// to "ok" or "low" before it reaches here.
func Context(n Now) string {
	parts := make([]string, 0, 4)
	if n.Clock != "" {
		parts = append(parts, "It is "+n.Clock)
	}
	if n.PartOfDay != "" {
		parts = append(parts, "in the "+n.PartOfDay)
	}
	if n.Capacity != "" {
		parts = append(parts, "capacity is "+n.Capacity)
	}
	if n.FreeUntil != nil {
		// Minutes rather than a clock time, because "in 25 minutes" is a
		// quantity you can fit something into and "at 15:05" is a fact you
		// then have to do arithmetic on.
		parts = append(parts, fmt.Sprintf("the next fixed thing is in %d minutes", *n.FreeUntil))
	}
	// Nil FreeUntil says nothing at all rather than "nothing is coming". Those
	// are different: one means the day is open, the other means nothing was
	// ever typed in, and the model must not read the second as the first.
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ") + "."
}
