package web

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Buddy, on the screen.
//
// The package and the types keep the word coach, which is what this is; Buddy
// is what it is called.
//
// A widget on every screen rather than a fourth door. DESIGN.md's rule against
// modals carries its own condition — "for anything that needs neither
// interruption nor protected focus" — and this meets it: a coach conversation
// happens when everything else on screen is noise, and protected focus is the
// whole point of the surface. The chore picker was refused a modal because
// choosing an interval needs neither. That reasoning is untouched.
//
// `/buddy` is a real page. The sheet is pile.js upgrading a real route, the
// same progressive enhancement the chore picker's <details> already uses: it
// works with scripting off, it deep-links, and it survives a reload. And
// because it is chrome rather than a destination, the home screen still has
// three doors.

// Exchange is one round of the conversation, as the screen says it.
//
// Declared here rather than imported for the same reason Store is: this
// package must not have to know that internal/coach exists, and internal/coach
// must not know a screen does. internal/boot converts between the two, which is
// the job it already does for the budget's log.
type Exchange struct {
	Said    string
	Replied string
}

// Answer is one turn's result: what the coach said, what actually changed, and
// what it wants permission for.
//
// Did is written by the application after a write succeeded, not by the model
// — a model saying "done" is not evidence anything happened. Propose is a
// thing it may not do on its own, rendered as one press.
type Answer struct {
	Text    string
	Did     []string
	Propose *Proposal
}

// Proposal is the four things the coach must ask about: a fixed point, because
// it will interrupt you later; a chore, because it comes back forever; a
// retirement, because it stops something recurring; and a drop, because it is
// disposal. None is undone by one press on a control already on screen, which
// is the whole of the test.
//
// Stored nowhere. It travels in the form that renders it, exactly as a split
// does, so an unanswered proposal lasts as long as the page it is on.
type Proposal struct {
	Do    string
	Said  string
	Text  string
	At    string
	Every string
	RefID int64
}

// coachAvailable reports whether there is anything behind the acorn.
//
// When there is not, the acorn is still drawn and `/buddy` still answers: the
// four chips are deterministic and the ladder behind them is what shipped
// before any of this existed. The only difference a missing key makes is that
// typing a sentence gets the four chips back instead of an answer.
func coachAvailable(opts Options) bool { return opts.Ask != nil }

// coachSayHandler is one turn.
//
// Two ways in, and they are the same route on purpose: a chip is the sentence
// you did not have to type. Someone at the moment of least capacity should not
// have to compose anything to be helped, which is what the four blockers are
// for — one press, and the answer still comes back about this task rather than
// in general.
// coachAskHandler is the chip: it asks for words and nothing else. The reply
// comes back through coachSayHandler, which is the same route the four
// blockers press.
func coachAskHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		// What you would be handed right now, so the question is about
		// something rather than about nothing. The picker is six rules and no
		// model, so asking costs nothing — which is what let the acorn be
		// pressed idly, and has to stay true of the chip.
		question := "What is going on?"
		if about := offerHint(s, opts, r); about != "" {
			question = "What is going on with " + about + "?"
		}
		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "ask Buddy"},
			askInWords(question, "/buddy/say", "say it", nil),
		}), "/")
	}
}

func coachSayHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// A chip first: it is a value this page offered, so it is trusted the
		// way a form value is. Free text is trusted the way typing is, which
		// is to say it is kept verbatim and never parsed for intent.
		if b, ok := squirrel.ParseBlocker(r.FormValue("why")); ok {
			answerBlocker(w, r, s, opts, personID, b)
			return
		}

		said := strings.TrimSpace(r.FormValue("said"))
		if said == "" {
			// An empty press says nothing, so nothing is said back. Not an
			// error: the box was there and you did not use it.
			answerWith(w, r, nil, "/")
			return
		}

		if !coachAvailable(opts) {
			// No coach, and the honest answer to a sentence is the four
			// chips: the ladder cannot read a paragraph, but it can be asked
			// which of four things is in the way. The words are kept.
			answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
				{Who: squirrel.SpeakerYou, Words: said},
				coachReply("Which of these is it?", true, false, nil, stepFor(s, opts, r)),
			}), "/")
			return
		}

		answer, err := opts.Ask(r.Context(), personID, "thread", said, subjectFor(s, opts, r))
		if err != nil {
			// The floor. What was said is kept — the record is what the box
			// used to be, and words that reached the server do not evaporate
			// because a model was unreachable.
			answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
				{Who: squirrel.SpeakerYou, Words: said},
				coachReply("I cannot think just now. Which of these is it?",
					true, false, nil, stepFor(s, opts, r)),
			}), "/")
			return
		}

		// What actually changed goes into the conversation alongside what was
		// said, because it is part of the same turn and because the next turn
		// should know it happened. The words are the application's — a model
		// saying "done" is not evidence anything did.
		remember(opts, personID, said, withDid(answer))

		// The reply, the proposal and where you are in a breakdown, as one
		// turn. A proposal is stored nowhere and travels in the form that
		// renders it, so a press is still the only thing that can apply one —
		// and in scrollback it has lost its button by the live edge rule,
		// which is the same guarantee the page gave by not surviving a reload.
		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: said},
			coachReplyCosting(withDid(answer), costLine(r.Context(), opts, personID),
				false, true, answer.Propose, stepFor(s, opts, r)),
		}), "/")
	}
}

// answerBlocker is a chip, answered by the ladder.
//
// Deterministic, and it stays deterministic: the four fixed answers are good
// precisely because they are fixed, and the worst case of having a coach at
// all should be that you press a chip and read the same sentence you would
// have read anyway.
func answerBlocker(w http.ResponseWriter, r *http.Request, s Store, opts Options, personID int64, b squirrel.Blocker) {
	// "Not today" is not an obstacle, it is a no — and it is the same no that
	// "not now" writes on the home screen, arrived at from another direction.
	if u := squirrel.UnstuckFor(b); u.Refuse {
		if o, found, err := s.PickNow(r.Context(), personID, now(), true); err == nil && found {
			if err := s.Refuse(r.Context(), personID, o.Kind, o.RefID, now()); err != nil {
				fail(w, err)
				return
			}
		}
		// Back to where the rest of the screen is. Turning something down is
		// the end of the conversation about it, not the start of one.
		forget(opts, personID)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Smaller first, when that is what this blocker wants and there is
	// something to break down. The redirect is the same either way: the step
	// is read back out of the store on the next render, so a reload shows
	// where you are rather than repeating a press.
	if o, found, err := s.PickNow(r.Context(), personID, now(), true); err == nil && found {
		smallerFor(s, opts, r, b, o)
	}

	// The answer travels in the address bar, the same way /now/stuck already
	// sends it — so the ladder's words live in one place, in the core, and the
	// control that comes with them is drawn from the same view the home screen
	// draws. A reload re-reads rather than repeating the press.
	remember(opts, personID, squirrel.BlockerWords[b], squirrel.UnstuckFor(b).Line)
	// The ladder's words come from the core, so they are the same wherever
	// they are read. The step, when there is one, is drawn on the same turn —
	// which is what the address bar used to carry.
	answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
		{Who: squirrel.SpeakerYou, Words: squirrel.BlockerWords[b]},
		coachReply(squirrel.UnstuckFor(b).Line, false, false, nil, stepFor(s, opts, r)),
	}), "/")
}

// Closing was a route. It is not one now: the sheet was a thing that could be
// open, and a conversation is not — you stop talking. What closing also did
// was forget the window, and the record is the window now.

// backTolerant is where a form's "from" may send you.
//
// Only a path this screen serves, and only ever a path: the value arrives from
// a form field and a form field is a place a stranger can type. An open
// redirect from a page behind forward-auth is still an open redirect.
// coachBadlyHandler records that the last thing Buddy said did not land.
//
// Principle 5 was opened on 20 August so the coach could be useful at the only
// thing a coach is for, and the cost was written down at the time: it can now
// say something that lands badly on a bad day. Every exchange has been kept
// since, for exactly that reason — and nothing has ever read one back.
//
// This is one press, and it is deliberately the smallest thing that could
// work. The moment it exists to serve is the moment there is least to spend on
// it: a bad reply, late, on a night that is already going badly. A comment box
// would be a form to fill in at the worst possible time.
//
// Nothing is rendered back except that it was heard. No count, no list, no
// history — what it feeds is the next prompt, where the model is shown the
// words that did not land rather than told about them.
func coachBadlyHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		// Fails soft on purpose. Saying "that landed badly" and being handed an
		// error page is the worst possible answer to it, and the thing being
		// recorded is not load-bearing for anything on the screen.
		heard, err := s.LandedBadlyLatest(r.Context(), personID, now())
		if err != nil {
			slog.Error("recording that a reply landed badly", "error", err)
		}
		// Nothing rendered back except that it was heard. No count, no list,
		// no history — what it feeds is the next prompt, where the model is
		// shown the words that did not land rather than told about them.
		words := squirrel.Say(squirrel.SayingHeard, now())
		if heard {
			words += " I will be shown that one."
		}
		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "that went badly"},
			{Who: squirrel.SpeakerBuddy, Words: words},
		}), "/")
	}
}

func backTolerant(from string) string {
	if !strings.HasPrefix(from, "/") || strings.HasPrefix(from, "//") {
		return "/"
	}
	return from
}

// subjectFor is what is on screen, for the model to answer about.
//
// Asked with showAnyway, because someone who has opened the coach and typed a
// sentence has already overridden the quiet of a low day by asking. A picker
// that cannot answer costs a hint and nothing else.
func subjectFor(s Store, opts Options, r *http.Request) string {
	personID, ok := opts.person()
	if !ok {
		return ""
	}
	o, found, err := s.PickNow(r.Context(), personID, now(), true)
	if err != nil || !found {
		return ""
	}
	return o.Text
}

func remember(opts Options, personID int64, said, replied string) {
	if opts.Remember != nil {
		opts.Remember(personID, said, replied)
	}
}

func forget(opts Options, personID int64) {
	if opts.Forget != nil {
		opts.Forget(personID)
	}
}

// costLine is what the coach has spent this month, for the reply that spent it.
//
// A figure that cannot be read is a figure not drawn, and no ceiling is a
// supported choice — "of €0.00" would read as one that had been reached.
func costLine(ctx context.Context, opts Options, personID int64) string {
	if opts.Spent == nil {
		return ""
	}
	spent, ceiling, ok := opts.Spent(ctx, personID)
	if !ok || spent == "" {
		return ""
	}
	if ceiling == "" {
		return spent
	}
	return spent + " of " + ceiling
}

// blockerChips is the four, in the order the ladder already uses. One press,
// no typing.
func blockerChips() []chipView {
	out := make([]chipView, 0, len(squirrel.Blockers))
	for _, b := range squirrel.Blockers {
		out = append(out, chipView{Why: string(b), Word: squirrel.BlockerWords[b]})
	}
	return out
}

// withDid is the reply plus what actually changed, as one thing said.
func withDid(a Answer) string {
	if len(a.Did) == 0 {
		return a.Text
	}
	return a.Text + " (" + strings.Join(a.Did, "; ") + ")"
}

// coachDoHandler applies a proposal, and only ever one that was pressed.
//
// Everything arrives back through the form, so everything is read the way a
// stranger's typing is read: the kind is checked against the four, the time and
// the rhythm go through the core's own parsers rather than being trusted, and
// anything that does not parse does nothing at all.
//
// Deliberately not a general "do what the model said" route. There are four
// things it can apply and they are named here, in a switch, so adding a fifth
// is a code change someone reviews.
func coachDoHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		var err error
		switch r.FormValue("do") {
		case "moment":
			err = keepMoment(r, s, opts, personID)
		case "chore":
			err = keepChore(r, s, personID)
		case "retire":
			err = retireProposed(r, s, personID)
		case "drop":
			err = dropProposed(r, s, personID)
		}
		if err != nil {
			fail(w, err)
			return
		}
		// What was applied is said, because a press with no answer is a press
		// you cannot tell landed.
		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "keep it"},
			{Who: squirrel.SpeakerBuddy, Words: "Kept."},
		}), "/")
	}
}

// keepMoment creates a fixed point, and the time is parsed by the core rather
// than trusted from the form. A guessed time is a missed appointment.
func keepMoment(r *http.Request, s Store, opts Options, personID int64) error {
	said := strings.TrimSpace(r.FormValue("at") + " " + r.FormValue("text"))
	m, ok := squirrel.ParseMomentIn(opts.Location, "at "+said, now())
	if !ok {
		// It did not parse, so nothing is created. The bar exists so a note is
		// never silently turned into something that interrupts you, and it is
		// not lowered because a model was the one asking.
		return nil
	}
	_, err := s.CreateMoment(r.Context(), personID, m)
	return err
}

// keepChore creates a recurring thing, with its rhythm parsed by the core.
func keepChore(r *http.Request, s Store, personID int64) error {
	name := strings.TrimSpace(r.FormValue("text"))
	every := strings.TrimSpace(r.FormValue("every"))
	if name == "" {
		return nil
	}
	if len(name) > choreNameLimit {
		name = name[:choreNameLimit]
	}
	_, interval, ok := squirrel.ParseEvery(every + ": " + name)
	if !ok {
		return nil
	}
	_, err := s.UpsertChore(r.Context(), personID, name, interval, interval/10)
	return err
}

// retireProposed stops a chore coming back, and only one that is yours.
func retireProposed(r *http.Request, s Store, personID int64) error {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		return nil
	}
	chores, err := s.ActiveChores(r.Context(), personID)
	if err != nil {
		return err
	}
	for _, c := range chores {
		if c.ID == id {
			return s.DeactivateChore(r.Context(), id)
		}
	}
	return nil
}

// dropProposed throws a note away, and only one that is yours. It reverses the
// way every other transition does.
func dropProposed(r *http.Request, s Store, personID int64) error {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		return nil
	}
	if _, found, err := s.ItemByID(r.Context(), personID, id); err != nil || !found {
		return err
	}
	return s.SetItemState(r.Context(), id, squirrel.ItemDropped, now())
}
