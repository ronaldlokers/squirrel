package boot

import (
	"context"
	"time"

	"github.com/ronaldlokers/squirrel/internal/coach"
	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The read tools, over the store. Here because internal/coach must not know a
// database exists and internal/squirrel must not know a model does.
//
// Two rules hold across all of it:
//
//   - Every cap is applied here, not asked for in the prompt. A cap the model is
//     asked to respect is a cap it can ignore.
//   - Every suppression is applied here too, so "not now" means the same thing
//     whether the picker or the coach is choosing.
type facts struct {
	store *squirrel.Store
	now   func() time.Time
}

func factsOver(store *squirrel.Store) *facts {
	return &facts{store: store, now: time.Now}
}

func (f *facts) Clock(ctx context.Context, personID int64) (coach.Now, error) {
	return nowFor(ctx, f.store, personID, f.now()), nil
}

// OpenWork is what could be done: tasks you decided on, and chores that are
// due and inside their asking window.
//
// Notes are deliberately absent. A note is a thought you have not decided
// about, and handing the pile to something that chooses would make the model
// decide for you — which is the one thing triage exists to stop it doing.
func (f *facts) OpenWork(ctx context.Context, personID int64, limit int) ([]coach.Work, error) {
	now := f.now()

	// Today's refusals, as the picker reads them. Same window, same key
	// format: the two must agree or "not now" means one thing on one path and
	// another on the other.
	suppressed, err := f.store.Suppressed(ctx, personID, squirrel.StartOfDay(now))
	if err != nil {
		return nil, err
	}

	work := make([]coach.Work, 0, limit)

	chores, err := f.store.DueChores(ctx, personID, now)
	if err != nil {
		return nil, err
	}
	for _, c := range chores {
		if suppressed[squirrel.SuppressionKey(squirrel.OfferChore, c.ID)] {
			continue
		}
		work = append(work, coach.Work{ID: c.ID, Kind: "chore", Text: c.Name})
	}

	tasks, _, err := f.store.Tasks(ctx, personID, limit)
	if err != nil {
		return nil, err
	}
	for _, t := range tasks {
		if t.State != squirrel.ItemOpen {
			continue
		}
		if suppressed[squirrel.SuppressionKey(squirrel.OfferTask, t.ID)] {
			continue
		}
		work = append(work, coach.Work{ID: t.ID, Kind: "task", Text: t.RawText})
	}

	if len(work) > limit {
		work = work[:limit]
	}
	return work, nil
}

// NextFixed is the next thing the world imposed, with the leave-by arithmetic
// already done. Handing the model the parts and letting it subtract would be
// two answers to one question, and the wrong one makes you late.
func (f *facts) NextFixed(ctx context.Context, personID int64) (coach.Fixed, bool, error) {
	now := f.now()
	m, found, err := f.store.NextMoment(ctx, personID, now)
	if err != nil || !found {
		return coach.Fixed{}, false, err
	}
	return coach.Fixed{
		Label:   m.Label,
		At:      m.Starts.Local().Format("15:04"),
		LeaveIn: int(m.LeaveAt().Sub(now).Minutes()),
		Guessed: m.Guessed,
	}, true, nil
}

// Lately is what was already done today, so the model does not hand back
// something that is finished or congratulate anyone on something they have
// not done.
func (f *facts) Lately(ctx context.Context, personID int64, limit int) ([]coach.Happened, error) {
	handled, err := f.store.HandledSince(ctx, personID, squirrel.StartOfDay(f.now()))
	if err != nil {
		return nil, err
	}

	out := make([]coach.Happened, 0, limit)
	for _, what := range append(append([]string{}, handled.Chores...), handled.Tasks...) {
		if len(out) == limit {
			break
		}
		out = append(out, coach.Happened{What: what})
	}
	// handled.Notes is deliberately dropped on the floor. It is the one count
	// this store returns, it exists for the evening message, and a number of
	// notes cleared is exactly the accruing figure this product refuses to put
	// in front of anyone — including a model that would then repeat it.
	return out, nil
}

// Typically is how long something usually takes, from timers that reached their
// end. The narrowness is the store's rule rather than this adapter's — it reads
// timer_runs, which only holds finished runs, and refuses to answer until there
// are enough for a median to mean something.
func (f *facts) Typically(ctx context.Context, personID int64, label string) (int, bool, error) {
	return f.store.TypicalMinutes(ctx, personID, label)
}

// Item is one thing, by id, and only ever this person's.
func (f *facts) Item(ctx context.Context, personID, id int64) (coach.Work, bool, error) {
	it, found, err := f.store.ItemByID(ctx, personID, id)
	if err != nil || !found {
		return coach.Work{}, false, err
	}
	kind := "task"
	if it.Kind != squirrel.ItemTask {
		// A note is not work, and saying so is better than pretending. The
		// model asked about an id it was given; it just cannot choose this one.
		kind = "note"
	}
	return coach.Work{ID: it.ID, Kind: kind, Text: it.RawText}, true, nil
}
