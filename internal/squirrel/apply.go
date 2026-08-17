package squirrel

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Sender posts a message the system initiated. A func type rather than an
// interface, so internal/boot can pass a transport's Send straight in and this
// package still imports nothing from internal/transport.
type Sender func(ctx context.Context, conversationID, text string) error

// undoWindow is how long `nvm` can reach back to remove a chore the matcher
// created from what was meant as a note.
const undoWindow = 10 * time.Minute

type Applier struct {
	store   *Store
	send    Sender
	chat    Chat
	onError func(error)
}

// NewApplier's chat parameter is unused until a later task adds tick — it is
// added here, once, so every caller from this task onward passes the same
// shape rather than the signature changing twice.
func NewApplier(store *Store, send Sender, chat Chat, onError func(error)) *Applier {
	if onError == nil {
		onError = func(error) {}
	}
	return &Applier{store: store, send: send, chat: chat, onError: onError}
}

// Apply runs after a capture has landed in Postgres. The raw text is already
// stored verbatim, so everything here is a derived view over it and nothing
// here can lose a thought.
//
// A panic anywhere below is recovered and reported as an error rather than
// left to propagate. It escaped once already — a byte-length bug in Match's
// chore-name parsing — and rode out through Drain.one and Drain.Run to exit
// the whole process. By then the spool file was already gone and the row
// already committed, so there was nothing left to retry, and every later
// digest attempt re-ran Match over that same row via CapturesSince and
// panicked the scheduler too, forever. A derived view failing must never be
// able to take capture down with it again.
func (a *Applier) Apply(ctx context.Context, item Item, personID *int64) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("applier panicked: %v", r)
		}
	}()
	return a.apply(ctx, item, personID)
}

func (a *Applier) apply(ctx context.Context, item Item, personID *int64) error {
	// Chores belong to a person. An unresolved identity means we do not know
	// whose they would be, so nothing is applied — the capture is already safe.
	if personID == nil || item.ConversationID == nil {
		return nil
	}

	// An action is not a thought and never reaches the matcher. It is checked
	// first so that a tap can never be reinterpreted as text. ParseAction
	// matches on text alone, so it cannot by itself tell a genuine tap from
	// someone typing the same shape into the room — isActionPayload is what
	// makes that distinction, from the payload the transport actually sent.
	if in, ok := ParseAction(item.RawText); ok && isActionPayload(item.Payload) {
		return a.applyAction(ctx, in, *personID)
	}

	intent := matchFn(item.RawText)
	reply, err := a.replyFor(ctx, intent, *personID, *item.ConversationID)
	if err != nil {
		return err
	}
	if reply == "" {
		return nil
	}
	return a.send(ctx, *item.ConversationID, reply)
}

func (a *Applier) replyFor(ctx context.Context, in Intent, personID int64, conversationID string) (string, error) {
	switch in.Kind {
	case IntentDefine:
		c, err := a.store.UpsertChore(ctx, personID, in.Name, in.Every, DefaultTolerance(in.Every))
		if err != nil {
			return "", err
		}
		return RenderDefined(c), nil

	case IntentComplete:
		return a.complete(ctx, in, personID, conversationID)

	case IntentStop:
		c, ok, err := a.store.ChoreAtPosition(ctx, personID, in.Position)
		if err != nil || !ok {
			return "I don't have a line " + fmt.Sprint(in.Position) + ".", err
		}
		if err := a.store.DeactivateChore(ctx, c.ID); err != nil {
			return "", err
		}
		return "Stopped " + c.Name + ".", nil

	case IntentQuery:
		chores, err := a.store.ActiveChores(ctx, personID)
		if err != nil {
			return "", err
		}
		if _, err := a.store.RecordPrompt(ctx, personID, conversationID, "query", time.Now(), nil, chores); err != nil {
			return "", err
		}
		return RenderList(chores), nil

	case IntentDrop:
		return a.undo(ctx, personID)
	}

	// IntentCapture: the squirrel already went out in the HTTP response.
	return "", nil
}

func (a *Applier) complete(ctx context.Context, in Intent, personID int64, conversationID string) (string, error) {
	if in.Position > 0 {
		c, ok, err := a.store.ChoreAtPosition(ctx, personID, in.Position)
		if err != nil {
			return "", err
		}
		if !ok {
			return fmt.Sprintf("I don't have a line %d.", in.Position), nil
		}
		if err := a.store.RecordCompletion(ctx, c.ID, personID, "ack", time.Now()); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s — next in %d days.", c.Name, c.EveryDays), nil
	}

	outstanding, err := a.store.OutstandingLines(ctx, personID)
	if err != nil {
		return "", err
	}
	switch len(outstanding) {
	case 0:
		return "Nothing outstanding.", nil
	case 1:
		c := outstanding[0]
		if err := a.store.RecordCompletion(ctx, c.ID, personID, "ack", time.Now()); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s — next in %d days.", c.Name, c.EveryDays), nil
	default:
		// Never guess. Re-number and ask, so the reply can be a bare digit.
		if _, err := a.store.RecordPrompt(ctx, personID, conversationID, "query", time.Now(), nil, outstanding); err != nil {
			return "", err
		}
		return "Which one?\n" + RenderList(outstanding), nil
	}
}

// undo removes a chore the matcher created from what was meant as a note. It
// only reaches back undoWindow, so `nvm` long after the fact is not a
// destructive surprise.
func (a *Applier) undo(ctx context.Context, personID int64) (string, error) {
	const q = `
		select id, name from chores
		 where person_id = $1 and active and created_at >= $2
		 order by created_at desc limit 1`

	var id int64
	var name string
	err := a.store.pool.QueryRow(ctx, q, personID, time.Now().Add(-undoWindow)).Scan(&id, &name)
	if err != nil {
		return "Nothing to undo.", nil
	}
	if err := a.store.DeactivateChore(ctx, id); err != nil {
		return "", err
	}
	return "Dropped " + name + ". It's still in your captures.", nil
}

// isActionPayload reports whether the payload the transport stored alongside
// this text really came from an action webhook. ParseAction matches on text
// alone, and a person typing "!action 451 done:2 true" into the room produces
// text byte-identical to a genuine tap — the payload's "type" field is the
// only thing that tells them apart. Anything that fails to unmarshal or
// carries a different type is treated as not an action, which sends the
// message down the normal matcher: a thought is never rejected.
func isActionPayload(payload json.RawMessage) bool {
	var p struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return false
	}
	return p.Type == "action"
}

// applyAction resolves a tap and applies it as a state assertion rather than a
// delta: "selected" means the completion should exist, "not selected" means it
// should not. Applying either twice lands in the same place, which is what
// makes a retried delivery harmless — the payload carries no event id, so a
// retry and a genuine second tap are indistinguishable.
//
// Every path here is silent. The boost is the receipt; a reply per tap would
// make the room unreadable.
func (a *Applier) applyAction(ctx context.Context, in ActionIntent, personID int64) error {
	prompt, ok, err := a.store.PromptByMessageID(ctx, personID, in.MessageID)
	if err != nil {
		return err
	}
	if !ok {
		// Someone else's prompt, or one we have no record of. Nothing to do,
		// and nothing to say: the capture is stored either way.
		return nil
	}

	c, ok, err := a.store.ChoreOnPrompt(ctx, prompt.ID, in.Position)
	if err != nil || !ok {
		return err
	}

	switch in.Kind {
	case "undefine":
		return a.store.DeactivateChore(ctx, c.ID)

	case "done":
		if !in.Selected {
			_, err := a.store.RetractCompletion(ctx, c.ID, personID, prompt.ID, time.Now())
			return err
		}
		done, err := a.store.CompletedSince(ctx, c.ID, prompt.ID)
		if err != nil || done {
			return err
		}
		return a.store.RecordCompletion(ctx, c.ID, personID, "tap", time.Now())
	}
	return nil
}
