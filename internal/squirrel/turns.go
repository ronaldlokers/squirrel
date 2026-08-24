package squirrel

import (
	"context"
	"fmt"
	"time"
)

// The conversation the screen is.
//
// The web surface stopped being a set of pages on 24 August 2026 and became one
// thread; see docs/superpowers/specs/2026-08-24-the-thread-design.md. Three
// things are done with it and no more: add to it, read the end of it, and walk
// back up it. There is deliberately no function here that edits a turn or
// removes one — history is not rewritten, and a mistake is answered by the next
// turn, which is how conversations work.

// Speaker is who said it. Two values, and a third would be a design decision
// rather than a constant.
type Speaker string

const (
	SpeakerBuddy Speaker = "buddy"
	SpeakerYou   Speaker = "you"
)

// Turn is one thing said, as it was said.
//
// Shown is JSON rather than a set of columns because what a turn drew varies
// with what kind of turn it is, and because nothing ever queries inside it — it
// is read back whole and handed to a template. It is never a pointer at another
// table; see the migration for why.
type Turn struct {
	ID     int64
	Who    Speaker
	Words  string
	Shown  []byte
	SaidAt time.Time
}

// AppendTurn writes one turn and returns it with its id and time filled in.
func (s *Store) AppendTurn(ctx context.Context, personID int64, t Turn) (Turn, error) {
	const q = `
		insert into turns (person_id, who, words, shown)
		values ($1, $2, $3, $4)
		returning id, said_at`

	// nil rather than an empty slice, so a turn that drew nothing stores SQL
	// null and reads back as nil.
	var shown any
	if len(t.Shown) > 0 {
		shown = t.Shown
	}
	if err := s.pool.QueryRow(ctx, q, personID, string(t.Who), t.Words, shown).
		Scan(&t.ID, &t.SaidAt); err != nil {
		return Turn{}, fmt.Errorf("saying it: %w", err)
	}
	return t, nil
}

// RecentTurns is the end of the conversation, oldest first.
//
// The limit is applied to the newest rows and the result is then reversed, so
// the cap keeps the end of the conversation rather than its beginning. more
// means there is something above what came back.
func (s *Store) RecentTurns(ctx context.Context, personID int64, limit int) ([]Turn, bool, error) {
	const q = `
		select id, who, words, shown, said_at
		  from turns
		 where person_id = $1
		 order by said_at desc, id desc
		 limit $2`
	return s.scanTurns(ctx, limit, q, personID, limit+1)
}

// TurnsBefore is the page above a turn you can already see.
func (s *Store) TurnsBefore(ctx context.Context, personID, beforeID int64, limit int) ([]Turn, bool, error) {
	const q = `
		select id, who, words, shown, said_at
		  from turns
		 where person_id = $1
		   and (said_at, id) < (select said_at, id from turns where id = $2)
		 order by said_at desc, id desc
		 limit $3`
	return s.scanTurns(ctx, limit, q, personID, beforeID, limit+1)
}

// scanTurns reads newest-first rows, reports whether one more than asked for
// came back, and hands the caller the rest oldest-first.
//
// The over-read is how "there is more" is answered without a second round trip.
// It predates counts being permitted and stays for that reason rather than for
// the old one: what is above you is not a thing you can act on, so the control
// says that there is more and not how much.
func (s *Store) scanTurns(ctx context.Context, limit int, q string, args ...any) ([]Turn, bool, error) {
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, false, fmt.Errorf("reading the conversation: %w", err)
	}
	defer rows.Close()

	var out []Turn
	for rows.Next() {
		var t Turn
		var who string
		if err := rows.Scan(&t.ID, &who, &t.Words, &t.Shown, &t.SaidAt); err != nil {
			return nil, false, fmt.Errorf("reading the conversation: %w", err)
		}
		t.Who = Speaker(who)
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("reading the conversation: %w", err)
	}

	more := len(out) > limit
	if more {
		out = out[:limit]
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, more, nil
}
