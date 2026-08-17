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
	store *Store
	// send is the phase 2 plain-text surface. apply() no longer calls it —
	// every reply is a Message now, sent through chat — but the field and
	// NewApplier's parameter stay so boot.go (rewired in a later task) and
	// phase 2 callers still compile.
	send    Sender
	chat    Chat
	onError func(error)
	// pending is the id of the prompt recorded earlier in this same apply()
	// call, if any — RecordPrompt commits before the message carrying its
	// buttons is sent, the same ordering the scheduler uses and for the same
	// reason, so the id has to survive the few lines between the two. Reset
	// to zero at the top of every apply().
	pending int64
}

// NewApplier's send parameter is a vestige of phase 2, kept only so callers
// built before Chat existed still compile — see the comment on Applier.send.
// chat is what every reply and receipt actually goes through now.
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
	a.pending = 0

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
	//
	// Both branches below funnel into the same tail: tick runs once handling
	// has succeeded, whether that handling was a tap resolved against a
	// prompt or a plain capture that reached Postgres. tick itself is the one
	// that knows a tap earns no ✅ — this join point does not need to.
	if in, ok := ParseAction(item.RawText); ok && isActionPayload(item.Payload) {
		if err := a.applyAction(ctx, in, *personID); err != nil {
			return err
		}
	} else {
		intent := matchFn(item.RawText)
		m, err := a.replyFor(ctx, intent, *personID, *item.ConversationID)
		if err != nil {
			return err
		}
		if m.Text != "" {
			messageID, err := a.sendMessage(ctx, *item.ConversationID, m)
			if err != nil {
				return err
			}
			if a.pending != 0 {
				if messageID == "" {
					// The transport reported success but returned no id to
					// hang the buttons off — see chatVia's messageIDFrom.
					// Still mark the prompt delivered below; just say, once,
					// that it can never be disabled and no tap can resolve
					// back to it, rather than storing a lie.
					a.onError(fmt.Errorf("prompt %d delivered with no addressable message id", a.pending))
				}
				if err := a.store.MarkPromptSent(ctx, a.pending, messageID, time.Now()); err != nil {
					return err
				}
				// Only a numbered surface closes the one before it. A define
				// names one chore and takes no position, so closing the
				// digest on its account would retire buttons the morning's
				// list still owns.
				if intent.Kind != IntentDefine {
					closePrevious(ctx, a.store, a.chat, a.onError, *personID, a.pending)
				}
			}
		}
	}

	a.tick(ctx, item)
	return nil
}

// sendMessage sends through Chat when the transport supports it, and falls
// back to the phase 2 plain-text Sender otherwise — Boost is already guarded
// this way in tick, and Update is already guarded this way in closePrevious;
// Send was the one field this method still called unconditionally, which is
// exactly why every phase 2 test that reaches this path had to build a Chat
// around its Sender in the first place. Falling back here instead makes
// "degrade to phase 2 behaviour when Chat carries no Send" true by
// construction, not by test scaffolding or deployment discipline, and gives
// the send field a reason to still exist.
func (a *Applier) sendMessage(ctx context.Context, conversationID string, m Message) (string, error) {
	if a.chat.Send == nil {
		return "", a.send(ctx, conversationID, m.Text)
	}
	return a.chat.Send(ctx, conversationID, m)
}

// tick is the second half of the receipt. 👀 said the thought was on disk; ✅
// says it has been handled — the drain reached Postgres and this ran. Both
// stay: the pair is a visible trail of two stages that genuinely happened,
// rather than one state overwriting its own history.
//
// Fail-open, like every part of the receipt. A boost that cannot be created
// never changes whether the capture was stored.
func (a *Applier) tick(ctx context.Context, item Item) {
	if a.chat.Boost == nil || item.ExternalID == nil || item.ConversationID == nil {
		return
	}
	if _, isTap := ParseAction(item.RawText); isTap {
		// A tap is not a message in the room; there is nothing to react to.
		return
	}
	if err := a.chat.Boost(ctx, *item.ConversationID, *item.ExternalID, "✅"); err != nil {
		a.onError(fmt.Errorf("ticking %s: %w", *item.ExternalID, err))
	}
}

func (a *Applier) replyFor(ctx context.Context, in Intent, personID int64, conversationID string) (Message, error) {
	switch in.Kind {
	case IntentDefine:
		c, err := a.store.UpsertChore(ctx, personID, in.Name, in.Every, DefaultTolerance(in.Every))
		if err != nil {
			return Message{}, err
		}
		// A define is a standalone surface: it names one chore, it is never
		// numbered, and it does not close the digest's buttons.
		id, err := a.store.RecordPrompt(ctx, personID, conversationID, "define", time.Now(), nil, []Chore{c})
		if err != nil {
			return Message{}, err
		}
		a.pending = id
		return DefinedMessage(c), nil

	case IntentComplete:
		return a.complete(ctx, in, personID, conversationID)

	case IntentStop:
		c, ok, err := a.store.ChoreAtPosition(ctx, personID, in.Position)
		if err != nil || !ok {
			return Message{Text: "I don't have a line " + fmt.Sprint(in.Position) + "."}, err
		}
		if err := a.store.DeactivateChore(ctx, c.ID); err != nil {
			return Message{}, err
		}
		return Message{Text: "Stopped " + c.Name + "."}, nil

	case IntentQuery:
		chores, err := a.store.ActiveChores(ctx, personID)
		if err != nil {
			return Message{}, err
		}
		id, err := a.store.RecordPrompt(ctx, personID, conversationID, "query", time.Now(), nil, chores)
		if err != nil {
			return Message{}, err
		}
		a.pending = id
		return ListMessage(chores), nil

	case IntentDrop:
		return a.undo(ctx, personID)
	}

	// IntentCapture: the squirrel already went out in the HTTP response.
	return Message{}, nil
}

func (a *Applier) complete(ctx context.Context, in Intent, personID int64, conversationID string) (Message, error) {
	if in.Position > 0 {
		c, ok, err := a.store.ChoreAtPosition(ctx, personID, in.Position)
		if err != nil {
			return Message{}, err
		}
		if !ok {
			return Message{Text: fmt.Sprintf("I don't have a line %d.", in.Position)}, nil
		}
		if err := a.store.RecordCompletion(ctx, c.ID, personID, "ack", time.Now()); err != nil {
			return Message{}, err
		}
		return Message{Text: fmt.Sprintf("%s — next in %d days.", c.Name, c.EveryDays)}, nil
	}

	outstanding, err := a.store.OutstandingLines(ctx, personID)
	if err != nil {
		return Message{}, err
	}
	switch len(outstanding) {
	case 0:
		return Message{Text: "Nothing outstanding."}, nil
	case 1:
		c := outstanding[0]
		if err := a.store.RecordCompletion(ctx, c.ID, personID, "ack", time.Now()); err != nil {
			return Message{}, err
		}
		return Message{Text: fmt.Sprintf("%s — next in %d days.", c.Name, c.EveryDays)}, nil
	default:
		// Never guess. Re-number and ask, so the reply can be a bare digit —
		// the same shape as IntentQuery, down to recording its own prompt.
		id, err := a.store.RecordPrompt(ctx, personID, conversationID, "query", time.Now(), nil, outstanding)
		if err != nil {
			return Message{}, err
		}
		a.pending = id
		m := ListMessage(outstanding)
		m.Text = "Which one?\n" + m.Text
		return m, nil
	}
}

// undo removes a chore the matcher created from what was meant as a note. It
// only reaches back undoWindow, so `nvm` long after the fact is not a
// destructive surprise.
func (a *Applier) undo(ctx context.Context, personID int64) (Message, error) {
	const q = `
		select id, name from chores
		 where person_id = $1 and active and created_at >= $2
		 order by created_at desc limit 1`

	var id int64
	var name string
	err := a.store.pool.QueryRow(ctx, q, personID, time.Now().Add(-undoWindow)).Scan(&id, &name)
	if err != nil {
		return Message{Text: "Nothing to undo."}, nil
	}
	if err := a.store.DeactivateChore(ctx, id); err != nil {
		return Message{}, err
	}
	return Message{Text: "Dropped " + name + ". It's still in your captures."}, nil
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
