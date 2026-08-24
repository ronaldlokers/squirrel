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

// The thread: the whole of the screen.
//
// This replaced home on 24 August 2026. Home's argument was that a front door
// showing what is waiting greets you with what is waiting; the owner retired
// that along with Principle 2, and the doors carry numbers now. What survives
// unchanged is that the doors are equals — one grid, four cells, the same stock
// — and that the slot is the way in.
//
// Only the newest Buddy turn carries controls. A card from this morning keeps
// its words and loses its buttons, because pressing DID IT on a card from a
// conversation three days old acts on a state nobody is looking at. See The
// live edge in docs/superpowers/specs/2026-08-24-the-thread-design.md.

// threadLimit is how much of the conversation one render holds. A bound rather
// than a page: everything above it is still there and one press away.
const threadLimit = 40

// shown is what a turn drew, as it was drawn.
//
// Decoded from the turn's own JSON and never re-read from another table. A turn
// holding a chore id would show today's chore inside yesterday's sentence,
// which is what "history is never rewritten" forbids.
type shown struct {
	Place string     `json:"place,omitempty"`
	Cards []cardView `json:"cards,omitempty"`
	Chips []turnChip `json:"chips,omitempty"`
}

type turnView struct {
	ID    int64
	Buddy bool
	Words string
	// Place is the <h2> when this turn opens one, and empty otherwise. The
	// thread has no <h1> — home's exemption, because nobody arrives at the
	// place they started wondering where they are — so these are what heading
	// navigation walks.
	Place string
	Cards []cardView
	Chips []turnChip
	// Live is the newest Buddy turn and nothing else.
	Live bool
}

type cardView struct {
	Title string    `json:"title"`
	Meta  string    `json:"meta,omitempty"`
	Acts  []actView `json:"acts,omitempty"`
}

// actView is one button on a card.
//
// Fields is a map rather than one name-and-value pair because the presses this
// has to carry are not all one field wide: /now/act wants kind, id and act
// together. A struct that can hold only one hidden input would be wrong again
// the first time a second one is needed.
type actView struct {
	Label  string            `json:"label"`
	Action string            `json:"action"`
	Fields map[string]string `json:"fields,omitempty"`
	Style  string            `json:"style,omitempty"`
}

// turnChip is a choice offered in the conversation, as a link.
//
// Not chipView: that is the pile's three reasons for setting something aside,
// and one type meaning two things is how a template ends up rendering the
// wrong one.
type turnChip struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

type doorView struct {
	Href  string
	Label string
	Art   string
	// Count is what is waiting behind the door. Zero renders no number at
	// all — a door reading "0" is a scoreboard, and that is what the retired
	// rule was actually protecting against.
	Count int
	Here  bool
}

func threadHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		ctx := r.Context()

		var (
			turns []squirrel.Turn
			more  bool
			err   error
		)
		// `?before=` walks up the conversation. It is in the address bar
		// rather than in a cursor because a page of the past is a place you
		// can send yourself back to.
		before, perr := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
		walkingBack := perr == nil && before > 0
		if walkingBack {
			turns, more, err = s.TurnsBefore(ctx, personID, before, threadLimit)
		} else {
			turns, more, err = s.RecentTurns(ctx, personID, threadLimit)
		}
		if err != nil {
			fail(w, err)
			return
		}

		v := view{
			Home:      true,
			Here:      "thread",
			Scrolling: true,
			Turns:     turnViews(turns),
			Rail:      railFor(ctx, s, personID, ""),
			MoreAbove: more,
		}
		if len(turns) > 0 {
			v.Oldest = turns[0].ID
		}
		renderWith(w, r, s, opts, "thread", v)
	}
}

// turnViews decodes each turn's own record of what it drew, and marks the live
// edge.
//
// The scan for the live edge runs backwards and stops at the first Buddy turn,
// so a run of your own turns at the bottom does not leave the conversation with
// nothing to press.
func turnViews(turns []squirrel.Turn) []turnView {
	out := make([]turnView, 0, len(turns))
	for _, t := range turns {
		v := turnView{ID: t.ID, Buddy: t.Who == squirrel.SpeakerBuddy, Words: t.Words}
		if len(t.Shown) > 0 {
			var sh shown
			if err := json.Unmarshal(t.Shown, &sh); err != nil {
				// A turn whose record cannot be read still said something, and
				// the words are the part that matters. Losing the cards is
				// better than losing the turn.
				slog.Error("reading what a turn drew", "turn", t.ID, "error", err)
			} else {
				v.Place, v.Cards, v.Chips = sh.Place, sh.Cards, sh.Chips
			}
		}
		out = append(out, v)
	}
	for i := len(out) - 1; i >= 0; i-- {
		if out[i].Buddy {
			out[i].Live = true
			break
		}
	}
	return out
}

// railFor is the four doors, with what is waiting behind each.
//
// A failed count is four doors and no numbers rather than an error page: the
// doors are how you get anywhere, and a database that cannot count is not a
// reason to take the navigation away.
func railFor(ctx context.Context, s Store, personID int64, here string) []doorView {
	rail := []doorView{
		{Href: "/pile", Label: "the pile", Art: "door-pile.png"},
		{Href: "/tasks", Label: "the tasks", Art: "door-tasks.png"},
		{Href: "/chores", Label: "the chores", Art: "door-chores.png"},
		{Href: "/at", Label: "the agenda", Art: "door-at.png"},
	}
	for i := range rail {
		rail[i].Here = rail[i].Label == here
	}
	waiting, err := s.Waiting(ctx, personID, now())
	if err != nil {
		slog.Error("counting what is waiting", "error", err)
		return rail
	}
	rail[0].Count = waiting.Pile
	rail[1].Count = waiting.Tasks
	rail[2].Count = waiting.Chores
	rail[3].Count = waiting.Agenda
	return rail
}

// threadSayHandler is the dock: one line in, two turns out.
//
// The words go to the spool and not to the pile directly, exactly as /capture
// does. The spool is the durability promise — it is what survives the database
// being unreachable — and a dock that wrote straight to Postgres would be a
// second capture path with weaker guarantees than the first.
//
// The spool is written first and the turns after. If the process dies between
// them the thought is safe and the conversation is missing a line, which is
// recoverable; the other order loses the thought, and losing thoughts is the
// one failure this product exists to prevent.
func threadSayHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		ctx := r.Context()
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		text := strings.TrimSpace(r.FormValue("words"))
		if text == "" {
			// An empty slot is not a turn. A blank bubble in a record that is
			// never rewritten is a blank bubble forever.
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if len(text) > captureLimit {
			text = text[:captureLimit]
		}

		// Whose it is, said in the transport's own vocabulary rather than as a
		// person id: the drain resolves every capture's owner from its sender,
		// and this one is no different for having been typed in the dock.
		sender := opts.Identity
		_, kept := opts.Spool.Write(squirrel.Capture{
			Transport:  squirrel.ScreenTransport,
			SenderID:   &sender,
			Text:       text,
			Payload:    []byte(squirrel.ScreenCapture),
			ReceivedAt: now(),
		})

		// Your words go into the record either way. The slot's old promise was
		// that a failed keep hands the text back to the box; the thread keeps
		// the same promise by keeping the turn.
		said := []squirrel.Turn{{Who: squirrel.SpeakerYou, Words: text}}

		// What Buddy says back, and it must never claim more than happened.
		reply := "Kept."
		if kept != nil {
			slog.Warn("a capture from the dock could not be spooled", "error", kept)
			reply = "Not kept — Squirrel cannot reach its memory. Your words are still here; try again in a moment."
		}
		said = append(said, squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: reply})

		for _, t := range said {
			if _, err := s.AppendTurn(ctx, personID, t); err != nil {
				slog.Error("keeping what was said", "error", err)
			}
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
