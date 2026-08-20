package squirrel

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Things you cannot act on, and why. See migration 0023.
//
// What is deliberately absent here, as everywhere else in this store: anything
// that counts them. There is no function that answers how many things you are
// waiting on, or how long one has been waiting, and there will not be. A
// number beside a list of stalled work is a reproach, and the whole point of
// setting something aside is to stop being asked about it.

// HeldItem is one thing you cannot act on, and what would unstick it.
type HeldItem struct {
	ID   int64
	Text string
	// State is which of the three.
	State ItemState
	// Because is what you are waiting on, in your words, or empty. Empty is
	// ordinary — someday is not waiting on anything, and sometimes you did not
	// say.
	Because string
	// Kind carries through so a surface can say whether this was a note or
	// something you had decided to do. The two read differently: a task you
	// are waiting on is work with a dependency, and a note is a thought you
	// have parked.
	Kind ItemKind
}

// Words is the whole reason in one phrase — "waiting on the vet" — or just the
// state when there is nothing to add.
func (h HeldItem) Words() string {
	words := HeldWords[h.State]
	if h.Because == "" {
		return words
	}
	return words + " " + h.Because
}

// HoldItem sets one of the three, with what you are waiting on.
//
// It refuses anything that is not one of the three rather than trusting the
// caller, because this writes a state and the column's own constraint is the
// only other thing standing between a typo and a row nothing will ever show
// again.
//
// Only your own item, and only one that is open: holding something already
// done, dropped or kept would be a transition nobody asked for, and the state
// it is in is a fact somebody already established.
func (s *Store) HoldItem(ctx context.Context, personID, itemID int64, state ItemState, because string, at time.Time) (bool, error) {
	if !IsHeld(state) {
		return false, fmt.Errorf("not a way to hold something: %q", state)
	}

	var held *string
	if because = strings.TrimSpace(because); because != "" {
		held = &because
	}

	tag, err := s.pool.Exec(ctx, `
		update items set state = $3, state_at = $4, held_because = $5
		 where id = $2 and person_id = $1 and state = 'open'`,
		personID, itemID, string(state), at, held)
	if err != nil {
		return false, fmt.Errorf("setting something aside: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// HeldItems is everything you have set aside, newest first.
//
// One query for all three rather than one each, because they are read together
// — the question this answers is "what is not moving", and splitting it into
// three lists would make you ask it three times.
//
// Capped and answering the same bare `more` boolean every other list here
// answers, for the same reason: a caller cannot render a total it never
// receives.
func (s *Store) HeldItems(ctx context.Context, personID int64, limit int) ([]HeldItem, bool, error) {
	rows, err := s.pool.Query(ctx, `
		select id, raw_text, state, coalesce(held_because, ''), kind from items
		 where person_id = $1 and raw_text <> ''
		   and state in ('waiting', 'blocked', 'someday')
		 order by state_at desc nulls last, id desc
		 limit $2`, personID, limit+1)
	if err != nil {
		return nil, false, fmt.Errorf("reading what you set aside: %w", err)
	}
	defer rows.Close()

	held := []HeldItem{}
	for rows.Next() {
		var h HeldItem
		var kind string
		if err := rows.Scan(&h.ID, &h.Text, &h.State, &h.Because, &kind); err != nil {
			return nil, false, fmt.Errorf("scanning what you set aside: %w", err)
		}
		h.Kind = ItemKind(kind)
		held = append(held, h)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("reading what you set aside: %w", err)
	}

	if len(held) > limit {
		return held[:limit], true, nil
	}
	return held, false, nil
}

// Unhold puts something back in the pile.
//
// There is no separate transition for this and there does not need to be:
// coming back is `open`, which is what undo already writes and what the pile's
// own PUT IT BACK already presses. The one thing it adds is clearing the
// reason, because "waiting on the vet" is not true of something you are now
// doing.
func (s *Store) Unhold(ctx context.Context, personID, itemID int64, at time.Time) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		update items set state = 'open', state_at = $3, held_because = null
		 where id = $2 and person_id = $1
		   and state in ('waiting', 'blocked', 'someday')`,
		personID, itemID, at)
	if err != nil {
		return false, fmt.Errorf("picking it back up: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}
