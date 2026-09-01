package web

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The rooms.
//
// The screen became one conversation on 24 August 2026 and seven rooms on
// 28 August; see docs/superpowers/specs/2026-08-28-rooms-design.md. A room is
// both a place that keeps its own conversation and a scope that narrows what
// Buddy can do in it.
//
// A slice rather than a map, because the rail has an order and a map has none
// — the order was the one thing the menu's map could not say. Everything
// first: it is where you talk, and where a press with nothing in mind should
// land. The four that hold things follow, in the order the menu carried them.
//
// Five rooms and not seven since 31 August 2026. The rail had been carrying
// two different kinds of thing under one shape: five were places you keep
// something, and two — the things you kept, what you set aside — were states a
// note is in, promoted to doors because there was nowhere else to put them. A
// state with a door on the rail reads as a fourth pile to stay on top of,
// which is the weight this product exists to remove. They are two chips inside
// the notes now; see shelfChips.
type room struct {
	// Key is the room in a URL and in the turns table.
	Key string
	// Name is what it is called. Never a hash-name: #chores carries Slack's
	// register and would bring its voice with it.
	Name string
	// Placeholder and Button are the dock, in this room.
	//
	// The button names the consequence rather than the act. The room decides
	// the filing with no confirmation step, which puts the whole weight of
	// "what will typing do here" on one control — and a grey placeholder
	// cannot carry it, because it is invisible by the third day. You read what
	// is about to happen at the moment you commit to it, which is the
	// confirmation without a press.
	Placeholder string
	Button      string
	// Action and Field are where the dock posts, and under what name.
	//
	// Three rooms are two-step and stay two-step: a chore needs a rhythm and
	// an appointment needs a day, and both pickers already exist. What the
	// dock replaces is the first step, which is a box asking for the words you
	// have already typed.
	Action string
	Field  string
}

var rooms = []room{
	// Named for what it holds rather than for who is in it. "general" was the
	// other candidate and is Slack's word — the same register #chores was
	// refused for when these rooms were built.
	{Key: "everything", Name: "everything",
		Placeholder: "what's going on?", Button: "Tell it",
		Action: "/capture", Field: "text"},
	{Key: "notes", Name: "the notes",
		Placeholder: "what is it", Button: "Put it in the notes",
		Action: "/capture", Field: "text"},
	{Key: "chores", Name: "the chores",
		Placeholder: "what comes back?", Button: "Make a chore",
		Action: "/chores/name", Field: "name"},
	{Key: "at", Name: "the agenda",
		Placeholder: "what is happening?", Button: "Put it in the agenda",
		Action: "/at/new", Field: "label"},
	{Key: "tasks", Name: "the tasks",
		Placeholder: "what did you decide?", Button: "Make a task",
		Action: "/tasks/new", Field: "text"},
}

// The two shelves, as they are reached now: a press inside the notes rather
// than a door on the rail.
//
// Nothing is kept or set aside by being typed. Both states are reached by
// deciding about a note that already exists, which is why neither was ever a
// place you could put something — a shelf with a dock had to borrow the pile's,
// and a shelf whose dock lied about its destination would have been worse.
//
// On the room's opening turn only. Two chips under every note would be two
// chips that are not about that note.
// One name for each, on the chip and on the place it draws. Two words for one
// shelf is two things as far as anybody reading the screen is concerned.
var shelves = []struct{ Key, Name string }{
	{"kept", "the things you kept"},
	{"held", "what you set aside"},
}

// placeName is what a place is called — the five rooms and the two shelves.
//
// A shelf is a place you can be shown and not a room you can go to, which is
// the whole distinction this change is about. Buddy can still open one by name
// and /open can still draw one; neither is a door on the rail.
func placeName(key string) (string, bool) {
	if name, ok := doorName(key); ok {
		return name, true
	}
	for _, sh := range shelves {
		if sh.Key == key {
			return sh.Name, true
		}
	}
	return "", false
}

// shelfChips is what the notes offer besides a new one.
func shelfChips() []turnChip {
	out := make([]turnChip, 0, len(shelves))
	for _, sh := range shelves {
		out = append(out, turnChip{
			Label: sh.Name, Action: "/notes/shelf",
			Fields: map[string]string{"shelf": sh.Key},
		})
	}
	return out
}

// shelfByKey is a shelf named by a form.
func shelfByKey(key string) bool {
	for _, sh := range shelves {
		if sh.Key == key {
			return true
		}
	}
	return false
}

func roomByKey(key string) (room, bool) {
	for _, r := range rooms {
		if r.Key == key {
			return r, true
		}
	}
	return room{}, false
}

// roomKey is the context key. Its own type, so nothing in any package can
// collide with it.
type roomKey struct{}

// roomOf is which room this request is in.
//
// Everything's when nobody said. A handler mounted outside a room still has to
// put its turn somewhere, and the somewhere is the room the whole conversation
// lived in before 28 August.
func roomOf(ctx context.Context) string {
	if r, ok := ctx.Value(roomKey{}).(string); ok && r != "" {
		return r
	}
	return "everything"
}

// withRoom is how the room gets onto the request.
//
// One place sets it and keepSaid is the one place that reads it. Thirty
// handlers append turns and threading a room parameter through all of them
// would be thirty chances to forget one — and a forgotten one is invisible,
// because a turn in the wrong room looks exactly like a room that is quiet.
// See TestOnlyKeepSaidPutsTurnsInARoom for the fence that keeps it that way.
func withRoom(key string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h(w, r.WithContext(withRoomIn(r.Context(), key)))
	}
}

// withRoomIn is the same thing for a handler that learns its room part way
// through — from a form field rather than from where it was mounted.
func withRoomIn(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, roomKey{}, key)
}

// roomRoute reads the room out of the path and puts it on the request before
// anything downstream can append a turn.
func roomRoute(s Store, opts Options) http.HandlerFunc {
	h := roomHandler(s, opts)
	return func(w http.ResponseWriter, r *http.Request) {
		withRoom(r.PathValue("room"), h)(w, r)
	}
}

// roomHandler is a room, entered.
//
// A GET, where a door was a POST. Entering a place is navigation and
// navigation must not write: the door appended "the pile" to the record on
// every press, which made a record of walking around rather than of anything
// said. The cost the door paid for being a POST — no new tab, no back through
// doors — comes back with it.
//
// What it does append is the room's own current state, and only when the
// conversation ends with nothing to act on. The same guard the offer uses on
// the thread, for the same reason: something already handed to you is
// something you already have.
func roomHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		here, ok := roomByKey(r.PathValue("room"))
		if !ok {
			// A typo, not a page. Not a redirect to Buddy: a URL that
			// silently becomes a different room is a URL you cannot trust in
			// a bookmark.
			http.NotFound(w, r)
			return
		}
		ctx := r.Context()

		var (
			turns []squirrel.Turn
			more  bool
			err   error
		)
		// `?before=` walks up this room's conversation, and it is in the
		// address bar for the same reason the thread's is: a page of the past
		// is a place you can send yourself back to.
		before, perr := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
		walkingBack := perr == nil && before > 0
		if walkingBack {
			turns, more, err = s.TurnsBefore(ctx, personID, here.Key, before, threadLimit)
		} else {
			turns, more, err = s.RecentTurns(ctx, personID, here.Key, threadLimit)
		}
		unreadable := err != nil
		if unreadable {
			slog.Error("reading a room", "room", here.Key, "error", err)
			turns, more = nil, false
		}

		// The room's own current state, drawn and not kept. Never while walking
		// back: reading the past must not put the present in the middle of it.
		//
		// There is no endsOpen guard any more and that is the whole fix. It
		// existed because the list was written into the record, so drawing it
		// twice meant saying the same thing twice — and refusing to draw it
		// meant a chore you had done was still on the screen asking. A list
		// that is drawn every time is current every time and duplicates
		// nothing, because it is nowhere to duplicate into.
		var edge []squirrel.Turn
		if !walkingBack && !unreadable {
			edge = roomEdge(ctx, s, opts, personID, here.Key)
		}

		// The edge on its own, for a press that has just changed what is in it.
		//
		// A press appends its answer to the conversation and leaves the list
		// above it describing the world as it was a moment ago — the same
		// staleness this change is about, one element down. The script asks
		// for this the moment a press lands.
		if r.Header.Get("X-Edge") == "1" {
			edgeOnly(w, r, atTheEdge(ctx, here.Key, edge))
			return
		}

		v := view{
			Room: here,
			// A conversation, but not the front door: the mark is a link back
			// to Buddy's room from here.
			Thread:    true,
			Here:      here.Key,
			Scrolling: true,
			Turns:     saidAndDone(turnViews(r.Context(), turns), len(edge) > 0),
			Edge:      atTheEdge(r.Context(), here.Key, edge),
			MoreAbove: more,
		}
		if len(turns) > 0 {
			v.Oldest = turns[0].ID
		}
		if unreadable {
			v.Turns = []turnView{{
				Buddy: true, Live: true, V: stamp(),
				Words: "I cannot reach what we said here. Tell me things anyway — they are kept, and they go in when I can.",
			}}
		}
		renderWith(w, r, s, opts, "thread", v)
	}
}

// railView is one room on the rail.
type railView struct {
	Key, Name string
	// Count is what is waiting, and zero draws no pill. Zero is no number and
	// not a nought: a room reading "0" is a scoreboard, which is the rule the
	// four doors carried and the rail inherits along with them.
	//
	// Kept against the evidence, deliberately, on 27 August 2026. A bad week
	// reads as a scoreboard, because pills in a column invite a total the eye
	// computes whether or not the product prints one — which is the shape
	// PRODUCT.md named when it retired the no-count rule. The trade is that
	// not knowing how much is waiting is its own weight.
	//
	// It reverses cleanly, which is why it is safe to have kept: delete the
	// Waiting read below and every number goes, because they are computed here
	// and stored nowhere.
	Count int
	// Current is the room you are in.
	Current bool
}

// roomsFor is the rail.
//
// Counts on the four that earn one. Everything carries none: nothing waits in
// the room you are standing in, and a number there would be counting the whole
// product at you on the way past.
func roomsFor(ctx context.Context, s Store, personID int64, here string) []railView {
	out := make([]railView, 0, len(rooms))
	for _, r := range rooms {
		out = append(out, railView{Key: r.Key, Name: r.Name, Current: r.Key == here})
	}
	waiting, err := s.Waiting(ctx, personID, now())
	if err != nil {
		// A count that cannot be read is a rail with no numbers on it, which
		// is what this was before 24 August. Everything still goes where it
		// goes.
		slog.Error("counting what is waiting, for the rail", "error", err)
		return out
	}
	for i := range out {
		switch out[i].Key {
		case "notes":
			out[i].Count = waiting.Pile
		case "at":
			out[i].Count = waiting.Agenda
		case "tasks":
			out[i].Count = waiting.Tasks
		case "chores":
			out[i].Count = waiting.Chores
		}
	}
	return out
}

// fromTheDock reads the room out of the posted form and puts it on the
// request.
//
// Every press carries the room it was made in, because the path cannot: there
// is one route per destination, not one per room. /pile/act is the same URL
// whether you pressed DROP in the pile or in the agenda, and turn.html puts
// the room on every form for exactly that reason.
//
// Without this the answer lands in Buddy's room, which is what roomOf falls
// back to, and nothing on the screen says so — the room you were in simply
// keeps the offer it already had, buttons and all, so a decision looks like it
// was taken and then forgotten. This was the dock's alone until 30 August 2026
// and it was wrong for every card action for as long as rooms have existed.
//
// r.ParseForm is idempotent, so the handlers' own calls still work — but it
// consumes a body it believes is a form, so a route whose body is JSON must not
// come through here. /push/subscribe is the only one and is mounted without it.
func inTheRoomItCameFrom(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err == nil {
			if _, ok := roomByKey(r.FormValue("room")); ok {
				withRoom(r.FormValue("room"), h)(w, r)
				return
			}
		}
		h(w, r)
	}
}

// backToTheRoom is where a press goes when the answer cannot be a fragment.
//
// Every one of these said "/", which is Buddy's room and was the only room
// there was until 28 August 2026. Typing in the agenda's dock filed the
// appointment in the agenda and then put you in Buddy, which reads as the
// press having gone to the wrong place — it had not; the way back had.
func backToTheRoom(r *http.Request) string {
	where := roomOf(r.Context())
	if where == "" {
		where = "everything"
	}
	return "/r/" + where
}

// roomEdge is what a room is showing you right now: its list, and the way to
// make one more.
//
// Its own function because it is what the room means, and because a test about
// what a room draws should be asking the thing the room asks rather than
// reading a record that no longer holds it.
func roomEdge(ctx context.Context, s Store, opts Options, personID int64, where string) []squirrel.Turn {
	reply, has := placeSaid(ctx, s, opts, personID, where, 0)
	if !has {
		return nil
	}
	return []squirrel.Turn{alsoOffer(reply, newChipFor(where)...)}
}
