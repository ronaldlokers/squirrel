package web

import (
	"embed"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

//go:embed templates/*.html
var templateFS embed.FS

// Each page parses layout, card and exactly one content template. Go's
// templates are a flat namespace, so two files both defining "content" cannot
// live in one set — the set is the page.
// dict lets one template be handed two values, which Go's templates otherwise
// cannot do. Only "step" needs it, and only because a step is drawn on two
// screens that disagree about where its buttons should come back to.
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

func page(files ...string) *template.Template {
	return template.Must(template.New("layout.html").Funcs(helpers).ParseFS(templateFS, files...))
}

var pages = map[string]*template.Template{
	"thread": page("templates/layout.html", "templates/turn.html", "templates/stopping.html", "templates/thread.html"),
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
	// Here is which of the two screens this is, so the lid can offer the other
	// one. A link that points at the page you are on is furniture.
	Here string
	// Scrolling is a list rather than a single card, so the field is
	// top-aligned. The deck centres because it holds exactly one thing and a
	// centred card is the shoebox's own composition; a list that grows from
	// the middle of the screen is just a list that is hard to read.
	Scrolling bool
	// The four sentences met most often, in today's wording. Habituation is
	// the documented enemy and the card stack was the only thing that moved;
	// these are art-and-phrasing novelty, which is what the roadmap sanctions.
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
	// Place is where you are, in the menu's own words, so the shut control can
	// say it without the template knowing the mapping.
	Place string
	// Home is the front door, where the lid carries no cross-link at all —
	// both doors are already the body of the page — and where the mark is not
	// a link, because it is a link to here.
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
	// Path is the page the acorn is being drawn on, so closing the coach
	// returns to it. renderWith fills it, so no handler can forget.
	Path string
	// Coach is the sheet's contents, and only the coach page fills it. Every
	// other page carries the acorn and nothing more — the sheet's markup
	// arrives when it is opened, because a conversation nobody has started is
	// Menu is everywhere else, behind the lid's one control. It carries what
	// the rail, the chip row and the stop link used to occupy the conversation
	// with — see layout.html for why a hamburger came back.
	Menu []turnChip
	// Also is the pair of chips at the foot of the conversation: asking Buddy
	// and looking something up. See thread.html.
	Also []turnChip
	// V stamps every asset URL on the page. render fills it, so no handler can
	// forget it and no template has to know where it comes from.
	V     string
	Query string
	Undo  *undoView
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

// linkView is one of the lid's cross-links.
type linkView struct {
	Href  string
	Label string
	// Here is the place you already are. It is still drawn, and it is not a
	// link: a nav whose items move as you move is a nav you have to read every
	// time, and the cheapest way to say where you are is to say it in the same
	// row as everywhere else you could be.
	Here bool
}

// placeName is where you are, in the words the menu uses for it.
//
// A screen that hangs off another one answers with its parent: the shelf is
// somewhere in the pile, the archive and the set-aside are somewhere in the
// tasks. The menu says which room you are in, and `views` says which corner.
func placeName(here string) string {
	switch here {
	case "pile", "kept", "bottom", "enough":
		return "the pile"
	case "tasks", "archive", "held":
		return "the tasks"
	case "chores":
		return "the chores"
	case "at":
		return "the agenda"
	case "buddy", "coach":
		return "buddy"
	}
	return "home"
}

// elsewhere is the map: the three places, with the one you are in marked.
//
// Behind a hamburger now rather than beside the mark. The lid was a row of
// words that cost most of a phone screen before anything you came for, and
// what it said — where you can go — is the thing you need least often and can
// always ask for.
//
// The place you are in is in the list and not a link. A menu that drops it
// has items that move as you move, so "the second one" means a different
// screen on every screen.
//
// Home is not in it: the mark is the way home and has been since the screen
// existed. Buddy is not in it either — it is one tap in the lid, because a
// conversation about what is in front of you should not be two.
func elsewhere(here string) []linkView {
	// One, since 24 August 2026. The tasks, the chores and the agenda stopped
	// being pages and became messages, and a menu that offered them would be
	// offering a link to somewhere that no longer exists. The way to them is
	// the rail, on the one screen you reach them from; the way back to that
	// screen is the mark, as it always was.
	//
	// None, since the deck came out. Every place is a message; the way to one
	// is the rail, and the way to the rail is the mark. A menu of one item
	// that is always where you are is furniture, and the lid has no room for
	// furniture.
	return nil
}

// views are the places that belong to the screen you are on, and nothing else
// reaches them. The pile keeps a shelf; the tasks keep what is finished and
// what is stalled. They sit under the title because that is what they are
// about — a link to the archive means nothing next to the chores.
func views(here string) []linkView {
	// None, since the shelf and the set-aside came into the conversation on
	// 25 August 2026. They were the last two screens with a view of their own,
	// and a place that is a message has no page to hang one on.
	return nil
}

// moodWeekView is one row of the readings grid: a label and seven days.
type moodWeekView struct {
	Week string
	Days []moodCellView
}

// moodCellView is one day on the grid. It carries no number and no position,
// for the same reason a face does not: these are not a scale.
//
// Nought is a day you said nothing on, which is drawn rather than skipped —
// the gaps are most of what is there and hiding them would be the flattery
// this page exists to avoid. Ahead is a day that has not happened, which is
// drawn as nothing at all: an empty Saturday next week is not a gap.
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
	// Unstuck is the ladder's answer, once one has been asked for. The offer
	// stays on the card underneath it: the thing you could not start is still
	// the thing, and taking it away would make "I can't start" a way of losing
	// it.
	Unstuck *unstuckView
}

// unstuckView is one line and at most one control. There is deliberately
// nowhere here to put a second step — the failure being avoided is the
// twelve-step plan, and a struct that cannot hold one cannot render one.
type unstuckView struct {
	Line    string
	Minutes int
	Ask     bool
	// Step is the ladder having got specific: one step of a stored sequence,
	// or nil. The line above it is what shows when this is nil, which is what
	// it did on its own before a model existed — the fixed answer is not
	// replaced, it is the floor this stands on.
	Step *stepView
}

// coachPanel was the sheet. It went with the page on 25 August 2026: Buddy is
// turns in the conversation now, and the parts of it that do something —
// the four blockers, the steps, the four proposals, "that went badly" and the
// spend — are drawn by askbuddy.go from the same values.

// chipView is one blocker as a press. Why is what the form sends; Word is what
// it says, and they differ because "not today" is an answer and `not today` is
// a value.
type chipView struct {
	Why  string
	Word string
}

type undoView struct {
	// ID is the note to move, and State is where to move it back to. The pair
	// is a plain transition — undo is not a special case, it is `act=open` on
	// a note that just left.
	ID    int64
	State string
	// Said is what happened, in the same words the card's own stamp uses.
	Said string
	// Action is where the way back is posted, because the three set-aside
	// states do not come back the way the other four do: `open` is a state a
	// note can be moved to, while a note that was set aside is *picked back
	// up*, which is its own verb on its own route.
	Action string
	// From is where the way back returns you to, for the route that asks.
	From string
}

// saidWords is the other half of pile.js's STATES table: what the screen says
// a transition did, once it has been done. The two have to match, because with
// JavaScript the phrase appears on the card and without it the phrase appears
// on the next page, for the same action.
var saidWords = map[string]string{
	"done":    "marked done",
	"kept":    "kept as reference",
	"dropped": "dropped",
	"chore":   "now a chore",
	// Deciding is not disposing, and it was the one answer on the card with no
	// word here at all — so the stamp fell through to the default and a note
	// promoted to a task announced itself as "marked done".
	"task": "now a task",
	"open": "back in the pile",
	// Setting one aside is a transition like any other, and until now it was
	// the only one that said nothing afterwards and offered nothing back.
	"waiting": "waiting on someone",
	"blocked": "blocked on a thing",
	"someday": "someday",
}

// backTo turns the state a note was in into the action word that returns it
// there. The form's vocabulary and the store's are deliberately not the same
// list — `keep` is what you do to a note, `kept` is what the note then is — so
// the round trip needs one place that knows both.
var backTo = map[squirrel.ItemState]string{
	squirrel.ItemOpen:    "open",
	squirrel.ItemDone:    "done",
	squirrel.ItemKept:    "keep",
	squirrel.ItemDropped: "drop",
}

// undoFrom reads the way back out of the query string, or answers nil.
//
// The parameters arrive from a redirect this package wrote, but they arrive
// through the address bar, so they are read as though a stranger typed them:
// an id that is not a number, or a state that is not one of the four, is no
// undo rather than a bad one.
func undoFrom(q url.Values) *undoView {
	id, err := strconv.ParseInt(q.Get("undo"), 10, 64)
	if err != nil || id == 0 {
		return nil
	}
	said := saidWords[q.Get("state")]
	// A promotion, which comes back by being made a note again rather than by
	// being moved to a state: what changed was the note's kind, and its state
	// never moved, so `act=open` would undo nothing at all.
	if q.Get("was") == "task" {
		return &undoView{ID: id, State: "note", Said: said, Action: "/pile/act"}
	}
	// Set aside, which comes back by being picked back up rather than by being
	// moved to a state. Checked first: the two vocabularies do not overlap, and
	// this one is not in `backTo` on purpose.
	if _, held := squirrel.ParseHeld(q.Get("was")); held {
		return &undoView{
			ID: id, State: "back", Said: said,
			Action: "/held/act", From: backTolerant(q.Get("from")),
		}
	}
	act, ok := backTo[squirrel.ItemState(q.Get("was"))]
	if !ok {
		return nil
	}
	return &undoView{ID: id, State: act, Said: said, Action: "/pile/act"}
}

// clashFrom says whether this render is answering a decision the pile had
// already overtaken. Read like every other parameter here: from an address
// bar, so anything that is not a number is no clash rather than a bad one.
func clashFrom(q url.Values) bool {
	id, err := strconv.ParseInt(q.Get("clash"), 10, 64)
	return err == nil && id > 0
}

// cursorFrom reads the skip position out of the query string. Nonsense is no
// cursor: this arrives from an address bar, and the pile is the right answer to
// a question that does not parse.
func cursorFrom(q url.Values) int64 {
	id, err := strconv.ParseInt(q.Get("after"), 10, 64)
	if err != nil || id < 1 {
		return 0
	}
	return id
}

// stateWords is the screen's half of the shared vocabulary. `open` is
// deliberately present: a search result still in the pile says so, and it wears
// Notebook Violet rather than one of the three exit colours.
// stateWords is what a row says it is. `open` is deliberately present: a
// search result still in the pile says so, and it wears Notebook Violet rather
// than one of the three exit colours.
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
	// The menu, on every screen that has a store to build it from. It carries
	// the counts, so it is filled here rather than in render() — which takes
	// neither a store nor a person and never could.
	if personID, ok := personOf(r); ok {
		v.Menu = menuFor(r.Context(), s, personID)
	}
	v.Timer = runningTimer(s, opts, r)
	v.PushKey = opts.PushKey
	v.Camera = opts.Photos != nil
	v.Path = r.URL.Path
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
	v.Place = placeName(v.Here)
	v.Scrolling = v.Scrolling || v.Query != "" ||
		v.Here == "tasks" || v.Here == "archive" || v.Here == "kept"
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
		// The way on. The screen is down and chat is not: the room writes
		// through the same spool this does, so capture still works while this
		// page is what the screen can offer. The service worker's offline page
		// has said so since it was written; this one, which fires on the same
		// failure from the other side, said nothing and left you on a page
		// with no way forward.
		`<p style="opacity:.75">Notes are still kept by talking to Squirrel in Campfire.</p>` +
		`</body></html>`))
}

// now is the clock, replaceable in a test that has to say what "right now"
// means. Nothing else in this package needs one, which is why it is here
// rather than threaded through Options.
var now = time.Now
