package web

import (
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

type boardView struct {
	V      string
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
	ID    int64
	Words string
	Mark  string
	Big   bool
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
		out = append(out, stripView{ID: it.ID, Words: it.RawText, Mark: markOfDay(it.ReceivedAt, at)})
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
		out = append(out, stripView{ID: it.ID, Words: it.RawText, Mark: markOfDay(it.ReceivedAt, at)})
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
		out = append(out, stripView{ID: c.ID, Words: c.Name, Mark: squirrel.Cadence(c.EveryDays)})
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
		out = append(out, stripView{ID: m.ID, Words: m.Label, Mark: markOfMoment(m, at), Big: true})
	}
	return out
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
