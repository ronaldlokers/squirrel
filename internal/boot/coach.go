package boot

import (
	"context"
	"log/slog"
	"time"

	"github.com/ronaldlokers/squirrel/internal/coach"
	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/ronaldlokers/squirrel/internal/web"
)

// Where the coach is joined to the store.
//
// internal/coach must not import internal/squirrel, or the core would depend on a
// model being reachable, and internal/squirrel must not import internal/coach for
// the same reason read the other way.
//
// Elsewhere the separation is free — internal/web declares the narrow interface
// it needs and *squirrel.Store satisfies it structurally. The budget's log cannot
// work that way: its parameter is a struct, and coach.Answer and
// squirrel.CoachAnswer are different types however identically written.
//
// So the conversion is written down here, in the package that exists to join
// things that must not know about each other.

// coachLog adapts the store to coach.Log.
type coachLog struct{ store *squirrel.Store }

func (l coachLog) RecordCoachAnswer(ctx context.Context, personID int64, a coach.Answer) error {
	return l.store.RecordCoachAnswer(ctx, personID, squirrel.CoachAnswer{
		Kind:       a.Kind,
		Model:      a.Model,
		Prompt:     a.Prompt,
		Reply:      a.Reply,
		InTokens:   a.InTokens,
		OutTokens:  a.OutTokens,
		CostMicros: a.CostMicros,
		Used:       a.Used,
		At:         a.At,
	})
}

func (l coachLog) CoachSpentSince(ctx context.Context, personID int64, since time.Time) (int64, error) {
	return l.store.CoachSpentSince(ctx, personID, since)
}

// budgetFor is the monthly ceiling, wired to the log that answers it. Two
// ceilings: the owner's, and a smaller one for everybody else, so two demo
// accounts are not two allowances.
//
// The owner is whoever SeedOwner made — there is no admin flag, because there is
// one owner and it is configuration. owner() is zero until Postgres answers, and
// a lookup before then sees a guest, which is the safe way round.
func budgetFor(cfg squirrel.CoachConfig, store *squirrel.Store, owner func() int64) coach.Budget {
	return coach.Budget{
		Log: coachLog{store: store},
		CeilingFor: func(personID int64) int64 {
			if personID != 0 && personID == owner() {
				return cfg.BudgetMicros
			}
			return cfg.GuestBudgetMicros
		},
	}
}

// coachFor builds the coach, or NoCoach — a shipping configuration, not a
// failure, so a missing key is logged at info rather than as a warning.
//
// A model the price table does not know does warrant one: the budget prices every
// call at zero and the ceiling silently stops existing.
func coachFor(cfg squirrel.CoachConfig, budget coach.Budget, store *squirrel.Store) coach.Coach {
	if !cfg.Enabled() {
		slog.Info("no coach configured; the picker and the ladder answer alone")
		return coach.NoCoach{}
	}
	for _, model := range []string{cfg.Fast, cfg.Deep} {
		if !coach.KnownModel(model) {
			slog.Warn("no price known for this model; it will count as free against the budget",
				"model", model)
		}
	}
	slog.Info("the coach is configured",
		"fast", cfg.Fast, "deep", cfg.Deep, "ceiling_micros", cfg.BudgetMicros)
	p := coach.NewProvider(cfg.BaseURL, cfg.APIKey, cfg.Fast, cfg.Deep, budget)
	p.Facts = factsOver(store)
	p.Hands = handsOver(store)
	return p
}

// deciding is the seam both surfaces choose through, or nil, and the way to drop
// a decision. Both halves come back together because they are only correct
// against the same cache: two lines naming different ones compile, run, and
// reproduce the exact bug this pair exists to fix.
//
// The cache is what makes it worth having: opening home is the most repeated
// action in the product and most opens change nothing.
//
// Keyed on the picker's own answer, which already reflects every invalidator the
// design listed — a check-in changes capacity, a timer changes rules 2 and 3, a
// completion removes the row. Hooks are the version of this that gets forgotten
// when a seventh write path is added.
func deciding(c coach.Coach, offers *coach.Offers) (squirrel.Decider, func(personID int64)) {
	if _, none := c.(coach.NoCoach); none {
		// No coach, no decision, and nothing to forget. Both nil, so the
		// screen's own nil checks take the path that shipped before either
		// existed.
		return nil, nil
	}

	decide := func(ctx context.Context, personID int64, pickedKind string, pickedRef int64,
		mayAsk bool) (string, int64, string, string, bool) {

		now := time.Now()
		basis := squirrel.SuppressionKey(squirrel.OfferKind(pickedKind), pickedRef)

		if d, ok := offers.Get(personID, basis, now); ok {
			return d.Kind, d.RefID, d.Text, d.Because, true
		}
		if !mayAsk {
			// A surface that must be free to open. It shows a decision that
			// was already paid for and otherwise shows the picker's — which
			// means the two surfaces agree whenever there is anything to agree
			// about, and nothing here is ever a reason to spend.
			return "", 0, "", "", false
		}

		d, err := c.Decide(ctx, personID)
		if err != nil {
			// Not cached. A failure is usually the network or the budget, and
			// holding "the coach could not answer" for half an hour would turn
			// a blip into an outage — the picker answers this open, and the
			// next one asks again.
			return "", 0, "", "", false
		}

		offers.Put(personID, basis, d, now)
		return d.Kind, d.RefID, d.Text, d.Because, true
	}

	return decide, offers.Forget
}

// asker is the seam the core reaches a model through: a closure over primitives,
// because internal/squirrel must not import internal/coach. Everything the model
// is told about the day is assembled here, which keeps the core from knowing that
// a prompt exists.
//
// Nil when there is no coach, and the core checks it: `!coach` is not advertised
// in help when there is nothing behind it.
func asker(c coach.Coach, store *squirrel.Store, talk *coach.Conversations, canOpen bool) turnFn {
	if _, none := c.(coach.NoCoach); none {
		return nil
	}

	return func(ctx context.Context, personID int64, kind, said, subject string) (coach.Reply, error) {
		now := time.Now()

		// The one place overwhelm is recognised, so the screen and the chat
		// cannot disagree about what it is. The caller says which surface
		// asked; whether this particular turn is a pile rather than a question
		// is a property of the words, not of the surface.
		deep := coach.Overwhelmed(said)
		if deep {
			kind = coach.KindOverwhelm
		}

		return c.Answer(ctx, coach.Turn{
			PersonID: personID,
			Kind:     kind,
			Deep:     deep,
			Now:      nowFor(ctx, store, personID, now),
			Said:     said,
			Subject:  subject,
			Recent:   talk.Recent(personID, now),
			// Whether a place can be drawn is a fact about the surface, and
			// the screen is the surface that can. See coachChat for the other
			// half: chat leaves this false, because a place there would be the
			// list the guard exists to refuse.
			CanOpen: canOpen,
		})
	}
}

// nowFor is the state of the day, as the four small facts the model is told.
//
// Every read here fails soft. A missing capacity or an unreachable moment
// costs the model a hint; it must not cost the person an answer, because the
// alternative to a slightly less informed reply is no reply at all.
func nowFor(ctx context.Context, store *squirrel.Store, personID int64, now time.Time) coach.Now {
	// The softest failure of all, and the one this function's own rule asks
	// for: no store is no hints, not no answer.
	if store == nil {
		return coach.Now{
			Clock:     now.Format("15:04"),
			PartOfDay: string(squirrel.PartOfDay(now)),
		}
	}

	n := coach.Now{
		Clock:     now.Format("15:04"),
		PartOfDay: string(squirrel.PartOfDay(now)),
		// Derived before it gets here, and derived deliberately: the model is
		// told "ok" or "low", never a mood word and never a history. A signal,
		// not a diagnosis.
		Capacity: string(store.Capacity(ctx, personID, now)),
	}

	// What has not landed here, in the model's own words. Fails soft: a reply that
	// cannot be read back costs the model a hint, never the person an answer.
	//
	// Three, because it is examples rather than a record.
	if said, err := store.BadlyLanded(ctx, personID, 3); err == nil {
		n.LandedBadly = said
	}

	// What a weekly read of the record concluded about how this person works.
	// Fails soft for the same reason as everything else here — and its absence
	// is the state the product was in until 25 August 2026, so a Buddy without
	// it is a Buddy that works.
	if known, err := store.Knowing(ctx, personID); err == nil {
		n.Knowing = known
	}

	if m, found, err := store.NextMoment(ctx, personID, now); err == nil && found {
		// Minutes to the thing itself rather than to when to leave for it.
		// Leave-by arithmetic is the product's own job and it is already done
		// on the screen; handing the model a second version of it invites two
		// answers to one question.
		if mins := int(m.Starts.Sub(now).Minutes()); mins >= 0 {
			n.FreeUntil = &mins
		}
	}

	return n
}

// breaker is the seam the ladder makes a thing smaller through, or nil. No cache:
// a breakdown is asked for by pressing something once, and holding an old
// sequence would hand you steps for the last thing you pressed on.
func breaker(c coach.Coach) squirrel.Breaker {
	if _, none := c.(coach.NoCoach); none {
		return nil
	}
	return func(ctx context.Context, personID int64, task, blocker string) ([]string, bool) {
		steps, err := c.Smaller(ctx, personID, task, blocker)
		if err != nil {
			return nil, false
		}
		return steps, true
	}
}

// reading is the three tiers the box judges with, as one value. Lifted out of an
// inline literal for the reason schedulerOptionsFor was: a field set in one
// cannot be checked by a test, and `AskedAQuestion` proved it by going missing
// while the whole suite stayed green.
type reading struct {
	Reads          func(context.Context, int64, string) (string, bool, string, error)
	AskedAQuestion func(context.Context, string) (bool, bool)
}

// readingWiring is what the screen is given.
func readingWiring(c coach.Coach, store *squirrel.Store, h *coach.House) reading {
	return reading{Reads: reader(c, store), AskedAQuestion: housed(h)}
}

// housed is the model on the cluster, or nil. Its own seam rather than a method
// on the coach: no key, no budget, no accounting, and it answers when the hosted
// one is absent entirely.
func housed(h *coach.House) func(context.Context, string) (bool, bool) {
	if h == nil {
		return nil
	}
	return h.AskedAQuestion
}

// reader is what the box is answered by, or nil — and captureHandler checks the
// nil, so a build without a key keeps the words and says "Kept."
//
// The state of the day goes in through nowFor, like every other turn: what
// somebody types at eleven at night reads differently from the same words at
// nine in the morning.
func reader(c coach.Coach, store *squirrel.Store) func(context.Context, int64, string) (string, bool, string, error) {
	if _, none := c.(coach.NoCoach); none {
		return nil
	}
	return func(ctx context.Context, personID int64, said string) (string, bool, string, error) {
		return c.Reads(ctx, personID, said, nowFor(ctx, store, personID, time.Now()))
	}
}

// learner is the weekly read of the record, or nil.
//
// Nil with no coach, and the nil is what the scheduler checks — so a build
// without a key never asks and never writes, which is the state the product
// was in for a month and works.
func learner(c coach.Coach) squirrel.Learner {
	if _, none := c.(coach.NoCoach); none {
		return nil
	}
	return c.Learn
}

// splitter is the seam a note is separated through, or nil. The cheap half is a
// rule and runs on every note the pile draws; the expensive half is a call and
// runs only when something is pressed.
func splitter(c coach.Coach) (
	func(context.Context, int64, string) ([]string, bool), func(string) bool) {

	if _, none := c.(coach.NoCoach); none {
		return nil, nil
	}
	split := func(ctx context.Context, personID int64, text string) ([]string, bool) {
		pieces, err := c.Split(ctx, personID, text)
		if err != nil {
			return nil, false
		}
		return pieces, true
	}
	// The same rule that recognises the overwhelm turn, because a brain dump
	// and an overwhelm turn are the same shape — and one definition of "this
	// is several things" is one place for it to be wrong.
	return split, coach.Overwhelmed
}

// turnFn is one conversational turn, in coach's own types. Both surfaces are
// adapted from it below, because they want different parts of the same answer:
// chat renders what changed as lines, and the screen also renders a proposal
// as a press.
type turnFn func(ctx context.Context, personID int64, kind, said, subject string) (coach.Reply, error)

// interrupter is the veto on a nudge the rules already allowed, or nil rather
// than a pass-through, so the scheduler's own nil check decides and there is no
// call at all.
//
// The Now handed over is deliberately thin: the model is deciding whether this
// moment is a bad one to speak into, and what is on the pile is not evidence
// about that.
func interrupter(c coach.Coach, store *squirrel.Store) squirrel.Interrupter {
	if _, none := c.(coach.NoCoach); none {
		return nil
	}
	return func(ctx context.Context, personID int64, about string, at time.Time) (string, bool) {
		return c.ShouldInterrupt(ctx, personID, about, nowFor(ctx, store, personID, at))
	}
}

// overFor answers "is this month's coach budget gone", for the room. A boolean
// rather than the figure: what it costs belongs on a surface you go to on
// purpose. Nil means "do not say so", because a month with no ceiling is never
// spent.
func overFor(c coach.Coach, budget coach.Budget) func(context.Context, int64) bool {
	if _, none := c.(coach.NoCoach); none {
		return nil
	}
	return func(ctx context.Context, personID int64) bool {
		spent, ceiling, ok := budget.Spent(ctx, personID, time.Now())
		if !ok || ceiling <= 0 {
			return false
		}
		return spent >= ceiling
	}
}

// spentFor is what the coach has cost this month, already rendered as money.
//
// Nil when there is no coach: "€0.00 of €10" under something that cannot call
// anything would be reporting on a thing that is not there.
func spentFor(c coach.Coach, budget coach.Budget) func(context.Context, int64) (string, string, bool) {
	if _, none := c.(coach.NoCoach); none {
		return nil
	}
	return func(ctx context.Context, personID int64) (string, string, bool) {
		spent, ceiling, ok := budget.Spent(ctx, personID, time.Now())
		if !ok {
			return "", "", false
		}
		if ceiling <= 0 {
			// No in-process ceiling is a supported choice, and "of €0.00"
			// would read as one that had been reached.
			return coach.Euros(spent), "", true
		}
		return coach.Euros(spent), coach.Euros(ceiling), true
	}
}

// coachWeb is the screen's half of the same seam. The screen declares its own
// Exchange, Answer and Proposal for the reason it declares its own Store, so the
// conversion lives here and boot stays the one place that knows both shapes.
func coachWeb(c coach.Coach, store *squirrel.Store, talk *coach.Conversations) (
	func(context.Context, int64, string, string, string) (web.Answer, error),
	func(int64) []web.Exchange, func(int64, string, string), func(int64)) {

	recent := func(personID int64) []web.Exchange {
		fresh := talk.Recent(personID, time.Now())
		out := make([]web.Exchange, 0, len(fresh))
		for _, e := range fresh {
			out = append(out, web.Exchange{Said: e.Said, Replied: e.Replied})
		}
		return out
	}
	remember := func(personID int64, said, replied string) {
		talk.Add(personID, said, replied, time.Now())
	}
	forget := func(personID int64) { talk.Forget(personID) }

	ask := asker(c, store, talk, true)
	if ask == nil {
		return nil, recent, remember, forget
	}

	webAsk := func(ctx context.Context, personID int64, kind, said, subject string) (web.Answer, error) {
		reply, err := ask(ctx, personID, kind, said, subject)
		if err != nil {
			return web.Answer{}, err
		}
		a := web.Answer{Text: reply.Text, Did: reply.Did, Open: reply.Open}
		if reply.Propose != nil {
			a.Propose = &web.Proposal{
				Do: reply.Propose.Do, Said: reply.Propose.Said, Text: reply.Propose.Text,
				At: reply.Propose.At, Every: reply.Propose.Every, RefID: reply.Propose.RefID,
			}
		}
		return a, nil
	}
	return webAsk, recent, remember, forget
}

// coachChat is chat's half. It wants what was said and what changed, and it
// has nowhere to render a proposal — so a turn that proposes something on this
// surface says so and does nothing, which is the honest shape rather than a
// silent drop.
func coachChat(ask turnFn) Asker {
	if ask == nil {
		return nil
	}
	return func(ctx context.Context, personID int64, kind, said, subject string) (string, []string, error) {
		reply, err := ask(ctx, personID, kind, said, subject)
		if err != nil {
			return "", nil, err
		}
		text := reply.Text
		if reply.Propose != nil {
			// Chat has no press to put it behind, and something that will
			// interrupt you later must never happen without one. Naming the
			// screen is better than pretending nothing was suggested.
			text += squirrel.OnTheScreen()
		}
		return text, reply.Did, nil
	}
}
