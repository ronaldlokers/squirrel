package web

import (
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

type boardView struct {
	V      string
	Light  int
	Tray   []trayView
	Kept   bool
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
	Writes   bool
	Rhythms  []rhythmView
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

type rhythmView struct {
	Days  int
	Words string
}

var theRhythms = []rhythmView{
	{Days: 1, Words: "a day"},
	{Days: 7, Words: "a week"},
	{Days: 14, Words: "2 weeks"},
	{Days: 30, Words: "a month"},
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
			Kept:   r.URL.Query().Get("kept") == "notes",
			Now:    at.Format("15:04"),
			Day:    at.Format("Monday 2 January"),
			Pulled: offerFor(s, opts, r, false, false),
			Timer:  runningTimer(s, opts, r),
			Tray:   trayStrips(r, s, opts, personID, at),
			Bays: []bayView{
				{Key: "notes", Name: "the notes", Question: "what is it", Writes: true, Shelves: true,
					Strips: noteStrips(r, s, personID, at)},
				{Key: "chores", Name: "the chores", Question: "what comes back?", Writes: true, Rhythms: theRhythms,
					Strips: choreStrips(r, s, personID)},
				{Key: "tasks", Name: "the tasks", Question: "what did you decide?", Writes: true,
					Strips: taskStrips(r, s, personID, at)},
				{Key: "agenda", Name: "the agenda", Question: "at 14:30 dentist", Writes: true,
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
	v.Light = squirrel.Light(now())
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
		it, mine, err := s.ItemByID(r.Context(), personID, id)
		if err != nil {
			return err
		}
		if !mine {
			return nil
		}
		_, err = s.MoveItemState(r.Context(), it.ID, it.State, state, at)
		return err
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
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/board", http.StatusSeeOther)
			return
		}
		if id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64); id != 0 {
			it, mine, err := s.ItemByID(r.Context(), personID, id)
			if err != nil {
				slog.Error("reading the strip to put back", "error", err)
				fail(w, err)
				return
			}
			if mine {
				if _, err := s.MoveItemState(r.Context(), it.ID, it.State, squirrel.ItemOpen, now()); err != nil {
					slog.Error("putting a strip back", "error", err)
					fail(w, err)
					return
				}
			}
		}
		http.Redirect(w, r, "/board", http.StatusSeeOther)
	}
}

// boardNewHandler is what a blank strip does. Its bay decides what the words
// become, which is the whole reason each rack asks its own question.
//
// The notes bay writes to the spool rather than to the database, because that
// is what capture is: the words reach fsynced disk before anything answers, and
// the drain resolves whose they are. The tasks bay does not — a task is a
// decision you already made about something, and a decision has no spool.
//
// The chores and agenda bays are not here. Both need a second answer before
// there is anything to keep — a rhythm, a day — and a blank strip that quietly
// dropped the words while it asked would be the one thing this product may
// never do.
func boardNewHandler(s Store, opts Options) http.HandlerFunc {
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
		words := strings.TrimSpace(r.FormValue("words"))
		if words == "" {
			http.Redirect(w, r, "/board", http.StatusSeeOther)
			return
		}
		if len(words) > captureLimit {
			words = words[:captureLimit]
		}

		switch r.FormValue("bay") {
		case "chores":
			if days, _ := strconv.Atoi(r.FormValue("every")); days > 0 {
				every := time.Duration(days) * 24 * time.Hour
				if _, err := s.UpsertChore(r.Context(), personID, words, every, every/10); err != nil {
					fail(w, err)
					return
				}
				break
			}
			if err := keepAsANote(r, opts, words); err != nil {
				fail(w, err)
				return
			}
			http.Redirect(w, r, "/board?kept=notes", http.StatusSeeOther)
			return
		case "agenda":
			if m, ok := squirrel.ParseMomentIn(opts.Location, words, now()); ok {
				if _, err := s.CreateMoment(r.Context(), personID, m); err != nil {
					fail(w, err)
					return
				}
				break
			}
			if err := keepAsANote(r, opts, words); err != nil {
				fail(w, err)
				return
			}
			http.Redirect(w, r, "/board?kept=notes", http.StatusSeeOther)
			return
		case "notes":
			if err := keepAsANote(r, opts, words); err != nil {
				fail(w, err)
				return
			}
		case "tasks":
			id, err := s.InsertItemReturningID(r.Context(), squirrel.Item{
				Transport: "screen", PersonID: &personID, RawText: words,
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
		}
		http.Redirect(w, r, "/board", http.StatusSeeOther)
	}
}

// keepAsANote is the floor under every blank strip: words that are not what the
// rack asked for are still a thought, and a thought goes to the spool. The one
// thing a bay may not do is drop what you typed because it was the wrong shape.
func keepAsANote(r *http.Request, opts Options, words string) error {
	sender := subOf(r)
	if _, err := opts.Spool.Write(squirrel.Capture{
		Transport:  squirrel.ScreenTransport,
		SenderID:   &sender,
		Text:       words,
		Payload:    []byte(squirrel.ScreenCapture),
		ReceivedAt: now(),
	}); err != nil {
		slog.Warn("a capture from the board could not be spooled", "error", err)
		return err
	}
	return nil
}
