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

// LandedBadly marks one reply as having landed badly, by the id it carries.
//
// Idempotent on purpose: pressing it twice is pressing it, and a second press
// on a bad night must not become an error to read. The timestamp is when you
// said so rather than when it was said, because those are different facts and
// the first is the one that dates the feedback.
func (s *Store) LandedBadly(ctx context.Context, personID, answerID int64, at time.Time) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		update coach_answers set landed_badly_at = $3
		where id = $1 and person_id = $2 and landed_badly_at is null`,
		answerID, personID, at)
	if err != nil {
		return false, fmt.Errorf("marking a reply as having landed badly: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// LandedBadlyLatest marks the most recent reply this person was actually shown.
//
// The conversation carries no row id — an exchange is two strings — and
// threading one through the window, the provider and the budget would be a lot
// of machinery for a press that means "the one I just read".
//
// So: the newest reply that reached a human. That is what the control is
// beside, and on the surface it lives on it is the only thing it can sensibly
// mean. `used` matters here — a reply the guard threw away was never seen, and
// saying it landed badly would be saying so about something nobody read.
func (s *Store) LandedBadlyLatest(ctx context.Context, personID int64, at time.Time) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		update coach_answers set landed_badly_at = $2
		where id = (
			select id from coach_answers
			where person_id = $1 and used and landed_badly_at is null
			order by said_at desc limit 1
		)`, personID, at)
	if err != nil {
		return false, fmt.Errorf("marking the last reply as having landed badly: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// BadlyLanded is the last few replies that landed badly, newest first, for the
// prompt to carry.
//
// A handful rather than all of them, and never a count. The coach is shown
// what does not land here — which is a fact about the words — and is never
// told how often, which would be a fact about the person and is what rule 2
// forbids on any surface.
func (s *Store) BadlyLanded(ctx context.Context, personID int64, limit int) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		select reply from coach_answers
		where person_id = $1 and landed_badly_at is not null
		order by landed_badly_at desc limit $2`, personID, limit)
	if err != nil {
		return nil, fmt.Errorf("reading what landed badly: %w", err)
	}
	defer rows.Close()

	var said []string
	for rows.Next() {
		var reply string
		if err := rows.Scan(&reply); err != nil {
			return nil, fmt.Errorf("reading what landed badly: %w", err)
		}
		said = append(said, reply)
	}
	return said, rows.Err()
}

func (s *Store) DeleteLatestCoachAnswer(ctx context.Context, personID int64) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		delete from coach_answers
		where id = (
			select id from coach_answers
			where person_id = $1
			order by said_at desc limit 1
		)`, personID)
	if err != nil {
		return false, fmt.Errorf("deleting the last thing the coach said: %w", err)
	}
	return tag.RowsAffected() > 0, nil
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
