package squirrel

// What a tap is, kept apart from what a sentence is.
//
// A tap carries no words to parse and earns no reply — the boost is the receipt —
// and what it does is turn a press into a state assertion against a row. Nothing
// here knows which surface delivered the press, and nothing here should learn.
//
// Adjacent to apply.go's command handling, it looked like more of the same. It is
// not, and this is the half that has to keep working unchanged as the room
// becomes best-effort.

import (
	"context"
	"encoding/json"
	"time"
)

// isActionPayload reports whether the payload really came from an action webhook.
// ParseAction matches on text alone, and someone typing "!action 451 done:2 true"
// produces byte-identical text — the payload's "type" is the only thing telling
// them apart. Anything that fails to unmarshal goes down the normal matcher, so a
// thought is never rejected.
func isActionPayload(payload json.RawMessage) bool {
	var p struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return false
	}
	return p.Type == "action"
}

// isTap reports whether item is a genuine tap rather than a lookalike message.
// Text alone is never enough — see isActionPayload — so every caller that must
// tell the two apart checks both.
func isTap(item Item) bool {
	_, ok := ParseAction(item.RawText)
	return ok && isActionPayload(item.Payload)
}

// applyAction applies a tap as a state assertion rather than a delta: applying
// either twice lands in the same place, which is what makes a retried delivery
// harmless — the payload carries no event id.
//
// Every path here is silent. The boost is the receipt; a reply per tap would make
// the room unreadable.
func (a *Applier) applyAction(ctx context.Context, in ActionIntent, personID int64) error {
	// A mood is answered before anything is looked up: it is not about a chore,
	// so there is no prompt line to resolve and no position to mean anything.
	if in.Kind == "mood" {
		if !in.Selected {
			// Deselecting is not a mood. Nothing to record and nothing to
			// undo — the reading simply stands until the next one.
			return nil
		}
		m, ok := ParseMood(in.Mood)
		if !ok {
			return nil
		}
		return a.store.RecordCheckin(ctx, personID, m, "chat", time.Now())
	}

	prompt, ok, err := a.store.PromptByMessageID(ctx, personID, in.MessageID)
	if err != nil {
		return err
	}
	if !ok {
		// Someone else's prompt, or one we have no record of. Nothing to do,
		// and nothing to say: the capture is stored either way.
		return nil
	}

	// Resolved as a line rather than a chore, because since the picker a button can
	// sit over a task. A `snooze` on a task cannot happen from any surface Squirrel
	// prints, so it is a no-op rather than an error.
	line, ok, err := a.store.LineOnPrompt(ctx, prompt.ID, in.Position)
	if err != nil || !ok {
		return err
	}
	if line.Item != nil {
		return a.applyItemAction(ctx, in, personID, prompt, *line.Item)
	}
	c := *line.Chore

	switch in.Kind {
	case "undefine":
		// DefinedMessage uses selection_mode "single", so deselecting delivers
		// "selected: false" for the same value. Only a selected tap is the correction
		// being asked for.
		if !in.Selected {
			return nil
		}
		return a.store.DeactivateChore(ctx, c.ID)

	case "snooze":
		// "Not today", pressed. Tomorrow rather than a duration, because the label is the
		// duration. Untapping clears it, the same shape as an unselected done retracting
		// a completion.
		when := tomorrow(time.Now())
		if !in.Selected {
			when = time.Now().Add(-time.Minute)
		}
		_, err := a.store.SnoozeChore(ctx, c.ID, personID, when)
		return err

	case "later":
		// The picker's refusal, on a chore. It changes nothing about the chore
		// — the clock runs, the nudge keeps its own budget — and it only tells
		// the picker not to hand you this one again today. Untapping takes it
		// back, like everything else here.
		if !in.Selected {
			return a.store.UnrefuseToday(ctx, personID, OfferChore, c.ID, time.Now())
		}
		return a.store.Refuse(ctx, personID, OfferChore, c.ID, time.Now())

	case "done":
		if !in.Selected {
			_, err := a.store.RetractCompletion(ctx, c.ID, personID, prompt.ID, time.Now())
			return err
		}
		done, err := a.store.CompletedSince(ctx, c.ID, prompt.ID)
		if err != nil || done {
			return err
		}
		if err := a.store.RecordCompletion(ctx, c.ID, personID, "tap", time.Now()); err != nil {
			return err
		}
		a.react(ctx, prompt)
		return nil
	}
	return nil
}

// applyItemAction is a tap that landed on a note or a task rather than a chore,
// because the picker can put a ✅ over something you decided to do. Same store
// call and same reversal as the tasks screen's own "did it".
func (a *Applier) applyItemAction(ctx context.Context, in ActionIntent, personID int64, prompt Prompt, it Item) error {
	switch in.Kind {
	case "later":
		if !in.Selected {
			return a.store.UnrefuseToday(ctx, personID, OfferTask, it.ID, time.Now())
		}
		return a.store.Refuse(ctx, personID, OfferTask, it.ID, time.Now())

	case "done":
		// A state assertion rather than a delta, as for a chore. Untapping returns it to
		// `open`; the kind is untouched, because undoing a completion is not undoing the
		// decision.
		if !in.Selected {
			return a.store.SetItemState(ctx, it.ID, ItemOpen, time.Now())
		}
		// Did is what completing an offer means, and this is completing one: the state
		// and the answer belong together or they drift apart.
		if err := a.store.Did(ctx, personID, Offer{Kind: OfferTask, RefID: it.ID}, time.Now()); err != nil {
			return err
		}
		a.react(ctx, prompt)
		return nil
	}
	return nil
}

// tomorrow is the start of the next day, locally. "Not today" means today's
// asking stops and tomorrow is a fresh question — not "in 24 hours", which
// would move the moment it asks a little later every time it was pressed.
func tomorrow(now time.Time) time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
}
