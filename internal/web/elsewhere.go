package web

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The shelf and the set-aside.
//
// The shelf holds what you kept rather than did or dropped; the set-aside holds
// what you cannot act on yet.
//
// The way to the shelf used to hang off the drawn card in the pile's turn, so
// the moment there was nothing to decide about it was reachable from nowhere at
// all. elsewhereFromThePile puts both on every branch of the notes' turn.

// elsewhereFromThePile is the way to both, and it is on every branch of the
// notes' turn — the one that hands you a note, the one that says there is
// nothing to decide about, and the bottom you reach by skipping.
//
// A press rather than a link since 31 August 2026, because a shelf stopped
// being a room you go to and became a thing you are shown where you already
// are. See shelfChips.
func elsewhereFromThePile() []turnChip { return shelfChips() }

// sayWithChips is a sentence that can still take you somewhere. Without it a
// turn with no cards has nowhere to hang a chip, which is the whole of the
// unreachable-shelf bug.
func sayWithChips(words string, chips []turnChip) squirrel.Turn {
	body, err := json.Marshal(drawn{Chips: chips})
	if err != nil {
		slog.Error("drawing the ways out", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: words}
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: words, Shown: body}
}

// keptTurn is the shelf, drawn in the conversation.
//
// One way off it, and it is the way back. The shelf exists because a kept note
// had nowhere to be read back; a kept note was never going to be done, so DONE
// here would answer a question nobody asked, and KEEP would be a button that
// does nothing. Putting it back in the pile is where every other decision gets
// made, and this is how you change your mind.
func keptTurn(ctx context.Context, s Store, personID int64, name string) squirrel.Turn {
	items, more, err := s.KeptItems(ctx, personID, listLimit)
	if err != nil {
		slog.Error("reading the shelf", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "I cannot reach the shelf just now."}
	}
	if len(items) == 0 {
		// Absence, not encouragement. An empty shelf is a normal state and
		// not a failure to set one up.
		return squirrel.Turn{
			Who:   squirrel.SpeakerBuddy,
			Words: "Nothing on the shelf yet. KEEP puts a note here instead of ending it.",
		}
	}

	sh := drawn{Place: name}
	for _, it := range items {
		v := toView(it)
		row := map[string]string{
			"id": strconv.FormatInt(v.ID, 10), "was": v.State, "from": "thread",
		}
		sh.Cards = append(sh.Cards, cardView{
			Title: v.Text, Photo: v.Photo, Meta: v.When,
			Acts: []actView{
				{Label: "back in the pile", Action: "/pile/act", Style: "go",
					Fields: with(row, "act", "open")},
			},
		})
	}

	words := "The things you kept."
	if more {
		// The same device every list here uses. It says there is more and
		// never how much more: the number above you is not a thing you can act
		// on, and a shelf with a count on it is a shelf you are behind on.
		words = "The things you kept, and there are more."
	}
	body, err := json.Marshal(sh)
	if err != nil {
		slog.Error("drawing the shelf", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: words}
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: words, Shown: body}
}

// heldTurn is what you cannot act on, drawn in the conversation.
//
// One list rather than the page's three groups. The page could afford headings;
// a turn cannot, because a heading with one row under it is a heading that
// looks like a mistake. Each card says which of the three it is and what would
// move it, which is the same information the grouping carried.
//
// No count anywhere. A number beside stalled work is a reproach, and the point
// of setting it aside was to stop being asked about it.
func heldTurn(ctx context.Context, s Store, personID int64, name string) squirrel.Turn {
	held, more, err := s.HeldItems(ctx, personID, listLimit)
	if err != nil {
		slog.Error("reading what is set aside", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "I cannot reach those just now."}
	}
	if len(held) == 0 {
		return squirrel.Turn{
			Who:   squirrel.SpeakerBuddy,
			Words: "Nothing set aside. When a note is waiting on somebody else, it comes here.",
		}
	}

	sh := drawn{Place: name}
	for _, h := range inTheCoresOrder(held) {
		photo := ""
		if h.PhotoName != "" {
			// By the row's id, never by the file's name: the name is the one
			// string in this product that becomes a path.
			photo = "/photo/" + strconv.FormatInt(h.ID, 10)
		}
		sh.Cards = append(sh.Cards, cardView{
			// Recessed and dashed rather than raised: something set aside is
			// present, and is not a thing you can pick up. It is the one body
			// that bends "cream card stock, never white" by not being stock.
			Kind: "held", Title: h.Text, Photo: photo, Meta: heldMeta(h),
			Acts: []actView{
				// Picking it back up is `open`, which is the transition
				// everything else in this product reverses through — so undo
				// works on it without anything new, and the pile it returns to
				// is where it came from.
				{Label: "pick it back up", Action: "/held/act", Style: "go",
					Fields: map[string]string{
						"id": strconv.FormatInt(h.ID, 10), "act": "back", "from": "thread",
					}},
			},
		})
	}

	words := "What you set aside."
	if more {
		words = "What you set aside, and there are more."
	}
	body, err := json.Marshal(sh)
	if err != nil {
		slog.Error("drawing what is set aside", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: words}
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: words, Shown: body}
}

// inTheCoresOrder puts the two that something outside you will end before the
// one only you can. The page carried that order in its headings; without it a
// single list would be in whatever order the query returned, and someday would
// sit among things that are actually moving.
func inTheCoresOrder(held []squirrel.HeldItem) []squirrel.HeldItem {
	out := make([]squirrel.HeldItem, 0, len(held))
	for _, state := range squirrel.Held {
		for _, h := range held {
			if h.State == state {
				out = append(out, h)
			}
		}
	}
	return out
}

// heldMeta is which of the three, and what would move it. The reason is often
// empty — someday is not waiting on anything — and then the card says only
// which kind it is rather than trailing a dash into nothing.
func heldMeta(h squirrel.HeldItem) string {
	word := strings.ToUpper(squirrel.HeldWords[h.State])
	if strings.TrimSpace(h.Because) == "" {
		return word
	}
	return word + " — " + h.Because
}

// shelfHandler is a shelf, asked for by name from inside the notes.
//
// One handler for both, because the only thing that differs is which turn is
// drawn — and two copies of the same eight lines is two places for one of them
// to drift. The same reasoning askNameHandler is written under.
//
// It is a press and it is kept, which is the difference between this and
// turning a calendar's month: asking to see what you set aside is something
// you did, and the answer is a thing you can act on.
func shelfHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil || !shelfByKey(r.FormValue("shelf")) {
			http.Redirect(w, r, backToTheRoom(r), http.StatusSeeOther)
			return
		}
		ctx := r.Context()
		said := placeTurn(ctx, s, opts, personID, r.FormValue("shelf"), 0)
		answerWith(w, r, keepSaid(ctx, s, personID, said), backToTheRoom(r))
	}
}
