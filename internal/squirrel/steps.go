package squirrel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// A thing broken into steps, handed back one at a time. See migration 0021.
//
// The important thing about this file is what is not in it: there is no
// function that returns the sequence. A model may produce a list; nothing in
// this product may show one, and the cheapest way to guarantee that is to make
// it unavailable — the same device that keeps a caller from rendering a count
// of the pile or a series of moods.

// Step is one thing to do, and the only one you are shown.
type Step struct {
	ID    int64
	Label string
	Body  string
	// Last says there is nothing after this one, so the surface can say so
	// rather than leaving you waiting for a step that never comes.
	Last bool
}

// SaveSteps replaces whatever sequence there was.
//
// Wholesale, in one transaction: a half-written sequence is worse than none,
// and two half-finished breakdowns is a list of things you did not finish.
func (s *Store) SaveSteps(ctx context.Context, personID int64, itemID *int64, label string, steps []string) error {
	if len(steps) == 0 {
		return nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("saving steps: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `delete from steps where person_id = $1`, personID); err != nil {
		return fmt.Errorf("clearing steps: %w", err)
	}
	for i, body := range steps {
		if _, err := tx.Exec(ctx, `
			insert into steps (person_id, item_id, label, position, body)
			values ($1, $2, $3, $4, $5)`, personID, itemID, label, i, body); err != nil {
			return fmt.Errorf("saving a step: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("saving steps: %w", err)
	}
	return nil
}

// NextStep is the one to show, and nothing else.
//
// It reports whether it is the last one by asking whether anything comes
// after, rather than by handing back a position out of a total — "step 2 of 5"
// is a count, and a count of what you have left to do is the accruing number
// this product refuses.
func (s *Store) NextStep(ctx context.Context, personID int64) (Step, bool, error) {
	var st Step
	var more int
	err := s.pool.QueryRow(ctx, `
		select id, label, body,
		       (select count(*) from steps later
		         where later.person_id = $1 and later.done_at is null
		           and later.position > steps.position)
		  from steps
		 where person_id = $1 and done_at is null
		 order by position
		 limit 1`, personID).Scan(&st.ID, &st.Label, &st.Body, &more)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Step{}, false, nil
		}
		return Step{}, false, fmt.Errorf("reading the next step: %w", err)
	}
	st.Last = more == 0
	return st, true, nil
}

// StepDone marks the one you were shown, and hands back whether there is
// another.
//
// Marking rather than deleting, for the reason everything else here reverses:
// pressing done on the last one should leave a finished sequence rather than
// an empty table that looks like nothing happened.
func (s *Store) StepDone(ctx context.Context, personID, stepID int64, at time.Time) error {
	if _, err := s.pool.Exec(ctx,
		`update steps set done_at = $3 where person_id = $1 and id = $2 and done_at is null`,
		personID, stepID, at); err != nil {
		return fmt.Errorf("marking a step done: %w", err)
	}
	return nil
}

// ClearSteps throws the sequence away.
//
// One press, no consequence, and no question asked back — the same shape "not
// now" already has. A breakdown you did not ask for and do not want must be as
// cheap to dismiss as it was to produce.
func (s *Store) ClearSteps(ctx context.Context, personID int64) error {
	if _, err := s.pool.Exec(ctx, `delete from steps where person_id = $1`, personID); err != nil {
		return fmt.Errorf("clearing steps: %w", err)
	}
	return nil
}
