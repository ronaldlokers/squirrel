package web

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A fixed point, and what is pointing at it.
//
// A browsable set of appointments is a calendar, and a calendar is a thing you
// are behind on — so this holds only what is still ahead. Nothing past, nothing
// done, no count, and nothing here has been missed.
//
// Notes point at an appointment rather than the appointment growing fields of
// its own: a thought living on an appointment instead of in the pile is a
// thought `!find` cannot reach.

// upcomingLimit caps the list the way every other list here is capped. The cap
// is what makes "there is more" truthful — and it is never rendered as a
// number.
const upcomingLimit = 30

func atOneHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
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
			slog.Error("reading what points at it", "error", err)
		}

		// The notification's own URL, and it keeps working forever: one sent
		// last week is still on a lock screen, and a link that 404s is worse
		// than one that lands somewhere true.
		//
		// It writes the turn and sends you to the conversation, so tapping a
		// warning puts the appointment at the live edge — the thing you are
		// about to leave for, with what to take on it, under everything else
		// that was said today.
		//
		// A GET that writes, which nothing else here does. It is a press: the
		// notification was the thing you tapped, and the redirect means a
		// reload of where you land does not write again.
		keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: m.Label},
			fixedPointTurn(m, notes),
		})
		http.Redirect(w, r, "/", http.StatusSeeOther)
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
		personID, ok := personOf(r)
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
		personID, ok := personOf(r)
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

// atOpenHandler draws one fixed point into the conversation: when to leave,
// what to take, and the notes pointing at it.
//
// `/at/{id}` stays a route because a notification sent yesterday is still on a
// lock screen — it writes the same turn and redirects here, so the tap and the
// press arrive at the same place. See TestTheNotificationsURLLandsInTheConversation.
func atOpenHandler(s Store, opts Options) http.HandlerFunc {
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

// atNewHandler asks which day.
//
// Turning to another month re-asks rather than answering: a page turned is not
// something you said, so it draws the question again without writing your half
// of it into the record.
func atNewHandler(s Store, opts Options) http.HandlerFunc {
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
		label := strings.TrimSpace(r.FormValue("label"))
		if label == "" {
			// The agenda's dock, pressed empty. Nothing rather than a
			// redirect, for the reason captureHandler states at the same
			// branch: the script follows a redirect without being told, and
			// what comes back is a whole page it then pastes into the room.
			// This route was the one dock that never got that guard.
			if wantsFragment(r) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		month := squirrel.StartOfDay(now())
		turning := false
		if want := r.FormValue("month"); want != "" {
			m, err := time.ParseInLocation("2006-01", want, now().Location())
			if err != nil {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
			month, turning = m, true
		}

		said := []squirrel.Turn{askForADay(label, month)}
		if !turning {
			said = append([]squirrel.Turn{{Who: squirrel.SpeakerYou, Words: label}}, said...)
		}
		answerWith(w, r, keepSaid(r.Context(), s, personID, said), "/")
	}
}

// atMakeHandler is the answer: a day and a time, composed into the sentence the
// parser already reads.
//
// No arithmetic here. MomentOn is where "14:30" means something, and a second
// place that built a time would be a second place to be wrong — the same
// reasoning composeEvery is written under.
func atMakeHandler(s Store, opts Options) http.HandlerFunc {
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
		label := strings.TrimSpace(r.FormValue("label"))
		at := r.FormValue("at")
		if label == "" || !offeredTime(at) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		day, err := time.ParseInLocation("2006-01-02", r.FormValue("day"), now().Location())
		if err != nil || day.Before(squirrel.StartOfDay(now())) {
			// A day nobody offered. The picker draws none in the past, and an
			// appointment you are already late for is the one thing this list
			// may not hold.
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		m, ok := squirrel.MomentOn(opts.Location, day, "at "+at+" "+label, now())
		if !ok {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		kept, err := s.CreateMoment(r.Context(), personID, m)
		if err != nil {
			fail(w, err)
			return
		}

		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: kept.Starts.Format("Monday 2 January") + ", " + at},
			fixedPointTurn(kept, nil),
		}), "/")
	}
}

// offeredTime is the guard on a value that arrives from a form. Only the three
// the picker draws; anything else is a sentence, which the dock reads.
func offeredTime(at string) bool {
	for _, t := range pickTimes {
		if t == at {
			return true
		}
	}
	return false
}
