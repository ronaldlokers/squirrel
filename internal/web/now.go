package web

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The one thing, on the screen.
//
// An earlier home screen showed the newest note — readable, not actionable,
// chosen by arrival order — and it was cut on sight with seven prohibitions
// written down. This breaks exactly one of them, "no action", and keeps the
// other six. The difference is the whole argument: that preview greeted you
// with a slice of your backlog, and this is one thing the product chose, with
// one way to do it and a one-press way to refuse it.
//
// The six that hold, plus the two this adds:
//
//   - never more than one, and never a list;
//   - never a count, a stack, or a "more";
//   - no state colour and no urgency copy: it is never late, never red;
//   - absent rather than empty when there is nothing to offer;
//   - refusing costs one press, has no consequence, and asks nothing back;
//   - it changes what is offered, never what is true;
//   - it is chosen deterministically and explains itself in one clause.
//
// offerFor reads it, and answers nil for every failure. A picker that cannot
// answer must not be able to take down a page that rendered without one for
// the product's whole life.
func offerFor(s Store, opts Options, r *http.Request, anyway bool) *offerView {
	personID, ok := opts.person()
	if !ok {
		return nil
	}
	o, found, err := s.PickNow(r.Context(), personID, now(), anyway)
	if err != nil || !found {
		return nil
	}
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

// nowActHandler is the offer's three answers.
//
// Every one of them is a form POST answered with a 303, the same shape the
// pile's and the chores' actions already use — so the whole region works with
// no script at all, and a reload after pressing re-reads rather than repeating
// the press.
func nowActHandler(s Store, opts Options) http.HandlerFunc {
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
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// nowStuckHandler is "I can't start", and it is the one branch of the offer
// that can end without anything having been done to anything.
//
// It answers with a redirect carrying the answer in the address bar rather
// than rendering here, so the response is a page you can reload, bookmark or
// come back to — and so the ladder's words live in one place, in the core,
// where the chat reads them too.
func nowStuckHandler(s Store, opts Options) http.HandlerFunc {
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
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/?stuck="+url.QueryEscape(string(b)), http.StatusSeeOther)
	}
}

// unstuckFrom reads the ladder's answer back out of the address bar.
//
// It arrives from a redirect this package wrote, but it arrives through the
// address bar, so it is read as though a stranger typed it: anything that is
// not one of the four is no answer rather than a bad one.
func unstuckFrom(q url.Values) *unstuckView {
	b, ok := squirrel.ParseBlocker(q.Get("stuck"))
	if !ok {
		return nil
	}
	u := squirrel.UnstuckFor(b)
	if u.Refuse {
		return nil
	}
	return &unstuckView{Line: u.Line, Minutes: u.Minutes, Ask: u.Ask}
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

// startFromOffer puts a timer on the thing being offered, and records that the
// offer is what started it.
//
// The same row `!timer` and the chores screen's own buttons write, so a body
// double started from home is the body double, not a second kind of one. The
// label is the offer's own words rather than anything this handler invents.
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
