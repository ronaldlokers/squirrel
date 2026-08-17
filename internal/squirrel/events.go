package squirrel

import (
	"context"
	"fmt"
	"time"
)

// RecordCompletion is the entire mechanism by which a chore resets. There is no
// "last done" column: the clock is derived from max(events.occurred_at), which
// is why a sensor writing an event later needs no new code.
//
// The insert is a select so it can assert ownership in the same statement. A
// chore id reaches here through a person-scoped lookup today, so this is
// defence in depth — but a button value is client-supplied, and "unreachable"
// should not depend on remembering that.
func (s *Store) RecordCompletion(ctx context.Context, choreID, personID int64, source string, at time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		insert into events (chore_id, person_id, source, occurred_at)
		select c.id, $2, $3, $4 from chores c
		 where c.id = $1 and c.person_id = $2`,
		choreID, personID, source, at)
	if err != nil {
		return fmt.Errorf("recording completion: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("recording completion: chore %d is not %d's", choreID, personID)
	}
	return nil
}

// RetractCompletion undoes every live completion since a given prompt. Rows
// stay; retracted_at is what the baseline query filters on, so the clock
// moves back without the log losing anything.
//
// It retracts all matching rows in one statement, not just the most recent
// one. That is what makes it a state assertion rather than an increment:
// "selected: false" means "there is no live completion from this prompt",
// and the only way to make that idempotent is to describe the end state
// directly. RecordCompletion is not itself idempotent, so a retried "done"
// delivery can leave two live completions in one window — retracting only
// the newest of those would undo the older one on a second call, landing in
// a different place than the first call did. Retracting the whole window
// every time means a second call finds nothing left live in it, affects
// zero rows, and returns false: the same place as staying put.
//
// The prompt lookup is joined to person_id, not just id, so a prompt
// belonging to someone else cannot widen the window on the caller's own
// chore. Ownership of the chore is already enforced by the join to chores,
// so this is defence in depth today — but promptID will soon come from a
// client-supplied message id, and "unreachable" should not depend on
// remembering that the other predicate covers it.
//
// Returns false when there was nothing live to retract. That is a no-op
// rather than an error — either nothing was ever completed since this
// prompt, or a previous call already retracted the whole window.
func (s *Store) RetractCompletion(ctx context.Context, choreID, personID, promptID int64, at time.Time) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		update events e
		   set retracted_at = $4
		  from chores c, prompts p
		 where c.id = e.chore_id
		   and e.chore_id = $1
		   and c.person_id = $2
		   and p.id = $3
		   and p.person_id = $2
		   and e.retracted_at is null
		   and e.occurred_at >= p.sent_at`,
		choreID, personID, promptID, at)
	if err != nil {
		return false, fmt.Errorf("retracting completion: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}
