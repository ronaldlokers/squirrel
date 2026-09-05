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
	Shelf    string
	Shelved  []stripView
	Opened   *stripView
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
	Faces    []faceView
	Told     []toldView
	Telling  bool
	AnyTold  bool
	You      whom
}

type toldView struct {
	Title string
	Body  string
	Mark  string
}

type bayView struct {
	// More says the rack holds more than it is showing. The words say that
	// there is more and never how much: what is further back is not a thing
	// you can act on, and a number beside it would be a count of what you have
	// not got to.
	More bool
	// Trouble says this rack could not be read. An empty rack and a rack that
	// failed look identical, and one of them is a lie: if the database is down
	// the screen says so rather than showing you a quiet morning.
	Trouble  bool
	Empty    string
	Camera   bool
	In       bool
	Key      string
	Name     string
	Question string
	Writes   bool
	Rhythms  []rhythmView
	Asking   string
	Settled  []settledView
	Strips   []stripView
}

type settledView struct {
	Name   string
	Empty  string
	Strips []stripView
}

type stripView struct {
	// Seen is the line it noticed about this thing, and SeenID is the line
	// itself so it can be refused. Empty is the ordinary case: most strips
	// have nothing worth saying about them, and a line under every one would
	// be a rack nobody reads.
	Seen   string
	SeenID int64
	// Back is a strip that has already left the pile: it carries the way back
	// and nothing else.
	Back    bool
	Resting bool
	State   string
	// Photo says this note has a photograph. The strip says so; opening it is
	// what shows it.
	Photo bool
	// Rhythms is the four intervals, on the one note that was asked how often
	// it comes back.
	Rhythms []rhythmView
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
		asking, _ := strconv.ParseInt(r.URL.Query().Get("chore"), 10, 64)
		find := strings.TrimSpace(r.URL.Query().Get("find"))
		shelf := r.URL.Query().Get("shelf")
		shelved := whatIsOnTheShelf(r, s, personID, shelf, at)
		blockers, unstuck := stuckView(r.URL.Query().Get("stuck"))
		v := boardView{
			In:       in,
			Find:     find,
			Shelf:    shelfNames[shelf],
			Shelved:  shelved,
			Opened:   openedStrip(r, s, personID, at),
			Found:    whatMatched(r, s, personID, find, at),
			Blockers: blockers,
			Unstuck:  unstuck,
			Kept:     r.URL.Query().Get("kept") == "1",
			Now:      at.Format("15:04"),
			Day:      at.Format("Monday 2 January"),
			// It notices without being asked. The press that spent this call
			// went on 3 September 2026: a line that only appears when you ask
			// for it is a thing you have to think of, and this product exists
			// to remove the thinking-of. The cache keys on the thing that was
			// picked, so the cost is one call per thing offered rather than
			// one per render, and the budget is what says no.
			Pulled:  offerFor(s, r, false),
			Timer:   runningTimer(s, opts, r),
			Tray:    trayStrips(r, s, opts, personID, at),
			Faces:   facesIfItIsTime(r, s, personID, at),
			Bays:    baysIn(in, oneBayOnly(r, theBaysOf(r, s, opts, personID, at, asking))),
			Telling: r.URL.Query().Get("told") == "1",
			You:     youFor(r.Context(), s, personID),
		}
		// One row is enough to mark the bell, and the whole list is only read
		// when you are looking at it: this runs on every board render.
		deep := 1
		if v.Telling {
			deep = boardDeep
		}
		v.Told = whatWasSaid(r, s, personID, at, deep)
		v.AnyTold = len(v.Told) > 0
		renderBoard(w, v)
	}
}

// shelfNames is what each shelf is called, and the map is the guard: a shelf
// nobody named is not a shelf, so an address with anything else in it draws the
// board.
var shelfNames = map[string]string{
	"kept": "the things you kept",
	"held": "what you set aside",
}

// whatIsOnTheShelf is a shelf, opened. It takes the racks' place the way search
// does — the board has one place where things are — and what it holds carries
// the way back and nothing else, because a thing on a shelf has already been
// decided about.
func whatIsOnTheShelf(r *http.Request, s Store, personID int64, shelf string, at time.Time) []stripView {
	switch shelf {
	case "kept":
		items, _, err := s.KeptItems(r.Context(), personID, boardDeep)
		if err != nil {
			slog.Error("reading what you kept", "error", err)
			return nil
		}
		out := make([]stripView, 0, len(items))
		for _, it := range items {
			out = append(out, stripView{
				ID: it.ID, What: "note", Words: it.RawText,
				State: markOfDay(it.ReceivedAt, at), Back: true, Photo: it.PhotoName != "",
			})
		}
		return out
	case "held":
		items, _, err := s.HeldItems(r.Context(), personID, boardDeep)
		if err != nil {
			slog.Error("reading what you set aside", "error", err)
			return nil
		}
		out := make([]stripView, 0, len(items))
		for _, it := range items {
			// The mark is what would move it, in his own words. A shelf that
			// counted them would be a reproach; a shelf that says when each
			// one ends is the reason there are three of these rather than one.
			out = append(out, stripView{
				ID: it.ID, What: "note", Words: it.Text,
				State: it.Words(), Back: true, Photo: it.PhotoName != "",
			})
		}
		return out
	}
	return nil
}

// whatIsSettled is what has already been decided about, under the notes that
// have not. Both shelves in the rack rather than behind a tab at the foot of
// it: a place you have to travel to is a place that stops being read, and what
// you set aside is exactly the thing that must not disappear.
func whatIsSettled(r *http.Request, s Store, personID int64, at time.Time) []settledView {
	out := make([]settledView, 0, 2)
	for _, shelf := range []struct{ key, empty string }{
		{"held", "nothing set aside"},
		{"kept", "nothing kept yet"},
	} {
		out = append(out, settledView{
			Name:   shelfNames[shelf.key],
			Empty:  shelf.empty,
			Strips: resting(whatIsOnTheShelf(r, s, personID, shelf.key, at)),
		})
	}
	return out
}

func resting(strips []stripView) []stripView {
	for i := range strips {
		strips[i].Resting = true
	}
	return strips
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

// askedForARhythm marks the one note that was asked how often it comes back.
// It is one strip, never all of them: a rack where every row asks a question is
// a rack you have to answer to read.
func askedForARhythm(strips []stripView, asking int64) []stripView {
	if asking == 0 {
		return strips
	}
	for i := range strips {
		if strips[i].ID == asking {
			strips[i].Rhythms = theRhythms
		}
	}
	return strips
}

// theBaysOf reads all four racks, each saying whether it could be read at all.
func theBaysOf(r *http.Request, s Store, opts Options, personID int64, at time.Time, asking int64) []bayView {
	rhythmFor := strings.TrimSpace(r.URL.Query().Get("rhythm"))
	whenFor := strings.TrimSpace(r.URL.Query().Get("when"))
	seen := whatWasNoticed(r, s, personID)
	notes, notesOK, moreNotes := noteStrips(r, s, personID, at)
	notes = marked(notes, "note", seen)
	settled := whatIsSettled(r, s, personID, at)
	chores, choresOK := choreStrips(r, s, personID)
	tasks, tasksOK, moreTasks := taskStrips(r, s, personID, at)
	agenda, agendaOK := agendaStrips(r, s, personID, at)
	return []bayView{
		{Key: "notes", Name: "the notes", Question: "what is it", Writes: true,
			Camera: opts.Photos != nil, Trouble: !notesOK, More: moreNotes,
			Empty: "nothing in the notes", Strips: askedForARhythm(notes, asking),
			Settled: settled},
		{Key: "chores", Name: "the chores", Question: "what comes back?", Writes: true,
			Rhythms: theRhythms, Trouble: !choresOK, Asking: rhythmFor,
			Empty: "nothing comes back today", Strips: chores},
		{Key: "tasks", Name: "the tasks", Question: "what did you decide?", Writes: true,
			Trouble: !tasksOK, More: moreTasks,
			Empty: "nothing in the tasks", Strips: tasks},
		{Key: "agenda", Name: "the agenda", Question: "at 14:30 dentist", Writes: true,
			Trouble: !agendaOK, Asking: whenFor,
			Empty: "nothing left today", Strips: agenda},
	}
}

func oneBayOnly(r *http.Request, bays []bayView) []bayView {
	if devDir == "" {
		return bays
	}
	only := strings.TrimSpace(r.URL.Query().Get("only"))
	if only == "" {
		return bays
	}
	for _, bay := range bays {
		if bay.Key == only {
			return []bayView{bay}
		}
	}
	return bays
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

func noteStrips(r *http.Request, s Store, personID int64, at time.Time) ([]stripView, bool, bool) {
	items, more, err := s.OpenItems(r.Context(), personID, boardDeep)
	if err != nil {
		slog.Error("reading the notes for the board", "error", err)
		return nil, false, false
	}
	out := make([]stripView, 0, len(items))
	for _, it := range items {
		out = append(out, stripView{
			ID: it.ID, What: "note", Words: it.RawText,
			Mark: markOfDay(it.ReceivedAt, at), Answers: noteAnswers,
			Photo: it.PhotoName != "",
		})
	}
	return out, true, more
}

func taskStrips(r *http.Request, s Store, personID int64, at time.Time) ([]stripView, bool, bool) {
	items, more, err := s.Tasks(r.Context(), personID, boardDeep)
	if err != nil {
		slog.Error("reading the tasks for the board", "error", err)
		return nil, false, false
	}
	out := make([]stripView, 0, len(items))
	for _, it := range items {
		out = append(out, stripView{
			ID: it.ID, What: "task", Words: it.RawText,
			Mark: markOfDay(it.ReceivedAt, at), Answers: taskAnswers,
			Photo: it.PhotoName != "",
		})
	}
	return out, true, more
}

func choreStrips(r *http.Request, s Store, personID int64) ([]stripView, bool) {
	chores, err := s.ActiveChores(r.Context(), personID)
	if err != nil {
		slog.Error("reading the chores for the board", "error", err)
		return nil, false
	}
	out := make([]stripView, 0, len(chores))
	for _, c := range chores {
		out = append(out, stripView{
			ID: c.ID, What: "chore", Words: c.Name,
			Mark: squirrel.Cadence(c.EveryDays), Answers: choreAnswers,
		})
	}
	return out, true
}

func agendaStrips(r *http.Request, s Store, personID int64, at time.Time) ([]stripView, bool) {
	soon, err := s.Upcoming(r.Context(), personID, at, boardDeep)
	if err != nil {
		slog.Error("reading what is coming for the board", "error", err)
		return nil, false
	}
	out := make([]stripView, 0, len(soon))
	for _, m := range soon {
		strip := stripView{
			ID: m.ID, What: "moment", Words: m.Label,
			Mark: markOfMoment(m, at), Big: true, Answers: momentAnswers,
		}
		// Inside the window the strip stops being a time and becomes a thing
		// you are about to leave for: the mark says when to go, and LEAVING is
		// the answer. Outside it there is nothing to press — a button that
		// closes a thing three hours early is one that gets pressed by
		// accident.
		if m.Open(at) {
			strip.Mark = "leave " + m.LeaveAt().Format("15:04")
			strip.Answers = leavingAnswers
		}
		out = append(out, strip)
	}
	return out, true
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

var leavingAnswers = []answerView{
	{Act: "leaving", Words: "leaving", Key: "D", Look: "did"},
	{Act: "over", Words: "it is over", Key: "X", Look: "no"},
}

var momentAnswers = []answerView{
	{Act: "over", Words: "it is over", Key: "D", Look: "did"},
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
	template.New("board.html").Funcs(helpers).ParseFS(templatesFS(), "templates/board.html", "templates/chips.html", "templates/strip.html"))

func renderBoard(w http.ResponseWriter, v boardView) {
	v.V = stamp()
	v.Light = squirrel.Light(now())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := boardPage
	if devDir != "" {
		reparsed, err := template.New("board.html").Funcs(helpers).ParseFS(templatesFS(), "templates/board.html", "templates/chips.html", "templates/strip.html")
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
	case "moment":
		switch answer {
		case "over":
			return s.MomentDone(r.Context(), personID, id, at)
		case "leaving":
			// The same write the conversation's LEAVING makes: you left, which
			// is what closes a fixed point and stops it being raised again.
			return s.Did(r.Context(), personID, squirrel.Offer{Kind: squirrel.OfferMoment, RefID: id}, at)
		}
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
			days := everyInDays(r.FormValue("every"), r.FormValue("unit"))
			if days <= 0 {
				// Asked for, never guessed at. Filing this as a note was the
				// old behaviour and it was the wrong kind of helpful: you typed
				// a chore, and what you got was a note in another rack, found
				// on the next refresh.
				http.Redirect(w, r, "/?bay=chores&rhythm="+url.QueryEscape(words), http.StatusSeeOther)
				return
			}
			every := time.Duration(days) * 24 * time.Hour
			if _, err := s.UpsertChore(r.Context(), personID, words, every, every/10); err != nil {
				fail(w, err)
				return
			}
		case "agenda":
			m, ok := momentFromPickers(opts.Location, words, r.FormValue("day"), clockFrom(r))
			if !ok {
				m, ok = squirrel.ParseMomentIn(opts.Location, words, now())
			}
			if !ok {
				// The same rule: an appointment with no time in it is a
				// question, not a note.
				http.Redirect(w, r, "/?bay=agenda&when="+url.QueryEscape(words), http.StatusSeeOther)
				return
			}
			if _, err := s.CreateMoment(r.Context(), personID, m); err != nil {
				fail(w, err)
				return
			}
		case "notes":
			if err := keepAsANote(r, s, personID, words); err != nil {
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
func keepAsANote(r *http.Request, s Store, personID int64, words string) error {
	return keptOnTheBoard(r, s, personID, words, "", "")
}

// keptOnTheBoard is the whole of capture from the screen: one row, written
// before the redirect, so the board you are sent back to has it on it.
//
// It went through the spool and a drain until 4 September 2026. The spool is
// still what Campfire's captures land in, because that path has no person in
// front of it and nothing to tell when a write fails. This one has both: it
// answers on the screen, and a row that did not land says so instead of being
// promised.
func keptOnTheBoard(r *http.Request, s Store, personID int64, words, photo, kind string) error {
	sender := subOf(r)
	_, err := s.InsertItem(r.Context(), squirrel.Item{
		Transport:  squirrel.ScreenTransport,
		SenderID:   &sender,
		PersonID:   &personID,
		RawText:    words,
		Payload:    []byte(squirrel.ScreenCapture),
		ReceivedAt: now(),
		PhotoName:  photo,
		PhotoType:  kind,
	})
	if err != nil {
		slog.Warn("a capture from the board could not be kept", "error", err)
	}
	return err
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
		case "later":
			if err := refuseTheOffer(r, s, personID, kind, refID); err != nil {
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
				if err := refuseTheOffer(r, s, personID, kind, refID); err != nil {
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

func refuseTheOffer(r *http.Request, s Store, personID int64, kind squirrel.OfferKind, refID int64) error {
	if !offerKinds[kind] {
		return nil
	}
	return s.Refuse(r.Context(), personID, kind, refID, now())
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

// boardCaptureHandler is the notes rack's own slot when it carries a
// photograph. It is the same path the conversation's capture takes — readCapture
// writes the bytes to the volume and fsyncs them before the row that points at
// them exists — because a second way into the pile would be a second way to
// lose a thought.
func boardCaptureHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		text, photo, kind, err := readCapture(r, opts)
		if err != nil {
			slog.Warn("a capture from the board was refused", "error", err)
			fail(w, err)
			return
		}
		// Nothing said and nothing photographed is nothing to keep. A
		// photograph on its own is a capture, which is most of the point of
		// having a camera.
		if text == "" && photo == "" {
			http.Redirect(w, r, "/?bay=notes", http.StatusSeeOther)
			return
		}
		if err := keptOnTheBoard(r, s, personID, text, photo, kind); err != nil {
			fail(w, err)
			return
		}
		http.Redirect(w, r, "/?bay=notes", http.StatusSeeOther)
	}
}

// opened is the one strip you asked to see, drawn whole: its photograph at the
// size a photograph needs, and the answers it would carry in its rack.
//
// A strip in a rack never carries the picture — it says it has one and this is
// what opening it does. Reading it back as yours is the same guard every other
// press has: a row that is not yours is not yours to look at either.
func openedStrip(r *http.Request, s Store, personID int64, at time.Time) *stripView {
	id, _ := strconv.ParseInt(r.URL.Query().Get("open"), 10, 64)
	if id == 0 {
		return nil
	}
	it, mine, err := s.ItemByID(r.Context(), personID, id)
	if err != nil {
		slog.Error("opening a strip", "error", err)
		return nil
	}
	if !mine {
		return nil
	}
	v := &stripView{
		ID: it.ID, What: "note", Words: it.RawText,
		Mark: markOfDay(it.ReceivedAt, at), Photo: it.PhotoName != "",
	}
	switch {
	case it.State != squirrel.ItemOpen:
		v.Back = true
		v.Mark = string(it.State)
	case it.Kind == squirrel.ItemTask:
		v.What, v.Answers = "task", taskAnswers
	default:
		v.Answers = noteAnswers
	}
	return v
}

// boardChoreHandler makes a chore out of a note that already exists. The note is
// the thing that was kept, so the rhythm can be asked for on its own strip
// without a thought sitting in a form waiting for the answer.
func boardChoreHandler(s Store, opts Options) http.HandlerFunc {
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
		id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
		days, _ := strconv.Atoi(r.FormValue("every"))
		if id == 0 || days <= 0 {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if _, mine, err := s.ItemByID(r.Context(), personID, id); err != nil || !mine {
			if err != nil {
				fail(w, err)
				return
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if _, _, err := s.PromoteItem(r.Context(), personID, id, time.Duration(days)*24*time.Hour); err != nil {
			fail(w, err)
			return
		}
		http.Redirect(w, r, "/?bay=chores", http.StatusSeeOther)
	}
}

// whatWasSaid is the record behind the bell: every push this person was sent,
// newest first, as it was sent.
func whatWasSaid(r *http.Request, s Store, personID int64, at time.Time, deep int) []toldView {
	said, err := s.WhatWasSaid(r.Context(), personID, deep)
	if err != nil {
		slog.Error("reading what was said", "error", err)
		return nil
	}
	out := make([]toldView, 0, len(said))
	for _, one := range said {
		out = append(out, toldView{Title: one.Title, Body: one.Body, Mark: markOfDay(one.At, at)})
	}
	return out
}

// everyInDays reads the rhythm off the blank strip: a count and what it counts.
// A bare number of days is still read, because the four shortcut chips send
// one.
func everyInDays(every, unit string) int {
	n, err := strconv.Atoi(strings.TrimSpace(every))
	if err != nil || n <= 0 {
		return 0
	}
	switch unit {
	case "weeks":
		return n * 7
	case "months":
		return n * 30
	}
	return n
}

// clockFrom is the time the pickers were set to. The hour and the minute are
// two fields rather than one, because a native time input renders in whatever
// the browser's locale says and no attribute can make it say 24 hours.
func clockFrom(r *http.Request) string {
	if clock := strings.TrimSpace(r.FormValue("time")); clock != "" {
		return clock
	}
	hour, minute := strings.TrimSpace(r.FormValue("hour")), strings.TrimSpace(r.FormValue("minute"))
	if hour == "" || minute == "" {
		return ""
	}
	if len(hour) == 1 {
		hour = "0" + hour
	}
	if len(minute) == 1 {
		minute = "0" + minute
	}
	return hour + ":" + minute
}

// momentFromPickers builds one out of the day and time beside the field, which
// is what the pickers are for: a sentence with a time in it is quicker when you
// have one, and unusable when you do not.
func momentFromPickers(loc *time.Location, words, day, clock string) (squirrel.Moment, bool) {
	if day == "" || clock == "" || words == "" {
		return squirrel.Moment{}, false
	}
	if loc == nil {
		loc = time.Local
	}
	starts, err := time.ParseInLocation("2006-01-02 15:04", day+" "+clock, loc)
	if err != nil {
		return squirrel.Moment{}, false
	}
	return squirrel.Moment{Label: words, Starts: starts, Guessed: true}, true
}

// facesIfItIsTime is the check-in on the board: the five faces at the tray's
// right end, and nothing at all while the last answer still describes now.
//
// A reading rather than a question, which is why it is drawn at the edge and
// never written into the record here — the record is the readings themselves.
func facesIfItIsTime(r *http.Request, s Store, personID int64, at time.Time) []faceView {
	c, found, err := s.LatestCheckin(r.Context(), personID)
	if err != nil {
		slog.Error("reading how you are", "error", err)
		return nil
	}
	if found && c.JustAsked(at) {
		return nil
	}
	return theFaces()
}

// boardMoodHandler keeps a reading and puts you back on the board.
//
// Nothing is said back. The conversation answers a check-in with a turn because
// a conversation is a record of what was said; the board is a record of what
// there is, and a reading is neither a strip nor something to answer.
func boardMoodHandler(s Store) http.HandlerFunc {
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
		m, ok := squirrel.ParseMood(r.FormValue("mood"))
		if !ok {
			// Not one of the five, so nothing is kept: this arrives from a
			// form, and a stranger's typing is read as no answer rather than
			// as a wrong one.
			http.Redirect(w, r, backToTheBay(r), http.StatusSeeOther)
			return
		}
		if err := s.RecordCheckin(r.Context(), personID, m, "screen", now()); err != nil {
			fail(w, err)
			return
		}
		http.Redirect(w, r, backToTheBay(r), http.StatusSeeOther)
	}
}

// whatWasNoticed is every line not refused, keyed by the thing it is about.
//
// One read for the whole board rather than one per strip: the lines are few by
// construction, and a query per row would be the thing that makes a rack slow.
func whatWasNoticed(r *http.Request, s Store, personID int64) map[string]squirrel.Noticed {
	lines, err := s.WhatWasNoticed(r.Context(), personID)
	if err != nil {
		// A rack without marginalia is the rack this product had for its whole
		// life. Nothing is said about the failure, because nothing was
		// promised.
		slog.Error("reading what was noticed", "error", err)
		return nil
	}
	out := make(map[string]squirrel.Noticed, len(lines))
	for _, one := range lines {
		out[one.Kind+":"+strconv.FormatInt(one.RefID, 10)] = one
	}
	return out
}

// marked hangs each line on the strip it names.
func marked(strips []stripView, kind string, seen map[string]squirrel.Noticed) []stripView {
	if len(seen) == 0 {
		return strips
	}
	for i := range strips {
		if one, ok := seen[kind+":"+strconv.FormatInt(strips[i].ID, 10)]; ok {
			strips[i].Seen, strips[i].SeenID = one.Words, one.ID
		}
	}
	return strips
}

// boardNotUsefulHandler is how a line is refused.
//
// It does not hide the line so much as answer it: the words stay, and the next
// pass is shown them as something not to write again. A refusal that only
// cleared the screen would leave the same line to be written tomorrow.
func boardNotUsefulHandler(s Store) http.HandlerFunc {
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
		if id <= 0 {
			http.Redirect(w, r, backToTheBay(r), http.StatusSeeOther)
			return
		}
		if _, err := s.NotUseful(r.Context(), personID, id, now()); err != nil {
			fail(w, err)
			return
		}
		http.Redirect(w, r, backToTheBay(r), http.StatusSeeOther)
	}
}
