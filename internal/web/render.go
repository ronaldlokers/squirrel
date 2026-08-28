package web

import (
	"embed"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

//go:embed templates/*.html
var templateFS embed.FS

var helpers = template.FuncMap{
	// add exists for one thing: telling the last item in a range from the
	// rest, so a control can sit on the newest reply and nowhere else.
	"add": func(a, b int) int { return a + b },
	"dict": func(pairs ...any) map[string]any {
		out := make(map[string]any, len(pairs)/2)
		for i := 0; i+1 < len(pairs); i += 2 {
			key, _ := pairs[i].(string)
			out[key] = pairs[i+1]
		}
		return out
	},
}

// page parses layout and exactly one content template: Go's templates are a flat
// namespace, so two files defining "content" cannot share a set.
func page(files ...string) *template.Template {
	return template.Must(template.New("layout.html").Funcs(helpers).ParseFS(templateFS, files...))
}

var pages = map[string]*template.Template{
	"thread": page("templates/layout.html", "templates/turn.html", "templates/thread.html"),
	"moods":  page("templates/layout.html", "templates/moods.html"),
	"enough": page("templates/layout.html", "templates/enough.html"),
}

// gatePage is on its own, outside `pages`, because it carries none of the
// app's frame and none of view's fields. Everything render() fills in — the
// menu, the sentences, the timer — is for people who are already inside.
var gatePage = template.Must(
	template.New("gate.html").Funcs(helpers).ParseFS(templateFS, "templates/gate.html"))

type noteView struct {
	ID   int64
	Text string
	// Photo is the note's own picture, or empty. The pile renders it above the
	// words, because a photograph of a letter is the note and the words beside
	// it are the caption.
	Photo string
	When  string
	State string
	// Task says this row was decided on rather than left in the pile. Search
	// reads every state and every kind, so a result has to say which it is:
	// without it an open task reported itself as IN THE PILE and was offered
	// the pile's verbs, and pressing KEEP moved it off the tasks screen.
	Task      bool
	StateWord string
}

// momentView is one fixed point with its arithmetic already done. Open is
// whether the leave-by window is running, worked out in the handler so a
// template cannot disagree with the core about when to leave.
type momentView struct {
	ID    int64
	Label string
	// Words is the one sentence a fixed point says, shared with chat and with
	// the notification so all three cannot drift.
	Words string
	// Take is what to bring, or empty. Its own field rather than a clause
	// appended to Words, because the thing you are standing in the hall
	// without deserves to be an element.
	Take string
	Open bool
}

type view struct {
	// Here is which screen this is, so the menu can say where you are.
	Here string
	// Scrolling is a list rather than a single card, so the field is
	// top-aligned: a list that grows from the middle of the screen is a list
	// that is hard to read.
	Scrolling bool
	// The sentences met most often, in today's wording: habituation is the
	// documented enemy, and phrasing is one of the two things allowed to move.
	// Every control label is deliberately absent from this list.
	SaySlot   string
	SayStop   string
	SayEnough string
	// The two things that move without being read. The stamp's lean and where
	// the room's light falls, both chosen from the day like the sentences are,
	// and both handed to the stylesheet as custom properties because a static
	// file cannot know what day it is. See squirrel.Tilt and squirrel.Light.
	Tilt  int
	Light int
	// Home is the front door, where the mark is not a link because it would be
	// a link to here.
	Home bool
	// Said carries words back to the slot when the write failed. A capture box
	// that clears on failure is a capture box that eats thoughts.
	Said string
	// Held is the worker having taken the words because there was no network.
	// A third state and not a flavour of the other two: the words are safe,
	// which failure is not, and they are not in the pile yet, which kept is.
	Held bool
	// Mood is the latest reading when it still describes now, and empty
	// otherwise. Never more than one, and never a date beside it.
	Mood string
	// Faces are the five, in the one order both surfaces use.
	Faces []faceView
	// Example is the worked example, on a conversation nobody has ever said
	// anything in, and empty every other time. It is drawn and never stored —
	// see internal/web/firstrun.go.
	Example []exampleTurn

	// Weeks is how you have been, as six weeks by seven days, and only the
	// readings page fills it.
	Weeks []moodWeekView
	// Camera is whether a photograph can be kept at all. False draws no
	// camera: a control that cannot work is worse than one never drawn.
	Camera bool
	// PushKey is the VAPID public key, or empty when pushing is not
	// configured. The script offers to subscribe only when there is one.
	PushKey string
	// Timer is what is running, on every screen, or nil.
	Timer *timerView
	// Rooms is the rail, on every screen. It replaced the lid's menu on
	// 28 August 2026: a room list you navigate to is a screen, and a room list
	// that is always there is furniture. See internal/web/rooms.go.
	Rooms []railView
	// Room is the one you are in, and it is what the dock reads its
	// placeholder, its button and its action from. Filled by the handler
	// rather than by renderWith, because renderWith cannot know.
	Room room
	// V stamps every asset URL on the page. render fills it, so no handler can
	// forget it and no template has to know where it comes from.
	V string
	// Turns is the conversation, oldest first. The screen is one page now;
	// see internal/web/thread.go.
	Turns []turnView
	// MoreAbove and Oldest are the page above this one. Oldest is the id the
	// "earlier" control walks back from.
	MoreAbove bool
	Oldest    int64
	// Clash says a decision arrived for a note that had already moved
	// somewhere else — from the room, while the card was still on the screen.
	// It is not an error and there is nothing to undo: what it says is that
	// the pile is not what this screen was showing.
	Clash bool
}

// choreView is a chore as the screen says it: what it is, how often it comes
// back, and when it was last done. No "due", no "late", no position in a queue.
type choreView struct {
	ID   int64
	Name string
	// Every is the rhythm as a person says it; Chip is which of the four
	// offered intervals that corresponds to, if any. Last is empty for a chore
	// that has never been done, and When for one with no preference about
	// being raised.
	Every string
	Chip  string
	Last  string
	When  string
}

// moodWeekView is one row of the readings grid: a label and seven days.
type moodWeekView struct {
	Week string
	Days []moodCellView
}

// moodCellView is one day on the grid, carrying no number and no position.
//
// Nought is a day you said nothing on, drawn rather than skipped: the gaps are
// most of what is there. Ahead is a day that has not happened, drawn as nothing —
// an empty Saturday next week is not a gap.
type moodCellView struct {
	Day    string
	Mood   string
	Word   string
	Nought bool
	Ahead  bool
}

// faceView is one of the five drawn answers. It carries no number and no
// position, because they are not a scale.
type faceView struct {
	Mood string
	Word string
}

// offerView is the one thing, as the screen says it.
//
// No state, no colour, no date and nothing about how long it has been waiting.
// Because is the clause that explains the choice and is not optional — an
// offer that cannot say why it is the offer is a demand.
type offerView struct {
	Kind    string
	RefID   int64
	Text    string
	Because string
	// Running means this is the timer you already started rather than
	// something chosen for you, so the card offers nothing to press: the lid
	// already carries the only control it needs.
	Running bool
}

// stateWords is what a row says it is. `open` is deliberately present: a search
// result still in the pile says so, and wears Notebook Violet rather than one of
// the three exit colours.
//
// A task is open too, and is not in the pile — see taskWords.
var stateWords = map[squirrel.ItemState]string{
	squirrel.ItemOpen:    "IN THE PILE",
	squirrel.ItemDone:    "DONE",
	squirrel.ItemDropped: "DROPPED",
	squirrel.ItemKept:    "KEPT",
}

// taskWords is the same map for a row that was decided on. Only `open`
// differs: a task you have not done is not in the pile, it is on the list of
// things you decided to do, and the two are different places with different
// verbs.
var taskWords = map[squirrel.ItemState]string{
	squirrel.ItemOpen:    "DECIDED",
	squirrel.ItemDone:    "DONE",
	squirrel.ItemDropped: "DROPPED",
	squirrel.ItemKept:    "KEPT",
}

func toView(it squirrel.Item) noteView {
	photo := ""
	if it.PhotoName != "" {
		// By the note's id, never by the file's name: the name is the only
		// string here that becomes a path, and a URL is a place a stranger can
		// type. The row is what says which file belongs to you.
		photo = "/photo/" + strconv.FormatInt(it.ID, 10)
	}
	words := stateWords
	if it.Kind == squirrel.ItemTask {
		words = taskWords
	}
	return noteView{
		ID:    it.ID,
		Photo: photo,
		Text:  it.RawText,
		// No `.Local()`. It carried one until 25 August 2026, which read the
		// *process* clock — and the pods run in UTC on purpose since #148, so
		// anything captured after ten in the evening wore yesterday's date.
		// The store hands this back in the person's clock now; converting
		// again here would be the same bug with an extra step.
		When:      strings.ToUpper(it.ReceivedAt.Format("2 January")),
		State:     string(it.State),
		Task:      it.Kind == squirrel.ItemTask,
		StateWord: words[it.State],
	}
}

// renderWith is render plus the one thing every screen shows regardless of
// what it is about: the timer, if one is running. Threading it through each
// handler would mean five places to forget it.
func renderWith(w http.ResponseWriter, r *http.Request, s Store, opts Options, name string, v view) {
	// The rail, on every screen that has a store to build it from. It carries
	// the counts, so it is filled here rather than in render() — which takes
	// neither a store nor a person and never could.
	if personID, ok := personOf(r); ok {
		v.Rooms = roomsFor(r.Context(), s, personID, roomOf(r.Context()))
	}
	v.Timer = runningTimer(s, opts, r)
	v.PushKey = opts.PushKey
	v.Camera = opts.Photos != nil
	render(w, name, v)
}

func render(w http.ResponseWriter, name string, v view) {
	t, ok := pages[name]
	if !ok {
		panic("no such page: " + name)
	}
	v.V = assetVersion
	// What the sentences say today. Chosen from the day, so both viewports
	// agree and a reload is not a slot machine — see squirrel.Say.
	v.SaySlot = squirrel.Say(squirrel.SayingSlot, now())
	v.SayStop = squirrel.Say(squirrel.SayingStop, now())
	v.SayEnough = squirrel.Say(squirrel.SayingEnough, now())
	v.Tilt = squirrel.Tilt(now())
	v.Light = squirrel.Light(now())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Never cached. The pile is state, and a back button that showed a note you
	// already triaged would be the two views disagreeing with themselves.
	w.Header().Set("Cache-Control", "no-store")
	if err := t.ExecuteTemplate(w, "layout", v); err != nil {
		slog.Error("rendering the pile", "page", name, "error", err)
	}
}

// errNoOwner is the screen before the database has ever answered: the routes
// are live because they were registered before Listen, but nobody knows whose
// pile this is yet. It reads as an unreachable database because that is what
// it is.
var errNoOwner = errors.New("the owner is not known yet")

// fail is what "the screen fails visibly and nothing is lost" looks like. The
// note is already durable; this is the exit, not the entrance.
func fail(w http.ResponseWriter, err error) {
	slog.Error("the pile could not be read", "error", err)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`<!doctype html><html lang="en"><head><meta charset="utf-8">` +
		`<title>Squirrel</title></head><body style="background:#58388a;color:#fffbf3;font:16px system-ui;padding:3rem">` +
		`<p>Squirrel cannot reach its memory right now. Nothing has been lost — everything you said is still there.</p>` +
		// The way on. The screen is down and chat is not: the room writes through the
		// same spool, so capture still works. The service worker's offline page has said
		// so since it was written; this one said nothing and left you with no way
		// forward.
		`<p style="opacity:.75">Notes are still kept by talking to Squirrel in Campfire.</p>` +
		`</body></html>`))
}

// now is the clock, replaceable in a test that has to say what "right now"
// means. Nothing else in this package needs one, which is why it is here
// rather than threaded through Options.
var now = time.Now
