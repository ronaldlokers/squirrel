package web

import (
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

type notesView struct {
	V       string
	Light   int
	Now     string
	Day     string
	Wall    []stripView
	Writer  bayView
	More    bool
	Trouble bool
	Find    string
	Dial    *dialView
	AnyTold bool
	Kept    bool
	You     whom
}

func notesHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			http.Error(w, "who are you", http.StatusForbidden)
			return
		}
		at := now().In(zoneOf(r.Context()))
		asking, _ := strconv.ParseInt(r.URL.Query().Get("chore"), 10, 64)
		v := notesView{
			Now:  at.Format("15:04"),
			Day:  at.Format("Monday 2 January"),
			Kept: r.URL.Query().Get("kept") == "1",
		}

		var (
			g     errgroup.Group
			seen  map[string]squirrel.Noticed
			notes []stripView
			told  []toldView
		)
		g.SetLimit(squirrel.MaxConns)
		g.Go(func() error { seen = whatWasNoticed(r, s, personID); return nil })
		g.Go(func() error { notes, v.Trouble, v.More = wallStrips(r, s, personID, at); return nil })
		g.Go(func() error { v.Dial = howYouAre(r, s, personID, at); return nil })
		g.Go(func() error { v.You = youFor(r.Context(), s, personID); return nil })
		g.Go(func() error { told = whatWasSaid(r, s, personID, at, 1); return nil })
		_ = g.Wait()

		v.AnyTold = len(told) > 0
		justAsked, _ := strconv.ParseInt(r.URL.Query().Get("answered"), 10, 64)
		rows := askable(notes, "notes", coachAvailable(opts))
		rows = answered(marked(marked(rows, "note", seen), "ask:note", seen), justAsked)
		v.Wall = askedForARhythm(rows, asking)
		v.Writer = bayView{
			Key: "notes", Question: "what is it", Writes: true,
			Camera:  opts.Photos != nil,
			Asking:  strings.TrimSpace(r.URL.Query().Get("nophoto")),
			Refused: r.URL.Query().Has("nophoto"),
			Offline: r.URL.Query().Get("offline") == "1",
		}
		renderNotes(w, v)
	}
}

func wallStrips(r *http.Request, s Store, personID int64, at time.Time) ([]stripView, bool, bool) {
	items, more, err := s.OpenItems(r.Context(), personID, wallDeep)
	if err != nil {
		slog.Error("reading the notes", "error", err)
		return nil, true, false
	}
	out := make([]stripView, 0, len(items))
	for _, it := range items {
		out = append(out, stripView{
			ID: it.ID, What: "note", Words: it.RawText,
			Mark: markOfDay(it.ReceivedAt, at), Answers: noteAnswers,
			Photo: it.PhotoName != "",
		})
	}
	return out, false, more
}

const wallDeep = 60

var notesPage = template.Must(template.New("notes.html").Funcs(helpers).ParseFS(templatesFS(),
	"templates/notes.html", "templates/board.html", "templates/chips.html", "templates/strip.html"))

func renderNotes(w http.ResponseWriter, v notesView) {
	v.V = stamp()
	v.Light = squirrel.Light(now())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := notesPage
	if devDir != "" {
		reparsed, err := template.New("notes.html").Funcs(helpers).ParseFS(templatesFS(),
			"templates/notes.html", "templates/board.html", "templates/chips.html", "templates/strip.html")
		if err != nil {
			slog.Error("re-reading the notes", "error", err)
		} else {
			t = reparsed
		}
	}
	if err := t.ExecuteTemplate(w, "notes", v); err != nil {
		slog.Error("drawing the notes", "error", err)
	}
}
