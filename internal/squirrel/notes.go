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

// ItemState is what a note has become. `open` is the pile; the other three are
// exits. `kept` is what stops the pile rebuilding itself — a serial number is not
// a task and will never be `done`.
//
// Every transition reverses, including back to `open`.
type ItemState string

const (
	ItemOpen    ItemState = "open"
	ItemDone    ItemState = "done"
	ItemDropped ItemState = "dropped"
	ItemKept    ItemState = "kept"

	// The three for a thing you cannot act on. See migration 0023 for why they
	// are three rather than one — they end differently, and that is the whole
	// of it.
	ItemWaiting ItemState = "waiting"
	ItemBlocked ItemState = "blocked"
	ItemSomeday ItemState = "someday"
)

// Held is the three, in the order every surface offers them: the two that
// something outside you will end, then the one only you can.
var Held = []ItemState{ItemWaiting, ItemBlocked, ItemSomeday}

// HeldWords is what each is called. The words are what you would say — "waiting
// on the vet", not "status: blocked" — because these are answers to "why is
// this not moving" rather than labels for a category.
var HeldWords = map[ItemState]string{
	ItemWaiting: "waiting on",
	ItemBlocked: "blocked on",
	ItemSomeday: "someday",
}

// ParseHeld reads one of the three, generously, the same way ParseBlocker does
// and for the same reason: this arrives from someone who has just found out
// they cannot do the thing, and being strict about a wording the product chose
// itself is a tax at the wrong moment.
func ParseHeld(s string) (ItemState, bool) {
	t := strings.ToLower(strings.TrimSpace(s))
	switch {
	case t == "":
		return "", false
	case strings.HasPrefix(t, "wait"):
		return ItemWaiting, true
	case strings.HasPrefix(t, "block"), strings.HasPrefix(t, "stuck on"):
		return ItemBlocked, true
	case strings.HasPrefix(t, "someday"), strings.HasPrefix(t, "some day"),
		strings.HasPrefix(t, "maybe"), strings.HasPrefix(t, "one day"):
		return ItemSomeday, true
	}
	return "", false
}

// IsHeld reports whether a state is one of the three. A map rather than a
// switch so an unknown state is a lookup miss rather than a default branch
// somebody later fills in.
func IsHeld(state ItemState) bool { return heldStates[state] }

var heldStates = map[ItemState]bool{
	ItemWaiting: true, ItemBlocked: true, ItemSomeday: true,
}

// ItemKind is what a row is, beside what state it is in. A column and not a
// second table, because undoing a decision must return the row rather than copy
// it back.
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

// SetItemState moves a note. Writing the state a note already holds is a no-op
// rather than an error: a tap is a state assertion, not a delta, and a
// redelivered webhook is byte-identical to a second tap.
func (s *Store) SetItemState(ctx context.Context, itemID int64, state ItemState, at time.Time) error {
	if _, err := s.pool.Exec(ctx, `
		update items set state = $2, state_at = $3 where id = $1`,
		itemID, string(state), at); err != nil {
		return fmt.Errorf("setting item state: %w", err)
	}
	return nil
}

// MoveItemState is SetItemState for a caller that knows what the note was when it
// decided, and answers whether the note was still there to move.
//
// The screen holds a tapped action for a beat, so its write lands about a second
// after the decision and the row is untouched for that window. A `!drop` typed
// inside it wrote the truth and the screen's unconditional overwrite replaced it.
//
// `state in (from, to)` rather than `state = from`: a second identical tap finds
// the note already at `to` and must still succeed. Only a note that moved
// somewhere else is refused.
func (s *Store) MoveItemState(ctx context.Context, itemID int64, from, to ItemState, at time.Time) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		update items set state = $3, state_at = $4
		where id = $1 and state in ($2, $3)`,
		itemID, string(from), string(to), at)
	if err != nil {
		return false, fmt.Errorf("moving item state: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// Reword changes what a note says. The id, arrival time, state and position stay
// — only the words were wrong.
//
// Deliberately not versioned: keeping the old text would make this a document
// with a history to read.
func (s *Store) Reword(ctx context.Context, personID, itemID int64, text string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		update items set raw_text = $3
		 where id = $1 and person_id = $2`, itemID, personID, text)
	if err != nil {
		return false, fmt.Errorf("rewording note: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// OpenItems is the pile: untriaged notes, newest first — oldest-first is a
// backlog you are behind on.
//
// At most limit rows and a bare boolean for whether more exist, never a count.
// The signature is what enforces it: the caller cannot render a total.
const stillMine = `(
	moment_id is null
	or not exists (
		select 1 from moments
		 where moments.id = items.moment_id
		   and moments.done_at is null
		   and moments.starts_at > now()
	)
)`

func (s *Store) OpenItems(ctx context.Context, personID int64, limit int) ([]Item, bool, error) {
	return s.itemsWhere(ctx,
		`person_id = $1 and kind = 'note' and state = 'open' and `+stillMine, limit, personID)
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

// AKeptItem is one kept note, chosen at random, for resurfacing and nothing else.
// Random rather than oldest-first: a queue would give the shelf a front, and a
// front is a place to be behind.
func (s *Store) AKeptItem(ctx context.Context, personID int64) (string, bool, error) {
	var text string
	err := s.pool.QueryRow(ctx, `
		select raw_text from items
		 where person_id = $1 and state = 'kept' and has_content
		 order by random() limit 1`, personID).Scan(&text)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("reading the shelf: %w", err)
	}
	return text, true, nil
}

// KeptItems is the shelf: notes kept rather than done or dropped. Newest first
// and capped, handing back the same bare boolean as the pile.
func (s *Store) KeptItems(ctx context.Context, personID int64, limit int) ([]Item, bool, error) {
	return s.itemsWhere(ctx, `person_id = $1 and state = 'kept'`, limit, personID)
}

// OpenItemsAfter is the pile from a point: untriaged notes older than the one the
// caller has already seen.
//
// Skipping is deliberately not a state — the note stays open and first from the
// top, and the position lives in the address bar and survives nothing.
//
// Compared on (received_at, id), the pair the ordering uses: ids come from a
// sequence and receipt times from the transport, and the spool lets them
// disagree. A cursor naming a row that does not exist is treated as no cursor.
func (s *Store) OpenItemsAfter(ctx context.Context, personID, afterID int64, limit int) ([]Item, bool, error) {
	if afterID == 0 {
		return s.OpenItems(ctx, personID, limit)
	}
	return s.itemsWhere(ctx, `person_id = $1 and kind = 'note' and state = 'open'
		 and (
		   not exists (select 1 from items where id = $2)
		   or (received_at, id) < (select received_at, id from items where id = $2)
		 )
		 and `+stillMine, limit, personID, afterID)
}

// SearchItems matches raw text across every state — `kept` exists so a reference
// note can leave triage and still be found.
//
// The term is typing, not a pattern: `%` and `_` are characters. That was once
// guaranteed by strpos, which hid the column from the planner and read every
// message ever received; migration 0010's index answers a LIKE directly, so this
// is a LIKE with the term escaped.
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

// itemsWhere reads newest-first and stops one row past what the caller wanted —
// that extra row answers "is there more" without a count(*).
//
// Filtered in Go rather than SQL because what makes a row a note is Match, which
// SQL cannot express. The drain stores every inbound message, so without this the
// pile fills with the commands you typed to look at the pile.
//
// CapturesSince applies exactly this test and the two must agree.
func (s *Store) itemsWhere(ctx context.Context, where string, limit int, args ...any) ([]Item, bool, error) {
	// has_content matches CapturesSince exactly. An attachment-only message lands as
	// an empty row, and without this the pile prints a blank numbered line while the
	// evening list does not.
	q := `select id, raw_text, received_at, payload, state, kind,
	             coalesce(attachment_path, ''), coalesce(attachment_type, '')
	        from items
	       where has_content and ` + where +
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
		if err := rows.Scan(&it.ID, &it.RawText, &it.ReceivedAt, &payload, &it.State, &it.Kind,
			&it.PhotoName, &it.PhotoType); err != nil {
			return nil, false, fmt.Errorf("scanning item: %w", err)
		}
		if !isNote(it.RawText, payload) {
			continue
		}
		// When it arrived, in the person's clock. See Store.here — this is the
		// same defect the fixed points had, on the other table.
		it.ReceivedAt = s.here(it.ReceivedAt)
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

// ItemByID reads one note, with the person part of the lookup rather than checked
// afterwards: a handler holding an id from a form must not be able to return
// someone else's row.
func (s *Store) ItemByID(ctx context.Context, personID, itemID int64) (Item, bool, error) {
	var it Item
	var payload json.RawMessage
	err := s.pool.QueryRow(ctx, `
		select id, raw_text, received_at, payload, state, kind,
		       coalesce(attachment_path, ''), coalesce(attachment_type, '')
		  from items
		 where id = $1 and person_id = $2`, itemID, personID).
		Scan(&it.ID, &it.RawText, &it.ReceivedAt, &payload, &it.State, &it.Kind,
			&it.PhotoName, &it.PhotoType)
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
	it.ReceivedAt = s.here(it.ReceivedAt)
	return it, true, nil
}

// PromoteItem turns a note into a chore: its text becomes the name, and the note
// becomes `done`.
//
// Chore first, then the note. A failure between them leaves the chore created and
// the note in the pile, so a retry upserts the same chore by name; the other
// order loses the chore.
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

// LastTriaged is the most recent note to have left the pile, ordered by state_at
// rather than by anything a caller remembers: two views share one pile, and a
// note triaged on the screen is what chat undoes.
//
// No history table. A note that left went to done, dropped or kept, and putting
// it back means open.
func (s *Store) LastTriaged(ctx context.Context, personID int64) (Item, bool, error) {
	rows, err := s.pool.Query(ctx, `
		select id, raw_text, received_at, payload, state, kind,
		       coalesce(attachment_path, ''), coalesce(attachment_type, '')
		  from items
		 where person_id = $1 and has_content
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
		if err := rows.Scan(&it.ID, &it.RawText, &it.ReceivedAt, &payload, &it.State, &it.Kind,
			&it.PhotoName, &it.PhotoType); err != nil {
			return Item{}, false, fmt.Errorf("scanning item: %w", err)
		}
		// The same test the pile applies. A typed command is stored like
		// anything else and can be triaged by a redelivery or a stray tap, and
		// undo must not answer with one.
		if isNote(it.RawText, payload) {
			it.ReceivedAt = s.here(it.ReceivedAt)
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
	// A note captured on the screen is a note whatever it says. The matcher reads
	// text because a chat room is ambiguous; the slot is not. Without this, typing
	// "every day vacuum" into the slot stored a row the pile then refused to show.
	if isScreenCapture(payload) {
		return true
	}
	return matchFn(text).Kind == IntentCapture
}

// ScreenCapture is the payload the screen writes, and the marker isNote reads.
const ScreenCapture = `{"type":"screen"}`

// ScreenTransport is what the screen's own captures are filed under.
//
// A transport like any other, now that the slot spools rather than writing
// straight to Postgres: the drain reads a capture's transport to resolve whose
// it is, so the screen needs a name and an identity under that name.
const ScreenTransport = "screen"

func isScreenCapture(payload json.RawMessage) bool {
	var p struct {
		Type string `json:"type"`
	}
	return json.Unmarshal(payload, &p) == nil && p.Type == "screen"
}
