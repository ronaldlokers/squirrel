package web

import (
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

type boardView struct {
	V        string
	Find     string
	Found    []stripView
	Blockers []blockerView
	Unstuck  string
	In       string
	Light    int
	Tray     []trayView
	Kept     bool
	Now      string
	Day      string
	Pulled   *offerView
	Timer    *timerView
	Bays     []bayView
}

type bayView struct {
	In       bool
	Key      string
	Name     string
	Question string
	Writes   bool
	Rhythms  []rhythmView
	Shelves  bool
	Strips   []stripView
}

type stripView struct {
	// Back is a strip that has already left the pile: it carries the way back
	// and nothing else.
	Back    bool
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
		in := r.URL.Query().Get("bay")
		find := strings.TrimSpace(r.URL.Query().Get("find"))
		blockers, unstuck := stuckView(r.URL.Query().Get("stuck"))
		v := boardView{
			In:       in,
			Find:     find,
			Found:    whatMatched(r, s, personID, find, at),
			Blockers: blockers,
			Unstuck:  unstuck,
			Kept:     r.URL.Query().Get("kept") == "1",
			Now:      at.Format("15:04"),
			Day:      at.Format("Monday 2 January"),
			Pulled:   offerFor(s, opts, r, false, r.URL.Query().Get("ask") == "1"),
			Timer:    runningTimer(s, opts, r),
			Tray:     trayStrips(r, s, opts, personID, at),
			Bays: baysIn(in, []bayView{
				{Key: "notes", Name: "the notes", Question: "what is it", Writes: true, Shelves: true,
					Strips: noteStrips(r, s, personID, at)},
				{Key: "chores", Name: "the chores", Question: "what comes back?", Writes: true, Rhythms: theRhythms,
					Strips: choreStrips(r, s, personID)},
				{Key: "tasks", Name: "the tasks", Question: "what did you decide?", Writes: true,
					Strips: taskStrips(r, s, personID, at)},
				{Key: "agenda", Name: "the agenda", Question: "at 14:30 dentist", Writes: true,
					Strips: agendaStrips(r, s, personID, at)},
			}),
		}
		renderBoard(w, v)
	}
}

// whatMatched is the search, which takes the racks' place rather than opening
// anywhere else: search is the only navigation on this board besides the four
// bays.
//
// Every state, on one screen, which is what the pile has always promised. What
// a result carries is decided by where it is: something still in the pile keeps
// its four answers, and something that already left carries the way back and
// nothing else, because offering the exits to a note that has taken one is
// asking a question that has been answered.
func whatMatched(r *http.Request, s Store, personID int64, find string, at time.Time) []stripView {
	if find == "" {
		return nil
	}
	out := []stripView{}
	items, _, err := s.SearchItems(r.Context(), personID, find, boardDeep)
	if err != nil {
		slog.Error("looking for something", "error", err)
		return nil
	}
	for _, it := range items {
		strip := stripView{ID: it.ID, Words: it.RawText, Mark: markOfDay(it.ReceivedAt, at)}
		switch {
		case it.State != squirrel.ItemOpen:
			strip.What, strip.Mark, strip.Back = "note", string(it.State), true
		case it.Kind == squirrel.ItemTask:
			strip.What, strip.Answers = "task", taskAnswers
		default:
			strip.What, strip.Answers = "note", noteAnswers
		}
		out = append(out, strip)
	}
	chores, err := s.SearchChores(r.Context(), personID, find, boardDeep)
	if err != nil {
		slog.Error("looking for a chore", "error", err)
		return out
	}
	for _, c := range chores {
		out = append(out, stripView{
			ID: c.ID, What: "chore", Words: c.Name,
			Mark: squirrel.Cadence(c.EveryDays), Answers: choreAnswers,
		})
	}
	return out
}

// baysIn lights the rack you are standing in, which is only ever one and is the
// notes when nothing says otherwise. It is a class rather than a filter: the
// desktop draws all four racks and the phone shows the lit one, so the same
// page serves both and neither needs a script.
func baysIn(in string, bays []bayView) []bayView {
	found := false
	for i := range bays {
		if bays[i].Key == in {
			bays[i].In, found = true, true
		}
	}
	if !found {
		bays[0].In = true
	}
	return bays
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
			http.Redirect(w, r, backToTheBay(r), http.StatusSeeOther)
			return
		}
		id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
		if err := answerOnTheBoard(r, s, personID, r.FormValue("what"), r.FormValue("answer"), id); err != nil {
			slog.Error("answering a strip", "error", err)
			fail(w, err)
			return
		}
		http.Redirect(w, r, backToTheBay(r), http.StatusSeeOther)
	}
}

// backToTheBay is the rack the press was made in, so a phone that shows one bay
// at a time does not answer a chore by putting you back in the notes.
func backToTheBay(r *http.Request) string {
	switch bay := r.FormValue("bay"); bay {
	case "notes", "chores", "tasks", "agenda":
		return "/?bay=" + bay
	}
	return "/"
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
			http.Redirect(w, r, backToTheBay(r), http.StatusSeeOther)
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
		http.Redirect(w, r, backToTheBay(r), http.StatusSeeOther)
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
			http.Redirect(w, r, backToTheBay(r), http.StatusSeeOther)
			return
		}
		words := strings.TrimSpace(r.FormValue("words"))
		if words == "" {
			http.Redirect(w, r, backToTheBay(r), http.StatusSeeOther)
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
			http.Redirect(w, r, "/?bay=notes&kept=1", http.StatusSeeOther)
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
			http.Redirect(w, r, "/?bay=notes&kept=1", http.StatusSeeOther)
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
		http.Redirect(w, r, backToTheBay(r), http.StatusSeeOther)
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

// boardNowHandler is the pulled strip's own three answers, and the ladder
// behind the third.
//
// Nothing is stored between being asked what is in the way and answering it:
// the blocker is in the address, so a reload shows the same sentence rather
// than repeating a press. The sentences are the core's, unchanged — a second
// ladder in the web package would be a second product.
func boardNowHandler(s Store, opts Options) http.HandlerFunc {
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
		refID, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)

		switch r.FormValue("act") {
		case "did":
			if err := s.Did(r.Context(), personID, squirrel.Offer{Kind: kind, RefID: refID}, now()); err != nil {
				fail(w, err)
				return
			}
			forgetOffer(opts, personID)
		case "later":
			if err := refuseTheOffer(r, s, opts, personID, kind, refID); err != nil {
				fail(w, err)
				return
			}
		case "stuck":
			why := r.FormValue("why")
			if why == "" {
				http.Redirect(w, r, "/?stuck=1", http.StatusSeeOther)
				return
			}
			b, ok := squirrel.ParseBlocker(why)
			if !ok {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
			// "Not today" is not an obstacle, it is a no, and it is the same no
			// that turning the offer down writes.
			if squirrel.UnstuckFor(b).Refuse {
				if err := refuseTheOffer(r, s, opts, personID, kind, refID); err != nil {
					fail(w, err)
					return
				}
				break
			}
			http.Redirect(w, r, "/?stuck="+url.QueryEscape(why), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func refuseTheOffer(r *http.Request, s Store, opts Options, personID int64, kind squirrel.OfferKind, refID int64) error {
	if !offerKinds[kind] {
		return nil
	}
	if err := s.Refuse(r.Context(), personID, kind, refID, now()); err != nil {
		return err
	}
	forgetOffer(opts, personID)
	return nil
}

// stuckView is what the pulled strip says while you are stuck: the four answers
// when nothing has been pressed, and one sentence when something has.
func stuckView(asked string) (blockers []blockerView, said string) {
	if asked == "" {
		return nil, ""
	}
	if b, ok := squirrel.ParseBlocker(asked); ok {
		return nil, squirrel.UnstuckFor(b).Line
	}
	for _, b := range squirrel.Blockers {
		blockers = append(blockers, blockerView{Why: squirrel.BlockerWords[b], Words: squirrel.BlockerWords[b]})
	}
	return blockers, ""
}

type blockerView struct {
	Why   string
	Words string
}

// boardBuddyHandler is the acorn on the pulled strip: the press that spends a
// call. Opening the board never does — a surface that has to cost nothing to
// open may not spend one — so asking is a press and its answer is drawn on the
// thing it is about.
//
// The press stores nothing. `?ask=1` is what lets the next render pay, and the
// cache behind Decide is what stops a reload paying twice.
func boardBuddyHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := personOf(r); !ok {
			fail(w, errNoOwner)
			return
		}
		http.Redirect(w, r, "/?ask=1", http.StatusSeeOther)
	}
}

// boardBadlyHandler is the one press that says the last thing Buddy said did
// not land. It marks the answer rather than arguing with it, and the marked
// ones are what the next prompt is shown as examples of what does not work.
func boardBadlyHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if _, err := s.LandedBadlyLatest(r.Context(), personID, now()); err != nil {
			slog.Error("marking what did not land", "error", err)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
