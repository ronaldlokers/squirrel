package squirrel

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// What Squirrel has come to know about how you work.
//
// See migrations/0028_knowing.sql for why this is sentences rather than
// fields, replaced rather than appended, and readable by the person it is
// about.

// KnowingMost is how many observations are kept.
//
// Six. The bound is not about storage: this text is prepended to every
// conversational turn, so it is paid for on every message, and a page of
// observations would cost more than the conversation it is meant to improve.
// Six is also about honesty — a model asked for twenty things it has noticed
// will produce twenty, and the last fourteen will be invented.
const KnowingMost = 6

// KnowingLongest is how long one observation may be, in characters. A
// paragraph about somebody is a paragraph the model wrote to itself.
const KnowingLongest = 160

// Knowing is what is known, oldest first.
func (s *Store) Knowing(ctx context.Context, personID int64) ([]string, error) {
	const q = `select said from knowing where person_id = $1 order by id`
	rows, err := s.pool.Query(ctx, q, personID)
	if err != nil {
		return nil, fmt.Errorf("reading what is known: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var said string
		if err := rows.Scan(&said); err != nil {
			return nil, fmt.Errorf("reading what is known: %w", err)
		}
		out = append(out, said)
	}
	return out, rows.Err()
}

// LearnedAt is when the last pass ran, or the zero time if none has.
//
// The scheduler asks this rather than keeping a marker of its own: the rows
// are the record, and a second place to store "when did this last happen" is a
// second place for it to be wrong.
func (s *Store) LearnedAt(ctx context.Context, personID int64) (time.Time, error) {
	const q = `select coalesce(max(learned_at), 'epoch'::timestamptz) from knowing where person_id = $1`
	var at time.Time
	if err := s.pool.QueryRow(ctx, q, personID).Scan(&at); err != nil {
		return time.Time{}, fmt.Errorf("reading when it last learned: %w", err)
	}
	return s.here(at), nil
}

// ReplaceKnowing writes what the newest pass concluded, and nothing else.
//
// One transaction, delete then insert, because the half-state is a person
// Squirrel knows nothing about — and a conversational turn landing in that
// window would be answered by a Buddy who had forgotten everything.
//
// An empty set is a legitimate answer and clears what was there. A pass that
// concluded nothing is a pass that concluded nothing; keeping last week's
// because this week's was empty would make the record older than it claims.
func (s *Store) ReplaceKnowing(ctx context.Context, personID int64, said []string, at time.Time) error {
	kept := HoldToShape(said)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("keeping what was learned: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `delete from knowing where person_id = $1`, personID); err != nil {
		return fmt.Errorf("keeping what was learned: %w", err)
	}
	for _, one := range kept {
		if _, err := tx.Exec(ctx,
			`insert into knowing (person_id, said, learned_at) values ($1, $2, $3)`,
			personID, one, at); err != nil {
			return fmt.Errorf("keeping what was learned: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("keeping what was learned: %w", err)
	}
	return nil
}

// ForgetKnowing throws it all away.
//
// One press and no confirmation, which is the same shape every other reversal
// in this product has. What it costs is a week: the next pass writes again.
func (s *Store) ForgetKnowing(ctx context.Context, personID int64) error {
	if _, err := s.pool.Exec(ctx, `delete from knowing where person_id = $1`, personID); err != nil {
		return fmt.Errorf("forgetting what was learned: %w", err)
	}
	return nil
}

// HoldToShape is the bound, applied where the rows are written rather than
// where they are produced.
//
// A model asked for six short observations will sometimes answer with nine, or
// with one that is a paragraph, or with a numbered list — and the guard that
// catches that has to be the one nearest the database, because that is the
// only place every writer has to pass through.
//
// Anything countable goes. "You have done this four times" is a fact about a
// person and rule 2 forbids one on any surface, including this one, which is
// mostly read by a model.
func HoldToShape(said []string) []string {
	out := make([]string, 0, KnowingMost)
	seen := map[string]bool{}
	for _, one := range said {
		one = strings.TrimSpace(strings.ReplaceAll(one, "\n", " "))
		one = strings.TrimLeft(one, "-*0123456789. )")
		one = strings.TrimSpace(one)
		if one == "" || len(one) > KnowingLongest || seen[strings.ToLower(one)] {
			continue
		}
		if counts(one) {
			continue
		}
		seen[strings.ToLower(one)] = true
		out = append(out, one)
		if len(out) == KnowingMost {
			break
		}
	}
	return out
}

// counts reports whether an observation is really a tally.
//
// Digits are the obvious half. The words are the half that matters: "always",
// "never" and "every time" are how a count is written without a number, and an
// absolute claim about a person is the one thing this table must not hold —
// it is wrong the first time it is contradicted and it never notices.
func counts(said string) bool {
	if strings.ContainsAny(said, "0123456789") {
		return true
	}
	low := strings.ToLower(said)
	for _, tally := range []string{
		"always", "never", "every time", "each time", "most days",
		"once again", "yet again", "twice", "three times", "many times",
	} {
		if strings.Contains(low, tally) {
			return true
		}
	}
	return false
}
