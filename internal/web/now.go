package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The one thing, on the screen. The rules it holds to — never a list, never a
// count, no state colour, absent rather than empty, refusable in one press —
// are argued in DESIGN.md under The Offer.
//
// offerFor answers nil for every failure. A picker that cannot answer must not
// take down a page that rendered without one for the product's whole life.
//
// mayAsk says whether this render may spend a model call. Home may; a surface
// that has to cost nothing to open may not.
func offerFor(s Store, opts Options, r *http.Request, anyway, mayAsk bool) *offerView {
	personID, ok := personOf(r)
	if !ok {
		return nil
	}
	o, found, err := s.PickNow(r.Context(), personID, now(), anyway)
	if err != nil || !found {
		// Nothing to hand over, and the coach is not asked: a model invited to find
		// something when the rules found nothing would be answering a different
		// question.
		return nil
	}
	o = judged(opts, r, personID, o, mayAsk)
	v := &offerView{
		Kind:    string(o.Kind),
		RefID:   o.RefID,
		Text:    o.Text,
		Because: o.Because,
	}
	// A running timer is a thing you are doing rather than a row that was
	// picked, so it carries no buttons of its own: the lid already has the one
	// control it needs, which is the way to stop.
	v.Running = o.Kind == squirrel.OfferTimer
	return v
}

// judged is the offer after a model has looked at it, or the same offer. Same
// type in and out, so nothing downstream can tell which produced it.
// JudgementHelps is the core's rule about which kinds a model may touch.
func judged(opts Options, r *http.Request, personID int64, o squirrel.Offer, mayAsk bool) squirrel.Offer {
	if opts.Decide == nil || !squirrel.JudgementHelps(o.Kind) {
		return o
	}
	kind, refID, text, because, ok := opts.Decide(r.Context(), personID, string(o.Kind), o.RefID, mayAsk)
	if !ok || text == "" || because == "" {
		return o
	}
	return squirrel.Offer{Kind: squirrel.OfferKind(kind), RefID: refID, Text: text, Because: because}
}

// nowActHandler is the offer's three answers, each a form POST answered with a
// 303 — so the region works with no script and a reload re-reads.
func nowActHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		kind := squirrel.OfferKind(r.FormValue("kind"))
		if !offerKinds[kind] {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		// A missing or unparseable id is zero, which is what an offer that
		// names no row already carries. It is not an error: the only thing that
		// can be done to such an offer is to turn it down.
		refID, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)

		var err error
		switch r.FormValue("act") {
		case "did":
			err = s.Did(r.Context(), personID, squirrel.Offer{Kind: kind, RefID: refID}, now())
		case "later":
			err = s.Refuse(r.Context(), personID, kind, refID, now())
		case "start":
			err = startFromOffer(s, r, personID)
		default:
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if err != nil {
			fail(w, err)
			return
		}
		forgetOffer(opts, personID)
		// What the two of you said about it. After the write, because a
		// conversation must not claim something happened that did not.
		answerWith(w, r, keepSaid(r.Context(), s, personID,
			saidAboutTheOffer(r.FormValue("act"), r.FormValue("label"))), "/")
	}
}

// forgetOffer drops the decision a model made. Called after the error check: an
// answer that did not land is not an answer, and throwing the decision away would
// swap the card underneath a press that failed.
func forgetOffer(opts Options, personID int64) {
	if opts.ForgetOffer == nil {
		return
	}
	opts.ForgetOffer(personID)
}

// nowStuckHandler is "I can't start", the one branch that can end without
// anything being done to anything. It redirects with the answer in the address
// bar, so the response is a page you can reload and the ladder's words live in
// the core where chat reads them too.
func nowStuckHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		b, ok := squirrel.ParseBlocker(r.FormValue("why"))
		if !ok {
			// Not one of the four. Nothing is done and nothing is said: this
			// arrives from a form, and a value that was never offered is read
			// the way a stranger's typing is read.
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// "Not today" is not an obstacle, it is a no — and it is the same no
		// that "not now" writes, arrived at from a different direction.
		if u := squirrel.UnstuckFor(b); u.Refuse {
			kind := squirrel.OfferKind(r.FormValue("kind"))
			refID, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
			if offerKinds[kind] {
				if err := s.Refuse(r.Context(), personID, kind, refID, now()); err != nil {
					fail(w, err)
					return
				}
			}
			// The same no as "not now", so the same decision has to go. The
			// three blockers below are not refusals and leave it alone: the
			// ladder answers underneath the card, and swapping the card there
			// would replace the thing you have just said you cannot start.
			forgetOffer(opts, personID)
			answerWith(w, r, keepSaid(r.Context(), s, personID,
				saidAboutTheOffer("later", r.FormValue("label"))), "/")
			return
		}
		// Broken into steps first, when that is what this blocker wants and
		// there is something to break down. The step is read back out of the
		// store rather than carried, so a reload shows where you actually are
		// rather than repeating a press.
		if o, found, err := s.PickNow(r.Context(), personID, now(), true); err == nil && found {
			smallerFor(s, opts, r, b, o)
		}
		answerWith(w, r, keepSaid(r.Context(), s, personID,
			saidAboutBeingStuck(s, opts, r, personID, b)), "/")
	}
}

// offerKinds is the vocabulary, as a map rather than a switch so an unknown
// kind is a lookup miss instead of a default branch someone later fills in
// with something destructive. The same device the pile's actions use.
var offerKinds = map[squirrel.OfferKind]bool{
	squirrel.OfferChore:  true,
	squirrel.OfferTask:   true,
	squirrel.OfferAgain:  true,
	squirrel.OfferMoment: true,
	squirrel.OfferTimer:  true,
}

// startFromOffer puts a timer on the thing being offered, writing the same row
// `!timer` does. The label is the offer's own words.
func startFromOffer(s Store, r *http.Request, personID int64) error {
	mins, err := strconv.Atoi(r.FormValue("minutes"))
	if err != nil || mins < 1 || mins > 180 {
		// Not one of the offered lengths. Nothing is started and nothing is
		// said: a button that was never drawn cannot have been pressed by
		// accident.
		return nil
	}
	label := strings.TrimSpace(r.FormValue("label"))
	if label == "" {
		label = "it"
	}
	if len(label) > choreNameLimit {
		label = label[:choreNameLimit]
	}
	if _, err := s.StartTimer(r.Context(), personID, label,
		time.Duration(mins)*time.Minute, now()); err != nil {
		return err
	}

	kind := squirrel.OfferKind(r.FormValue("kind"))
	refID, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	return s.RecordAnswer(r.Context(), personID, kind, refID, squirrel.AnswerStarted, now())
}
