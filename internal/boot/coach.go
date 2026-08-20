package boot

import (
	"context"
	"log/slog"
	"time"

	"github.com/ronaldlokers/squirrel/internal/coach"
	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Where the coach is joined to the store.
//
// internal/coach must not import internal/squirrel — the core would then depend
// on a model being reachable, which is the whole thing this architecture
// refuses. internal/squirrel must not import internal/coach either, for the
// same reason read the other way.
//
// Elsewhere that separation is free: internal/web declares the narrow interface
// it needs and *squirrel.Store satisfies it structurally, because every
// parameter is either a primitive or a squirrel type. The budget's log cannot
// work that way. Its parameter is a struct, and coach.Answer and
// squirrel.CoachAnswer are different types however identically they are
// written, so no structural match is possible in either direction.
//
// So the conversion is written down, here, in the package that already exists
// to join things that must not know about each other. Twelve lines of copying
// is a smaller price than either package importing the other, and it is the
// only place the two structs have to agree.

// coachLog adapts the store to coach.Log.
type coachLog struct{ store *squirrel.Store }

func (l coachLog) RecordCoachAnswer(ctx context.Context, personID int64, a coach.Answer) error {
	return l.store.RecordCoachAnswer(ctx, personID, squirrel.CoachAnswer{
		Kind:       a.Kind,
		Model:      a.Model,
		Prompt:     a.Prompt,
		Reply:      a.Reply,
		InTokens:   a.InTokens,
		OutTokens:  a.OutTokens,
		CostMicros: a.CostMicros,
		Used:       a.Used,
		At:         a.At,
	})
}

func (l coachLog) CoachSpentSince(ctx context.Context, personID int64, since time.Time) (int64, error) {
	return l.store.CoachSpentSince(ctx, personID, since)
}

// budgetFor is the monthly ceiling, wired to the log that answers it.
func budgetFor(cfg squirrel.CoachConfig, store *squirrel.Store) coach.Budget {
	return coach.Budget{Log: coachLog{store: store}, CeilingMicros: cfg.BudgetMicros}
}

// coachFor builds the coach, or NoCoach.
//
// NoCoach is a shipping configuration, not a failure: with no key the picker
// still chooses, the ladder still answers, and every screen still works. So the
// absence of a key is logged at info — a warning would be telling someone off
// for a choice they made.
//
// A model the price table does not know is different, and does warrant a
// warning. It means the budget will price every call at zero and the monthly
// ceiling silently stops existing, which is a thing to hear about at start
// rather than discover on an invoice.
func coachFor(cfg squirrel.CoachConfig) coach.Coach {
	if !cfg.Enabled() {
		slog.Info("no coach configured; the picker and the ladder answer alone")
		return coach.NoCoach{}
	}
	for _, model := range []string{cfg.Fast, cfg.Deep} {
		if !coach.KnownModel(model) {
			slog.Warn("no price known for this model; it will count as free against the budget",
				"model", model)
		}
	}
	// Phase A ships the skeleton and no provider. A key that is configured
	// before there is anything to call it with must still leave the product
	// working, so this is NoCoach until phase B replaces it.
	slog.Info("coach configured; no provider built yet", "fast", cfg.Fast, "deep", cfg.Deep)
	return coach.NoCoach{}
}
