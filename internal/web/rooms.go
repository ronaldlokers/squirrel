package web

import (
	"context"
	"net/http"
)

// The rooms.
//
// The screen became one conversation on 24 August 2026 and seven rooms on
// 28 August; see docs/superpowers/specs/2026-08-28-rooms-design.md. A room is
// both a place that keeps its own conversation and a scope that narrows what
// Buddy can do in it.
//
// A slice rather than a map, because the rail has an order and a map has none
// — the order was the one thing the menu's map could not say. Buddy first: he
// is where you talk, and where a press with nothing in mind should land. The
// four that hold things next, in the order the menu carried them. The two
// shelves last, because they are where a decision has already been made.
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
	{Key: "buddy", Name: "Buddy",
		Placeholder: "what's going on?", Button: "Tell it",
		Action: "/capture", Field: "text"},
	{Key: "pile", Name: "the pile",
		Placeholder: "what is it", Button: "Put it in the pile",
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
	// The two shelves. Nothing is kept or set aside by being typed — both
	// states are reached by deciding about a note that already exists. A shelf
	// with no dock would be the one screen where the thumb has nowhere to go,
	// and a shelf whose dock lied about its destination would be worse than
	// either. So the room still decides, and a shelf's decision is the pile.
	{Key: "held", Name: "what you set aside",
		Placeholder: "what is it", Button: "Put it in the pile",
		Action: "/capture", Field: "text"},
	{Key: "kept", Name: "the things you kept",
		Placeholder: "what is it", Button: "Put it in the pile",
		Action: "/capture", Field: "text"},
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
// Buddy's when nobody said. A handler mounted outside a room still has to put
// its turn somewhere, and the somewhere is the room the whole conversation
// lived in before 28 August.
func roomOf(ctx context.Context) string {
	if r, ok := ctx.Value(roomKey{}).(string); ok && r != "" {
		return r
	}
	return "buddy"
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
		h(w, r.WithContext(context.WithValue(r.Context(), roomKey{}, key)))
	}
}
