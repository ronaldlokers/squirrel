package squirrel

import (
	"context"
	"fmt"
	"time"
)

// The store's half of the coach's log. See migration 0020 for what is kept and
// why, and internal/coach for who reads it.
//
// This package does not import internal/coach and never will: the core must not
// depend on a model being reachable, which is the same rule that keeps it from
// importing internal/transport. Structural matching cannot bridge the two here,
// because the parameter is a struct and the two packages' structs are different
// types however identical their fields.
//
// So internal/boot adapts between them, which is what boot is for — it is
// already the only package that imports both a transport and the core.

// CoachAnswer is one recorded call. It mirrors coach.Answer field for field,
// and boot's adapter is what keeps the two honest.
type CoachAnswer struct {
	Kind       string
	Model      string
	Prompt     string
	Reply      string
	InTokens   int
	OutTokens  int
	CostMicros int64
	Used       bool
	At         time.Time
}

// RecordCoachAnswer stores one call, used or not.
func (s *Store) RecordCoachAnswer(ctx context.Context, personID int64, a CoachAnswer) error {
	at := a.At
	if at.IsZero() {
		at = time.Now()
	}
	if _, err := s.pool.Exec(ctx, `
		insert into coach_answers
		    (person_id, kind, model, prompt, reply, in_tokens, out_tokens, cost_micros, used, said_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		personID, a.Kind, a.Model, a.Prompt, a.Reply,
		a.InTokens, a.OutTokens, a.CostMicros, a.Used, at); err != nil {
		return fmt.Errorf("recording what the coach said: %w", err)
	}
	return nil
}

// CoachSpentSince is the sum of what the coach has cost since an instant, in
// micro-euros.
//
// A sum over the rows rather than a running counter, so there is nothing to
// drift: what was spent is what was recorded, and a row that failed to insert
// is a call that is not billed to the budget — which is the safe direction,
// because a call that failed to insert almost certainly failed to happen.
func (s *Store) CoachSpentSince(ctx context.Context, personID int64, since time.Time) (int64, error) {
	var spent int64
	if err := s.pool.QueryRow(ctx, `
		select coalesce(sum(cost_micros), 0) from coach_answers
		 where person_id = $1 and said_at >= $2`, personID, since).Scan(&spent); err != nil {
		return 0, fmt.Errorf("reading what the coach has cost: %w", err)
	}
	return spent, nil
}
