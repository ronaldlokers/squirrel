package coach

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// A conversational turn that can act.
//
// The write tools are the ones that pass the policy's test — already a
// button, undone by one press — plus the four kinds that do not, which the
// model may only propose.
//
// Nothing here can reword a note, touch a check-in, or delete a row. Those are
// not omissions to be filled in later: rewriting your own words is what `!fix`
// is for and it is yours alone; how you feel is said by you and never
// inferred; and nothing in this product deletes, it retracts.

// writeTools are the ones that run when asked, and the one that proposes.
//
// The proposal is a single tool with a kind rather than four tools, so the
// four things that need confirming are unmistakably one category in the wire
// description as well as in the policy.
var writeTools = []map[string]any{
	spec("complete", "Mark a task done. Only ever a task id a tool returned.",
		map[string]any{"item_id": map[string]any{"type": "integer"}}),
	spec("complete_chore", "Record that a chore was done. Only ever a chore id a tool returned.",
		map[string]any{"chore_id": map[string]any{"type": "integer"}}),
	spec("start_timer", "Start the body double on something, for a number of minutes.",
		map[string]any{
			"label":   map[string]any{"type": "string"},
			"minutes": map[string]any{"type": "integer"},
		}),
	spec("refuse", "Turn something down for today. It is offered again tomorrow.",
		map[string]any{
			"kind":   map[string]any{"type": "string", "enum": []string{"task", "chore"}},
			"ref_id": map[string]any{"type": "integer"},
		}),
	spec("snooze_chore", "Stop raising a chore for a number of hours.",
		map[string]any{
			"chore_id": map[string]any{"type": "integer"},
			"hours":    map[string]any{"type": "integer"},
		}),
	spec("create_task", "Record something they have decided to do.",
		map[string]any{"text": map[string]any{"type": "string"}}),
	spec("propose", "Ask before doing something that interrupts them later, recurs forever, stops something recurring, or throws something away.",
		map[string]any{
			"do":     map[string]any{"type": "string", "enum": []string{"moment", "chore", "retire", "drop"}},
			"said":   map[string]any{"type": "string", "description": "One sentence, what you want to do and why."},
			"text":   map[string]any{"type": "string", "description": "The label or name, for a moment or a chore."},
			"at":     map[string]any{"type": "string", "description": "When a moment starts, in their words: \"14:30\", \"tomorrow 09:00\"."},
			"every":  map[string]any{"type": "string", "description": "A chore's rhythm, in their words: \"every 2 weeks\"."},
			"ref_id": map[string]any{"type": "integer", "description": "The row a retirement or a drop names."},
		}),
	spec("say", "Answer without doing anything. Use this whenever nothing needs doing.",
		map[string]any{"text": map[string]any{"type": "string"}}),
}

// openTool shows them one of their places, whole.
//
// It exists because the coach is forbidden to recite a list — Guard refuses
// bullets, numbers and headings, and the brief is two sentences — and asking
// to see your tasks is one of the most ordinary things anybody says here. The
// coach knew the answer, was not allowed to type it, and said it could not
// help. It could: the same places the menu opens are turns full of cards, and
// this is the press.
//
// Not a write, and it asks no permission, because it changes nothing. Say
// something as well: this draws the place, it does not answer the question.
var openTool = spec("open",
	"Show them one of their places, whole, as cards they can act on. Use this whenever they ask to see something rather than asking about it. Say something as well.",
	map[string]any{
		"where": map[string]any{
			"type":        "string",
			"enum":        []string{"notes", "tasks", "chores", "at", "held", "kept"},
			"description": "pile: everything untriaged. tasks: what is open. chores: what comes back. at: what is at a fixed time. held: what they set aside. kept: what they kept for later.",
		},
	})

// actPreamble is added when a turn can act.
//
// The last line is the one that matters. A model given tools will use them,
// and most of what someone says to a coach wants an answer rather than an
// action — so saying nothing happened has to be an explicit, easy thing to do
// rather than the absence of a choice.
const actPreamble = `

You can act on their behalf. Only ever on something a tool returned in
this conversation.

Some things you must ask about first: anything that will interrupt them
later, anything that comes back on its own forever, anything that stops
something recurring, and anything that throws something away. Use propose
for those, and say in one sentence what you want to do.

Most of the time nothing needs doing. Call say and answer them.`

// inTheRoom is what Buddy is told about where he is.
//
// Named rather than described. "You are in the chores" and a toolset that only
// touches chores agree with each other, so the model reading both is told the
// same thing twice rather than handed a rule to follow — and the tools are the
// half that is enforced.
//
// Buddy's own room says nothing. It is not a room he is confined to; it is
// where he is, and every other room is the narrowing.
func inTheRoom(room string) string {
	name := RoomName(room)
	if name == "" {
		return ""
	}
	return "\n\nYou are in " + name + ". Everything here is about " + name +
		", and the tools you have are the ones that room uses. If they ask " +
		"about something that lives somewhere else, say where it is rather " +
		"than answering for it."
}

// actRounds is how many times a turn may go round. Two, not three: a
// conversational turn is not a search — it has already been handed what is on
// screen — and a loop is a place for a model to talk itself into acting.
const actRounds = 2

// answerActing is Answer for a turn that has hands.
//
// It returns the same Reply the plain path does, plus a proposal when there is
// one. Everything about it degrades to the plain path and from there to the
// deterministic floor: no hands, no tools; no tool call, no action.
func (p *Provider) answerActing(ctx context.Context, t Turn) (Reply, error) {
	now := p.now()
	permit, err := p.Budget.Ask(ctx, t.PersonID, now, "the deterministic answers take over")
	if err != nil {
		return Reply{}, ErrUnavailable
	}
	defer permit.Release()

	model := p.modelFor(t)
	handed := map[string]Work{}
	msgs := p.messages(t)
	msgs[0].Content += actPreamble + inTheRoom(t.Room)

	// Narrowed to the room. See toolsfor.go — this is the half of a room that
	// makes it worth having, and Turn.CanOpen is still what decides whether a
	// place can be drawn at all.
	tools := toolsFor(t.Room, t.CanOpen)

	var inTotal, outTotal int
	var reply Reply
	var proposal *Proposal
	var acted []string
	var open string

	defer func() {
		if inTotal+outTotal == 0 {
			return
		}
		if err := p.Budget.Record(ctx, t.PersonID, Answer{
			Kind: t.Kind, Model: model, Prompt: t.Said, Reply: reply.Text,
			InTokens: inTotal, OutTokens: outTotal, Used: reply.Text != "", At: now,
		}); err != nil {
			slog.Error("recording what the coach said", "error", err)
		}
	}()

	for round := 0; round < actRounds; round++ {
		text, calls, in, out, err := p.completionWithTools(ctx, permit, model, msgs, tools)
		inTotal += in
		outTotal += out
		if err != nil {
			slog.Error("the coach", "kind", t.Kind, "model", model,
				"why", Why(err), "error", err)
			return Reply{}, err
		}

		if len(calls) == 0 {
			// It answered in prose without calling say. That is fine and it is
			// the ordinary shape of the plain path — guard it and use it.
			guarded, ok := Guard(text)
			if !ok {
				slog.Warn("the coach said something the wrong shape; using the fixed answer",
					"kind", t.Kind, "model", model, "said", text)
				return Reply{}, ErrUnavailable
			}
			reply = Reply{
				Text: guarded, Model: model, InTokens: inTotal, OutTokens: outTotal,
				Did: acted, Propose: proposal, Open: open,
			}
			return reply, nil
		}

		msgs = append(msgs, chatMessage{Role: "assistant", Calls: calls})

		for _, call := range calls {
			switch call.Function.Name {
			case "say":
				guarded, ok := Guard(saidIn(call))
				if !ok {
					slog.Warn("the coach said something the wrong shape; using the fixed answer",
						"kind", t.Kind, "model", model)
					return Reply{}, ErrUnavailable
				}
				reply = Reply{
					Text: guarded, Model: model, InTokens: inTotal, OutTokens: outTotal,
					Did: acted, Propose: proposal, Open: open,
				}
				return reply, nil

			case "open":
				// Taken only when the surface offered it, and only for a
				// place that exists. A model naming something else is told so
				// and gets to try again, the same as any other refusal here.
				if where, ok := openIn(call); ok && t.CanOpen {
					open = where
					msgs = append(msgs, chatMessage{
						Role: "tool", ToolCallID: call.ID,
						Content: `{"shown":true}`,
					})
					break
				}
				msgs = append(msgs, chatMessage{
					Role: "tool", ToolCallID: call.ID,
					Content: refused("that is not one of their places"),
				})

			case "propose":
				proposal = proposalIn(call)
				msgs = append(msgs, chatMessage{
					Role: "tool", ToolCallID: call.ID,
					Content: `{"asked":true}`,
				})

			default:
				said, done := p.runTool(ctx, t.PersonID, t.Room, call, handed)
				if done != "" {
					acted = append(acted, done)
				}
				msgs = append(msgs, chatMessage{
					Role: "tool", ToolCallID: call.ID, Content: said,
				})
			}
		}
	}

	slog.Warn("the coach used its turn on tools and never answered", "kind", t.Kind)
	return Reply{}, ErrUnavailable
}

// runTool runs one tool and reports what it said back, plus a plain sentence
// naming what actually changed when something did.
//
// The second return is what the surface shows underneath the reply. A model
// saying "done" is not evidence anything happened; a line the application
// wrote after the write succeeded is.
func (p *Provider) runTool(ctx context.Context, personID int64, room string, call toolCall, handed map[string]Work) (string, string) {
	// Offering fewer tools is most of the narrowing; refusing the ones the
	// room was not offered is the rest. A model can name a function that was
	// never in its list, and providers do — so a narrowing that lives only in
	// the request is not the only thing between the chores and a deleted task.
	if !mayUse(room, call.Function.Name) {
		slog.Warn("a tool was called in a room that does not have it",
			"room", room, "tool", call.Function.Name)
		return refused("that is not something this room does"), ""
	}
	if p.Hands == nil || !writes[call.Function.Name] {
		return p.answerTool(ctx, personID, room, call, handed), ""
	}

	var args struct {
		ItemID  int64  `json:"item_id"`
		ChoreID int64  `json:"chore_id"`
		RefID   int64  `json:"ref_id"`
		Label   string `json:"label"`
		Text    string `json:"text"`
		Kind    string `json:"kind"`
		Minutes int    `json:"minutes"`
		Hours   int    `json:"hours"`
	}
	_ = json.Unmarshal([]byte(call.Function.Arguments), &args)

	switch call.Function.Name {
	case "complete":
		if _, ok := handed[key("task", args.ItemID)]; !ok {
			return refused("that is not something you were shown"), ""
		}
		what, err := p.Hands.Complete(ctx, personID, args.ItemID)
		return outcome(err), didSay(err, what+" is done")

	case "complete_chore":
		if _, ok := handed[key("chore", args.ChoreID)]; !ok {
			return refused("that is not something you were shown"), ""
		}
		what, err := p.Hands.CompleteChore(ctx, personID, args.ChoreID)
		return outcome(err), didSay(err, what+" is done")

	case "start_timer":
		if args.Minutes < 1 || args.Minutes > 180 {
			return refused("a timer is between 1 and 180 minutes"), ""
		}
		err := p.Hands.StartTimer(ctx, personID, args.Label, args.Minutes)
		return outcome(err), didSay(err,
			fmt.Sprintf("%d minutes on %s, running", args.Minutes, args.Label))

	case "refuse":
		if _, ok := handed[key(args.Kind, args.RefID)]; !ok {
			return refused("that is not something you were shown"), ""
		}
		err := p.Hands.Refuse(ctx, personID, args.Kind, args.RefID)
		return outcome(err), didSay(err, "put off until tomorrow")

	case "snooze_chore":
		if _, ok := handed[key("chore", args.ChoreID)]; !ok {
			return refused("that is not something you were shown"), ""
		}
		what, err := p.Hands.SnoozeChore(ctx, personID, args.ChoreID, args.Hours)
		return outcome(err), didSay(err, what+" is quiet for a while")

	case "create_task":
		text := strings.TrimSpace(args.Text)
		if text == "" {
			return refused("a task needs words"), ""
		}
		err := p.Hands.CreateTask(ctx, personID, text)
		return outcome(err), didSay(err, "kept as a task: "+text)
	}

	return "{}", ""
}

// writes names the tools that change something, so runTool can tell them from
// the reads without a second switch that could disagree with the first.
var writes = map[string]bool{
	"complete": true, "complete_chore": true, "start_timer": true,
	"refuse": true, "snooze_chore": true, "create_task": true,
}

func readTools() []map[string]any {
	out := make([]map[string]any, 0, len(toolSpecs)-1)
	for _, spec := range toolSpecs {
		fn, _ := spec["function"].(map[string]any)
		if name, _ := fn["name"].(string); name == "choose" {
			continue
		}
		out = append(out, spec)
	}
	return out
}

func outcome(err error) string {
	if err != nil {
		// Told, not hidden. A model that thinks it acted will say so, and
		// saying so when nothing happened is the worst failure available here.
		return `{"done":false,"why":"it did not work"}`
	}
	return `{"done":true}`
}

func didSay(err error, what string) string {
	if err != nil {
		return ""
	}
	return what
}

func refused(why string) string {
	raw, _ := json.Marshal(map[string]any{"done": false, "why": why})
	return string(raw)
}

func saidIn(call toolCall) string {
	var args struct {
		Text string `json:"text"`
	}
	_ = json.Unmarshal([]byte(call.Function.Arguments), &args)
	return args.Text
}

// places is the vocabulary the open tool accepts, checked here as well as
// stated in the enum.
//
// A map rather than a switch, so a name that is not one of them is a lookup
// miss instead of a default branch: the same device doorNames uses on the
// screen. The screen holds the same six and is the only thing that draws them
// — this list exists so a model naming "inbox" is refused here rather than
// producing an empty turn there.
var places = map[string]bool{
	"notes": true, "tasks": true, "chores": true, "at": true, "held": true, "kept": true,
}

// openIn reads which place, and reports whether it is one.
func openIn(call toolCall) (string, bool) {
	var args struct {
		Where string `json:"where"`
	}
	_ = json.Unmarshal([]byte(call.Function.Arguments), &args)
	return args.Where, places[args.Where]
}

// proposalIn reads a proposal, or nil.
//
// The `said` sentence goes through the same guard a reply does: it is a
// sentence the person reads before pressing something, so it is held to the
// same shape as everything else they read here.
func proposalIn(call toolCall) *Proposal {
	var args struct {
		Do    string `json:"do"`
		Said  string `json:"said"`
		Text  string `json:"text"`
		At    string `json:"at"`
		Every string `json:"every"`
		RefID int64  `json:"ref_id"`
	}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return nil
	}
	if !KnownProposal(args.Do) {
		return nil
	}
	said, ok := Guard(args.Said)
	if !ok {
		return nil
	}
	return &Proposal{
		Do: args.Do, Said: said, Text: strings.TrimSpace(args.Text),
		At: strings.TrimSpace(args.At), Every: strings.TrimSpace(args.Every),
		RefID: args.RefID,
	}
}
