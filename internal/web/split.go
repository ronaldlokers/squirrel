package web

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Four thoughts captured as one note, offered as four.
//
// Nothing here is automatic and nothing is silent. A note that looks like
// several things gets one extra press on its card; pressing it proposes; the
// pieces are only written when a second press says so. Capture is never
// rewritten — the original stays, and it stays in your words.
//
// The proposal is not stored anywhere. It travels in the form that renders it,
// which is what makes it impossible for one to be applied without a press, and
// what means there is no pending state to expire. It also answers the
// architecture's open question about how long an unanswered proposal lasts:
// as long as the page it is on.

// splitHandler is both halves: proposing, and keeping what was proposed.
func splitHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/pile", http.StatusSeeOther)
			return
		}
		id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
		if err != nil {
			http.Redirect(w, r, "/pile", http.StatusSeeOther)
			return
		}

		switch r.FormValue("act") {
		case "propose":
			proposeSplit(w, r, s, opts, personID, id)
		case "keep":
			keepSplit(w, r, s, opts, personID, id)
		default:
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	}
}

// proposeSplit asks, and renders the answer next to the note it came from.
//
// Rendered rather than redirected, because the proposal lives in the page. A
// reload asks again, which costs one cheap call and is the honest behaviour:
// there is nothing stored to come back to.
func proposeSplit(w http.ResponseWriter, r *http.Request, s Store, opts Options, personID, id int64) {
	it, found, err := s.ItemByID(r.Context(), personID, id)
	if err != nil || !found {
		http.Redirect(w, r, "/pile", http.StatusSeeOther)
		return
	}
	if opts.Split == nil {
		http.Redirect(w, r, "/pile", http.StatusSeeOther)
		return
	}

	pieces, ok := opts.Split(r.Context(), personID, it.RawText)
	if !ok {
		// It could not, or it decided the note is really one thought. Either
		// way the note is exactly as it was.
		if fromThread(r) {
			answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
				{Who: squirrel.SpeakerYou, Words: "is this more than one thing?"},
				{Who: squirrel.SpeakerBuddy, Words: "It reads as one thing to me."},
			}), "/")
			return
		}
		http.Redirect(w, r, "/pile", http.StatusSeeOther)
		return
	}
	if fromThread(r) {
		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "is this more than one thing?"},
			proposeInThread(id, pieces),
		}), "/")
		return
	}
	pileInto(w, r, s, opts, personID, &splitView{ID: id, Said: it.RawText, Pieces: pieces})
}

// keepSplit writes the pieces and files the original.
//
// The original is kept rather than dropped, and that is the whole of decision
// 8 in one line: what you actually typed stays readable, on the shelf, next to
// the four things it turned into. Dropping it would make a model's reading of
// your words the only surviving version of them.
//
// Every piece is an ordinary note. Nothing marks them as a machine's doing,
// because by the time they exist you pressed a button to say they are yours.
func keepSplit(w http.ResponseWriter, r *http.Request, s Store, opts Options, personID, id int64) {
	pieces := r.Form["piece"]
	if len(pieces) < 2 {
		http.Redirect(w, r, "/pile", http.StatusSeeOther)
		return
	}

	for _, piece := range pieces {
		text := strings.TrimSpace(piece)
		if text == "" {
			continue
		}
		if _, err := s.InsertItem(r.Context(), squirrel.Item{
			PersonID: &personID, RawText: text, ReceivedAt: now(),
			// The same transport the slot's own captures carry: a piece is a
			// note the screen wrote, and pretending otherwise would put a
			// fourth transport in the table for one feature.
			Transport: "web", Payload: []byte(`{}`),
		}); err != nil {
			fail(w, err)
			return
		}
	}

	// Kept, not dropped, and it goes through the same transition every other
	// exit uses — so undo works on it exactly as it works on anything else.
	if err := s.SetItemState(r.Context(), id, squirrel.ItemKept, now()); err != nil {
		fail(w, err)
		return
	}
	if fromThread(r) {
		answerWith(w, r, keepSaid(r.Context(), s, personID, append(
			[]squirrel.Turn{
				{Who: squirrel.SpeakerYou, Words: "use these"},
				{Who: squirrel.SpeakerBuddy, Words: "Kept, and the note itself is on the shelf."},
			},
			pileTurn(r.Context(), s, opts, personID, 0, ""),
		)), "/")
		return
	}
	http.Redirect(w, r, "/pile?undo="+strconv.FormatInt(id, 10)+"&was=open&state=kept",
		http.StatusSeeOther)
}

// splitView is a proposal, on the card it came from.
//
// It carries the original as well as the pieces, because the question being
// asked is "is this what you meant" and it cannot be answered without both.
type splitView struct {
	ID     int64
	Said   string
	Pieces []string
}
