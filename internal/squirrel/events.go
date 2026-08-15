package squirrel

import (
	"context"
	"fmt"
	"time"
)

// RecordCompletion writes the event that resets a chore's clock. Nothing
// updates the chore row: the clock is derived, so a sensor writing here later
// works without touching this file.
func (s *Store) RecordCompletion(ctx context.Context, choreID, personID int64, source string, at time.Time) error {
	_, err := s.pool.Exec(ctx, `
		insert into events (chore_id, person_id, source, occurred_at)
		values ($1, $2, $3, $4)`, choreID, personID, source, at)
	if err != nil {
		return fmt.Errorf("recording completion: %w", err)
	}
	return nil
}
