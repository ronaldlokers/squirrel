package coach

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func namesOf(t *testing.T, specs []map[string]any) []string {
	t.Helper()
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		fn, ok := s["function"].(map[string]any)
		require.True(t, ok)
		name, ok := fn["name"].(string)
		require.True(t, ok)
		out = append(out, name)
	}
	return out
}

func TestTheChoresCannotTouchATask(t *testing.T) {
	got := namesOf(t, toolsFor("chores", true))
	require.NotContains(t, got, "complete", "the chores can complete a task")
	require.NotContains(t, got, "create_task", "the chores can make a task")
	require.NotContains(t, got, "start_timer")
	require.Contains(t, got, "complete_chore")
	require.Contains(t, got, "snooze_chore")
	require.Contains(t, got, "say")
}

// An appointment is a fixed point and rule 1 is to prefer it, so a model that
// can move one can move the thing everything else is arranged around. It asks,
// and you press.
func TestTheAgendaCannotWrite(t *testing.T) {
	got := namesOf(t, toolsFor("at", true))
	require.NotContains(t, got, "complete")
	require.NotContains(t, got, "complete_chore")
	require.NotContains(t, got, "start_timer")
	require.NotContains(t, got, "create_task")
	require.Contains(t, got, "propose",
		"the agenda cannot ask either, which leaves it able to do nothing")
	require.Contains(t, got, "say")
}

// The way off a shelf is a card's own button. Nothing here writes.
// A shelf is not a room and is narrowed by nothing, because there is nothing
// left to narrow: it is drawn inside the notes, under the notes' own toolset.
// It had two of its own until 31 August 2026, when the rail stopped carrying a
// note's state as if it were a place.
func TestAShelfIsNotARoom(t *testing.T) {
	for _, shelf := range []string{"held", "kept"} {
		require.Empty(t, RoomName(shelf), "%s is still a room the coach narrows in", shelf)
		_, narrowed := roomTools[shelf]
		require.False(t, narrowed, "%s still has a narrowing of its own", shelf)
	}
}

// Everything is the room that is not narrowed. It is where you talk, and the
// other four are the narrowing.
func TestBuddysOwnRoomKeepsEverything(t *testing.T) {
	all := namesOf(t, toolsFor("everything", true))
	for _, name := range namesOf(t, append(readTools(), writeTools...)) {
		require.Contains(t, all, name, "Buddy's own room lost %q", name)
	}
	require.Contains(t, all, "open")
}

// The half that would be forgotten. A model can name a function that was never
// in its list, and providers do, so the narrowing has to be enforced where the
// call is dispatched and not only where the request is built.
func TestAToolTheRoomWasNotOfferedIsRefused(t *testing.T) {
	require.False(t, mayUse("chores", "complete"))
	require.False(t, mayUse("notes", "complete_chore"))
	require.False(t, mayUse("at", "start_timer"))
	require.True(t, mayUse("chores", "complete_chore"))
	require.True(t, mayUse("everything", "complete"))
}

// A narrowing that leaves the wide enum in place only looks like one.
func TestTheAgendaMayOnlyProposeAMoment(t *testing.T) {
	require.Equal(t, []string{"moment"}, enumOf(t, toolsFor("at", true), "propose", "do"))
	require.Equal(t, []string{"chore", "retire"},
		enumOf(t, toolsFor("chores", true), "propose", "do"))
	require.Equal(t, []string{"task"},
		enumOf(t, toolsFor("tasks", true), "refuse", "kind"))
}

// And narrowing one room's copy must not narrow the next room's. The specs are
// package-level maps shared by every turn.
func TestNarrowingOneRoomLeavesTheSpecsAlone(t *testing.T) {
	_ = toolsFor("at", true)
	require.Equal(t, []string{"moment", "chore", "retire", "drop"},
		enumOf(t, toolsFor("buddy", true), "propose", "do"),
		"the agenda's narrowing leaked into every later turn")
}

// A room drawn on the screen and forgotten here would silently get Buddy's
// whole toolset, which is the failure the narrowing exists to prevent.
func TestEveryRoomHasAToolset(t *testing.T) {
	for _, room := range RoomKeys() {
		require.NotEmpty(t, roomTools[room].Tools, "%s has no toolset", room)
		require.NotEmpty(t, toolsFor(room, true), "%s is offered nothing", room)
		require.Less(t, len(toolsFor(room, true)), len(toolsFor("buddy", true)),
			"%s is not narrowed at all", room)
	}
}

func TestTheChoresAreNotHandedATask(t *testing.T) {
	work := []Work{
		{ID: 1, Kind: "task", Text: "a task"},
		{ID: 2, Kind: "chore", Text: "a chore"},
	}
	require.Equal(t, []Work{{ID: 2, Kind: "chore", Text: "a chore"}}, onlyKind("chores", work))
	require.Equal(t, []Work{{ID: 1, Kind: "task", Text: "a task"}}, onlyKind("tasks", work))
	require.Len(t, onlyKind("buddy", work), 2, "Buddy's room lost half its work")
}

// Asking to see something inside a room shows that room. Buddy's keeps all six
// — it is the room the tool was written for.
func TestOpenIsTheRoomYouAreIn(t *testing.T) {
	require.Equal(t, []string{"chores"}, enumOf(t, toolsFor("chores", true), "open", "where"))
	require.Len(t, enumOf(t, toolsFor("buddy", true), "open", "where"), 6)
}

func enumOf(t *testing.T, specs []map[string]any, tool, arg string) []string {
	t.Helper()
	for _, s := range specs {
		fn := s["function"].(map[string]any)
		if fn["name"] != tool {
			continue
		}
		props := fn["parameters"].(map[string]any)["properties"].(map[string]any)
		switch e := props[arg].(map[string]any)["enum"].(type) {
		case []string:
			return e
		case []any:
			out := make([]string, 0, len(e))
			for _, v := range e {
				out = append(out, v.(string))
			}
			return out
		}
		t.Fatalf("%s has no enum on %s", tool, arg)
	}
	t.Fatalf("no %s in this room", tool)
	return nil
}

// Buddy is told where he is, in the room's own name.
//
// The tools are the half that is enforced and this is the half that is said,
// and they have to agree: a model given only the chores' tools and told
// nothing spends a round discovering what it cannot do.
func TestBuddyIsToldWhichRoomHeIsIn(t *testing.T) {
	require.Contains(t, inTheRoom("chores"), "the chores")
	require.Contains(t, inTheRoom("at"), "the agenda")
	require.Contains(t, inTheRoom("notes"), "the notes")

	// His own room says nothing. It is not a room he is confined to.
	require.Empty(t, inTheRoom("everything"))
	require.Empty(t, inTheRoom("kept"), "a shelf is not a room he is in")
	require.Empty(t, inTheRoom(""))
	require.Empty(t, inTheRoom("nowhere"))
}

// Every room this package narrows in can say its own name, and every room it
// can name it narrows in. Either half alone is a room that half-exists.
func TestEveryRoomTheCoachNarrowsInCanSayItsName(t *testing.T) {
	for _, room := range RoomKeys() {
		require.NotEmpty(t, RoomName(room), "%s has no name", room)
		require.NotEmpty(t, inTheRoom(room), "%s cannot say where it is", room)
		require.NotEmpty(t, roomTools[room].Tools, "%s has no toolset", room)
	}
	for room := range roomTools {
		require.NotEmpty(t, RoomName(room), "%s has a toolset and no name", room)
	}
}
