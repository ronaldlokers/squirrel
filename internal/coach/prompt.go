package coach

import (
	"fmt"
	"strings"
)

// What the model is told, and nothing else.
//
// The preamble is short because its job is narrow, and because a short prefix
// is a cheap cached prefix. Every line in it earns its place:
//
//   - "one manageable thing at a time" is the product in a sentence.
//   - The ban on plans is the failure mode this whole thing exists to avoid.
//     It is said here as well as enforced in the guard, because a model that
//     is asked not to is cheaper than a model whose answer gets thrown away.
//   - "should", "just" and "simply" are the three words that turn help into a
//     reprimand. Unlike the plan ban, nothing enforces this — it is a matter
//     of tone, and a shape guard cannot judge tone.
//   - The last line matters most. A model allowed to decline produces silence,
//     and silence is the deterministic answer taking over, which is the floor
//     everything here stands on.
const preamble = `You are Buddy. You help one person with ADHD by handing them one
manageable thing at a time.

Never produce a plan, a checklist, or numbered steps in what you say.
One thing. Two sentences at most. Plain words.
Never say "should", "just", or "simply".
If you cannot answer usefully, say nothing rather than something generic.`

// lowVoice is appended when capacity is low.
//
// Worth knowing what it implies: a plainer voice is itself legible. When
// Squirrel goes flat, that is a visible sign it noticed something. That is
// allowed — Principle 5 was opened on 20 August 2026 — but it is a signal
// rather than a neutral setting, and it will be read as one.
const lowVoice = `

When capacity is "low", drop warmth and character. Shorter sentences,
plainer words, no turns of phrase. Say the thing and stop.`

// System is the preamble plus the parts that vary with the turn.
//
// Both additions are appended rather than substituted, and they compose: an
// overwhelm turn on a low day gets the ordering rule and the plainer voice,
// which is exactly the turn where both matter most.
func System(n Now, kind string) string {
	var b strings.Builder
	b.WriteString(preamble)
	if kind == KindOverwhelm {
		b.WriteString(overwhelmVoice)
	}
	if n.Capacity == "low" {
		b.WriteString(lowVoice)
	}
	return b.String()
}

// Context is the state of now, as one short line.
//
// A line rather than a block, because it is prepended to what the person said
// and a block would read as a document being handed over. Roughly thirty
// tokens, which is the whole of what the model is told about the day when no
// tool has been called.
//
// What is deliberately absent: any mood word, any history, any count. Capacity
// is already derived to "ok" or "low" before it reaches here — the model gets a
// signal, never a diagnosis.
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
