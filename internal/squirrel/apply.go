package squirrel

import (
	"context"
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
	onError func(error)
}

func NewApplier(store *Store, send Sender, onError func(error)) *Applier {
	if onError == nil {
		onError = func(error) {}
	}
	return &Applier{store: store, send: send, onError: onError}
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
