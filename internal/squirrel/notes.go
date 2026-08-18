package squirrel

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// ItemState is what a note has become.
//
// `open` is the pile. The other three are exits, and `kept` is the one that
// keeps the pile from rebuilding itself: a serial number or a link is not a
// task and will never be `done`, so without somewhere for reference notes to
// go they would sit in triage forever.
//
// Every transition reverses, including back to `open`. Undo is a transition
// rather than a special case, which is the lesson phase 3 paid two review
// rounds for on chore completions.
type ItemState string

const (
	ItemOpen    ItemState = "open"
	ItemDone    ItemState = "done"
	ItemDropped ItemState = "dropped"
	ItemKept    ItemState = "kept"
)

// SetItemState moves a note.
//
// Writing the state a note already holds is a no-op rather than an error: a tap
// is a state assertion, not a delta, and a redelivered webhook is byte-identical
// to a second tap. Any variant of this that reports "already done" turns a
// retry into a failure.
func (s *Store) SetItemState(ctx context.Context, itemID int64, state ItemState, at time.Time) error {
	if _, err := s.pool.Exec(ctx, `
		update items set state = $2, state_at = $3 where id = $1`,
		itemID, string(state), at); err != nil {
		return fmt.Errorf("setting item state: %w", err)
	}
	return nil
}

// OpenItems is the pile: untriaged notes, newest first.
//
// Newest first, not oldest: oldest-first is a backlog you are behind on, and
// newest-first is context you still remember writing.
//
// It returns at most limit rows and a bare boolean saying whether more exist.
// Deliberately not a count. A number that grows while you are not looking,
// beside an implied target of zero, is the accumulating mechanism this project
// bans — and because the caller is handed a bool, it cannot render a total even
// if a later author wanted one. The rule is enforced by the signature rather
// than by a comment someone has to read.
func (s *Store) OpenItems(ctx context.Context, personID int64, limit int) ([]Item, bool, error) {
	return s.itemsWhere(ctx, `person_id = $1 and state = 'open'`, limit, personID)
}

// SearchItems matches raw text across every state, newest first.
//
// Every state, deliberately: `kept` exists so a reference note can leave triage
// and still be found later, so filtering by state here would defeat the state.
//
// The match is a plain substring test rather than a LIKE pattern. Interpolating
// the term into `'%' || $2 || '%'` would make a typed `%` a wildcard and return
// the whole pile, which looks like a working search until you notice every note
// is in the results.
func (s *Store) SearchItems(ctx context.Context, personID int64, query string, limit int) ([]Item, bool, error) {
	return s.itemsWhere(ctx,
		`person_id = $1 and strpos(lower(raw_text), lower($2)) > 0`,
		limit, personID, query)
}

// itemsWhere asks for one row more than the caller wanted, so "is there more"
// costs no second query and never needs a count(*) — which is also the only
// reason no code path anywhere has a total to leak.
func (s *Store) itemsWhere(ctx context.Context, where string, limit int, args ...any) ([]Item, bool, error) {
	q := `select id, raw_text, received_at from items where ` + where +
		` order by received_at desc, id desc limit $` + strconv.Itoa(len(args)+1)

	rows, err := s.pool.Query(ctx, q, append(args, limit+1)...)
	if err != nil {
		return nil, false, fmt.Errorf("listing items: %w", err)
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.RawText, &it.ReceivedAt); err != nil {
			return nil, false, fmt.Errorf("scanning item: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("listing items: %w", err)
	}

	more := len(items) > limit
	if more {
		items = items[:limit]
	}
	return items, more, nil
}
