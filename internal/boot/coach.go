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
// internal/coach must not import internal/squirrel — the core would then depend
// on a model being reachable, which is the whole thing this architecture
// refuses. internal/squirrel must not import internal/coach either, for the
// same reason read the other way.
//
// Elsewhere that separation is free: internal/web declares the narrow interface
// it needs and *squirrel.Store satisfies it structurally, because every
// parameter is either a primitive or a squirrel type. The budget's log cannot
// work that way. Its parameter is a struct, and coach.Answer and
// squirrel.CoachAnswer are different types however identically they are
// written, so no structural match is possible in either direction.
//
// So the conversion is written down, here, in the package that already exists
// to join things that must not know about each other. Twelve lines of copying
// is a smaller price than either package importing the other, and it is the
// only place the two structs have to agree.

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

// budgetFor is the monthly ceiling, wired to the log that answers it.
func budgetFor(cfg squirrel.CoachConfig, store *squirrel.Store) coach.Budget {
	return coach.Budget{Log: coachLog{store: store}, CeilingMicros: cfg.BudgetMicros}
}

// coachFor builds the coach, or NoCoach.
//
// NoCoach is a shipping configuration, not a failure: with no key the picker
// still chooses, the ladder still answers, and every screen still works. So the
// absence of a key is logged at info — a warning would be telling someone off
// for a choice they made.
//
// A model the price table does not know is different, and does warrant a
// warning. It means the budget will price every call at zero and the monthly
// ceiling silently stops existing, which is a thing to hear about at start
// rather than discover on an invoice.
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

// decider is the seam both surfaces choose through, or nil.
//
// The cache is the reason this is worth having at all. Opening home is the
// most repeated action in the product and most opens change nothing, so
// without it the deep model is paid over and over for the same answer — and
// pick.go's own argument applies too: an offer that changes on every reload
// reads as the product changing its mind, and a model asked twice will not
// answer identically.
//
// What it is keyed on is the picker's own answer. PickNow already reflects
// every event the design listed as an invalidator — a check-in changes
// capacity, a timer changes rules 2 and 3, a completion or a refusal removes
// the row, a moment entering its leave-by window outranks everything — so
// comparing its answer catches all of them without a hook at a single write
// site. Hooks are the version of this that gets forgotten when a seventh write
// path is added.
func decider(c coach.Coach, offers *coach.Offers) squirrel.Decider {
	if _, none := c.(coach.NoCoach); none {
		return nil
	}

	return func(ctx context.Context, personID int64, pickedKind string, pickedRef int64,
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
}

// asker is the seam the core reaches a model through.
//
// A closure over primitives, because internal/squirrel must not import
// internal/coach. Everything the model is told about the day is assembled here
// rather than there, which is what keeps the core from knowing that a prompt
// is a thing that exists.
//
// It returns nil when there is no coach at all, and the nil is meaningful: the
// core checks it and does not advertise `!coach` in help when there is nothing
// behind it.
func asker(c coach.Coach, store *squirrel.Store, talk *coach.Conversations) turnFn {
	if _, none := c.(coach.NoCoach); none {
		return nil
	}

	return func(ctx context.Context, personID int64, kind, said, subject string) (coach.Reply, error) {
		now := time.Now()

		// The one place overwhelm is recognised, so the sheet and the chat
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
		})
	}
}

// nowFor is the state of the day, as the four small facts the model is told.
//
// Every read here fails soft. A missing capacity or an unreachable moment
// costs the model a hint; it must not cost the person an answer, because the
// alternative to a slightly less informed reply is no reply at all.
func nowFor(ctx context.Context, store *squirrel.Store, personID int64, now time.Time) coach.Now {
	n := coach.Now{
		Clock:     now.Format("15:04"),
		PartOfDay: string(squirrel.PartOfDay(now)),
		// Derived before it gets here, and derived deliberately: the model is
		// told "ok" or "low", never a mood word and never a history. A signal,
		// not a diagnosis.
		Capacity: string(store.Capacity(ctx, personID, now)),
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

// breaker is the seam the ladder makes a thing smaller through, or nil.
//
// No cache here, deliberately. A breakdown is asked for by pressing something,
// once, at the moment of least capacity — there is no idle repeated open to
// protect against, and holding an old sequence would mean pressing "too big"
// on a different thing and being handed steps for the last one.
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

// splitter is the seam a note is separated through, or nil.
//
// Two halves. The cheap one is a rule and runs on every note the pile draws;
// the expensive one is a call and runs only when something is pressed. Which
// is the interruption pre-filter's argument applied to triage: rules narrow,
// and the model answers the few that survive.
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

// interrupter is the veto on a nudge the rules already allowed, or nil.
//
// Nil rather than a pass-through when there is no coach, so the scheduler's
// own nil check is the one that decides and there is no call at all — the
// nudge path stays exactly what it was, which is the point of failing open.
//
// The Now handed over is deliberately thin: the clock, the part of the day and
// capacity. The model is deciding whether *this moment* is a bad one to speak
// into, and what is on the pile is not evidence about that.
func interrupter(c coach.Coach, store *squirrel.Store) squirrel.Interrupter {
	if _, none := c.(coach.NoCoach); none {
		return nil
	}
	return func(ctx context.Context, personID int64, about string, at time.Time) (string, bool) {
		return c.ShouldInterrupt(ctx, personID, about, nowFor(ctx, store, personID, at))
	}
}

// spentFor is what the coach has cost this month, already rendered as money.
//
// Nil when there is no coach: a line saying "€0.00 of €10" under a sheet that
// cannot call anything would be reporting on a thing that is not there.
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

// coachWeb is the screen's half of the same seam.
//
// The screen declares its own Exchange, Answer and Proposal for the same
// reason it declares its own Store: it must not have to know internal/coach
// exists. So the conversion lives here, next to the budget's, and boot stays
// the one place that knows both shapes.
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

	ask := asker(c, store, talk)
	if ask == nil {
		return nil, recent, remember, forget
	}

	webAsk := func(ctx context.Context, personID int64, kind, said, subject string) (web.Answer, error) {
		reply, err := ask(ctx, personID, kind, said, subject)
		if err != nil {
			return web.Answer{}, err
		}
		a := web.Answer{Text: reply.Text, Did: reply.Did}
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
