package boot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The write tools, over the store.
//
// Every one of these calls a function the screen's own buttons already call.
// Nothing here is a new way to change something — it is the same write, asked
// for in words instead of pressed, which is the whole of what makes the
// confirmation policy safe: the action existed, its undo existed, and the only
// thing that changed is which finger pressed it.
//
// Each returns what it acted on, so the surface can say what happened in the
// application's words rather than repeating the model's claim about it.
type hands struct {
	store *squirrel.Store
	now   func() time.Time
}

func handsOver(store *squirrel.Store) *hands {
	return &hands{store: store, now: time.Now}
}

var errNotYours = errors.New("that is not yours")

func (h *hands) Complete(ctx context.Context, personID, itemID int64) (string, error) {
	it, found, err := h.store.ItemByID(ctx, personID, itemID)
	if err != nil {
		return "", err
	}
	if !found {
		return "", errNotYours
	}
	if err := h.store.SetItemState(ctx, itemID, squirrel.ItemDone, h.now()); err != nil {
		return "", err
	}
	return it.RawText, nil
}

// CompleteChore records a completion the same way `!did` and the chores
// screen's own button do, source and all — so a retraction works on it exactly
// as it works on either of those.
func (h *hands) CompleteChore(ctx context.Context, personID, choreID int64) (string, error) {
	name, err := h.choreName(ctx, personID, choreID)
	if err != nil {
		return "", err
	}
	if err := h.store.RecordCompletion(ctx, choreID, personID, "coach", h.now()); err != nil {
		return "", err
	}
	return name, nil
}

func (h *hands) StartTimer(ctx context.Context, personID int64, label string, minutes int) error {
	if label == "" {
		label = "it"
	}
	_, err := h.store.StartTimer(ctx, personID, label,
		time.Duration(minutes)*time.Minute, h.now())
	return err
}

func (h *hands) Refuse(ctx context.Context, personID int64, kind string, refID int64) error {
	k := squirrel.OfferKind(kind)
	if k != squirrel.OfferTask && k != squirrel.OfferChore {
		// The vocabulary the picker itself uses, checked rather than trusted:
		// a kind that was never offered is a value nobody pressed.
		return errNotYours
	}
	return h.store.Refuse(ctx, personID, k, refID, h.now())
}

// SnoozeChore is bounded here rather than in the prompt. A model asked not to
// snooze something for a year is a model that can; a ceiling in code is one
// that cannot.
func (h *hands) SnoozeChore(ctx context.Context, personID, choreID int64, hours int) (string, error) {
	if hours < 1 {
		hours = 1
	}
	if hours > 24*14 {
		hours = 24 * 14
	}
	name, err := h.choreName(ctx, personID, choreID)
	if err != nil {
		return "", err
	}
	ok, err := h.store.SnoozeChore(ctx, choreID, personID,
		h.now().Add(time.Duration(hours)*time.Hour))
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errNotYours
	}
	return name, nil
}

// CreateTask writes what the screen's own "decide something outright" writes:
// an item of kind task, in the person's words, through the ordinary path.
func (h *hands) CreateTask(ctx context.Context, personID int64, text string) error {
	id, err := h.store.InsertItemReturningID(ctx, squirrel.Item{
		PersonID: &personID, RawText: text, ReceivedAt: h.now(),
		Transport: "coach", Payload: []byte(`{}`),
	})
	if err != nil {
		return err
	}
	_, err = h.store.SetItemKind(ctx, personID, id, squirrel.ItemTask)
	return err
}

// choreName is the name, and the ownership check at the same time. Two
// purposes in one query because they are the same question: a chore that is
// not in your active list is not a chore you can act on.
func (h *hands) choreName(ctx context.Context, personID, choreID int64) (string, error) {
	chores, err := h.store.ActiveChores(ctx, personID)
	if err != nil {
		return "", err
	}
	for _, c := range chores {
		if c.ID == choreID {
			return c.Name, nil
		}
	}
	return "", fmt.Errorf("%w: chore %d", errNotYours, choreID)
}
