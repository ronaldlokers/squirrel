package web

import (
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

//go:embed templates/*.html
var templateFS embed.FS

// Each page parses layout, card and exactly one content template. Go's
// templates are a flat namespace, so two files both defining "content" cannot
// live in one set — the set is the page.
var pages = map[string]*template.Template{
	"pile":    template.Must(template.ParseFS(templateFS, "templates/layout.html", "templates/card.html", "templates/pile.html")),
	"empty":   template.Must(template.ParseFS(templateFS, "templates/layout.html", "templates/empty.html")),
	"results": template.Must(template.ParseFS(templateFS, "templates/layout.html", "templates/results.html")),
}

type noteView struct {
	ID        int64
	Text      string
	When      string
	State     string
	StateWord string
}

type view struct {
	Path    string
	Query   string
	Note    *noteView
	More    bool
	Results []noteView
	Undo    *undoView
}

type undoView struct {
	ID    int64
	Said  string
	State string
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

func render(w http.ResponseWriter, name string, v view) {
	t, ok := pages[name]
	if !ok {
		panic("no such page: " + name)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Never cached. The pile is state, and a back button that showed a note you
	// already triaged would be the two views disagreeing with themselves.
	w.Header().Set("Cache-Control", "no-store")
	if err := t.ExecuteTemplate(w, "layout", v); err != nil {
		slog.Error("rendering the pile", "page", name, "error", err)
	}
}

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
