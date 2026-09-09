package web

import (
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

type boardView struct {
	V        string
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
	Moodful  bool
	Racks    []rackView
	Rhythm   string
	When     string
	// Adding is what you had typed in the bar when you pressed the plus. The
	// bar keeps a thought on its own; the plus is how you say it is something
	// else, and the words have to travel with the press or you type them
	// twice.
	Adding string
	// Notes is how many are on the wall, on the chip that leads there.
	Notes int
	// Rhythms is what the writer offers when a thing comes back: presets, in
	// the product's own words, because a number box and a unit dropdown is
	// two decisions where the answer is nearly always one of four.
	Rhythms []rhythmView
	Dial    *dialView
	Coming  *comingView
	Rail    *railView
	// Loose is a row for the things the picker offers that have no row of their
	// own: the breadcrumb, and a timer whose thing is not on the board. They
	// used to share the offer's card; with the card gone they get a row, at
	// the head, because a thing you cannot reach is worse than a thing in a
	// place of its own.
	Loose   []stripView
	Told    []toldView
	Telling bool
	AnyTold bool
	You     whom
}

type toldView struct {
	Title string
	Body  string
	Mark  string
}

// rackView is one of the three racks the chores are cut into. It is not a
// bayView: a rack holds one kind of thing, is never written into, and carries
// a count of what it is deliberately not showing.
type rackView struct {
	Key     string
	Name    string
	In      bool
	Trouble bool
	Empty   string
	// Resting is what a quiet day is holding back. Never a number of things
	// you are behind on — it is the size of the part of your life the board
	// has decided not to put in front of you today.
	Resting int
	// Wants is what is asking for you today; Rest is the rest of the rack.
	// Two lists rather than one list and a flag, because the seam between
	// them is a thing the template draws, and a flag would make it guess
	// where.
	Wants  []stripView
	Rest   []stripView
	Strips []stripView
	More   bool
}

// comingView is what is ahead of you, in the sidebar. Everything still to
// come, not only today: this took the agenda door's place, and the door held
// the same.
// Hoisted is the one inside its leave-by window, lifted above the dial. It is
// the only thing that reorders that column, and it does so on the same rule
// the picker's first rule already uses — so nothing new decides when the
// world is calling.
type comingView struct {
	Hoisted *apptView
	Appts   []apptView
	Count   int
	More    bool
	Trouble bool
}

// apptView is one fixed point. The time is a deadline and is set as one; the
// leave-by is arithmetic about a distance and stays quiet, and admits when the
// distance was never given.
type apptView struct {
	ID      int64
	Time    string
	Day     string
	Label   string
	LeaveBy string
	Repeats string
	Big     bool
	Late    bool
}

// dialView is how you have been, on the board rather than only on the page
// about you: today's face, and the seven days behind it.
type dialView struct {
	Mood string
	Word string
	Said bool
	Days []moodCellView
	Ring []ringSegView
	// Faces is the five, always. Asking is whether Squirrel wants an answer,
	// which is a separate thing from whether you may give one — the faces used
	// to appear only when it did, so saying how you were was something you had
	// to wait to be asked for.
	Faces  []faceView
	Asking bool
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
	Refused  bool
	Offline  bool
	Strips   []stripView
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
	// Photo says this note has a photograph. The strip says so; opening it is
	// what shows it.
	Photo bool
	// Rhythms is the four intervals, on the one note that was asked how often
	// it comes back.
	Rhythms []rhythmView
	// Why is where this row is in its rack and what put it there, said on
	// the rows that are asking and nowhere else.
	Why string
	// Wants says this row is asking for you today. What a resting row stops
	// carrying is its reason and its usual time — both are still true, and
	// both are one press away on the row itself.
	Wants bool
	// Because is the offer's own sentence, on the row the offer is about. The
	// card it used to sit on went on 9 September 2026: the offer was always a
	// thing that already had a row, and drawing it twice was the duplication
	// the picker itself was cured of.
	Because string
	// Chosen says the picker chose this row. Separate from Because, which is
	// the sentence: an offer with nothing to say for itself is still an offer,
	// and gating the ladder on the sentence is how it disappeared once.
	Chosen bool
	// Blockers and Unstuck are the ladder, on the row the offer is about. On
	// the row rather than reached through the board, because the strip
	// template is its own scope and a rung that has to be threaded through
	// every call site is a rung that gets dropped at one.
	Blockers []blockerView
	Unstuck  string
	// Offerable says this is a thing the picker could have offered, so the
	// opened strip may carry the two answers that used to be on its card.
	Offerable bool
	// When is the hour a rail row hangs at, or the part of the day when that
	// is all there is to say. Empty everywhere but the phone's rail.
	When string
	// because a colour is a thing you have to already know how to read.
	Due      bool
	Late     bool
	ID       int64
	What     string
	Words    string
	Mark     string
	Big      bool
	Answers  []answerView
	Room     string
	Held     []answerView
	Reword   bool
	Answered bool
	Before   []string
	Step     *stepView
}

type rhythmView struct {
	Days  int
	Words string
}

const noticedKept = 6

// writerRhythms is what the writer offers, which is the four the notes' own
// promotion offers plus the weekday the bins live on.
var writerRhythms = []rhythmView{
	{Days: 1, Words: "every day"},
	{Days: 7, Words: "every week"},
	{Days: 14, Words: "2 weeks"},
	{Days: 30, Words: "a month"},
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
		at := now().In(zoneOf(r.Context()))
		in := r.URL.Query().Get("bay")
		if in == "notes" || r.URL.Query().Has("shelf") {
			http.Redirect(w, r, "/notes", http.StatusMovedPermanently)
			return
		}
		find := strings.TrimSpace(r.URL.Query().Get("find"))
		blockers, unstuck := stuckView(r.URL.Query().Get("stuck"))
		v := boardView{
			In:       in,
			Find:     find,
			Blockers: blockers,
			Unstuck:  unstuck,
			Kept:     r.URL.Query().Get("kept") == "1",
			Now:      at.Format("15:04"),
			Day:      at.Format("Monday 2 January"),
			Telling:  r.URL.Query().Get("told") == "1",
			Adding:   strings.TrimSpace(r.URL.Query().Get("words")),
			Rhythms:  writerRhythms,
			Rhythm:   strings.TrimSpace(r.URL.Query().Get("rhythm")),
			When:     strings.TrimSpace(r.URL.Query().Get("when")),
		}
		// The same gate the picker reads, and CapacityLow is wiped or
		// frazzled rather than the mood called low — capacity.go says why.
		// On such a day the racks show what comes back today and count the
		// rest.
		quiet := s.Capacity(r.Context(), personID, at) == squirrel.CapacityLow
		deep := 1
		if v.Telling {
			deep = boardDeep
		}

		var g errgroup.Group
		g.SetLimit(squirrel.MaxConns)
		g.Go(func() error { v.Opened = openedStrip(r, s, personID, at); return nil })
		g.Go(func() error { v.Found = whatMatched(r, s, personID, find, at); return nil })
		g.Go(func() error { v.Pulled = offerFor(s, r, false); return nil })
		g.Go(func() error { v.Tray = trayStrips(r, s, opts, personID, at); return nil })
		g.Go(func() error { v.Dial = howYouAre(r, s, personID, at); return nil })
		g.Go(func() error { v.Coming = whatIsComing(r, s, personID, at); return nil })
		var notesCount int
		g.Go(func() error { notesCount, _ = howManyFolded(r, s, personID); return nil })
		var (
			racksOK bool
			usually map[int64]squirrel.Usually
		)
		g.Go(func() error { v.Racks, usually, racksOK = choreRacks(r, s, personID, at, quiet); return nil })
		bays := fetchBays(&g, r, s, personID, at)
		g.Go(func() error { v.You = youFor(r.Context(), s, personID); return nil })
		g.Go(func() error { v.Told = whatWasSaid(r, s, personID, at, deep); return nil })
		_ = g.Wait()

		v.Racks = marginalia(r, opts, bays.seen, troubled(v.Racks, racksOK))
		v.Racks = onlyOne(r, append(v.Racks, bays.once(r, opts)))
		v.Moodful = in == "mood" && v.Dial != nil
		v.Racks = standingIn(in, v.Racks)
		v.Notes = notesCount
		v.Rail = railFor(v.Racks, v.Coming, usually, notesCount)
		theOffer(&v)
		v.AnyTold = len(v.Told) > 0
		renderBoard(w, v)
	}
}

// whatMatched is the search, which takes the racks' place rather than opening
// anywhere else: search is the only navigation on this board besides the four
// bays.
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

type bayFetch struct {
	seen               map[string]squirrel.Noticed
	tasks              []stripView
	tasksOK, moreTasks bool
}

func fetchBays(g *errgroup.Group, r *http.Request, s Store, personID int64, at time.Time) *bayFetch {
	f := &bayFetch{}
	g.Go(func() error { f.seen = whatWasNoticed(r, s, personID); return nil })
	g.Go(func() error { f.tasks, f.tasksOK, f.moreTasks = taskStrips(r, s, personID, at); return nil })
	return f
}

func (f *bayFetch) once(r *http.Request, opts Options) rackView {
	justAsked, _ := strconv.ParseInt(r.URL.Query().Get("answered"), 10, 64)
	tasks := askable(f.tasks, "tasks", coachAvailable(opts))
	tasks = answered(marked(tasks, "ask:task", f.seen), justAsked)
	return onceRack(tasks, f.tasksOK, f.moreTasks)
}

func onceRack(tasks []stripView, ok, more bool) rackView {
	rack := rackView{
		Key: "once", Name: "once", Trouble: !ok, More: more,
		Empty:  "nothing to do just the one time",
		Strips: tasks,
	}
	for i := range rack.Strips {
		// Once, not the hour it was captured at. When a thing comes back is
		// what a mark on this board says, and for these the answer is never.
		rack.Strips[i].Mark = "once"
		if i == 0 {
			rack.Strips[i].Wants = true
			rack.Strips[i].Why = "the last thing you decided"
		}
		if rack.Strips[i].Wants {
			rack.Wants = append(rack.Wants, rack.Strips[i])
		} else {
			rack.Rest = append(rack.Rest, rack.Strips[i])
		}
	}
	return rack
}

// onlyOne is the development board answering ?only= with a single rack, so an
// element can be picked without four others on the screen. It is inert in a
// shipped binary, which is what the first test in onebay_test.go is for.
func onlyOne(r *http.Request, racks []rackView) []rackView {
	if devDir == "" {
		return racks
	}
	only := strings.TrimSpace(r.URL.Query().Get("only"))
	if only == "" {
		return racks
	}
	for _, rack := range racks {
		if rack.Key == only {
			return []rackView{rack}
		}
	}
	return racks
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

// choreRacks is the whole of the chores, cut into three by how often they come
// back and ordered inside each by the rules in rhythm.go.
// One read of the chores and one of when you usually do them, for all three
// racks: a query per rack would be three times the work for the same rows, and
// the racks are a cut of one list rather than three lists.
func choreRacks(r *http.Request, s Store, personID int64, at time.Time, quiet bool) ([]rackView, map[int64]squirrel.Usually, bool) {
	chores, err := s.ActiveChores(r.Context(), personID)
	if err != nil {
		slog.Error("reading the chores for the board", "error", err)
		return nil, nil, false
	}
	usually, err := s.WhenYouUsuallyDo(r.Context(), personID)
	if err != nil {
		// The racks are still the racks without it. What is lost is the line
		// under each row saying when you tend to do the thing, and a rack that
		// cannot say that is worth more than no rack at all.
		slog.Error("reading when you usually do things", "error", err)
		usually = nil
	}

	racks := squirrel.RacksOf(chores, usually, at, quiet)

	// The phone's first tab, built here rather than in the template: it is the
	// same rows under the same rules, cut differently, and a cut the screen
	// invented would be a fourth place the order could go wrong.
	now := choreRows(squirrel.Now(racks), at)
	out := []rackView{{
		Key: "now", Name: "now",
		Empty: "nothing comes back today",
		Wants: now, Strips: now,
	}}
	for _, rack := range racks {
		out = append(out, rackView{
			Key: string(rack.Rhythm), Name: string(rack.Rhythm),
			Empty: emptyRack[rack.Rhythm], Resting: rack.Resting,
			Strips: choreRows(rack.Waiting, at),
		})
	}
	return out, usually, true
}

// emptyRack is what each rack says when it is holding nothing. Three sentences
// rather than one, because "nothing here" three times down a screen reads as a
// fault, and what is true of each is different: a rack with no daily chores in
// it is a fact about your life rather than a gap in the day.
var emptyRack = map[squirrel.Rhythm]string{
	squirrel.Daily:  "nothing every day",
	squirrel.Weekly: "nothing this often",
	squirrel.Seldom: "nothing that comes back slowly",
}

func choreRows(standing []squirrel.Standing, at time.Time) []stripView {
	out := make([]stripView, 0, len(standing))
	for _, one := range standing {
		row := stripView{
			ID: one.Chore.ID, What: "chore", Words: one.Chore.Name,
			Mark: squirrel.Cadence(one.Chore.EveryDays), Answers: choreAnswers,
			Why: one.Because, Wants: one.WantsYouToday(),
			Due:  one.Chore.EverDone && one.Chore.SinceDays >= one.Chore.EveryDays,
			Late: one.Chore.LateToday(at),
		}
		// No second line. Every rank that asks for you already carries a
		// reason, and that reason names the usual time whenever the usual
		// time is why the row is where it is. A separate line for it was
		// saying the same thing twice under one name.
		out = append(out, row)
	}
	return out
}

// doorOpened is a door standing open, which takes the racks' place the way
// search and the shelves do: the board has one place where things are, and
// what is in a door is a list you came to read rather than something the
// board is holding in front of you.
func doorOpened(in string, bays []bayView) *bayView {
	for i := range bays {
		if bays[i].Key == in {
			return &bays[i]
		}
	}
	return nil
}

// halved splits a rack in two. The order inside each half is the order
// rhythm.go gave it, which is the whole reason that file exists.
func halved(rows []stripView, want bool) []stripView {
	out := []stripView{}
	for _, row := range rows {
		if row.Wants == want {
			out = append(out, row)
		}
	}
	return out
}

// standingIn lights the rack the phone is standing in. Same shape as baysIn
// and for the same reason: one page, drawn whole on the desk and one rack at a
// time on the phone, and no script deciding which.
// Nothing lit is the ordinary case and not a fallback: the phone's first tab
// is "now", which is a cut across all three rather than any one of them.
func standingIn(in string, racks []rackView) []rackView {
	lit := false
	for i := range racks {
		racks[i].In = racks[i].Key == in
		lit = lit || racks[i].In
	}
	// Nothing named, so the phone stands in "now". A tab bar with none of its
	// tabs lit is a screen showing nothing at all at that width.
	if !lit && len(racks) > 0 {
		racks[0].In = true
	}
	return racks
}

var noteAnswers = []answerView{
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

var heldAnswers = []answerView{
	{Act: "waiting", Words: squirrel.HeldWords[squirrel.ItemWaiting], Key: "W"},
	{Act: "blocked", Words: squirrel.HeldWords[squirrel.ItemBlocked], Key: "B"},
	{Act: "someday", Words: squirrel.HeldWords[squirrel.ItemSomeday], Key: "Y"},
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

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

const choreNameLimit = 200

func theFaces() []faceView {
	out := make([]faceView, 0, len(squirrel.Moods))
	for _, m := range squirrel.Moods {
		out = append(out, faceView{Mood: string(m), Word: squirrel.Words[m]})
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

// backToTheStrip is the strip the ladder was climbed on. The ladder lived on
// the offer's own card until 9 September 2026 and had one address; it is on the
// opened strip now, so it has to come back to the strip it was pressed from.
func backToTheStrip(r *http.Request) string {
	switch from := strings.TrimSpace(r.FormValue("from")); {
	case strings.HasPrefix(from, "/?open="):
		return from
	default:
		return "/?pulled=1"
	}
}

// backToTheBay is the rack the press was made in, so a phone that shows one bay
// at a time does not answer a chore by putting you back in the notes.
func backToTheBay(r *http.Request) string {
	switch bay := r.FormValue("bay"); bay {
	case "notes":
		return "/notes"
	case "tasks", "now", "daily", "weekly", "seldom", "once", "mood":
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
	"done":    squirrel.ItemDone,
	"drop":    squirrel.ItemDropped,
	"waiting": squirrel.ItemWaiting,
	"blocked": squirrel.ItemBlocked,
	"someday": squirrel.ItemSomeday,
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
// The notes bay writes to the spool rather than to the database, because that
// is what capture is: the words reach fsynced disk before anything answers, and
// the drain resolves whose they are. The tasks bay does not — a task is a
// decision you already made about something, and a decision has no spool.
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
		case "daily", "weekly", "seldom":
			days := everyInDays(r.FormValue("every"), r.FormValue("unit"))
			if days <= 0 {
				// Asked for, never guessed at. Filing this as a note was the
				// old behaviour and it was the wrong kind of helpful: you typed
				// a chore, and what you got was a note in another rack, found
				// on the next refresh.
				http.Redirect(w, r, "/?bay=daily&rhythm="+url.QueryEscape(words), http.StatusSeeOther)
				return
			}
			every := time.Duration(days) * 24 * time.Hour
			if _, err := s.UpsertChore(r.Context(), personID, words, every, every/10); err != nil {
				fail(w, err)
				return
			}
		case "agenda":
			m, ok := momentFromPickers(opts.Location, words, dayFrom(r), clockFrom(r))
			if !ok {
				m, ok = squirrel.ParseMomentIn(opts.Location, words, now())
			}
			if !ok {
				// The same rule: an appointment with no time in it is a
				// question, not a note.
				http.Redirect(w, r, "/?bay=agenda&when="+url.QueryEscape(words), http.StatusSeeOther)
				return
			}
			m.EveryWeeks = weeksFrom(r.FormValue("weeks"))
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
		case "wrong":
			if err := notThisOne(r, s, personID, kind, refID); err != nil {
				fail(w, err)
				return
			}
		case "stuck":
			why := r.FormValue("why")
			back := backToTheStrip(r)
			if why == "" {
				http.Redirect(w, r, back+"&stuck=1", http.StatusSeeOther)
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
			if o, found, err := s.PickNow(r.Context(), personID, now(), true); err == nil && found {
				if smallerFor(s, opts, r, b, o) != nil && o.Kind == squirrel.OfferTask && o.RefID != 0 {
					http.Redirect(w, r, "/?open="+strconv.FormatInt(o.RefID, 10), http.StatusSeeOther)
					return
				}
			}
			http.Redirect(w, r, back+"&stuck="+url.QueryEscape(why), http.StatusSeeOther)
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

func notThisOne(r *http.Request, s Store, personID int64, kind squirrel.OfferKind, refID int64) error {
	if !offerKinds[kind] {
		return nil
	}
	return s.NotThisOne(r.Context(), personID, kind, refID, now())
}

// stuckView is what the pulled strip says while you are stuck: the four answers
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
		text, photo, kind, _, err := readCapture(r, opts)
		if err != nil {
			slog.Warn("a capture from the board was refused", "error", err)
			if refusedPhotograph(w, r, err, text) {
				return
			}
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
		http.Redirect(w, r, "/notes?kept=1", http.StatusSeeOther)
	}
}

// opened is the one strip you asked to see, drawn whole: its photograph at the
// size a photograph needs, and the answers it would carry in its rack.
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
		v.What, v.Answers, v.Held = "task", taskAnswers, heldAnswers
		v.Offerable = true
	default:
		v.Answers = noteAnswers
	}
	v.Reword = r.URL.Query().Get("reword") == "1"
	if !v.Back {
		v.Step = stepForItem(s, r, it.ID)
	}
	if newest, before := whatWasNoticedAbout(r, s, personID, "ask:"+v.What, it.ID); newest.ID != 0 {
		v.Seen, v.SeenID, v.Before = newest.Words, newest.ID, before
	}
	return v
}

func boardFixHandler(s Store, opts Options) http.HandlerFunc {
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
		text := strings.TrimSpace(r.FormValue("text"))
		if err != nil || id < 1 || text == "" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if len(text) > captureLimit {
			text = text[:captureLimit]
		}
		if _, err := s.Reword(r.Context(), personID, id, text); err != nil {
			fail(w, err)
			return
		}
		http.Redirect(w, r, "/?open="+strconv.FormatInt(id, 10), http.StatusSeeOther)
	}
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

func dayFrom(r *http.Request) string {
	if day := strings.TrimSpace(r.FormValue("day")); day != "" {
		return day
	}
	dd, mo := strings.TrimSpace(r.FormValue("dd")), strings.TrimSpace(r.FormValue("mo"))
	if dd == "" || mo == "" {
		return ""
	}
	day, err := strconv.Atoi(dd)
	if err != nil || day < 1 || day > 31 {
		return ""
	}
	month, err := strconv.Atoi(mo)
	if err != nil || month < 1 || month > 12 {
		return ""
	}
	today := now()
	at := time.Date(today.Year(), time.Month(month), day, 0, 0, 0, 0, today.Location())
	if at.Before(time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())) {
		at = at.AddDate(1, 0, 0)
	}
	return at.Format("2006-01-02")
}

// weeksFrom reads how often a fixed point comes round. Anything that is not one
// of the offered numbers is nought, which is a fixed point that happens once —
// the answer the writer defaults to and the only safe thing to make of a value
// nobody offered.
func weeksFrom(said string) int {
	weeks, err := strconv.Atoi(strings.TrimSpace(said))
	if err != nil || weeks < 1 || weeks > 52 {
		return 0
	}
	return weeks
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

// boardMoodHandler keeps a reading and puts you back on the board.
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

func whatWasNoticedAbout(r *http.Request, s Store, personID int64, kind string, refID int64) (squirrel.Noticed, []string) {
	lines, err := s.NoticedAbout(r.Context(), personID, kind, refID, noticedKept)
	if err != nil {
		slog.Error("reading what was noticed about this", "error", err)
		return squirrel.Noticed{}, nil
	}
	if len(lines) == 0 {
		return squirrel.Noticed{}, nil
	}
	before := make([]string, 0, len(lines)-1)
	for _, one := range lines[1:] {
		before = append(before, one.Words)
	}
	return lines[0], before
}

func answered(strips []stripView, id int64) []stripView {
	if id == 0 {
		return strips
	}
	for i := range strips {
		if strips[i].ID == id && strips[i].Seen != "" {
			strips[i].Answered = true
		}
	}
	return strips
}

func askable(strips []stripView, room string, on bool) []stripView {
	if !on {
		return strips
	}
	for i := range strips {
		strips[i].Room = room
	}
	return strips
}

// boardNotUsefulHandler is how a line is refused.
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

func boardAskHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil || !coachAvailable(opts) {
			http.Redirect(w, r, backToTheBay(r), http.StatusSeeOther)
			return
		}
		id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
		what := r.FormValue("what")
		words := strings.TrimSpace(r.FormValue("words"))
		room := r.FormValue("room")
		if _, ok := theBays[room]; !ok || id <= 0 || what == "" || words == "" {
			http.Redirect(w, r, backToTheBay(r), http.StatusSeeOther)
			return
		}
		answer, err := opts.Ask(r.Context(), personID, "strip", room, words, "")
		if err != nil {
			slog.Error("asking about a strip", "error", err)
			http.Redirect(w, r, backToTheBay(r), http.StatusSeeOther)
			return
		}
		remember(opts, personID, room, words, withDid(answer))
		if err := s.Notice(r.Context(), personID, "ask:"+what, id, withDid(answer), now()); err != nil {
			fail(w, err)
			return
		}
		back := backToTheBay(r)
		sep := "&"
		if !strings.Contains(back, "?") {
			sep = "?"
		}
		http.Redirect(w, r, back+sep+"answered="+strconv.FormatInt(id, 10), http.StatusSeeOther)
	}
}

// theOffer puts the picker's sentence on the row the pick is about, and lifts
// that row to the head of the rail.
//
// Only a chore or a thing you do once: those are the two kinds with a row of
// their own. A fixed point is the world's business and is drawn in the diary.
func theOffer(v *boardView) {
	if v.Pulled == nil {
		return
	}
	if v.Pulled.Kind != "chore" && v.Pulled.Kind != "task" {
		return
	}
	found := false
	mark := func(rows []stripView) {
		for i := range rows {
			if rows[i].What == v.Pulled.Kind && rows[i].ID == v.Pulled.RefID {
				rows[i].Because, rows[i].Chosen = v.Pulled.Because, true
				rows[i].Blockers, rows[i].Unstuck = v.Blockers, v.Unstuck
				found = true
			}
		}
	}
	for i := range v.Racks {
		mark(v.Racks[i].Strips)
		mark(v.Racks[i].Wants)
		mark(v.Racks[i].Rest)
	}
	if !found {
		// The pick is not on the board — a task further back than a rack
		// draws, or a chore a quiet day is holding. It still has to be
		// reachable, so it gets a row of its own.
		v.Loose = append(v.Loose, stripView{
			What: v.Pulled.Kind, ID: v.Pulled.RefID, Words: v.Pulled.Text,
			Because: v.Pulled.Because, Chosen: true, Answers: answersFor(v.Pulled.Kind),
			Blockers: v.Blockers, Unstuck: v.Unstuck,
		})
	}
	if v.Rail == nil {
		return
	}
	mark(v.Rail.Hung)
	for i := range v.Rail.Folded {
		mark(v.Rail.Folded[i].Rows)
	}
	if v.Rail.Head != nil && v.Rail.Head.What == v.Pulled.Kind && v.Rail.Head.ID == v.Pulled.RefID {
		v.Rail.Head.Because, v.Rail.Head.Chosen = v.Pulled.Because, true
		v.Rail.Head.Blockers, v.Rail.Head.Unstuck = v.Blockers, v.Unstuck
		return
	}
	for _, from := range []*[]stripView{&v.Rail.Hung} {
		for i, row := range *from {
			if row.What != v.Pulled.Kind || row.ID != v.Pulled.RefID {
				continue
			}
			was := v.Rail.Head
			lifted := row
			v.Rail.Head = &lifted
			*from = append((*from)[:i], (*from)[i+1:]...)
			if was != nil {
				v.Rail.Hung = append([]stripView{*was}, v.Rail.Hung...)
			}
			return
		}
	}
}

// howManyFolded is the two numbers the phone's folded rows carry. A read that
// fails counts nothing rather than failing the board: a row saying nought is a
// smaller lie than a screen that will not draw.
func howManyFolded(r *http.Request, s Store, personID int64) (notes, once int) {
	notes, once, err := s.HowMany(r.Context(), personID)
	if err != nil {
		slog.Error("counting what is folded away", "error", err)
		return 0, 0
	}
	return notes, once
}

func answersFor(kind string) []answerView {
	if kind == "chore" {
		return choreAnswers
	}
	return taskAnswers
}

// marginalia puts on a rack's rows the three things every other strip on the
// board carries: what was noticed about it, the press that asks Buddy, and the
// focus that lands on an answer just given.
// Separate from assemble because the racks are read separately, and the reason
// it is not simply left out is that leaving it out is what happened first: the
// chores lost their marginalia and their ask press the moment they stopped
// being a bay, and nothing on the screen said so.
func marginalia(r *http.Request, opts Options, seen map[string]squirrel.Noticed, racks []rackView) []rackView {
	askOn := coachAvailable(opts)
	justAsked, _ := strconv.ParseInt(r.URL.Query().Get("answered"), 10, 64)
	for i := range racks {
		rows := askable(racks[i].Strips, "chores", askOn)
		racks[i].Strips = answered(marked(marked(rows, "chore", seen), "ask:chore", seen), justAsked)
		// Split last. Marginalia writes onto the rows, and halving before it
		// leaves the template rendering copies nobody wrote to — which is
		// how the chores lost their ask press the first time.
		racks[i].Wants = halved(racks[i].Strips, true)
		racks[i].Rest = halved(racks[i].Strips, false)
	}
	return racks
}

// troubled says so on every rack when the chores could not be read. An empty
// rack and a rack that failed look identical, and one of them is a lie.
func troubled(racks []rackView, ok bool) []rackView {
	if ok {
		return racks
	}
	if len(racks) == 0 {
		for _, key := range []string{"now", string(squirrel.Daily), string(squirrel.Weekly), string(squirrel.Seldom)} {
			racks = append(racks, rackView{Key: key, Name: key})
		}
	}
	for i := range racks {
		racks[i].Trouble = true
	}
	return racks
}

func comesRoundWords(weeks int) string {
	switch {
	case weeks <= 0:
		return ""
	case weeks == 1:
		return "every week"
	}
	return "every " + strconv.Itoa(weeks) + " weeks"
}

// whatIsComing is every fixed point still ahead, soonest first.
// Not only today. The rule the list was allowed under is that it holds only
// what is still in front of you — nothing past, nothing done, never a count of
// what you did not do — and that rule does not care how far ahead it reaches.
func whatIsComing(r *http.Request, s Store, personID int64, at time.Time) *comingView {
	soon, err := s.Upcoming(r.Context(), personID, at, comingDeep+1)
	if err != nil {
		slog.Error("reading what is coming for the board", "error", err)
		return &comingView{Trouble: true}
	}
	c := &comingView{Count: len(soon)}
	if len(soon) > comingDeep {
		c.More, soon, c.Count = true, soon[:comingDeep], comingDeep
	}
	for i, m := range soon {
		one := apptView{
			ID: m.ID, Time: m.Starts.Format("15:04"), Label: m.Label,
			Big: i == 0 && sameDay(m.Starts, at),
		}
		if !sameDay(m.Starts, at) {
			one.Day = markOfMoment(m, at)
		}
		one.LeaveBy = leaveWords(m, at)
		one.Repeats = comesRoundWords(m.EveryWeeks)
		if m.Late(at) {
			one.Late, one.LeaveBy = true, ""
		}
		// Inside the window where leaving matters it comes out of the list and
		// sits above the dial. One thing, in one place — a fixed point shown
		// twice in one column is the duplication the picker was cured of.
		if (m.Open(at) || one.Late) && c.Hoisted == nil {
			lifted := one
			lifted.Big = true
			c.Hoisted = &lifted
			continue
		}
		c.Appts = append(c.Appts, one)
	}
	return c
}

// comingDeep is how far ahead the sidebar draws before it says there is more.
// Four is what fits beside the dial; the number is the column's, not a rule.
const comingDeep = 4

// leaveWords is arithmetic about a distance, and says so when the distance was
// a guess. Never called a deadline: the appointment's own time is the deadline
// and is set as one.
func leaveWords(m squirrel.Moment, at time.Time) string {
	if m.Travel == 0 {
		return ""
	}
	leave := "leave " + m.LeaveAt().Format("15:04")
	if m.Guessed {
		return leave + " — if it is a quarter of an hour away"
	}
	return leave
}

// howYouAre is the dial: today's face, the seven days behind it, and the way
// to the whole record.
// Always pressable, which is the change. The faces used to appear only when
// Squirrel wanted an answer, so saying how you were was something you waited to
// be asked for. Being asked is still a separate thing — Faces is what carries
// that, and it is empty while the last answer still describes now.
func howYouAre(r *http.Request, s Store, personID int64, at time.Time) *dialView {
	d := &dialView{Faces: theFaces()}

	latest, found, err := s.LatestCheckin(r.Context(), personID)
	if err != nil {
		slog.Error("reading how you are", "error", err)
		return nil
	}
	if found && sameDay(latest.SaidAt, at) {
		d.Mood, d.Word, d.Said = string(latest.Mood), squirrel.Words[latest.Mood], true
	}
	d.Asking = !found || !latest.JustAsked(at)

	readings, err := s.CheckinsSince(r.Context(), personID, at.AddDate(0, 0, -dialDays))
	if err != nil {
		// The face is the dial's job; the week behind it is the extra. One
		// without the other is worth drawing.
		slog.Error("reading how you have been", "error", err)
		return d
	}
	d.Days = moodDaysBefore(readings, at)
	d.Ring = moodRing(d.Days)
	return d
}
