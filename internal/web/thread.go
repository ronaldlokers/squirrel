package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

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

// drawn is what a turn drew, as it was drawn.
//
// Decoded from the turn's own JSON and never re-read from another table. A turn
// holding a chore id would show today's chore inside yesterday's sentence,
// which is what "history is never rewritten" forbids.
type drawn struct {
	Place string     `json:"place,omitempty"`
	Cards []cardView `json:"cards,omitempty"`
	Chips []turnChip `json:"chips,omitempty"`
	// Pick is a question with its answers on it: rows of choices in one form
	// with one submit. See askHowOften for why it is one form.
	Pick *pickView `json:"pick,omitempty"`
	// Cal is the day picker. See askForADay.
	Cal *calView `json:"cal,omitempty"`
	// Faces is the check-in's five drawings. A flag rather than five chips
	// because they are the product's own faces and the markup for them
	// already exists; rendering them as words would be a different control
	// wearing the same question.
	Faces bool `json:"faces,omitempty"`
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
	Faces []faceView
	Pick  *pickView
	Cal   *calView
	// V stamps the asset URLs a turn draws. Filled here rather than by render,
	// because a turn is also rendered on its own as a fragment, where there is
	// no page around it to carry it.
	V string
	// Live is the newest Buddy turn and nothing else.
	Live bool
}

type cardView struct {
	// Kind is what sort of thing the card holds, or empty for an ordinary one.
	//
	// "chore" makes it render as `article.chore` — the same element the chores
	// screen used — so pile.js's chore keys and the stylesheet's chore rules
	// keep working on it. The screen went; the thing on it did not.
	Kind  string `json:"kind,omitempty"`
	Title string `json:"title"`
	Meta  string `json:"meta,omitempty"`
	// Photo is the note's own picture, or empty. A note with no words at all
	// is a perfectly good note, and a task made from one would otherwise be a
	// card saying nothing.
	Photo string `json:"photo,omitempty"`
	// Take is what to bring, on its own line: it is the thing you are standing
	// in the hall without, so it is a line rather than a tail on the meta.
	Take string    `json:"take,omitempty"`
	Acts []actView `json:"acts,omitempty"`
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
	// Href for somewhere to go; Action and Fields for something to do. A chip
	// that acts is a form, because a GET that writes writes again on every
	// reload — the same reason a door is a press.
	Href   string            `json:"href,omitempty"`
	Action string            `json:"action,omitempty"`
	Fields map[string]string `json:"fields,omitempty"`
}

type doorView struct {
	// Where is the door's own word, posted to /open. Not an href: a door is
	// pressed rather than followed, and a field that could be used as one
	// would invite exactly that.
	Where string
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
		// A record that cannot be read is not a reason to take the screen
		// away: the doors still work, and the dock still writes to the spool,
		// which is the whole of what an unreachable database must not stop.
		// Said out loud rather than rendered as an empty conversation, because
		// an empty conversation looks like your history is gone.
		unreadable := err != nil
		if unreadable {
			slog.Error("reading the conversation", "error", err)
			turns, more = nil, false
		}

		// The one thing Buddy opens with, and only while the last answer no
		// longer describes now. Written rather than rendered, because a
		// question that is not in the record is a question the record cannot
		// show you answering.
		//
		// Never while walking back: a page of the past is being read, and
		// reading it must not add to it.
		asked := false
		if !walkingBack && !unreadable {
			if t, ask := checkinTurn(ctx, s, personID, r.URL.Query().Get("ask") != ""); ask {
				asked = true
				if saved, err := s.AppendTurn(ctx, personID, t); err == nil {
					turns = append(turns, saved)
				} else {
					slog.Error("asking how you are", "error", err)
				}
			}
		}

		// Only once the question has been answered. Asking how you are and
		// then handing you a job in the same breath is the interruption this
		// product exists to reduce — and the answer is what shapes the offer
		// anyway.
		if !walkingBack && !asked && !unreadable && !endsOpen(turns) {
			if t, has := offerTurn(s, opts, r); has {
				if saved, err := s.AppendTurn(ctx, personID, t); err == nil {
					turns = append(turns, saved)
				} else {
					slog.Error("offering it", "error", err)
				}
			}
		}

		v := view{
			Home: true,
			// The worker having taken the words because there was no network.
			// From the address bar, read the way a stranger's typing is read:
			// a present flag and nothing else.
			Held:      r.URL.Query().Get("held") != "",
			Here:      "thread",
			Scrolling: true,
			Turns:     turnViews(turns),
			Rail:      railFor(ctx, s, personID, ""),
			MoreAbove: more,
		}
		if len(turns) > 0 {
			v.Oldest = turns[0].ID
		}
		if unreadable {
			v.Turns = []turnView{{
				Buddy: true, Live: true, V: assetVersion,
				Words: "I cannot reach what we said. Tell me things anyway — they are kept, and they go in when I can.",
			}}
		}
		renderWith(w, r, s, opts, "thread", v)
	}
}

// endsOpen says the conversation already ends with something to act on.
//
// Without it the offer was appended on every single load: a reload put a second
// copy in the record, and — worse — it stole the live edge from whatever Buddy
// had just said, so pressing "too big" got you the way through with its timer
// button already taken off it.
//
// The question it asks is deliberately about shape rather than about which
// offer: anything Buddy has put on the table and you have not answered is a
// reason not to put something else there.
func endsOpen(turns []squirrel.Turn) bool {
	if len(turns) == 0 {
		return false
	}
	last := turns[len(turns)-1]
	if last.Who != squirrel.SpeakerBuddy || len(last.Shown) == 0 {
		return false
	}
	var sh drawn
	if err := json.Unmarshal(last.Shown, &sh); err != nil {
		// A turn whose record cannot be read is treated as open. Saying
		// nothing more is the safe direction: the other one talks over it.
		return true
	}
	return len(sh.Cards) > 0 || len(sh.Chips) > 0 || sh.Faces || sh.Pick != nil || sh.Cal != nil
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
		v := turnView{ID: t.ID, Buddy: t.Who == squirrel.SpeakerBuddy, Words: t.Words, V: assetVersion}
		if len(t.Shown) > 0 {
			var sh drawn
			if err := json.Unmarshal(t.Shown, &sh); err != nil {
				// A turn whose record cannot be read still said something, and
				// the words are the part that matters. Losing the cards is
				// better than losing the turn.
				slog.Error("reading what a turn drew", "turn", t.ID, "error", err)
			} else {
				v.Place, v.Cards, v.Chips, v.Pick, v.Cal = sh.Place, sh.Cards, sh.Chips, sh.Pick, sh.Cal
				if sh.Faces {
					v.Faces = theFaces()
				}
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
		{Where: "pile", Label: "the pile", Art: "door-pile.png"},
		{Where: "tasks", Label: "the tasks", Art: "door-tasks.png"},
		{Where: "chores", Label: "the chores", Art: "door-chores.png"},
		{Where: "at", Label: "the agenda", Art: "door-at.png"},
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

// theFaces is the five, in the one order both surfaces use.
func theFaces() []faceView {
	out := make([]faceView, 0, len(squirrel.Moods))
	for _, m := range squirrel.Moods {
		out = append(out, faceView{Mood: string(m), Word: squirrel.Words[m]})
	}
	return out
}

// checkinTurn is Buddy's question, while the last answer no longer describes
// now.
//
// It is a turn rather than a region, and that is the change: the answer used to
// replace the question and the morning was gone. Both stay now, which is what a
// record that is never rewritten buys.
//
// How is right now — right now, and not today. That rule is unchanged; what
// changed is only where the question lives.
func checkinTurn(ctx context.Context, s Store, personID int64, again bool) (squirrel.Turn, bool) {
	c, found, err := s.LatestCheckin(ctx, personID)
	if err != nil {
		slog.Error("reading how you are", "error", err)
		return squirrel.Turn{}, false
	}
	if found && c.Fresh(now()) && !again {
		return squirrel.Turn{}, false
	}
	body, err := json.Marshal(drawn{Faces: true})
	if err != nil {
		slog.Error("drawing the faces", "error", err)
		return squirrel.Turn{}, false
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "how do you feel?", Shown: body}, true
}

// threadMoodHandler is the answer to Buddy's question.
//
// The reading is recorded first and the turns after: an answer that reached the
// conversation but not the readings would make /moods disagree with the thread,
// and the readings are what the picker actually consults.
func threadMoodHandler(s Store, opts Options) http.HandlerFunc {
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
		m, ok := squirrel.ParseMood(r.FormValue("mood"))
		if !ok {
			// Not one of the five. This arrives from a form, so it is read the
			// way a stranger's typing is read: no answer rather than a wrong
			// one, and nothing said about it in the record.
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if err := s.RecordCheckin(ctx, personID, m, "screen", now()); err != nil {
			fail(w, err)
			return
		}
		// "Noted" is the word this screen has always answered with, and it is
		// the whole of what it may say: this reports what you said and may
		// never characterise you.
		//
		// The way to change your mind travels with it, because the answer is
		// about to be scrollback and scrollback carries no controls. Changing
		// your mind is not a special case; it is the same answer given twice.
		// "how you felt before" and not "what you said before": the pile's own
		// door is what you said, and the same words naming two different
		// things on one screen is what the rename fixed.
		again, err := json.Marshal(drawn{Chips: []turnChip{
			{Label: "say something else", Href: "/?ask=1"},
			{Label: "how you felt before", Href: "/moods"},
		}})
		if err != nil {
			slog.Error("drawing the way back", "error", err)
		}
		answerWith(w, r, keepSaid(ctx, s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: squirrel.Words[m]},
			{Who: squirrel.SpeakerBuddy, Words: "Noted.", Shown: again},
		}), "/")
	}
}

// offerTurn is the one thing Squirrel picked, or nothing.
//
// Nothing renders nothing: being handed nothing is a normal state, and a
// reassuring sentence in its place would be the product deciding you ought to
// be busy. That was home's rule and it is unchanged by the move.
func offerTurn(s Store, opts Options, r *http.Request) (squirrel.Turn, bool) {
	// "Show me anyway" lifts the capacity gate for this render and nothing
	// else. It lives in the address bar rather than in the database on
	// purpose: a person who says they are wiped and then works anyway has not
	// changed their answer, and storing it would make them re-decide tomorrow
	// that they meant it.
	anyway := r.URL.Query().Get("anyway") != ""
	o := offerFor(s, opts, r, anyway, true)
	if o == nil {
		// Nothing, or something being withheld because of how you said you
		// were. Those are different and only the second has anything to say:
		// having nothing to be handed is a normal state, and a reassuring
		// sentence in its place would be the product deciding you ought to be
		// busy.
		if !anyway && offerFor(s, opts, r, true, false) != nil {
			return wayThrough()
		}
		return squirrel.Turn{}, false
	}

	card := cardView{Title: o.Text, Meta: o.Because}
	// A running timer carries no buttons of its own: it is a thing you are
	// doing rather than a row that was picked, and the lid already has the one
	// control it needs, which is the way to stop.
	if !o.Running {
		row := map[string]string{"kind": o.Kind, "id": strconv.FormatInt(o.RefID, 10), "label": o.Text}
		did := "DID IT"
		if o.Kind == string(squirrel.OfferMoment) {
			// A fixed point is not done, it is left for. The word the card has
			// always used, kept.
			did = "LEAVING"
		}
		switch o.Kind {
		case string(squirrel.OfferAgain):
			// A breadcrumb is not a job. It is the thing you were on a little
			// while ago, and the only two answers are picking it back up and
			// not — there is nothing here to have done.
			card.Acts = []actView{
				{Label: "PICK IT UP", Action: "/now/act", Style: "go", Fields: with(with(row, "act", "start"), "minutes", "10")},
				{Label: "not now", Action: "/now/act", Style: "later", Fields: with(row, "act", "later")},
			}
		default:
			card.Acts = []actView{
				{Label: did, Action: "/now/act", Style: "did", Fields: with(row, "act", "did")},
				{Label: "10 MIN", Action: "/now/act", Style: "go", Fields: with(with(row, "act", "start"), "minutes", "10")},
				{Label: "not now", Action: "/now/act", Style: "later", Fields: with(row, "act", "later")},
			}
		}
		// The four ways of not being able to start. They are on the card
		// rather than behind a disclosure because the card is about to be
		// scrollback, and scrollback carries no controls — a ladder you can
		// only reach by pressing something that has already gone is a ladder
		// nobody reaches.
		for _, b := range blockersFor(o.Kind) {
			card.Acts = append(card.Acts, actView{
				Label: squirrel.BlockerWords[b], Action: "/now/stuck", Style: "why",
				Fields: with(row, "why", string(b)),
			})
		}
	}

	body, err := json.Marshal(drawn{Cards: []cardView{card}})
	if err != nil {
		slog.Error("drawing the offer", "error", err)
		return squirrel.Turn{}, false
	}
	return squirrel.Turn{
		Who: squirrel.SpeakerBuddy, Words: squirrel.Say(squirrel.SayingOffer, now()), Shown: body,
	}, true
}

// with is one field added to a row's own fields, copied rather than mutated —
// three buttons share the row and each needs a different act.
func with(fields map[string]string, name, value string) map[string]string {
	out := make(map[string]string, len(fields)+1)
	for k, v := range fields {
		out[k] = v
	}
	out[name] = value
	return out
}

// saidAboutTheOffer is what the two of you said when a press landed.
//
// Turning it down is in the record beside doing it, and they do not read the
// same: which answer you gave is the whole of what happened, and stopping
// partway is a normal ending rather than an absence.
func saidAboutTheOffer(act, label string) []squirrel.Turn {
	if label == "" {
		label = "that"
	}
	switch act {
	case "did":
		return []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "did it — " + label},
			{Who: squirrel.SpeakerBuddy, Words: "Good."},
		}
	case "later":
		return []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "not now — " + label},
			{Who: squirrel.SpeakerBuddy, Words: "Fine. It will come back."},
		}
	case "start":
		return []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "picked it up — " + label},
			{Who: squirrel.SpeakerBuddy, Words: "Ten minutes. Stop whenever."},
		}
	}
	return nil
}

// keepSaid writes what was said, and says so in the log when it cannot.
//
// A press that changed the pile and failed to reach the conversation is a
// conversation with a hole in it, which is recoverable; refusing the press
// because the record could not be written would not be.
func keepSaid(ctx context.Context, s Store, personID int64, said []squirrel.Turn) []squirrel.Turn {
	out := make([]squirrel.Turn, 0, len(said))
	for _, t := range said {
		saved, err := s.AppendTurn(ctx, personID, t)
		if err != nil {
			slog.Error("keeping what was said", "error", err)
			continue
		}
		out = append(out, saved)
	}
	return out
}

// saidAboutBeingStuck is the ladder, as two turns.
//
// It used to be a region under the offer, reached back through the address
// bar. It is in the record now for the same reason everything else is: the
// thing you could not start, and what was said about it, are part of the
// conversation rather than a state the next render happens to be in.
//
// The line stays whatever the core says it is. One sentence, lower case, no
// exclamation — and the step, when there is one, sits above it rather than
// replacing it: the fixed answer is the floor, not a draft.
func saidAboutBeingStuck(s Store, opts Options, r *http.Request, personID int64, b squirrel.Blocker) []squirrel.Turn {
	u := squirrel.UnstuckFor(b)
	if u.Refuse {
		return nil
	}

	card := cardView{Title: u.Line}
	if step := stepFor(s, opts, r); step != nil {
		// One step, and there is deliberately nowhere here to put a second:
		// the failure being avoided is the twelve-step plan.
		card.Title = step.Body
		card.Meta = u.Line
	}
	if u.Minutes > 0 {
		card.Acts = []actView{{
			Label: strconv.Itoa(u.Minutes) + " MIN", Action: "/timer", Style: "go",
			Fields: map[string]string{"minutes": strconv.Itoa(u.Minutes), "label": r.FormValue("label")},
		}}
	}

	body, err := json.Marshal(drawn{Cards: []cardView{card}})
	if err != nil {
		slog.Error("drawing the way through", "error", err)
		return nil
	}
	words := "Right."
	if u.Ask {
		// The one branch that captures: "what would I have to find out first"
		// is a thought, and thoughts go in the pile. The dock is always there
		// now, so there is nothing to offer — only something to ask.
		words = "What would you have to find out first? Tell me and I will keep it."
	}
	return []squirrel.Turn{
		{Who: squirrel.SpeakerYou, Words: squirrel.BlockerWords[b]},
		{Who: squirrel.SpeakerBuddy, Words: words, Shown: body},
	}
}

// blockersFor is the four ways of not being able to start, on the things they
// are about.
//
// Not on a breadcrumb: "I can't start" about a thing you were already doing is
// a question about something else, and the answer to it is the breadcrumb's own
// "not now".
func blockersFor(kind string) []squirrel.Blocker {
	if kind == string(squirrel.OfferAgain) {
		return nil
	}
	return squirrel.Blockers
}

// wayThrough is what a low day gets instead of a job.
//
// The gate is real and it is kept: nothing is handed to you. What is offered is
// the way past it, once, in your own hands — because a person who says they are
// wiped and then decides to work anyway has not changed their answer.
func wayThrough() (squirrel.Turn, bool) {
	body, err := json.Marshal(drawn{Chips: []turnChip{
		{Label: "show me something anyway", Href: "/?anyway=1"},
	}})
	if err != nil {
		slog.Error("drawing the way through", "error", err)
		return squirrel.Turn{}, false
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "Nothing from me today.", Shown: body}, true
}

// wantsFragment is a press made by the script rather than by the browser's own
// form machinery.
//
// A header rather than a second route: one URL per action, one handler, one
// write. A `/capture/fragment` twin would be a second place the write can drift
// from the first.
func wantsFragment(r *http.Request) bool { return r.Header.Get("X-Thread") == "fragment" }

// answerWith is what a press gets back.
//
// The same HTML the page would have rendered for those turns, from the same
// templates. There is no JSON here and no client-side template; a second
// description of a card is how the two ends grow apart.
//
// The live edge moves to the last turn that came back, so the controls follow
// the conversation rather than staying where the page was painted.
func answerWith(w http.ResponseWriter, r *http.Request, said []squirrel.Turn, back string) {
	if !wantsFragment(r) {
		http.Redirect(w, r, back, http.StatusSeeOther)
		return
	}
	vs := turnViews(said)
	for i := range vs {
		vs[i].Live = false
	}
	for i := len(vs) - 1; i >= 0; i-- {
		if vs[i].Buddy {
			vs[i].Live = true
			break
		}
	}

	t := pages["thread"]
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	for _, v := range vs {
		if err := t.ExecuteTemplate(w, "turn", v); err != nil {
			slog.Error("drawing a turn", "turn", v.ID, "error", err)
			return
		}
	}
}

// listLimit is how many cards one turn draws.
//
// A bound rather than a page, and it matters more here than anywhere else: a
// turn is frozen the moment it is written, so a turn holding forty cards is
// forty cards in the record forever.
const listLimit = 12

// doorNames is the vocabulary, as a map rather than a switch so an unknown door
// is a lookup miss instead of a default branch someone later fills in with
// something destructive. The same device the offer's kinds use.
var doorNames = map[string]string{
	"pile": "the pile", "tasks": "the tasks", "chores": "the chores", "at": "the agenda",
}

// openHandler is a door being pressed.
//
// A POST, and not a link, because opening a place is an utterance: it goes into
// the record like anything else you say. A GET that wrote to the record would
// write again on every reload and on every walk back through the past.
//
// What it costs, stated rather than discovered: a door cannot be opened in a
// new tab, and the back button does not step through doors. That is the
// ordinary trade for one page, and it is the only thing the rail gave up.
func openHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		said := placeTurn(r.Context(), s, personID, r.FormValue("where"))
		if len(said) == 0 {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		answerWith(w, r, keepSaid(r.Context(), s, personID, said), "/")
	}
}

// placeTurn is what you said and what Buddy answered, or nothing at all.
func placeTurn(ctx context.Context, s Store, personID int64, where string) []squirrel.Turn {
	name, ok := doorNames[where]
	if !ok {
		return nil
	}
	var reply squirrel.Turn
	switch where {
	case "chores":
		reply = choresTurn(ctx, s, personID, name)
	case "tasks":
		reply = tasksTurn(ctx, s, personID, name)
	case "at":
		reply = agendaTurn(ctx, s, personID, name)
	case "pile":
		reply = pileTurn(ctx, s, personID, 0, name)
	default:
		// The pile and the agenda are phase 3. Until then the doors that are
		// not built say so rather than answering with silence, which reads as
		// a press that did not land.
		reply = squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "Not yet — that one is still a page."}
	}
	return []squirrel.Turn{{Who: squirrel.SpeakerYou, Words: name}, reply}
}

// choresTurn is what comes back, as cards.
func choresTurn(ctx context.Context, s Store, personID int64, name string) squirrel.Turn {
	chores, err := s.ActiveChores(ctx, personID)
	if err != nil {
		slog.Error("reading what comes back", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "I cannot reach the chores just now."}
	}
	if len(chores) == 0 {
		// A fact, not a nudge. It says where chores come from and offers the
		// other way of making one — the same words the empty state used.
		body, err := json.Marshal(drawn{Place: name})
		if err != nil {
			slog.Error("drawing the chores", "error", err)
		}
		return squirrel.Turn{
			Who:   squirrel.SpeakerBuddy,
			Words: "Nothing comes back on its own. When a note becomes a chore, it lives here. " + makeOne,
			Shown: body,
		}
	}

	more := false
	if len(chores) > listLimit {
		chores, more = chores[:listLimit], true
	}
	sh := drawn{Place: name}
	for _, c := range chores {
		sh.Cards = append(sh.Cards, choreCard(toChoreView(c)))
	}
	if more {
		sh.Chips = []turnChip{{Label: "the rest", Href: "/?open=chores"}}
	}

	body, err := json.Marshal(sh)
	if err != nil {
		slog.Error("drawing the chores", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "I cannot draw the chores just now."}
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: choreLead(len(sh.Cards)), Shown: body}
}

// makeOne is how you make one from nothing.
//
// The new-chore form went with the screen, and what replaced it is the
// sentence: the dock already understands this, and saying so is where you find
// that out. A guided version is a multi-turn flow with state to keep, which is
// a bigger thing than the interval picker and not what was asked for.
//
// It is a sentence rather than a chip because a chip would have to lead
// somewhere, and there is nowhere for it to lead that the dock does not already
// answer better.
//
// Said only when there is nothing there. Telling you how to make one every time
// you look at the chores you already keep is nagging, and the empty list is
// exactly when it is worth knowing.
const makeOne = "Tell me another like every 2 weeks: descale the kettle."

// choreCard is one chore, drawn the one way.
//
// Written once and used by both the list and the reply to making a new one: a
// chore read back out of the store and a chore made from nothing must not look
// different, which is the sort of difference nobody notices until one of them
// grows a button the other has not.
func choreCard(v choreView) cardView {
	row := map[string]string{"id": strconv.FormatInt(v.ID, 10), "label": v.Name}
	return cardView{
		Kind: "chore", Title: v.Name, Meta: choreMeta(v),
		Acts: []actView{
			{Label: "DID IT", Action: "/chores/act", Style: "did", Fields: with(row, "act", "done")},
			{Label: "HOW OFTEN", Action: "/chores/often", Style: "go", Fields: row},
			{Label: "STOP ASKING", Action: "/chores/act", Style: "stop", Fields: with(row, "act", "retire")},
		},
	}
}

// choreMeta is the rhythm, and what has happened, on the card's own line.
//
// What has not happened is not reported: a chore nobody has ever done shows its
// rhythm and stops there.
func choreMeta(v choreView) string {
	out := v.Every
	if v.Last != "" {
		out += " · LAST DONE " + v.Last
	}
	if v.When != "" {
		out += " · " + v.When
	}
	return out
}

// choreLead is Buddy counting, which he is allowed to do: Principle 5 permitted
// it in speech on 20 August 2026 and Principle 2's retirement permitted it
// everywhere else on the 24th.
func choreLead(n int) string {
	if n == 1 {
		return "One comes back."
	}
	return fmt.Sprintf("%d come back.", n)
}

// saidAboutAChore is what the two of you said about one.
//
// Its own function rather than the offer's, because the answers are different
// facts: an offer is a thing you were handed, and a chore is a thing that comes
// back whatever you do about it. "Stop asking" especially — it is the one press
// here that ends something, and it must not read like finishing it.
//
// The name comes from the stored chore rather than from the form, so what the
// record says happened cannot be something the press claimed.
func saidAboutAChore(act, name string) []squirrel.Turn {
	switch act {
	case "done":
		return []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "did it — " + name},
			{Who: squirrel.SpeakerBuddy, Words: "Good. It will come back."},
		}
	case "retire":
		return []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "stop asking — " + name},
			{Who: squirrel.SpeakerBuddy, Words: "It will not come back. Tell me if you want it again."},
		}
	}
	return nil
}

// madeAChore is a chore you made from nothing, drawn the way the list draws
// one — the same choreCard, so the two cannot look different.
func madeAChore(c squirrel.Chore) squirrel.Turn {
	v := toChoreView(c)
	body, err := json.Marshal(drawn{Cards: []cardView{choreCard(v)}})
	if err != nil {
		slog.Error("drawing a new chore", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "Kept. It comes back " + v.Every + "."}
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "Kept.", Shown: body}
}

// pickView is a question with its answers on it.
//
// One form and one submit, deliberately: a picker that wrote a turn every time
// you pressed a number would fill a record that is never rewritten with the
// sound of somebody deciding. You are asked once and you answer once.
type pickView struct {
	Action string            `json:"action"`
	Fields map[string]string `json:"fields,omitempty"`
	Rows   []pickRow         `json:"rows"`
	Do     string            `json:"do"`
}

type pickRow struct {
	Lead    string   `json:"lead"`
	Name    string   `json:"name"`
	Options []string `json:"options"`
	// Chosen is what is true now, or empty. Empty rather than a nearest guess:
	// marking the wrong one would say the thing is something it is not.
	Chosen string `json:"chosen,omitempty"`
}

// pickNumbers and pickUnits are what the interval picker offers.
//
// Six numbers and three units, and no way to type one: six covers what anyone
// reaches for, and `every 9 weeks` is a sentence rather than a control.
// ParseEvery accepts fortnights, quarters and years too, and those stay
// available through the sentence at no cost in buttons.
var (
	pickNumbers = []string{"1", "2", "3", "4", "6", "8"}
	pickUnits   = []string{"days", "weeks", "months"}
)

// askHowOften is the question, as one form with two rows.
func askHowOften(id int64, count, unit string) squirrel.Turn {
	body, err := json.Marshal(drawn{Pick: &pickView{
		Action: "/chores/act",
		Fields: map[string]string{"id": strconv.FormatInt(id, 10)},
		Do:     "that's it",
		Rows: []pickRow{
			{Lead: "every", Name: "count", Options: pickNumbers, Chosen: count},
			{Lead: "of these", Name: "unit", Options: pickUnits, Chosen: unit},
		},
	}})
	if err != nil {
		slog.Error("drawing the question", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "Tell me how often, in words."}
	}
	return squirrel.Turn{
		Who: squirrel.SpeakerBuddy, Words: "How often should it come back?", Shown: body,
	}
}

// rhythmOf is the interval a chore has now, as the picker's own two answers, so
// the question opens on what is true rather than on a blank form.
//
// Anything that does not land on an offered pair — a fortnight, a quarter —
// leaves both empty rather than rounding to the nearest offered thing.
//
// Units are tried largest first. Days last, deliberately: 14 days is two weeks
// to a person, and trying days first would answer "14", which is not an offered
// number and would then fall through to nothing at all.
func rhythmOf(every time.Duration) (count, unit string) {
	for i := len(pickUnits) - 1; i >= 0; i-- {
		u := pickUnits[i]
		step := unitStep(u)
		if step == 0 || every == 0 || every%step != 0 {
			continue
		}
		n := strconv.FormatInt(int64(every/step), 10)
		if oneOf(pickNumbers, n) {
			return n, u
		}
	}
	return "", ""
}

// unitStep is how long each offered unit is.
//
// Thirty days for a month, exactly as the core reads it — this is a nudge, not
// a calendar. See unitDurations in internal/squirrel/intent.go, and do not let
// the two drift: TestThePickerAndTheSentenceAgree is what notices if they do.
func unitStep(unit string) time.Duration {
	switch unit {
	case "days":
		return 24 * time.Hour
	case "weeks":
		return 7 * 24 * time.Hour
	case "months":
		return 30 * 24 * time.Hour
	}
	return 0
}

// composeEvery turns the picker's two answers into an interval, through the
// same parser a typed sentence goes through.
//
// Not arithmetic here: ParseEvery is where "every 3 weeks" means something, and
// a second place deciding what a week was would be a second place to be wrong.
// Both answers are checked against what was offered first, because they arrive
// from a form.
func composeEvery(count, unit string) (time.Duration, bool) {
	if !oneOf(pickNumbers, count) || !oneOf(pickUnits, unit) {
		return 0, false
	}
	_, every, ok := squirrel.ParseEvery("every " + count + " " + unit + ": x")
	return every, ok
}

func oneOf(list []string, v string) bool {
	for _, o := range list {
		if o == v {
			return true
		}
	}
	return false
}

// tasksTurn is what you decided and have not done, as cards.
//
// Newest first, like the pile: a task decided this morning is the one you still
// remember deciding.
func tasksTurn(ctx context.Context, s Store, personID int64, name string) squirrel.Turn {
	items, more, err := s.Tasks(ctx, personID, listLimit)
	if err != nil {
		slog.Error("reading what you decided", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "I cannot reach the tasks just now."}
	}
	if len(items) == 0 {
		return squirrel.Turn{
			Who:   squirrel.SpeakerBuddy,
			Words: "Nothing decided yet. A task is a note you said yes to.",
		}
	}

	sh := drawn{Place: name}
	for _, it := range items {
		v := toView(it)
		row := map[string]string{"id": strconv.FormatInt(v.ID, 10)}
		sh.Cards = append(sh.Cards, cardView{
			Title: v.Text, Meta: "decided " + v.When, Photo: v.Photo,
			Acts: []actView{
				{Label: "did it", Action: "/tasks/act", Style: "did", Fields: with(row, "act", "done")},
				// "back" rather than "later": this is not a deferral, it is a
				// decision reversed. The class matters twice — the tasks are
				// never late, and `later` has the word inside it.
				{Label: "not a task", Action: "/tasks/act", Style: "back", Fields: with(row, "act", "untask")},
			},
		})
	}
	// The way to what you cannot act on. It hung off the tasks screen, and
	// without it here /held is reachable from nowhere in the product — which
	// is the bug the mood history had for an afternoon.
	sh.Chips = []turnChip{{Label: "what you cannot act on", Href: "/held"}}
	if more {
		sh.Chips = append(sh.Chips, turnChip{Label: "the rest", Href: "/?open=tasks"})
	}

	body, err := json.Marshal(sh)
	if err != nil {
		slog.Error("drawing the tasks", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "I cannot draw the tasks just now."}
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: taskLead(len(sh.Cards)), Shown: body}
}

func taskLead(n int) string {
	if n == 1 {
		return "One thing you decided."
	}
	return fmt.Sprintf("%d things you decided.", n)
}

// saidAboutATask is what the two of you said about one.
//
// "Not a task" is not a failure and must not read like one: it is a note that
// went back to being a note, which is a decision reversed rather than a thing
// given up on.
func saidAboutATask(act, text string) []squirrel.Turn {
	if text == "" {
		text = "that"
	}
	switch act {
	case "done":
		return []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "did it — " + text},
			{Who: squirrel.SpeakerBuddy, Words: "Done."},
		}
	case "open":
		return []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "not done after all — " + text},
			{Who: squirrel.SpeakerBuddy, Words: "Back on the list."},
		}
	case "untask":
		return []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "not a task — " + text},
			{Who: squirrel.SpeakerBuddy, Words: "Back in the pile."},
		}
	}
	return nil
}

// agendaTurn is what is still ahead, as cards.
//
// The list this product spent its whole life refusing, and what makes it
// allowed is unchanged by its moving into a turn: it holds only what is still
// coming. Nothing past, nothing done, and nothing here has been missed —
// because a thing you have not reached yet is not a thing you are late for.
func agendaTurn(ctx context.Context, s Store, personID int64, name string) squirrel.Turn {
	coming, err := s.Upcoming(ctx, personID, now(), listLimit)
	if err != nil {
		slog.Error("reading what is coming", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "I cannot reach the agenda just now."}
	}
	if len(coming) == 0 {
		return squirrel.Turn{
			Who:   squirrel.SpeakerBuddy,
			Words: "When something has a time you can be late for, it will be here.",
		}
	}

	sh := drawn{Place: name}
	for _, m := range coming {
		row := map[string]string{"id": strconv.FormatInt(m.ID, 10)}
		// The core's own sentence, shared with chat and with the notification,
		// so the three cannot drift apart about when to leave.
		card := cardView{Title: m.Label, Meta: squirrel.LeaveWords(m)}
		card.Acts = []actView{{Label: "OPEN", Action: "/at/open", Style: "go", Fields: row}}
		if m.Open(now()) {
			// Only inside the window. Outside it the appointment is not yet
			// something you can act on, and a button that closes a thing three
			// hours early is one that gets pressed by accident.
			card.Acts = append(card.Acts, actView{
				Label: "LEAVING", Action: "/now/act", Style: "did",
				Fields: map[string]string{
					"kind": string(squirrel.OfferMoment),
					"id":   strconv.FormatInt(m.ID, 10),
					"act":  "did", "label": m.Label,
				},
			})
		}
		sh.Cards = append(sh.Cards, card)
	}

	body, err := json.Marshal(sh)
	if err != nil {
		slog.Error("drawing the agenda", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "I cannot draw the agenda just now."}
	}
	// Buddy counts what is ahead, and says nothing about what is behind: there
	// is no count of what was missed because nothing here can be.
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: comingLead(len(sh.Cards)), Shown: body}
}

func comingLead(n int) string {
	if n == 1 {
		return "One thing has a time."
	}
	return fmt.Sprintf("%d things have a time.", n)
}

// fixedPointTurn is one appointment and what is pointing at it.
//
// What to take is its own line rather than a clause after a middot: it is the
// thing you are standing in the hall without, so it gets to be a line rather
// than a tail. Absent when there is nothing to take — an empty label is a row
// that says nothing.
func fixedPointTurn(m squirrel.Moment, notes []squirrel.Item) squirrel.Turn {
	card := cardView{Title: m.Label, Meta: squirrel.LeaveWords(m)}
	if m.Bring != "" {
		card.Take = "take " + m.Bring
	}
	if m.Open(now()) {
		card.Acts = []actView{{
			Label: "LEAVING", Action: "/now/act", Style: "did",
			Fields: map[string]string{
				"kind": string(squirrel.OfferMoment),
				"id":   strconv.FormatInt(m.ID, 10),
				"act":  "did", "label": m.Label,
			},
		}}
	}
	sh := drawn{Cards: []cardView{card}}

	// The notes pointing at it, each with the way back. Every transition in
	// this product reverses, and the pointer was the whole of the change.
	for _, it := range notes {
		v := toView(it)
		sh.Cards = append(sh.Cards, cardView{
			Title: v.Text, Photo: v.Photo, Meta: v.When,
			Acts: []actView{{
				Label: "BACK IN THE PILE", Action: "/at/" + strconv.FormatInt(m.ID, 10) + "/detach",
				Style: "back", Fields: map[string]string{"id": strconv.FormatInt(v.ID, 10)},
			}},
		})
	}

	body, err := json.Marshal(sh)
	if err != nil {
		slog.Error("drawing a fixed point", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: squirrel.LeaveWords(m)}
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: squirrel.LeaveWords(m), Shown: body}
}

// calView is the day picker: a month to choose out of, and a time.
//
// One form and one submit, exactly as the interval picker. Turning to another
// month is the one control that posts on its own — it is not an answer, it is
// turning a page, and it re-asks rather than writing a second question into a
// record that is never rewritten.
type calView struct {
	Action string            `json:"action"`
	Fields map[string]string `json:"fields,omitempty"`
	Month  string            `json:"month"`
	// The months either side, as 2026-07. Empty for a month with nothing to
	// go back to: the picker offers no day in the past, so there is nothing
	// behind this one.
	Prev string `json:"prev,omitempty"`
	Next string `json:"next,omitempty"`
	// Pad is the blanks before the first, Monday first.
	Pad   int      `json:"pad"`
	Days  []calDay `json:"days"`
	Times []string `json:"times"`
	Do    string   `json:"do"`
}

// Padding is the blanks before the first, as a range a template can walk:
// html/template has no way to count to a number.
func (c calView) Padding() []struct{} { return make([]struct{}, c.Pad) }

// calDay is one cell. Past days are drawn and not offered: a month with holes
// in it is harder to read than one where some days cannot be pressed.
type calDay struct {
	Day  int    `json:"day"`
	Date string `json:"date,omitempty"`
	Gone bool   `json:"gone,omitempty"`
}

// pickTimes are the hours offered.
//
// Three and a way out, like the numbers on the interval picker: these are the
// times most appointments actually are, and anything else is a sentence — the
// dock already understands "at 08:15 dentist".
var pickTimes = []string{"09:00", "14:30", "18:00"}

// askForADay is the question.
//
// Monday first, because that is how a week is read here. Days already gone are
// drawn and not offered — a month with holes in it is harder to read than one
// where some days cannot be pressed — and there is no way back past this month
// for the same reason: the list may hold nothing you are already late for.
func askForADay(label string, month time.Time) squirrel.Turn {
	first := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
	// Monday is 0 here; Go's Sunday is 0.
	pad := (int(first.Weekday()) + 6) % 7

	today := squirrel.StartOfDay(now())
	cal := calView{
		Action: "/at/make",
		Fields: map[string]string{"label": label},
		Month:  first.Format("January 2006"),
		Pad:    pad,
		Times:  pickTimes,
		Do:     "that's it",
	}
	if first.After(today) {
		cal.Prev = first.AddDate(0, -1, 0).Format("2006-01")
	}
	cal.Next = first.AddDate(0, 1, 0).Format("2006-01")

	for d := first; d.Month() == first.Month(); d = d.AddDate(0, 0, 1) {
		cell := calDay{Day: d.Day()}
		if d.Before(today) {
			cell.Gone = true
		} else {
			cell.Date = d.Format("2006-01-02")
		}
		cal.Days = append(cal.Days, cell)
	}

	body, err := json.Marshal(drawn{Cal: &cal})
	if err != nil {
		slog.Error("drawing the days", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "Tell me when, like at 14:30 dentist."}
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "Which day?", Shown: body}
}

// pileTurn is one note, with the four ways out of it.
//
// One at a time, exactly as the deck does it: the pile is a thing you decide
// about, and a list of things to decide about is a list you are behind on.
// What changes in a conversation is only that the next one arrives underneath
// the last rather than replacing it.
func pileTurn(ctx context.Context, s Store, personID, after int64, name string) squirrel.Turn {
	items, _, err := s.OpenItemsAfter(ctx, personID, after, 1)
	if err != nil {
		slog.Error("reading the pile", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "I cannot reach the pile just now."}
	}
	if len(items) == 0 {
		if after != 0 {
			// The bottom, reached by skipping. Not the same as an empty pile:
			// what is above you is still there.
			return squirrel.Turn{
				Who:   squirrel.SpeakerBuddy,
				Words: "That is the end of them. The ones you skipped are still there.",
			}
		}
		return squirrel.Turn{
			Who:   squirrel.SpeakerBuddy,
			Words: "Nothing to decide about. Anything you tell me lands here.",
		}
	}

	v := toView(items[0])
	row := map[string]string{
		"id": strconv.FormatInt(v.ID, 10), "was": v.State, "from": "thread",
	}
	sh := drawn{
		Place: name,
		Cards: []cardView{{
			Title: v.Text, Photo: v.Photo, Meta: v.When,
			Acts: []actView{
				{Label: "DONE", Action: "/pile/act", Style: "did", Fields: with(row, "act", "done")},
				{Label: "KEEP", Action: "/pile/act", Style: "go", Fields: with(row, "act", "keep")},
				{Label: "DROP", Action: "/pile/act", Style: "stop", Fields: with(row, "act", "drop")},
				{Label: "A TASK", Action: "/pile/act", Style: "go", Fields: with(row, "act", "task")},
			},
		}},
		// Later is not a decision. It leaves the note where it was and hands
		// you the next, which is the deck's own LATER.
		Chips: []turnChip{{
			Label: "later", Action: "/pile/later",
			Fields: map[string]string{"after": strconv.FormatInt(v.ID, 10)},
		}},
	}

	body, err := json.Marshal(sh)
	if err != nil {
		slog.Error("drawing the pile", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: v.Text}
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "This one.", Shown: body}
}

// saidAboutANote is what the two of you said, and the way to change your mind.
//
// The way back travels with the answer because the card is about to be
// scrollback, and scrollback carries no controls: an undo you can only reach by
// pressing something that has already gone is an undo nobody reaches.
func saidAboutANote(act, text string, id int64, was string) []squirrel.Turn {
	words, reply := "", ""
	switch act {
	case "done":
		words, reply = "done — "+text, "Good."
	case "keep":
		words, reply = "keep — "+text, "On the shelf."
	case "drop":
		words, reply = "drop — "+text, "Gone."
	case "task":
		words, reply = "a task — "+text, "It is a task now."
	default:
		return nil
	}

	// Back to where it was. "note" rather than "open" for a decision, because
	// what changed was the note's kind and not its state — the same distinction
	// actHandler makes, said the same way.
	backAct := "open"
	if act == "task" {
		backAct = "note"
	}
	body, err := json.Marshal(drawn{Chips: []turnChip{{
		Label: "put it back", Action: "/pile/undo",
		Fields: map[string]string{
			"id": strconv.FormatInt(id, 10), "act": backAct, "was": was,
		},
	}}})
	if err != nil {
		slog.Error("drawing the way back", "error", err)
	}
	return []squirrel.Turn{
		{Who: squirrel.SpeakerYou, Words: words},
		{Who: squirrel.SpeakerBuddy, Words: reply, Shown: body},
	}
}
