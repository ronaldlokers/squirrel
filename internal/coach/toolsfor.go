package coach

// What each room lets Buddy do.
//
// The narrowing is the point of a room. A room is both a place that keeps its
// own conversation and a scope on Buddy — see
// docs/superpowers/specs/2026-08-28-rooms-design.md — and a room that kept its
// own history while answering with the whole product would be a filter on a
// transcript.
//
// A table rather than seven constructors: "what may the chores do" should be
// answerable by reading one screen, and it is the question this whole feature
// is. Buddy's own room is absent on purpose and is handled below as the room
// that is not narrowed.
//
// Two shapes here look like omissions and are not:
//
//   - The agenda cannot write. An appointment is a fixed point and the
//     product's rule 1 is to prefer it, so a model that can move one can move
//     the thing everything else is arranged around. It proposes, and you press.
//   - The two shelves cannot write at all. The way off a shelf is a card's own
//     button, which is what the shelf turns already draw.
var roomTools = map[string]struct {
	// Tools is every tool name this room may use, reads and writes together.
	Tools []string
	// Propose narrows propose's `do` enum, and Refuse narrows refuse's `kind`.
	// A propose offered in the agenda that still accepts do: "chore" is a
	// narrowing that only looks like one.
	Propose []string
	Refuse  []string
}{
	"pile": {
		Tools:   []string{"now", "item", "open_work", "typically", "create_task", "propose", "say", "open"},
		Propose: []string{"chore", "moment", "drop"},
	},
	"chores": {
		Tools:   []string{"now", "item", "open_work", "typically", "complete_chore", "snooze_chore", "propose", "say", "open"},
		Propose: []string{"chore", "retire"},
	},
	"tasks": {
		Tools:  []string{"now", "item", "open_work", "typically", "lately", "complete", "create_task", "start_timer", "refuse", "say", "open"},
		Refuse: []string{"task"},
	},
	"at": {
		Tools:   []string{"now", "next_fixed", "propose", "say", "open"},
		Propose: []string{"moment"},
	},
	"held": {Tools: []string{"now", "item", "say", "open"}},
	"kept": {Tools: []string{"now", "item", "say", "open"}},
}

// mayUse says this room is allowed this tool.
//
// The other half of the narrowing, and the half that would be forgotten.
// Offering fewer tools is most of it and all of the token saving, but a model
// can name a function that was never in its list, and providers do — so the
// check has to live where the call is dispatched as well as where the request
// is built.
func mayUse(room, tool string) bool {
	if RoomName(room) == "" {
		// Buddy's own room, or a room this package does not know. Not
		// narrowed: the fallback has to be the room you talk in, because a
		// turn that arrived without a room is a turn from a surface that has
		// none.
		return true
	}
	for _, name := range roomTools[room].Tools {
		if name == tool {
			return true
		}
	}
	return false
}

// toolsFor is the tools this room's Buddy is offered.
func toolsFor(room string, canOpen bool) []map[string]any {
	all := append(append([]map[string]any{}, readTools()...), writeTools...)
	if canOpen {
		all = append(all, openFor(room))
	}
	if RoomName(room) == "" {
		// Buddy's own room keeps everything. It is where you talk, and the
		// narrowing is what the other six are.
		return all
	}
	narrowed := roomTools[room]
	out := make([]map[string]any, 0, len(narrowed.Tools))
	for _, s := range all {
		name, _ := s["function"].(map[string]any)["name"].(string)
		if !mayUse(room, name) {
			continue
		}
		switch name {
		case "propose":
			s = withEnum(s, "do", narrowed.Propose)
		case "refuse":
			s = withEnum(s, "kind", narrowed.Refuse)
		}
		out = append(out, s)
	}
	return out
}

// openFor is the place this room can draw, which is itself.
//
// Buddy's own room keeps all six: it is the room you talk in, and asking to see
// something there is the ask this tool was written for. Inside a room it is
// close to redundant — the rail is one press — but it is what happens when
// somebody asks to see the chores while standing in them, and drawing the room
// is the right answer to that.
func openFor(room string) map[string]any {
	if RoomName(room) == "" {
		return openTool
	}
	return withEnum(openTool, "where", []string{room})
}

// withEnum copies a spec with one argument's enum replaced.
//
// A copy, and it has to be: the specs are package-level maps shared by every
// turn, and narrowing one in place would narrow it for every room after.
func withEnum(s map[string]any, arg string, values []string) map[string]any {
	fn, _ := s["function"].(map[string]any)
	params, _ := fn["parameters"].(map[string]any)
	props, _ := params["properties"].(map[string]any)
	was, _ := props[arg].(map[string]any)

	nextProp := map[string]any{}
	for k, v := range was {
		nextProp[k] = v
	}
	nextProp["enum"] = values

	nextProps := map[string]any{}
	for k, v := range props {
		nextProps[k] = v
	}
	nextProps[arg] = nextProp

	nextParams := map[string]any{}
	for k, v := range params {
		nextParams[k] = v
	}
	nextParams["properties"] = nextProps

	nextFn := map[string]any{}
	for k, v := range fn {
		nextFn[k] = v
	}
	nextFn["parameters"] = nextParams

	return map[string]any{"type": s["type"], "function": nextFn}
}

// onlyKind is the room's own kind of work, or all of it.
//
// The chores and the tasks both read open_work and it returns both kinds. The
// chores being handed a task is the narrowing failing quietly: the model is
// told the task exists, which is the fact the room was drawn to keep out.
func onlyKind(room string, work []Work) []Work {
	want := map[string]string{"chores": "chore", "tasks": "task"}[room]
	if want == "" {
		return work
	}
	out := make([]Work, 0, len(work))
	for _, w := range work {
		if w.Kind == want {
			out = append(out, w)
		}
	}
	return out
}
