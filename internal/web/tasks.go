package web

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// taskLimit caps both lists. The cap is what makes "there is more" truthful,
// the same device the deck and the shelf use.
const taskLimit = 30

// What you decided: things you chose to do once, which left the pile when you
// chose them and are archived when you have done them.
//
// The pile holds what you have not decided about — that is what triage is for,
// and why stopping partway is fine. This holds what you have. Keeping both in
// one list would mean triage stopped meaning anything.
func tasksHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		items, more, err := s.Tasks(r.Context(), personID, taskLimit)
		if err != nil {
			fail(w, err)
			return
		}
		v := view{Here: "tasks", More: more}
		for _, it := range items {
			v.Results = append(v.Results, toView(it))
		}
		renderWith(w, r, s, opts, "tasks", v)
	}
}

// What you have done. Never how many, and never crossed out — striking through
// your own words would be the product marking them as spent.
func archiveHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		items, more, err := s.ArchivedTasks(r.Context(), personID, taskLimit)
		if err != nil {
			fail(w, err)
			return
		}
		v := view{Here: "archive", More: more}
		for _, it := range items {
			v.Results = append(v.Results, toView(it))
		}
		renderWith(w, r, s, opts, "archive", v)
	}
}

// Two ways out of a task, and only two.
//
// Dropping is deliberately absent: a task you no longer want is a note you no
// longer want, and it gets there by ceasing to be a task first. One way to say
// a thing.
func taskActHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
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

		switch r.FormValue("act") {
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
			http.Redirect(w, r, "/tasks", http.StatusSeeOther)
			return
		}
		if err != nil {
			fail(w, err)
			return
		}
		http.Redirect(w, r, backToTasks(r), http.StatusSeeOther)
	}
}

// backToTasks returns to the screen the button was on.
func backToTasks(r *http.Request) string {
	if r.FormValue("from") == "archive" {
		return "/tasks/done"
	}
	return "/tasks"
}

// Decided outright, in the slot's own shape rather than the new-chore form's:
// a task is a sentence, not a configuration.
//
// It does not reuse /capture. That route makes notes; this one makes tasks,
// and sharing it would mean a flag — which is how a capture route grows a
// second meaning.
func newTaskHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/tasks", http.StatusSeeOther)
			return
		}
		text := strings.TrimSpace(r.FormValue("text"))
		if text == "" {
			http.Redirect(w, r, "/tasks", http.StatusSeeOther)
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
		http.Redirect(w, r, "/tasks", http.StatusSeeOther)
	}
}
