package web

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// taskLimit caps both lists, which is what makes "there is more" truthful.
const taskLimit = 30

// Two ways out of a task, and only two.
//
// Dropping is deliberately absent: a task you no longer want is a note you no
// longer want, and it gets there by ceasing to be a task first. One way to say
// a thing.
func taskActHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/tasks", http.StatusSeeOther)
			return
		}
		id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
		if err != nil || id < 1 {
			http.Redirect(w, r, "/tasks", http.StatusSeeOther)
			return
		}

		// The row is read before it is written to, which the words need anyway
		// and which scopes the write: ItemByID is the person's, and
		// SetItemState takes a bare id. A row that is not yours is not yours
		// to act on.
		it, found, err := s.ItemByID(r.Context(), personID, id)
		if err != nil {
			fail(w, err)
			return
		}
		if !found {
			http.Redirect(w, r, backToTheRoom(r), http.StatusSeeOther)
			return
		}

		act := r.FormValue("act")
		switch act {
		case "done":
			err = s.SetItemState(r.Context(), id, squirrel.ItemDone, now())
		case "open":
			// Out of the archive: still a task, not done any more.
			err = s.SetItemState(r.Context(), id, squirrel.ItemOpen, now())
		case "untask":
			// Deciding was the mistake. It goes back to the pile as the note
			// it was, in the state it was in — undoing a decision must not
			// require finishing it.
			_, err = s.SetItemKind(r.Context(), personID, id, squirrel.ItemNote)
		default:
			http.Redirect(w, r, backToTheRoom(r), http.StatusSeeOther)
			return
		}
		if err != nil {
			fail(w, err)
			return
		}
		answerWith(w, r, keepSaid(r.Context(), s, personID,
			saidAboutATask(act, it.RawText)), backToTheRoom(r))
	}
}

// Decided outright, in the slot's own shape rather than the new-chore form's:
// a task is a sentence, not a configuration.
//
// It does not reuse /capture. That route makes notes; this one makes tasks,
// and sharing it would mean a flag — which is how a capture route grows a
// second meaning.
func newTaskHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, backToTheRoom(r), http.StatusSeeOther)
			return
		}
		text := strings.TrimSpace(r.FormValue("text"))
		if text == "" {
			// Nothing to decide on. Silence rather than a scolding.
			answerWith(w, r, nil, backToTheRoom(r))
			return
		}
		if len(text) > captureLimit {
			text = text[:captureLimit]
		}

		id, err := s.InsertItemReturningID(r.Context(), squirrel.Item{
			Transport: "screen", PersonID: &personID, RawText: text,
			Payload: []byte(squirrel.ScreenCapture), ReceivedAt: now(),
		})
		if err != nil {
			fail(w, err)
			return
		}
		if _, err := s.SetItemKind(r.Context(), personID, id, squirrel.ItemTask); err != nil {
			fail(w, err)
			return
		}
		// The task you just decided on, as a card, so it is on the screen
		// rather than somewhere you have to go and look at it. The screen it
		// used to send you back to is a message now.
		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: text},
			{Who: squirrel.SpeakerBuddy, Words: "On the list."},
		}), backToTheRoom(r))
	}
}
