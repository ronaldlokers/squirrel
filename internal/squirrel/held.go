package squirrel

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Things you cannot act on, and why. See migration 0023.
//
// Deliberately absent: anything that counts them. No function answers how many
// things you are waiting on, or how long one has waited, and none will.

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
	// PhotoName is the picture this row carries, or empty. It follows the note
	// here for the same reason it follows it everywhere: a note with no words
	// and only a photograph is a perfectly good note, and a screen that drops
	// the picture shows an empty row.
	PhotoName string
	// Since is how long it has been in this state, at the moment it was read.
	// Only filled by GoneQuiet; the list does not carry it, because a list of
	// elapsed times is a list of reproaches.
	Since time.Duration
}

// QuietAfter is how long each state may go unmentioned.
//
// Three weeks is right for a referral and absurd for a text message, so it is a
// property of the state rather than one number. Blocked is shorter because a
// thing you are blocked on is usually something you can unblock; waiting is
// somebody else's move.
//
// **someday is absent, and that is the design.** Someday is the state that
// means "not now, and do not ask me" — a product that came back to it in three
// weeks would have taken the one place you can put something down and turned it
// into a delayed nag.
var QuietAfter = map[ItemState]time.Duration{
	ItemWaiting: 21 * 24 * time.Hour,
	ItemBlocked: 14 * 24 * time.Hour,
}

// rowsToHeld finishes a scanned row. Shared so that the list and the single
// read cannot disagree about what a kind is.
func rowsToHeld(h *HeldItem, kind string) error {
	h.Kind = ItemKind(kind)
	return nil
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
		select id, raw_text, state, coalesce(held_because, ''), kind,
		       coalesce(attachment_path, '') from items
		 where person_id = $1 and has_content
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
		if err := rows.Scan(&h.ID, &h.Text, &h.State, &h.Because, &kind, &h.PhotoName); err != nil {
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

// GoneQuiet is something you set aside that nobody has mentioned since, if it
// has been long enough to be worth mentioning.
//
// The three states shipped in August as a one-way door: you park something
// waiting on the surgery precisely so that you do not have to hold it, and that
// only works if something else is holding it. Nothing was.
//
// The oldest one, not all of them. This is a sentence in the opening turn, not
// a list to work through — a screen that handed back everything you had ever
// parked would be a second pile wearing a different word.
func (s *Store) GoneQuiet(ctx context.Context, personID int64, at time.Time) (HeldItem, bool, error) {
	var h HeldItem
	var kind string
	var since time.Time
	err := s.pool.QueryRow(ctx, `
		select id, raw_text, state, coalesce(held_because, ''), kind,
		       coalesce(attachment_path, ''), state_at
		  from items
		 where person_id = $1 and has_content
		   and state in ('waiting', 'blocked')
		   and state_at is not null
		   -- Cast, because a parameter inside a case arm is inferred from the
		   -- arm and make_interval takes a double. Without it Postgres decides
		   -- both are text and refuses the statement outright — the same trap
		   -- this codebase hit three times during the OIDC work.
		   and state_at <= $2::timestamptz - make_interval(secs =>
		         case state when 'waiting' then $3::double precision
		                    else $4::double precision end)
		 order by state_at
		 limit 1`,
		personID, at,
		int64(QuietAfter[ItemWaiting]/time.Second),
		int64(QuietAfter[ItemBlocked]/time.Second)).
		Scan(&h.ID, &h.Text, &h.State, &h.Because, &kind, &h.PhotoName, &since)

	if errors.Is(err, pgx.ErrNoRows) {
		return HeldItem{}, false, nil
	}
	if err != nil {
		return HeldItem{}, false, fmt.Errorf("reading what has gone quiet: %w", err)
	}
	if err := rowsToHeld(&h, kind); err != nil {
		return HeldItem{}, false, err
	}
	h.Since = at.Sub(s.here(since))
	return h, true, nil
}

// StillHolding is "still waiting" — the answer that costs nothing.
//
// It moves the clock and touches nothing else: the state is unchanged, the
// reason is unchanged, and the note does not come back to the pile. Being able
// to say "yes, still" without it becoming a task is the whole reason this is
// safe to mention at all.
func (s *Store) StillHolding(ctx context.Context, personID, itemID int64, at time.Time) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		update items set state_at = $3
		 where id = $1 and person_id = $2 and state in ('waiting', 'blocked', 'someday')`,
		itemID, personID, at)
	if err != nil {
		return false, fmt.Errorf("keeping something waiting: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}
