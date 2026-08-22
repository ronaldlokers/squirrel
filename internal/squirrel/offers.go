package squirrel

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// What the picker offered and what you said back. See migration 0016 for why
// only the answers are stored and never the showings.

// OfferAnswer is what a person did with an offer.
type OfferAnswer string

const (
	// AnswerLater is the only one that suppresses. It costs one press, it has
	// no consequence, and it is never followed by a question.
	AnswerLater OfferAnswer = "later"
	// AnswerDid and AnswerStarted are recorded because they are free and
	// because "did the picker actually cause anything" is worth being able to
	// ask. Neither suppresses: a chore done is suppressed by its own clock, and
	// a task done leaves the list by changing state.
	AnswerDid     OfferAnswer = "did"
	AnswerStarted OfferAnswer = "started"
)

// RecordAnswer stores what was said about an offer.
//
// refID is zero for an offer that names no row of its own. It is written as
// null rather than as 0, so a suppression key can never collide with a real
// chore or item id.
func (s *Store) RecordAnswer(ctx context.Context, personID int64, kind OfferKind, refID int64, answer OfferAnswer, at time.Time) error {
	var ref *int64
	if refID != 0 {
		ref = &refID
	}
	if _, err := s.pool.Exec(ctx, `
		insert into offers (person_id, kind, ref_id, answer, answered_at)
		values ($1, $2, $3, $4, $5)`,
		personID, string(kind), ref, string(answer), at); err != nil {
		return fmt.Errorf("recording an offer's answer: %w", err)
	}
	return nil
}

// RefusedSince reports whether this kind of offer was turned down after a
// moment.
//
// It exists for the breadcrumb, which is the one offer with no row of its own:
// "what you were on" names a label rather than a chore or a task, so its
// suppression key is the same string every time and the set below cannot tell
// this morning's breadcrumb from this afternoon's. Asking "was this kind
// turned down since the moment this one appeared" can, because a later focus
// ends later than the refusal that was about the earlier one.
//
// A boolean, and deliberately not a count. The constraint on Suppressed
// applies here for the same reason: the picker asks "may I offer this", and
// there is no question here that could be answered with a number about
// somebody's day.
func (s *Store) RefusedSince(ctx context.Context, personID int64, kind OfferKind, since time.Time) (bool, error) {
	var refused bool
	if err := s.pool.QueryRow(ctx, `
		select exists (
			select 1 from offers
			 where person_id = $1 and kind = $2
			   and answer = 'later' and answered_at > $3)`,
		personID, string(kind), since).Scan(&refused); err != nil {
		return false, fmt.Errorf("reading whether this was turned down: %w", err)
	}
	return refused, nil
}

// Suppressed is the set of things turned down since a moment, as
// "kind:ref_id" keys.
//
// A set rather than a list, and one window rather than a history: the picker
// asks "may I offer this", and there is deliberately no function here that
// could answer "how often has this been turned down", which is a sentence
// about the person.
//
// Only 'later' suppresses. A completion is already invisible to the picker for
// its own reasons — a chore's clock resets, a task's state changes — and
// treating "did it" as a suppression would hide a daily chore the day after
// you did it for a second, wrong reason.
func (s *Store) Suppressed(ctx context.Context, personID int64, since time.Time) (map[string]bool, error) {
	rows, err := s.pool.Query(ctx, `
		select kind, coalesce(ref_id, 0) from offers
		 where person_id = $1 and answered_at >= $2 and answer = 'later'`,
		personID, since)
	if err != nil {
		return nil, fmt.Errorf("reading refusals: %w", err)
	}
	defer rows.Close()

	out := map[string]bool{}
	for rows.Next() {
		var kind string
		var ref int64
		if err := rows.Scan(&kind, &ref); err != nil {
			return nil, fmt.Errorf("scanning a refusal: %w", err)
		}
		out[suppressionKey(OfferKind(kind), ref)] = true
	}
	return out, rows.Err()
}

// UnrefuseToday takes a refusal back.
//
// Every state transition in this product reverses, and a refusal is a
// transition like any other — untapping "not now" is the same shape as
// untapping a done, and the alternative is one press that cannot be undone
// until midnight.
//
// It deletes today's refusals for that thing rather than writing a second row
// that cancels the first. The table's whole purpose is "may I offer this", and
// a cancelling row would make that a question about ordering — which is how a
// two-row history becomes a thing someone eventually renders.
func (s *Store) UnrefuseToday(ctx context.Context, personID int64, kind OfferKind, refID int64, now time.Time) error {
	var ref *int64
	if refID != 0 {
		ref = &refID
	}
	if _, err := s.pool.Exec(ctx, `
		delete from offers
		 where person_id = $1 and kind = $2 and ref_id is not distinct from $3
		   and answer = 'later' and answered_at >= $4`,
		personID, string(kind), ref, startOfDay(now)); err != nil {
		return fmt.Errorf("taking back a refusal: %w", err)
	}
	return nil
}

// suppressionKey is the one place the key's shape is decided, so a writer and
// a reader cannot spell it differently.
func suppressionKey(kind OfferKind, refID int64) string {
	return string(kind) + ":" + strconv.FormatInt(refID, 10)
}

// SuppressionKey is the same, for the one caller outside this package.
//
// internal/boot needs it because the coach's read tools apply today's
// refusals before the model ever sees the work — suppression belongs in the
// fact, not in a prompt, or "not now" would mean less depending on which of
// the two things answered. Exported rather than reimplemented there, because
// two spellings of this key is exactly the bug the unexported one exists to
// prevent.
func SuppressionKey(kind OfferKind, refID int64) string { return suppressionKey(kind, refID) }
