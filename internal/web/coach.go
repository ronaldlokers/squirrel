package web

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The coach, on the screen.
//
// A widget on every screen rather than a fourth door. DESIGN.md's rule against
// modals carries its own condition — "for anything that needs neither
// interruption nor protected focus" — and this meets it: a coach conversation
// happens when everything else on screen is noise, and protected focus is the
// whole point of the surface. The chore picker was refused a modal because
// choosing an interval needs neither. That reasoning is untouched.
//
// `/coach` is a real page. The sheet is pile.js upgrading a real route, the
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

// coachAvailable reports whether there is anything behind the acorn.
//
// When there is not, the acorn is still drawn and `/coach` still answers: the
// four chips are deterministic and the ladder behind them is what shipped
// before any of this existed. The only difference a missing key makes is that
// typing a sentence gets the four chips back instead of an answer.
func coachAvailable(opts Options) bool { return opts.Ask != nil }

// coachHandler is the page.
//
// It paints with whatever the picker would hand you right now, and painting
// costs nothing: no model is called on open. Idle opens have to be free or the
// widget becomes a thing you think about before pressing, and thinking about it
// is the cost this product spends everything else avoiding.
func coachHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		renderCoach(w, r, s, opts, personID, coachView{})
	}
}

// coachSayHandler is one turn.
//
// Two ways in, and they are the same route on purpose: a chip is the sentence
// you did not have to type. Someone at the moment of least capacity should not
// have to compose anything to be helped, which is what the four blockers are
// for — one press, and the answer still comes back about this task rather than
// in general.
func coachSayHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/coach", http.StatusSeeOther)
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
			http.Redirect(w, r, "/coach", http.StatusSeeOther)
			return
		}

		if !coachAvailable(opts) {
			// No coach, and the honest answer to a sentence is the four
			// chips: the ladder cannot read a paragraph, but it can be asked
			// which of four things is in the way. The words are kept.
			renderCoach(w, r, s, opts, personID, coachView{Said: said, AskWhich: true})
			return
		}

		reply, err := opts.Ask(r.Context(), personID, "sheet", said, subjectFor(s, opts, r))
		if err != nil {
			// The floor. What was typed stays in the box — a box that clears
			// on failure is a box that eats thoughts, which is the slot's rule
			// and it holds here for the same reason.
			renderCoach(w, r, s, opts, personID, coachView{Said: said, AskWhich: true})
			return
		}

		remember(opts, personID, said, reply)
		// Redirect rather than render, so the answer survives a reload and
		// pressing back does not offer to ask again. The conversation is in
		// the window; this page is a view of it.
		http.Redirect(w, r, "/coach", http.StatusSeeOther)
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

	// The answer travels in the address bar, the same way /now/stuck already
	// sends it — so the ladder's words live in one place, in the core, and the
	// control that comes with them is drawn from the same view the home screen
	// draws. A reload re-reads rather than repeating the press.
	remember(opts, personID, squirrel.BlockerWords[b], squirrel.UnstuckFor(b).Line)
	http.Redirect(w, r, "/coach?stuck="+url.QueryEscape(string(b)), http.StatusSeeOther)
}

// coachCloseHandler is the way out, and closing means one thing: the
// conversation is over.
//
// It does not mean anything else. The widget never initiates — you opened it —
// so three opens with no engagement is not a signal, the acorn does not dim,
// and nothing goes quiet. Reading meaning into how a button gets used would be
// the product forming an opinion about you from behaviour rather than from
// something you said, and the check-in already exists for saying it.
func coachCloseHandler(opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if personID, ok := opts.person(); ok {
			forget(opts, personID)
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, backTolerant(r.FormValue("from")), http.StatusSeeOther)
	}
}

// backTolerant is where closing returns to.
//
// Only a path this screen serves, and only ever a path: the value arrives from
// a form field and a form field is a place a stranger can type. An open
// redirect from a page behind forward-auth is still an open redirect.
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

// coachView is what the page needs beyond the conversation itself.
type coachView struct {
	// Said is what is in the box: preserved across a failure, and empty on an
	// ordinary open.
	Said string
	// AskWhich means the four chips are the answer this time — no coach, or a
	// coach that could not say anything usable.
	AskWhich bool
}

// renderCoach draws the page: the offer first, then the conversation, then the
// chips and the box.
func renderCoach(w http.ResponseWriter, r *http.Request, s Store, opts Options, personID int64, cv coachView) {
	v := view{
		Here:      "coach",
		Scrolling: true,
		Said:      cv.Said,
		Coach: &coachPanel{
			// Painted from the picker, not from a model. Opening costs
			// nothing and has to keep costing nothing.
			Offer:    offerFor(s, opts, r, true),
			Talking:  coachAvailable(opts),
			AskWhich: cv.AskWhich || !coachAvailable(opts),
			Blockers: blockerChips(),
			From:     backTolerant(r.URL.Query().Get("from")),
			// The ladder's control, when a chip was the last thing pressed.
			// Read out of the address bar and read the way a stranger's typing
			// is read: anything that is not one of the four is no answer.
			Unstuck: unstuckFrom(r.URL.Query()),
		},
	}
	if opts.Recent != nil {
		v.Coach.Said = opts.Recent(personID)
	}
	renderWith(w, r, s, opts, "coach", v)
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
