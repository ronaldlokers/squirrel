package squirrel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

// ItemKind is what a row is, beside what state it is in.
//
// A note is a thought you had; a task is a thing you decided to do, once. The
// two are the same row at different moments, which is why this is a column and
// not a second table: undoing a decision has to return the row to where it
// was, not copy it back.
type ItemKind string

const (
	ItemNote ItemKind = "note"
	ItemTask ItemKind = "task"
)

// SetItemKind promotes a note to a task, or demotes it back.
//
// Scoped by person like every other write here, and idempotent for the same
// reason SetItemState is: saying a thing twice is saying it, not an error.
func (s *Store) SetItemKind(ctx context.Context, personID, itemID int64, k ItemKind) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`update items set kind = $3 where id = $1 and person_id = $2`,
		itemID, personID, string(k))
	if err != nil {
		return false, fmt.Errorf("setting item kind: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

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

// Reword changes what a note says, keeping everything else about it.
//
// PRODUCT.md listed this as explicitly undecided for a long time, and the
// hesitation was worth having: a note is a record of what you said, and a
// system that lets the record be rewritten is a system whose record cannot be
// trusted. What settles it is the failure it fixes — a thought typed on a
// phone, one-handed, arrives half-written or autocorrected into something
// else, and the only way to fix it was to drop it and say it again, which
// costs the arrival time and the place in the pile.
//
// So: the text changes, and nothing else does. The id, the arrival time, the
// state and the position in the pile all stay, because those are the facts
// about the note and only the words were wrong.
//
// Deliberately not versioned. Keeping the old text would make this a document
// with a history to read, which is a second thing to look at and a second
// place a thought can hide — and the thing being corrected is usually a typo.
func (s *Store) Reword(ctx context.Context, personID, itemID int64, text string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		update items set raw_text = $3
		 where id = $1 and person_id = $2`, itemID, personID, text)
	if err != nil {
		return false, fmt.Errorf("rewording note: %w", err)
	}
	return tag.RowsAffected() > 0, nil
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
	return s.itemsWhere(ctx, `person_id = $1 and kind = 'note' and state = 'open'`, limit, personID)
}

// Tasks is what you decided and have not done. Newest first, like the pile: a
// task decided this morning is the one you still remember deciding.
func (s *Store) Tasks(ctx context.Context, personID int64, limit int) ([]Item, bool, error) {
	return s.itemsWhere(ctx, `person_id = $1 and kind = 'task' and state = 'open'`, limit, personID)
}

// ArchivedTasks is what you did. Never a note that got done — those were never
// decided on, and the archive is a record of decisions carried out.
func (s *Store) ArchivedTasks(ctx context.Context, personID int64, limit int) ([]Item, bool, error) {
	return s.itemsWhere(ctx, `person_id = $1 and kind = 'task' and state = 'done'`, limit, personID)
}

// KeptItems is the shelf: the notes that were kept rather than done or
// dropped.
//
// `kept` is the state that exists because a serial number or a link is not a
// task and will never be done — without it every reference note sits in triage
// forever. Having made that argument, the product then gave it nowhere to be
// read back: search could find a kept note if you already knew a word in it,
// which is exactly the case where you do not.
//
// Newest first and capped like every other list here, and it hands back the
// same bare boolean for the same reason: the caller cannot render a total even
// if a later author wanted one.
func (s *Store) KeptItems(ctx context.Context, personID int64, limit int) ([]Item, bool, error) {
	return s.itemsWhere(ctx, `person_id = $1 and state = 'kept'`, limit, personID)
}

// OpenItemsAfter is the pile from a point: untriaged notes older than the one
// the caller has already looked at.
//
// This is skipping, and skipping is deliberately not a state. A skipped note is
// untouched — still open, still first the next time the pile is opened from the
// top — because "not now" is not something a note becomes. The position lives
// in the caller's address bar and nowhere else, which is also why it survives
// nothing: reload and you are back at the newest.
//
// The comparison is on (received_at, id) rather than id alone, because that is
// the pair the ordering uses. Ids come from an insert sequence and receipt
// times come from the transport, and the spool means the two can disagree —
// comparing on id alone would skip past a note that sorts later.
//
// A cursor naming a row that does not exist is treated as no cursor. It arrives
// through a URL, so it can be stale or invented, and the alternative is a pile
// that reports itself empty while holding everything.
func (s *Store) OpenItemsAfter(ctx context.Context, personID, afterID int64, limit int) ([]Item, bool, error) {
	if afterID == 0 {
		return s.OpenItems(ctx, personID, limit)
	}
	return s.itemsWhere(ctx, `person_id = $1 and kind = 'note' and state = 'open'
		 and (
		   not exists (select 1 from items where id = $2)
		   or (received_at, id) < (select received_at, id from items where id = $2)
		 )`, limit, personID, afterID)
}

// SearchItems matches raw text across every state, newest first.
//
// Every state, deliberately: `kept` exists so a reference note can leave triage
// and still be found later, so filtering by state here would defeat the state.
//
// The term is a person's typing, not a pattern: `%` and `_` are characters
// here, and a search for "80%" finds the note about the tank rather than every
// note there is. That used to be guaranteed by using strpos instead of LIKE.
// strpos hides the column from the planner, though, so every search read every
// message ever received — and the number of those only goes up. The index in
// migration 0010 answers a LIKE directly, so this is a LIKE now, with the term
// escaped rather than trusted.
func (s *Store) SearchItems(ctx context.Context, personID int64, query string, limit int) ([]Item, bool, error) {
	pattern := "%" + likeEscape(strings.ToLower(query)) + "%"
	return s.itemsWhere(ctx,
		`person_id = $1 and lower(raw_text) like $2 escape '\'`,
		limit, personID, pattern)
}

// likeEscape makes a typed string mean itself. The backslash goes first, or it
// would escape the escapes added after it.
func likeEscape(term string) string {
	term = strings.ReplaceAll(term, `\`, `\\`)
	term = strings.ReplaceAll(term, "%", `\%`)
	return strings.ReplaceAll(term, "_", `\_`)
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
	q := `select id, raw_text, received_at, payload, state, kind from items
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
		if err := rows.Scan(&it.ID, &it.RawText, &it.ReceivedAt, &payload, &it.State, &it.Kind); err != nil {
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
		select id, raw_text, received_at, payload, state, kind from items
		 where id = $1 and person_id = $2`, itemID, personID).
		Scan(&it.ID, &it.RawText, &it.ReceivedAt, &payload, &it.State, &it.Kind)
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

// LastTriaged is the most recent note to have left the pile, whichever surface
// moved it.
//
// It is ordered by state_at rather than by anything the caller remembers,
// because the two views share one pile: a note triaged on the screen is what
// chat undoes, and the reverse. Anything else would be each surface keeping its
// own idea of "last", which is the disagreement Principle 4 forbids.
//
// There is no history table to walk. A note that left the pile went to done,
// dropped or kept, and putting it back means open — which is also the only
// destination that makes sense for an undo, since the states are exits and open
// is the pile.
func (s *Store) LastTriaged(ctx context.Context, personID int64) (Item, bool, error) {
	rows, err := s.pool.Query(ctx, `
		select id, raw_text, received_at, payload, state, kind from items
		 where person_id = $1 and raw_text <> ''
		   and state <> 'open' and state_at is not null
		 order by state_at desc, id desc
		 limit 20`, personID)
	if err != nil {
		return Item{}, false, fmt.Errorf("reading the last triaged note: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var it Item
		var payload json.RawMessage
		if err := rows.Scan(&it.ID, &it.RawText, &it.ReceivedAt, &payload, &it.State, &it.Kind); err != nil {
			return Item{}, false, fmt.Errorf("scanning item: %w", err)
		}
		// The same test the pile applies. A typed command is stored like
		// anything else and can be triaged by a redelivery or a stray tap, and
		// undo must not answer with one.
		if isNote(it.RawText, payload) {
			return it, true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return Item{}, false, fmt.Errorf("reading the last triaged note: %w", err)
	}
	return Item{}, false, nil
}

// isNote is the pile's definition of a note, and it is deliberately the same
// one CapturesSince uses: a genuine tap is not a thought, and neither is a
// command. ParseAction matches on text alone, so isActionPayload is what tells
// a real tap from someone typing the same shape — which stays a thought.
func isNote(text string, payload json.RawMessage) bool {
	if _, isTap := ParseAction(text); isTap && isActionPayload(payload) {
		return false
	}
	// A note captured on the screen is a note whatever it says.
	//
	// The matcher below reads text, and it has to: in a chat room the only
	// thing distinguishing "done 2" the command from "done 2" the thought is
	// the words. The slot has no such ambiguity — it is a capture surface and
	// nothing else, with no commands to be confused with — so a thought that
	// happens to read like one is still a thought. Without this, typing
	// "every day vacuum" into the slot would store a row the pile then refuses
	// to show: a thought lost silently, which is the failure this product
	// exists to prevent. Chat has the leading dot for the same problem; the
	// screen needs no escape hatch because it never interprets.
	if isScreenCapture(payload) {
		return true
	}
	return matchFn(text).Kind == IntentCapture
}

// ScreenCapture is the payload the screen writes, and the marker isNote reads.
const ScreenCapture = `{"type":"screen"}`

func isScreenCapture(payload json.RawMessage) bool {
	var p struct {
		Type string `json:"type"`
	}
	return json.Unmarshal(payload, &p) == nil && p.Type == "screen"
}
