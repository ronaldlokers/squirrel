package squirrel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Where you got to.
//
// See migrations/0030_runs.sql for why there is one row per person and no
// count in it.

// Run is a triage run somebody is part way through.
type Run struct {
	Place string
	// Since is how long ago it was last touched, at the moment it was read.
	Since time.Duration
}

// RunPile is the only place that loops today.
const RunPile = "pile"

// KeepingPlace is how long a run stays worth offering.
//
// Three hours, and the number is the design rather than a tuning knob. Coming
// back to a half-finished pile twenty minutes later is resuming; coming back
// to yesterday's is being nagged about an afternoon you have already had. If
// this is ever changed it should get shorter.
const KeepingPlace = 3 * time.Hour

// MarkRun says you are part way through something.
//
// Called on every answer rather than once at the start, so the clock measures
// silence rather than duration: a run you are actively working is never stale,
// and one you walked away from ages out whether or not you finished it.
func (s *Store) MarkRun(ctx context.Context, personID int64, place string, at time.Time) error {
	_, err := s.pool.Exec(ctx, `
		insert into runs (person_id, place, started_at, seen_at)
		values ($1, $2, $3, $3)
		on conflict (person_id) do update
		   set place = excluded.place,
		       seen_at = excluded.seen_at,
		       -- A run in a different place is a different run.
		       started_at = case when runs.place = excluded.place
		                        then runs.started_at else excluded.started_at end`,
		personID, place, at)
	if err != nil {
		return fmt.Errorf("keeping your place: %w", err)
	}
	return nil
}

// RunFor is the run worth offering, if there is one.
//
// Expiry is in the query rather than in the caller, the same as a session's:
// a check every call site has to remember is one some call site will forget.
func (s *Store) RunFor(ctx context.Context, personID int64, at time.Time) (Run, bool, error) {
	var out Run
	var seen time.Time
	err := s.pool.QueryRow(ctx, `
		select place, seen_at from runs
		 where person_id = $1 and seen_at > $2`, personID, at.Add(-KeepingPlace)).
		Scan(&out.Place, &seen)

	if errors.Is(err, pgx.ErrNoRows) {
		return Run{}, false, nil
	}
	if err != nil {
		return Run{}, false, fmt.Errorf("reading where you got to: %w", err)
	}
	out.Since = at.Sub(seen)
	return out, true, nil
}

// EndRun forgets it. Finishing, stopping and starting fresh are all the same
// write, because they are all "there is nothing to come back to".
func (s *Store) EndRun(ctx context.Context, personID int64) error {
	if _, err := s.pool.Exec(ctx, `delete from runs where person_id = $1`, personID); err != nil {
		return fmt.Errorf("forgetting where you got to: %w", err)
	}
	return nil
}
