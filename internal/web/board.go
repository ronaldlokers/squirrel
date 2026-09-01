package web

import (
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

type boardView struct {
	V      string
	Tray   []trayView
	Now    string
	Day    string
	Pulled *offerView
	Timer  *timerView
	Bays   []bayView
}

type bayView struct {
	Key      string
	Name     string
	Question string
	Shelves  bool
	Strips   []stripView
}

type stripView struct {
	ID      int64
	What    string
	Words   string
	Mark    string
	Big     bool
	Answers []answerView
}

type answerView struct {
	Act   string
	Words string
	Key   string
	Look  string
}

type trayView struct {
	ID     int64
	What   string
	Words  string
	Left   string
	Newest bool
}

func boardHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			http.Error(w, "who are you", http.StatusForbidden)
			return
		}
		at := now()
		v := boardView{
			Now:    at.Format("15:04"),
			Day:    at.Format("Monday 2 January"),
			Pulled: offerFor(s, opts, r, false, false),
			Timer:  runningTimer(s, opts, r),
			Tray:   trayStrips(r, s, opts, personID, at),
			Bays: []bayView{
				{Key: "notes", Name: "the notes", Question: "what is it", Shelves: true,
					Strips: noteStrips(r, s, personID, at)},
				{Key: "chores", Name: "the chores", Question: "what comes back?",
					Strips: choreStrips(r, s, personID)},
				{Key: "tasks", Name: "the tasks", Question: "what did you decide?",
					Strips: taskStrips(r, s, personID, at)},
				{Key: "agenda", Name: "the agenda", Question: "what is happening?",
					Strips: agendaStrips(r, s, personID, at)},
			},
		}
		renderBoard(w, v)
	}
}

func noteStrips(r *http.Request, s Store, personID int64, at time.Time) []stripView {
	items, _, err := s.OpenItems(r.Context(), personID, boardDeep)
	if err != nil {
		slog.Error("reading the notes for the board", "error", err)
		return nil
	}
	out := make([]stripView, 0, len(items))
	for _, it := range items {
		out = append(out, stripView{
			ID: it.ID, What: "note", Words: it.RawText,
			Mark: markOfDay(it.ReceivedAt, at), Answers: noteAnswers,
		})
	}
	return out
}

func taskStrips(r *http.Request, s Store, personID int64, at time.Time) []stripView {
	items, _, err := s.Tasks(r.Context(), personID, boardDeep)
	if err != nil {
		slog.Error("reading the tasks for the board", "error", err)
		return nil
	}
	out := make([]stripView, 0, len(items))
	for _, it := range items {
		out = append(out, stripView{
			ID: it.ID, What: "task", Words: it.RawText,
			Mark: markOfDay(it.ReceivedAt, at), Answers: taskAnswers,
		})
	}
	return out
}

func choreStrips(r *http.Request, s Store, personID int64) []stripView {
	chores, err := s.ActiveChores(r.Context(), personID)
	if err != nil {
		slog.Error("reading the chores for the board", "error", err)
		return nil
	}
	out := make([]stripView, 0, len(chores))
	for _, c := range chores {
		out = append(out, stripView{
			ID: c.ID, What: "chore", Words: c.Name,
			Mark: squirrel.Cadence(c.EveryDays), Answers: choreAnswers,
		})
	}
	return out
}

func agendaStrips(r *http.Request, s Store, personID int64, at time.Time) []stripView {
	soon, err := s.Upcoming(r.Context(), personID, at, boardDeep)
	if err != nil {
		slog.Error("reading what is coming for the board", "error", err)
		return nil
	}
	out := make([]stripView, 0, len(soon))
	for _, m := range soon {
		out = append(out, stripView{ID: m.ID, What: "moment", Words: m.Label, Mark: markOfMoment(m, at), Big: true})
	}
	return out
}

var noteAnswers = []answerView{
	{Act: "done", Words: "done", Key: "D", Look: "did"},
	{Act: "keep", Words: "keep", Key: "K"},
	{Act: "drop", Words: "drop", Key: "X", Look: "no"},
}

var taskAnswers = []answerView{
	{Act: "done", Words: "done", Key: "D", Look: "did"},
	{Act: "drop", Words: "drop", Key: "X", Look: "no"},
}

var choreAnswers = []answerView{
	{Act: "did", Words: "did it", Key: "D", Look: "did"},
	{Act: "later", Words: "later", Key: "L", Look: "no"},
}

var leftWords = map[squirrel.ItemState]string{
	squirrel.ItemDone:    "done",
	squirrel.ItemKept:    "kept",
	squirrel.ItemDropped: "dropped",
}

func trayStrips(r *http.Request, s Store, opts Options, personID int64, at time.Time) []trayView {
	gone, err := s.TriagedSince(r.Context(), personID, dayOpened(at, opts.Location))
	if err != nil {
		slog.Error("reading today's tray", "error", err)
		return nil
	}
	out := make([]trayView, 0, len(gone))
	for i, it := range gone {
		out = append(out, trayView{
			ID: it.ID, What: "note", Words: it.RawText,
			Left: leftWords[it.State], Newest: i == 0,
		})
	}
	return out
}

// dayOpened is when today began where the person is, which is what the tray
// empties on. Not midnight UTC: a note kept at 23:30 belongs to the evening it
// was kept in.
func dayOpened(at time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.Local
	}
	there := at.In(loc)
	return time.Date(there.Year(), there.Month(), there.Day(), 0, 0, 0, 0, loc)
}

func markOfDay(said, today time.Time) string {
	switch {
	case sameDay(said, today):
		return said.Format("15:04")
	case sameDay(said, today.AddDate(0, 0, -1)):
		return "yest"
	case today.Sub(said) < 7*24*time.Hour:
		return said.Format("Mon")
	}
	return said.Format("2 Jan")
}

func markOfMoment(m squirrel.Moment, at time.Time) string {
	if sameDay(m.Starts, at) {
		return m.Starts.Format("15:04")
	}
	if m.Starts.Sub(at) < 7*24*time.Hour {
		return m.Starts.Format("Mon")
	}
	return m.Starts.Format("2 Jan")
}

const boardDeep = 40

var boardPage = template.Must(
	template.New("board.html").Funcs(helpers).ParseFS(templatesFS(), "templates/board.html"))

func renderBoard(w http.ResponseWriter, v boardView) {
	v.V = stamp()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := boardPage
	if devDir != "" {
		reparsed, err := template.New("board.html").Funcs(helpers).ParseFS(templatesFS(), "templates/board.html")
		if err != nil {
			slog.Error("re-reading the board", "error", err)
		} else {
			t = reparsed
		}
	}
	if err := t.ExecuteTemplate(w, "board", v); err != nil {
		slog.Error("drawing the board", "error", err)
	}
}

func boardActHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/board", http.StatusSeeOther)
			return
		}
		id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
		if err := answerOnTheBoard(r, s, personID, r.FormValue("what"), r.FormValue("answer"), id); err != nil {
			slog.Error("answering a strip", "error", err)
			fail(w, err)
			return
		}
		http.Redirect(w, r, "/board", http.StatusSeeOther)
	}
}

func answerOnTheBoard(r *http.Request, s Store, personID int64, what, answer string, id int64) error {
	at := now()
	switch what {
	case "note", "task":
		state, ok := boardStates[answer]
		if !ok {
			return nil
		}
		return s.SetItemState(r.Context(), id, state, at)
	case "chore":
		switch answer {
		case "did":
			return s.RecordCompletion(r.Context(), id, personID, "board", at)
		case "later":
			return s.Refuse(r.Context(), personID, squirrel.OfferChore, id, at)
		}
	}
	return nil
}

var boardStates = map[string]squirrel.ItemState{
	"done": squirrel.ItemDone,
	"keep": squirrel.ItemKept,
	"drop": squirrel.ItemDropped,
}

// boardUndoHandler puts a strip back in the rack it left. The state it goes
// back to is `open` for every exit, because the three exits are the same door
// from the pile's side and nothing else is remembered about which one was taken.
func boardUndoHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := personOf(r); !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/board", http.StatusSeeOther)
			return
		}
		if id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64); id != 0 {
			if err := s.SetItemState(r.Context(), id, squirrel.ItemOpen, now()); err != nil {
				slog.Error("putting a strip back", "error", err)
				fail(w, err)
				return
			}
		}
		http.Redirect(w, r, "/board", http.StatusSeeOther)
	}
}
