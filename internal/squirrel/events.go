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

// RetractCompletion undoes the chore's most recent live completion. The row
// stays; retracted_at is what the baseline query filters on, so the clock moves
// back without the log losing anything.
//
// Returns false when there was nothing live to retract. That is a no-op rather
// than an error: an un-tap with nothing behind it is a user changing their mind
// twice, not a failure.
func (s *Store) RetractCompletion(ctx context.Context, choreID, personID int64, at time.Time) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		update events set retracted_at = $3
		 where id = (
		   select e.id from events e
		     join chores c on c.id = e.chore_id
		    where e.chore_id = $1 and c.person_id = $2 and e.retracted_at is null
		    order by e.occurred_at desc, e.id desc
		    limit 1)`,
		choreID, personID, at)
	if err != nil {
		return false, fmt.Errorf("retracting completion: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}
