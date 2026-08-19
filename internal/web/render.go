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
var pages = map[string]*template.Template{
	"home":    template.Must(template.ParseFS(templateFS, "templates/layout.html", "templates/home.html")),
	"chores":  template.Must(template.ParseFS(templateFS, "templates/layout.html", "templates/chores.html")),
	"kept":    template.Must(template.ParseFS(templateFS, "templates/layout.html", "templates/kept.html")),
	"bottom":  template.Must(template.ParseFS(templateFS, "templates/layout.html", "templates/bottom.html")),
	"pile":    template.Must(template.ParseFS(templateFS, "templates/layout.html", "templates/every.html", "templates/card.html", "templates/pile.html")),
	"empty":   template.Must(template.ParseFS(templateFS, "templates/layout.html", "templates/empty.html")),
	"results": template.Must(template.ParseFS(templateFS, "templates/layout.html", "templates/every.html", "templates/results.html")),
}

type noteView struct {
	ID        int64
	Text      string
	When      string
	State     string
	StateWord string
}

type view struct {
	// Here is which of the two screens this is, so the lid can offer the other
	// one. A link that points at the page you are on is furniture.
	Here string
	// Home is the front door, where the lid carries no cross-link at all —
	// both doors are already the body of the page — and where the mark is not
	// a link, because it is a link to here.
	Home bool
	// Said carries words back to the slot when the write failed. A capture box
	// that clears on failure is a capture box that eats thoughts.
	Said string
	// Kept and NoKeep are what the slot says back, and they are deliberately
	// two booleans rather than one message: the failure is not the success
	// with different words, and only one of them keeps the words.
	Kept   bool
	NoKeep bool
	// Held is the worker having taken the words because there was no network.
	// A third state and not a flavour of the other two: the words are safe,
	// which failure is not, and they are not in the pile yet, which kept is.
	Held bool
	// Mood is the latest reading when it still describes now, and empty
	// otherwise. Never more than one, and never a date beside it.
	Mood     string
	MoodWord string
	// Faces are the five, in the one order both surfaces use.
	Faces []faceView
	// Timer is what is running, on every screen, or nil.
	Timer *timerView
	// V stamps every asset URL on the page. render fills it, so no handler can
	// forget it and no template has to know where it comes from.
	V string
	// After is the note space moved past, carried so that a transition made
	// while skipped comes back to the same place rather than to the top.
	After   int64
	Query   string
	Note    *noteView
	More    bool
	Results []noteView
	Chores  []choreView
	Undo    *undoView
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

// faceView is one of the five drawn answers. It carries no number and no
// position, because they are not a scale.
type faceView struct {
	Mood string
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
	"open":    "back in the pile",
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
	act, ok := backTo[squirrel.ItemState(q.Get("was"))]
	if !ok {
		return nil
	}
	return &undoView{ID: id, State: act, Said: saidWords[q.Get("state")]}
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
var stateWords = map[squirrel.ItemState]string{
	squirrel.ItemOpen:    "IN THE PILE",
	squirrel.ItemDone:    "DONE",
	squirrel.ItemDropped: "DROPPED",
	squirrel.ItemKept:    "KEPT",
}

func toView(it squirrel.Item) noteView {
	return noteView{
		ID:        it.ID,
		Text:      it.RawText,
		When:      strings.ToUpper(it.ReceivedAt.Local().Format("2 January")),
		State:     string(it.State),
		StateWord: stateWords[it.State],
	}
}

// renderWith is render plus the one thing every screen shows regardless of
// what it is about: the timer, if one is running. Threading it through each
// handler would mean five places to forget it.
func renderWith(w http.ResponseWriter, r *http.Request, s Store, opts Options, name string, v view) {
	v.Timer = runningTimer(s, opts, r)
	render(w, name, v)
}

func render(w http.ResponseWriter, name string, v view) {
	t, ok := pages[name]
	if !ok {
		panic("no such page: " + name)
	}
	v.V = assetVersion
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
		`</body></html>`))
}

// now is the clock, replaceable in a test that has to say what "right now"
// means. Nothing else in this package needs one, which is why it is here
// rather than threaded through Options.
var now = time.Now
