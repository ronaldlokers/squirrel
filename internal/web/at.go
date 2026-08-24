package web

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A fixed point, and what is pointing at it.
//
// The list this product spent its whole life refusing — `PRODUCT.md` said a
// browsable set of your appointments is a calendar and a calendar is a thing
// you are behind on. The owner overturned that on 24 August 2026, and what the
// rule protected against is kept rather than argued away: this holds only what
// is still ahead. Nothing past, nothing done, no count, and nothing here has
// been missed.
//
// Notes point at an appointment rather than the appointment growing fields of
// its own, because a thought that lives on an appointment instead of in the
// pile is a thought `!find` cannot reach. See the spec at
// docs/superpowers/specs/2026-08-24-fixed-point-detail-design.md.

// upcomingLimit caps the list the way every other list here is capped. The cap
// is what makes "there is more" truthful — and it is never rendered as a
// number.
const upcomingLimit = 30

func atHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		coming, err := s.Upcoming(r.Context(), personID, now(), upcomingLimit)
		if err != nil {
			fail(w, err)
			return
		}
		renderWith(w, r, s, opts, "at", view{
			Here: "at", Scrolling: true, Upcoming: upcomingViews(coming),
		})
	}
}

func atOneHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		m, found, err := s.MomentByID(r.Context(), personID, id)
		if err != nil {
			fail(w, err)
			return
		}
		if !found {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		notes, err := s.NotesFor(r.Context(), personID, id)
		if err != nil {
			fail(w, err)
			return
		}
		renderWith(w, r, s, opts, "atone", view{
			Here: "at", Scrolling: true,
			Moment:   momentViewOf(m),
			Attached: attachedViews(notes),
		})
	}
}

// atNoteHandler is the slot on a fixed point.
//
// Two writes rather than one, the same way the new-task line does it: the spool
// answers "was this kept" and not "which row is it", and this needs the row to
// point it. What that costs is the spool's durability on this one path, and it
// is the same cost the tasks screen already accepted for the same reason.
//
// No picker anywhere. A picker would need a browsable list of appointments to
// choose from, which is exactly the shape the record refuses — here the
// appointment is the page you are already on.
func atNoteHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		back := "/at/" + strconv.FormatInt(id, 10)
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, back, http.StatusSeeOther)
			return
		}
		text := strings.TrimSpace(r.FormValue("words"))
		if text == "" {
			http.Redirect(w, r, back, http.StatusSeeOther)
			return
		}
		if len(text) > captureLimit {
			text = text[:captureLimit]
		}

		itemID, err := s.InsertItemReturningID(r.Context(), squirrel.Item{
			Transport: "screen", PersonID: &personID, RawText: text,
			Payload: []byte(squirrel.ScreenCapture), ReceivedAt: now(),
		})
		if err != nil {
			fail(w, err)
			return
		}
		if _, err := s.AttachNote(r.Context(), personID, itemID, id); err != nil {
			fail(w, err)
			return
		}
		http.Redirect(w, r, back, http.StatusSeeOther)
	}
}

// atDetachHandler puts a note back in the pile, which is the whole reversal.
func atDetachHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/at/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
			return
		}
		itemID, perr := strconv.ParseInt(r.FormValue("id"), 10, 64)
		if perr != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		moved, err := s.DetachNote(r.Context(), personID, itemID)
		if err != nil {
			fail(w, err)
			return
		}
		if !moved {
			// Not yours, or not pointing anywhere. Nothing happened, so
			// nothing is said about it.
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "back in the pile"},
			{Who: squirrel.SpeakerBuddy, Words: "It is in the pile again."},
		}), "/")
	}
}

// momentViewOf is one fixed point, with the arithmetic already done.
//
// Whether the window is open is worked out here rather than in the template,
// because a template that does arithmetic is a template that can disagree with
// the core about when to leave — and the core is where LeaveWords lives.
func momentViewOf(m squirrel.Moment) *momentView {
	return &momentView{
		ID:    m.ID,
		Label: m.Label,
		Words: squirrel.LeaveWords(m),
		Take:  m.Bring,
		Open:  m.Open(now()),
	}
}

// attachedViews reuses the pile's own note shape, because an attached note is
// an ordinary note and rendering it any other way would be the first step
// towards it becoming a different kind of thing.
func attachedViews(items []squirrel.Item) []noteView {
	out := make([]noteView, 0, len(items))
	for _, it := range items {
		out = append(out, toView(it))
	}
	return out
}

func upcomingViews(ms []squirrel.Moment) []momentView {
	out := make([]momentView, 0, len(ms))
	for _, m := range ms {
		out = append(out, *momentViewOf(m))
	}
	return out
}

// atOpenHandler draws one fixed point into the conversation.
//
// The same three things the page shows — when to leave, what to take, and the
// notes pointing at it — said rather than navigated to. `/at/{id}` stays a real
// page until phase 4: a notification sent yesterday is still on a lock screen.
func atOpenHandler(s Store, opts Options) http.HandlerFunc {
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
		id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		m, found, err := s.MomentByID(r.Context(), personID, id)
		if err != nil {
			fail(w, err)
			return
		}
		if !found {
			// Not yours, or gone. Nothing is drawn and nothing is said.
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		notes, err := s.NotesFor(r.Context(), personID, id)
		if err != nil {
			slog.Error("reading what points at it", "error", err)
		}

		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: m.Label},
			fixedPointTurn(m, notes),
		}), "/")
	}
}
