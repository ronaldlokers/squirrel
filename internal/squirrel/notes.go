package squirrel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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

// itemsWhere reads newest-first and stops as soon as it has one row more than
// the caller wanted — that extra row is how "is there more" is answered without
// a count(*), which is also why no code path anywhere has a total to leak.
//
// The rows are filtered in Go rather than in SQL because what makes a row a
// note is `Match`, and Match is prose rules and regexes that SQL cannot
// express. The drain stores *every* inbound message as an item: `!notes`, `?`,
// `done 2` and a tap's own "!action …" text all land in this table. Without
// this filter the pile would fill with the things you typed to look at the
// pile, and searching for "done" would return your own commands.
//
// CapturesSince applies exactly this test for the evening message, and the two
// have to agree — a note the evening list shows and the pile hides, or the
// reverse, is the kind of disagreement phase 3 spent a fix round on.
//
// The cost is that a long run of commands is scanned before enough notes are
// found. Bounded by how many commands were typed consecutively, which is small,
// and there is no SQL limit to make it wrong when it is not.
func (s *Store) itemsWhere(ctx context.Context, where string, limit int, args ...any) ([]Item, bool, error) {
	// raw_text <> '' matches CapturesSince exactly. An attachment-only message
	// lands as an empty row — campfire.go takes the body's plain text with no
	// guard — and without this the pile prints a blank numbered line while the
	// evening list, which has always filtered them, does not. The two surfaces
	// disagreeing about what a note is is the thing this function's comment
	// warns about.
	q := `select id, raw_text, received_at, payload, state from items
	       where raw_text <> '' and ` + where +
		` order by received_at desc, id desc`

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, false, fmt.Errorf("listing items: %w", err)
	}
	defer rows.Close()

	items := []Item{}
	for rows.Next() && len(items) <= limit {
		var it Item
		var payload json.RawMessage
		if err := rows.Scan(&it.ID, &it.RawText, &it.ReceivedAt, &payload, &it.State); err != nil {
			return nil, false, fmt.Errorf("scanning item: %w", err)
		}
		if !isNote(it.RawText, payload) {
			continue
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

// ItemByID reads one note, scoped to its owner.
//
// The person is part of the lookup rather than checked afterwards. A handler
// that receives an id from a form has been handed a number by whoever is on
// the other end, and the only safe shape is a query that cannot return someone
// else's row in the first place.
func (s *Store) ItemByID(ctx context.Context, personID, itemID int64) (Item, bool, error) {
	var it Item
	var payload json.RawMessage
	err := s.pool.QueryRow(ctx, `
		select id, raw_text, received_at, payload, state from items
		 where id = $1 and person_id = $2`, itemID, personID).
		Scan(&it.ID, &it.RawText, &it.ReceivedAt, &payload, &it.State)
	if errors.Is(err, pgx.ErrNoRows) {
		return Item{}, false, nil
	}
	if err != nil {
		return Item{}, false, fmt.Errorf("reading item: %w", err)
	}
	// The same test itemsWhere applies, for the same reason: the drain stores
	// every inbound message as a row, so an id can point at "!notes" as easily
	// as at a thought. A lookup that answered for one would let a caller act on
	// a row the pile itself refuses to show, and the two read paths disagreeing
	// about what a note is is the drift this file's other comments warn about.
	if !isNote(it.RawText, payload) {
		return Item{}, false, nil
	}
	return it, true, nil
}

// PromoteItem turns a note into a chore: the note's own text becomes the
// chore's name, and the note becomes `done` — it did its job by turning into
// something that comes back on its own.
//
// Chore first, then the note. A failure between them leaves the chore created
// and the note still in the pile, so a second attempt upserts the same chore by
// name and finishes the job. The other order would leave a note marked done
// with no chore to show for it, which is the one of the two that loses
// something.
//
// Both the chat command and the screen call this. Two implementations of "a
// note becomes a chore" is the disagreement Principle 4 forbids, and the chat
// path already paid for the ordering above.
func (s *Store) PromoteItem(ctx context.Context, personID, itemID int64, every time.Duration) (Chore, bool, error) {
	it, ok, err := s.ItemByID(ctx, personID, itemID)
	if err != nil || !ok {
		return Chore{}, false, err
	}
	c, err := s.UpsertChore(ctx, personID, it.RawText, every, DefaultTolerance(every))
	if err != nil {
		return Chore{}, false, err
	}
	if err := s.SetItemState(ctx, it.ID, ItemDone, time.Now()); err != nil {
		return Chore{}, false, err
	}
	return c, true, nil
}

// isNote is the pile's definition of a note, and it is deliberately the same
// one CapturesSince uses: a genuine tap is not a thought, and neither is a
// command. ParseAction matches on text alone, so isActionPayload is what tells
// a real tap from someone typing the same shape — which stays a thought.
func isNote(text string, payload json.RawMessage) bool {
	if _, isTap := ParseAction(text); isTap && isActionPayload(payload) {
		return false
	}
	return matchFn(text).Kind == IntentCapture
}
