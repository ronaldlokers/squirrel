package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The thread: the whole of the screen. The slot is the way in.
//
// Only the newest Buddy turn carries controls. A card from this morning keeps
// its words and loses its buttons, because pressing DID IT on a card from a
// conversation three days old acts on a state nobody is looking at. See The
// live edge in DESIGN.md.

// threadLimit is how much of the conversation one render holds. A bound rather
// than a page: everything above it is still there and one press away.
const threadLimit = 40

// drawn is what a turn drew, decoded from the turn's own JSON and never re-read
// from another table: a turn holding a chore id would show today's chore inside
// yesterday's sentence.
type drawn struct {
	// Cost is what the coach has spent, on the reply that spent it. Only ever
	// there — see coachReplyCosting.
	Cost string `json:"cost,omitempty"`
	// Opened marks a turn Buddy started himself, and carries what it was
	// about, so the conversation can be asked whether it has said this
	// already. See openingTurn.
	Opened string     `json:"opened,omitempty"`
	Place  string     `json:"place,omitempty"`
	Cards  []cardView `json:"cards,omitempty"`
	Chips  []turnChip `json:"chips,omitempty"`
	// Pick is a question with its answers on it: rows of choices in one form
	// with one submit. See askHowOften for why it is one form.
	Pick *pickView `json:"pick,omitempty"`
	// Cal is the day picker. See askForADay.
	Cal *calView `json:"cal,omitempty"`
	// Say is a question whose answer is words. See askForWords.
	Say *sayView `json:"say,omitempty"`
	// Cut is a proposal to split a note. See proposeInThread.
	Cut *cutView `json:"cut,omitempty"`
	// Hits are search results, which are deliberately not cards. A card is
	// something you act on; a hit is something you are finding, and making
	// them one object is why search felt like a second pile. See searchTurn.
	Hits []hitView `json:"hits,omitempty"`
	// Faces is the check-in's five drawings. A flag rather than five chips
	// because they are the product's own faces and the markup for them
	// already exists; rendering them as words would be a different control
	// wearing the same question.
	Faces bool `json:"faces,omitempty"`
}

type turnView struct {
	ID int64
	// Room is where this was said, and every form drawn on this turn carries
	// it. Threading it through the twenty places that build a Fields map would
	// be twenty chances to forget one, and a forgotten one is invisible: the
	// answer lands in Buddy's room, which is what roomOf falls back to.
	//
	// From the turn rather than from the page, because a fragment is rendered
	// with no page around it — the same reason V is filled here.
	Room string
	// You is who is reading, for the face on your own turns. On the turn for
	// the same reason Room and V are: a fragment is rendered with no page
	// around it.
	You   whom
	Buddy bool
	Words string
	// Cost is what this reply cost, on the reply that cost it.
	Cost string
	// Opens marks the turn that begins a run by either speaker, which is where a
	// face goes: consecutive turns are one utterance, and a face on every one
	// is wallpaper by the third day.
	Opens bool
	// When is the clock time this was said, on the turn that opens a run and
	// empty on the rest — the same rule the face follows, for the same reason.
	// A time on every line turns a conversation into a log of your day, which
	// is the shape the run-resumption sentence refuses a clock time to avoid.
	//
	// A run is one utterance, so one time is the whole truth about it.
	When string
	// Day is the divider above this turn, and empty when the day has not
	// changed. A time alone cannot answer "when was this" once you have
	// scrolled past midnight, and a date on every line to answer it would be
	// the log again.
	Day string
	// Place is the <h2> when this turn opens one, and empty otherwise. The
	// thread has no <h1> — home's exemption, because nobody arrives at the
	// place they started wondering where they are — so these are what heading
	// navigation walks.
	Place string
	Cards []cardView
	Hits  []hitView
	Chips []turnChip
	Faces []faceView
	Pick  *pickView
	Cal   *calView
	Say   *sayView
	Cut   *cutView
	// V stamps the asset URLs a turn draws. Filled here rather than by render,
	// because a turn is also rendered on its own as a fragment, where there is
	// no page around it to carry it.
	V string
	// Live is the newest Buddy turn and nothing else.
	Live bool
}

type cardView struct {
	// Kind is what sort of thing the card holds, or empty for an ordinary one.
	// "chore" renders as `article.chore`, which pile.js's chore keys and the
	// stylesheet's chore rules already work on.
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

// actView is one button on a card. Fields is a map because /now/act wants kind,
// id and act together.
type actView struct {
	Label  string            `json:"label"`
	Action string            `json:"action"`
	Fields map[string]string `json:"fields,omitempty"`
	Style  string            `json:"style,omitempty"`
}

// turnChip is a choice offered in the conversation.
type turnChip struct {
	Label string `json:"label"`
	// Href for somewhere to go; Action and Fields for something to do. A chip
	// that acts is a form, because a GET that writes writes again on every
	// reload — the same reason a door is a press.
	Href   string            `json:"href,omitempty"`
	Action string            `json:"action,omitempty"`
	Fields map[string]string `json:"fields,omitempty"`
	// Count is a number beside the label, in the menu only. Zero is no number
	// and not a nought: a place reading "0" is a scoreboard.
	Count int `json:"-"`
}

// hitView is one search result: a quiet line that opens into a card.
type hitView struct {
	Title string `json:"title"`
	// Meta is which of the seven it is in, because a result read without it is
	// a note whose verbs would be wrong.
	Meta   string            `json:"meta,omitempty"`
	Action string            `json:"action"`
	Fields map[string]string `json:"fields,omitempty"`
}

func threadHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
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
			turns, more, err = s.TurnsBefore(ctx, personID, "everything", before, threadLimit)
		} else {
			turns, more, err = s.RecentTurns(ctx, personID, "everything", threadLimit)
		}
		// A record that cannot be read is not a reason to take the screen away: the dock
		// still writes to the spool. Said out loud rather than rendered as an empty
		// conversation, which looks like your history is gone.
		unreadable := err != nil
		if unreadable {
			slog.Error("reading the conversation", "error", err)
			turns, more = nil, false
		}

		// say appends one of Buddy's own turns. A turn that fails to reach the
		// record leaves a hole in the conversation, which is recoverable;
		// refusing the page over it would not be.
		say := func(t squirrel.Turn, doing string) {
			saved, err := s.AppendTurn(ctx, personID, "everything", t)
			if err != nil {
				slog.Error(doing, "error", err)
				return
			}
			turns = append(turns, saved)
		}

		// Whether this is a first run is decided before anything is appended, because
		// every turn below is about to make the record non-empty.
		//
		// Provisional: anything Squirrel then finds to say about your own world puts it
		// out. Never while walking back, and never when the record could not be read.
		first := !walkingBack && !unreadable && !more && len(turns) == 0

		// The check-in, written rather than rendered: a question that is not in the
		// record is one the record cannot show you answering.
		//
		// Never while walking back — reading the past must not add to it — and never
		// twice over, which put the same question on screen three deep.
		asked := false
		if !walkingBack && !unreadable && !alreadyAsking(turns) {
			if t, ask := checkinTurn(ctx, s, personID, r.URL.Query().Get("ask") != ""); ask {
				asked = true
				say(t, "asking how you are")
			}
		}

		// Only once the question has been answered. Asking how you are and then handing
		// you a job in the same breath is the interruption this product exists to
		// reduce, and the answer shapes the offer anyway.
		if !walkingBack && !unreadable && !endsOpen(turns) {
			if st := stepFor(s, opts, r); st != nil {
				say(coachReply("Where you were.", false, false, nil, st),
					"drawing where you were")
			}
		}

		// The opening line, before the offer: what is happening, then what to do about
		// it.
		if !walkingBack && !unreadable && !endsAsking(turns) {
			if t, has := openingTurn(ctx, s, opts, personID, turns); has {
				// Squirrel has something to say about your world, so you have
				// one.
				first = false
				say(t, "opening")
			}
		}

		if !walkingBack && !asked && !unreadable && !endsOpen(turns) {
			if t, has := offerTurn(s, opts, r); has {
				// Something to hand you is something you already have. A
				// worked example above it would be explaining a product that
				// is mid-sentence about your own things.
				first = false
				say(t, "offering it")
			}
		}

		// Everything has no list of its own, so its edge is empty — but the
		// script asks whichever room it is standing in, and a handler that
		// ignored the ask would answer with the whole page and have it pasted
		// into the element the ask was about.
		if r.Header.Get("X-Edge") == "1" {
			edgeOnly(w, r, nil)
			return
		}

		buddy, _ := roomByKey("everything")
		v := view{
			Home:   true,
			Thread: true,
			Room:   buddy,
			// The worker having taken the words because there was no network.
			// From the address bar, read the way a stranger's typing is read:
			// a present flag and nothing else.
			Held:      r.URL.Query().Get("held") != "",
			Here:      "thread",
			Scrolling: true,
			Turns:     turnViews(r.Context(), turns),
			MoreAbove: more,
		}
		if first {
			v.Example = worked()
		}
		if len(turns) > 0 {
			v.Oldest = turns[0].ID
		}
		if unreadable {
			v.Turns = []turnView{{
				Buddy: true, Live: true, V: stamp(),
				Words: "I cannot reach what we said. Tell me things anyway — they are kept, and they go in when I can.",
			}}
		}
		renderWith(w, r, s, opts, "thread", v)
	}
}

// endsOpen says the conversation already ends with something to act on.
//
// Without it the offer was appended on every load, and it stole the live edge
// from whatever Buddy had just said. The question is about shape rather than
// about which offer: anything unanswered is a reason not to add another.
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
	// An opening line is not something on the table: it says what is true and asks
	// nothing. Counting its chip as open meant that on any day with an agenda, a due
	// chore or a pile — most days — Squirrel opened with a line and then offered
	// nothing.
	if sh.Opened != "" {
		return false
	}
	return len(sh.Cards) > 0 || len(sh.Chips) > 0 || sh.Faces ||
		sh.Pick != nil || sh.Cal != nil || sh.Say != nil || sh.Cut != nil
}

// turnViews decodes each turn's record of what it drew and marks the live edge.
// The scan runs backwards and stops at the first Buddy turn, so a run of your own
// turns at the bottom leaves something to press.
func turnViews(ctx context.Context, turns []squirrel.Turn) []turnView {
	out := make([]turnView, 0, len(turns))
	for _, t := range turns {
		v := turnView{ID: t.ID, Room: t.Room, Buddy: t.Who == squirrel.SpeakerBuddy, Words: t.Words, V: stamp(), You: whomOf(ctx)}
		if len(t.Shown) > 0 {
			var sh drawn
			if err := json.Unmarshal(t.Shown, &sh); err != nil {
				// A turn whose record cannot be read still said something, and
				// the words are the part that matters. Losing the cards is
				// better than losing the turn.
				slog.Error("reading what a turn drew", "turn", t.ID, "error", err)
			} else {
				v.Place, v.Cards, v.Chips = sh.Place, sh.Cards, sh.Chips
				v.Hits = sh.Hits
				v.Cost = sh.Cost
				v.Pick, v.Cal, v.Say, v.Cut = sh.Pick, sh.Cal, sh.Say, sh.Cut
				if sh.Faces {
					v.Faces = theFaces()
				}
			}
		}
		out = append(out, v)
	}
	// Where Buddy starts speaking. The first turn of a run and every turn
	// after somebody else spoke.
	for i := range out {
		// Either speaker, since 30 August 2026: yours carries a face now too,
		// and a run of yours has to hold one right edge for the same reason a
		// run of his holds one left edge.
		out[i].Opens = i == 0 || out[i-1].Buddy != out[i].Buddy
	}

	// The clock, on the openers only, and a divider wherever the day turns
	// over. Read in your zone rather than the container's — see zoneOf.
	where := zoneOf(ctx)
	for i := range out {
		said := turns[i].SaidAt
		if said.IsZero() {
			continue
		}
		said = said.In(where)
		if out[i].Opens {
			out[i].When = said.Format("15:04")
		}
		if i == 0 || !sameDay(turns[i-1].SaidAt.In(where), said) {
			out[i].Day = dayCalled(said, now().In(where))
		}
	}

	for i := len(out) - 1; i >= 0; i-- {
		if out[i].Buddy {
			out[i].Live = true
			break
		}
	}
	return out
}

// sameDay is whether two times fall on the same date, in whatever zone they are
// already carrying. Compared as a date rather than as a gap: two turns eleven
// hours apart can be one day or two, and only the calendar knows which.
func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// dayCalled is the divider's words.
//
// Today and yesterday by name, because they are the two days you are most
// likely to be looking for and a date makes you work out which one it was.
// Everything older gets the date it has: "Monday 25 August" is what the agenda
// already says, and one vocabulary for a day is what keeps two screens from
// disagreeing about it.
func dayCalled(said, today time.Time) string {
	switch {
	case sameDay(said, today):
		return "today"
	case sameDay(said, today.AddDate(0, 0, -1)):
		return "yesterday"
	}
	return said.Format("Monday 2 January")
}

// menuFor is everywhere else, behind the lid's one control. Order is by how
// often a thing is wanted; stopping is not in it — the template puts it last,
// under a rule.
// theFaces is the five, in the one order both surfaces use.
func theFaces() []faceView {
	out := make([]faceView, 0, len(squirrel.Moods))
	for _, m := range squirrel.Moods {
		out = append(out, faceView{Mood: string(m), Word: squirrel.Words[m]})
	}
	return out
}

// checkinTurn is Buddy's question, while the last answer no longer describes now.
// A turn rather than a region, so the question and the answer both stay.
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

// threadMoodHandler records the reading first and the turns after: an answer that
// reached the conversation but not the readings would make /moods disagree with
// the thread, and the readings are what the picker consults.
func threadMoodHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		ctx := r.Context()
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, backToTheRoom(r), http.StatusSeeOther)
			return
		}
		m, ok := squirrel.ParseMood(r.FormValue("mood"))
		if !ok {
			// Not one of the five. This arrives from a form, so it is read the
			// way a stranger's typing is read: no answer rather than a wrong
			// one, and nothing said about it in the record.
			http.Redirect(w, r, backToTheRoom(r), http.StatusSeeOther)
			return
		}
		if err := s.RecordCheckin(ctx, personID, m, "screen", now()); err != nil {
			fail(w, err)
			return
		}
		// This reports what you said and may never characterise you.
		//
		// The way to change your mind travels with it, because the answer is about to be
		// scrollback and scrollback carries no controls.
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
		}), backToTheRoom(r))
	}
}

// offerTurn is the one thing Squirrel picked, or nothing. Nothing renders
// nothing: a reassuring sentence would be the product deciding you ought to be
// busy.
func offerTurn(s Store, opts Options, r *http.Request) (squirrel.Turn, bool) {
	// "Show me anyway" lifts the capacity gate for this render only, and lives in the
	// address bar rather than the database: storing it would make somebody re-decide
	// tomorrow that they meant it.
	anyway := r.URL.Query().Get("anyway") != ""
	o := offerFor(s, opts, r, anyway, true)
	if o == nil {
		// Nothing, or something withheld because of how you said you were. Only the
		// second has anything to say.
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
				{Label: "PICK IT UP", Action: "/now/act", Style: "go",
					Fields: with(row, "act", "start", "minutes", quickTimer)},
				{Label: "not now", Action: "/now/act", Style: "later", Fields: with(row, "act", "later")},
			}
		default:
			card.Acts = []actView{
				{Label: did, Action: "/now/act", Style: "did", Fields: with(row, "act", "did")},
				{Label: quickTimer + " MIN", Action: "/now/act", Style: "go",
					Fields: with(row, "act", "start", "minutes", quickTimer)},
				{Label: "not now", Action: "/now/act", Style: "later", Fields: with(row, "act", "later")},
			}
		}
		// The four ways of not being able to start, on the card rather than behind a
		// disclosure: the card is about to be scrollback, and scrollback carries no
		// controls.
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

// with is a row's own fields plus some more, copied rather than mutated: the
// buttons on one card share the row and each needs a different act.
//
// Pairs, so a button that adds two fields reads as one call rather than as with
// wrapped around with.
func with(fields map[string]string, pairs ...string) map[string]string {
	out := make(map[string]string, len(fields)+len(pairs)/2)
	for k, v := range fields {
		out[k] = v
	}
	for i := 0; i+1 < len(pairs); i += 2 {
		out[pairs[i]] = pairs[i+1]
	}
	return out
}

// quickTimer is what the offer's own timer button starts. Ten minutes, which is
// the number on the button.
const quickTimer = "10"

// saidAboutTheOffer is what the two of you said when a press landed. Turning it
// down is in the record beside doing it, and they do not read the same.
func saidAboutTheOffer(act, label string) []squirrel.Turn {
	if label == "" {
		label = "that"
	}
	switch act {
	case "did":
		return []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "did it — " + label},
			{Who: squirrel.SpeakerBuddy, Words: squirrel.Say(squirrel.SayingDid, now())},
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

// keepSaid writes what was said into the room the request is in, and logs when
// it cannot. A press that changed the pile and failed to reach the
// conversation is recoverable; refusing the press because the record could not
// be written would not be.
//
// The room comes from the context rather than from a parameter because thirty
// handlers call this and a thirty-first would be added without one. A turn in
// the wrong room is invisible — it looks exactly like a room that is quiet —
// so there is one place that decides, and TestOnlyKeepSaidPutsTurnsInARoom
// fences it.
func keepSaid(ctx context.Context, s Store, personID int64, said []squirrel.Turn) []squirrel.Turn {
	out := make([]squirrel.Turn, 0, len(said))
	for _, t := range said {
		saved, err := s.AppendTurn(ctx, personID, roomOf(ctx), t)
		if err != nil {
			slog.Error("keeping what was said", "error", err)
			continue
		}
		out = append(out, saved)
	}
	return out
}

// saidAboutBeingStuck is the ladder, as two turns.
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

// blockersFor is the four blockers, on the things they are about. Not on a
// breadcrumb: the answer there is the breadcrumb's own "not now".
func blockersFor(kind string) []squirrel.Blocker {
	if kind == string(squirrel.OfferAgain) {
		return nil
	}
	return squirrel.Blockers
}

// wayThrough is what a low day gets instead of a job: the gate holds, and what is
// offered is the way past it, once, in your own hands.
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

// wantsFragment is a press made by the script rather than the browser's own form
// machinery. A header rather than a second route: one URL per action, one write.
func wantsFragment(r *http.Request) bool { return r.Header.Get("X-Thread") == "fragment" }

// answerWith is what a press gets back: a fragment for the script, a redirect
// for a browser posting a form.
// insteadOf says this answer re-draws something already on the screen rather
// than adding to it, and reports whether it can.
//
// Turning the calendar's month appended a whole new "Which day?" every time,
// so paging through to November left five of them in a record that is never
// rewritten. Paging is not something you said. The turn comes back under the
// id it already had and the script swaps it in place, and nothing is kept — so
// a reload shows the question once, in the month it was first asked in.
//
// Only for a press the script made. Without one there is nothing on the page
// to swap, and the honest fallback is the old behaviour: keep it, and let the
// redirect draw a conversation with the question in it. Enhancement only, the
// way everything else on this screen is.
func insteadOf(w http.ResponseWriter, r *http.Request, said *squirrel.Turn, replacing int64) bool {
	if !wantsFragment(r) || replacing <= 0 {
		return false
	}
	said.ID = replacing
	w.Header().Set("X-Replaces", "turn-"+strconv.FormatInt(replacing, 10))
	return true
}

func answerWith(w http.ResponseWriter, r *http.Request, said []squirrel.Turn, back string) {
	if !wantsFragment(r) {
		http.Redirect(w, r, back, http.StatusSeeOther)
		return
	}
	vs := turnViews(r.Context(), said)
	for i := range vs {
		vs[i].Live = false
	}
	// A fragment is appended to a conversation that is already on screen, and
	// the turn above it is not in this batch — so the divider turnViews marks
	// on the first turn would say "today" under a "today" that is already
	// there. Cleared here rather than not computed there, because the page
	// path wants it and this is the one path that does not.
	//
	// What that costs: a press made after midnight with the page still open
	// gets no divider until the next reload. A duplicate one on every press is
	// the worse of the two.
	if len(vs) > 0 {
		vs[0].Day = ""
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

// listLimit is how many cards one turn draws. A bound rather than a page: what
// is past it is one press away, and never a count.
const listLimit = 5

// doorName is the vocabulary, and it is the rooms'. Two lists of the same
// seven names is one list that goes stale.
func doorName(key string) (string, bool) {
	r, ok := roomByKey(key)
	return r.Name, ok
}

// openHandler is two things that used to be one.
//
// With a `from`, it is "the rest" — the next page of a list you are already
// looking at. That is paging inside a room rather than navigation to one, and
// it is genuinely something you said, so it still writes: the record says you
// asked for the rest of the tasks, not that you opened the tasks twice.
//
// Without one, it is the door as it was until 28 August 2026, and it writes
// nothing — it sends you to the room. Kept rather than deleted because an
// installed home screen holds a cached page whose forms still post here, and a
// 405 on the first press after an update is an app that looks broken.
func openHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, backToTheRoom(r), http.StatusSeeOther)
			return
		}
		where := r.FormValue("where")
		if _, ok := placeName(where); !ok {
			http.Redirect(w, r, backToTheRoom(r), http.StatusSeeOther)
			return
		}
		from, _ := strconv.Atoi(r.FormValue("from"))
		if from <= 0 {
			http.Redirect(w, r, "/r/"+where, http.StatusSeeOther)
			return
		}
		// The rest belongs in the room the list is in, so it is written there
		// rather than wherever the press happened to come from.
		ctx := withRoomIn(r.Context(), where)
		said := placeTurn(ctx, s, opts, personID, where, from)
		if len(said) == 0 {
			http.Redirect(w, r, "/r/"+where, http.StatusSeeOther)
			return
		}
		answerWith(w, r, keepSaid(ctx, s, personID, said), "/r/"+where)
	}
}

// placeTurn is what you said and what Buddy answered, or nothing at all.
func placeTurn(ctx context.Context, s Store, opts Options, personID int64, where string, from int) []squirrel.Turn {
	name, ok := placeName(where)
	if !ok {
		return nil
	}
	reply, _ := placeSaid(ctx, s, opts, personID, where, from)
	// And the way to make one more. On every branch, including the one that
	// says there is nothing here — an empty list is the moment you are most
	// likely to want to add to it. Not on "the rest", which is the middle of
	// a list rather than the top of one.
	said := name
	if from > 0 {
		return []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "the rest of " + name}, reply,
		}
	}
	return []squirrel.Turn{
		{Who: squirrel.SpeakerYou, Words: said},
		alsoOffer(reply, newChipFor(where)...),
	}
}

// placeSaid is the place itself, without the utterance that opened it.
//
// Split out because a place has two ways of being opened and only one of them
// is something you said. Pressing the menu is an utterance and goes into the
// record as one; Buddy opening it because you asked him to is his answer, and
// putting "the tasks" in your mouth would be the record inventing a sentence
// you did not say. See coachSayHandler.
func placeSaid(ctx context.Context, s Store, opts Options, personID int64, where string, from int) (squirrel.Turn, bool) {
	name, ok := placeName(where)
	if !ok {
		return squirrel.Turn{}, false
	}
	var reply squirrel.Turn
	switch where {
	case "chores":
		reply = choresTurn(ctx, s, opts, personID, name, from)
	case "tasks":
		reply = tasksTurn(ctx, s, opts, personID, name, from)
	case "at":
		reply = agendaTurn(ctx, s, personID, name, from)
	case "notes":
		reply = pileTurn(ctx, s, opts, personID, 0, name)
	// The two shelves. Not rooms since 31 August 2026, and still places: a
	// shelf is something you can be shown, which is what the chips inside the
	// notes and Buddy's own `open` both ask for.
	case "kept":
		reply = keptTurn(ctx, s, personID, name)
	case "held":
		reply = heldTurn(ctx, s, personID, name)
	default:
		// Unreachable while this switch covers doorNames, which is checked
		// above. Here so that a name added to the map and forgotten here says
		// so rather than answering with silence, which reads as a press that
		// did not land.
		reply = squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "Not yet — that one is still a page."}
	}
	return reply, true
}

// choresTurn is what comes back, as cards.
func choresTurn(ctx context.Context, s Store, opts Options, personID int64, name string, from int) squirrel.Turn {
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

	chores, more := slice(chores, from)
	if len(chores) == 0 {
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "That is all of them."}
	}
	sh := drawn{Place: name}
	for _, c := range chores {
		sh.Cards = append(sh.Cards, choreCard(toChoreView(c)))
	}
	if more {
		sh.Chips = []turnChip{theRest("chores", from+listLimit)}
	}

	body, err := json.Marshal(sh)
	if err != nil {
		slog.Error("drawing the chores", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "I cannot draw the chores just now."}
	}
	return squirrel.Turn{
		Who:   squirrel.SpeakerBuddy,
		Words: withNotice(ctx, opts, personID, "the chores", choreLead(len(sh.Cards)), sh.Cards),
		Shown: body,
	}
}

// makeOne is how you make one from nothing: a sentence, because the dock already
// understands it and a chip would have nowhere better to lead.
//
// Said only when there is nothing there.
const makeOne = "Tell me another like every 2 weeks: descale the kettle."

// choreCard is one chore, drawn the one way. Shared by the list and the reply to
// making a new one, so the two cannot grow different buttons.
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

// saidAboutAChore is its own function rather than the offer's: a chore comes back
// whatever you do about it, and "stop asking" must not read like finishing it.
//
// The name comes from the stored chore rather than the form, so the record cannot
// say something the press only claimed.
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

// pickView is a question with its answers on it. One form and one submit: a
// picker that wrote a turn per press would fill the record with the sound of
// somebody deciding.
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

// pickNumbers and pickUnits are what the interval picker offers. Six numbers and
// three units, and no way to type one: `every 9 weeks` is a sentence, and
// ParseEvery accepts fortnights, quarters and years through it.
var (
	pickNumbers = []string{"1", "2", "3", "4", "6", "8"}
	pickUnits   = []string{"days", "weeks", "months"}
	// The day a chore comes back on. "any day" first and chosen by default, so
	// naming a day is something you add rather than dismiss.
	//
	// Only offered against weeks: a day is meaningless on "every 3 days" and wrong on
	// "every 6 months".
	pickDays = []string{"any day", "mon", "tue", "wed", "thu", "fri", "sat", "sun"}
)

// askHowOften is the question, as one form with two rows.
func askHowOften(action string, fields map[string]string, count, unit, day string) squirrel.Turn {
	body, err := json.Marshal(drawn{Pick: &pickView{
		Action: action,
		Fields: fields,
		Do:     "that's it",
		Rows: []pickRow{
			{Lead: "every", Name: "count", Options: pickNumbers, Chosen: count},
			{Lead: "of these", Name: "unit", Options: pickUnits, Chosen: unit},
			// The day a chore comes back on. Unmarked unless the chore actually has one:
			// "any day" is the first option rather than a chosen one, because a mark is a
			// fact about this chore and not a default.
			{Lead: "on a", Name: "day", Options: pickDays, Chosen: day},
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

// rhythmOf is the interval a chore has now, as the picker's own two answers.
//
// Anything not landing on an offered pair leaves both empty rather than rounding.
// Units are tried largest first: days last, or 14 days answers "14".
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

// unitStep is how long each offered unit is. Thirty days for a month, exactly as
// the core reads it. See unitDurations in internal/squirrel/intent.go —
// TestThePickerAndTheSentenceAgree notices if they drift.
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

// composeEvery turns the picker's two answers into an interval through the same
// parser a typed sentence goes through. Both answers are checked against what was
// offered, because they arrive from a form.
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
func tasksTurn(ctx context.Context, s Store, opts Options, personID int64, name string, from int) squirrel.Turn {
	// Over-read by the offset, because Tasks caps rather than skips. A list
	// this size is one query either way.
	all, _, err := s.Tasks(ctx, personID, from+listLimit+1)
	if err != nil {
		slog.Error("reading what you decided", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "I cannot reach the tasks just now."}
	}
	if len(all) == 0 {
		return squirrel.Turn{
			Who:   squirrel.SpeakerBuddy,
			Words: "Nothing decided yet. A task is a note you said yes to.",
		}
	}
	items, more := slice(all, from)
	if len(items) == 0 {
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "That is all of them."}
	}

	sh := drawn{Place: name}
	for _, it := range items {
		v := toView(it)
		row := map[string]string{"id": strconv.FormatInt(v.ID, 10)}
		sh.Cards = append(sh.Cards, cardView{
			// A task wears the notebook's page tab, which is the device this
			// system already owns for "this row was decided on".
			Kind: "task", Title: v.Text, Meta: "decided " + v.When, Photo: v.Photo,
			Acts: []actView{
				{Label: "did it", Action: "/tasks/act", Style: "did", Fields: with(row, "act", "done")},
				// "back" rather than "later": this is not a deferral, it is a
				// decision reversed. The class matters twice — the tasks are
				// never late, and `later` has the word inside it.
				{Label: "not a task", Action: "/tasks/act", Style: "back", Fields: with(row, "act", "untask")},
			},
		})
	}
	// The way to what you set aside, from the room a task is set aside out of.
	// A press, because a shelf is drawn where you are standing rather than
	// entered — so this one draws it here, in the tasks.
	sh.Chips = []turnChip{{
		Label: "what you set aside", Action: "/notes/shelf",
		Fields: map[string]string{"shelf": "held"},
	}}
	if more {
		sh.Chips = append(sh.Chips, theRest("tasks", from+listLimit))
	}

	body, err := json.Marshal(sh)
	if err != nil {
		slog.Error("drawing the tasks", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "I cannot draw the tasks just now."}
	}
	return squirrel.Turn{
		Who:   squirrel.SpeakerBuddy,
		Words: withNotice(ctx, opts, personID, "the tasks", taskLead(len(sh.Cards)), sh.Cards),
		Shown: body,
	}
}

func taskLead(n int) string {
	if n == 1 {
		return "One thing you decided."
	}
	return fmt.Sprintf("%d things you decided.", n)
}

// saidAboutATask is what the two of you said about one. "Not a task" is a note
// that went back to being a note, and must not read as a failure.
func saidAboutATask(act, text string) []squirrel.Turn {
	if text == "" {
		text = "that"
	}
	switch act {
	case "done":
		return []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "did it — " + text},
			{Who: squirrel.SpeakerBuddy, Words: squirrel.Say(squirrel.SayingDid, now())},
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

// agendaTurn holds only what is still coming: nothing past, nothing done, and
// nothing here has been missed.
func agendaTurn(ctx context.Context, s Store, personID int64, name string, from int) squirrel.Turn {
	coming, err := s.Upcoming(ctx, personID, now(), from+listLimit+1)
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

	page, more := slice(coming, from)
	if len(page) == 0 {
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "That is all of them."}
	}

	sh := drawn{Place: name}
	for _, m := range page {
		row := map[string]string{"id": strconv.FormatInt(m.ID, 10)}
		// The core's own sentence, shared with chat and the notification, so the three
		// cannot drift about when to leave. A ticket, not a card — see DESIGN.md, The Six
		// Bodies.
		card := cardView{Kind: "at", Title: m.Label, Meta: squirrel.LeaveWords(m)}
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
	if more {
		sh.Chips = append(sh.Chips, theRest("at", from+listLimit))
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

// fixedPointTurn is one appointment and what is pointing at it. What to take is
// its own line, and absent when there is nothing to take.
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

// calView is the day picker. One form and one submit, like the interval picker;
// turning to another month posts on its own and re-asks rather than writing a
// second question into the record.
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

// pickTimes are the three the picker puts within one press. They are a
// shortcut and not the vocabulary: the field beside them takes any time, which
// is what an appointment at 11:15 needs and what three chips could never say.
var pickTimes = []string{"09:00", "14:30", "18:00"}

// aTimeOfDay is the guard on a value that arrives from a form: 24-hour, on the
// clock, and nothing else. It was a membership test against the three chips,
// which is why the field could not have existed beside them.
func aTimeOfDay(at string) bool {
	if len(at) != 5 || at[2] != ':' {
		return false
	}
	h, err := strconv.Atoi(at[:2])
	if err != nil || h > 23 {
		return false
	}
	m, err := strconv.Atoi(at[3:])
	return err == nil && m <= 59
}

// askForADay is the question. Monday first. Days already gone are drawn and not
// offered, and there is no way back past this month.
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

// pileTurn is one note, with the four ways out of it. One at a time: a list of
// things to decide about is a list you are behind on.
func pileTurn(ctx context.Context, s Store, opts Options, personID, after int64, name string) squirrel.Turn {
	items, _, err := s.OpenItemsAfter(ctx, personID, after, 1)
	if err != nil {
		slog.Error("reading the pile", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "I cannot reach the pile just now."}
	}
	if len(items) == 0 {
		// An empty pile is exactly when the other two places are worth reaching, and it
		// was the one branch that could not reach them: the chips hung off the drawn
		// card.
		words := "Nothing to decide about. Anything you tell me lands here."
		if after != 0 {
			// The bottom, reached by skipping. Not the same as an empty pile:
			// what is above you is still there.
			words = "That is the end of them. The ones you skipped are still there."
		}
		return sayWithChips(words, elsewhereFromThePile())
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
				// Four, and only four. The three that ask a question rather than ending the note
				// are behind `something else?` and answer as a turn.
			},
		}},
		Chips: append([]turnChip{
			// Later is not a decision. It leaves the note where it was and
			// hands you the next one.
			{
				Label: "later", Action: "/pile/later",
				Fields: map[string]string{"after": strconv.FormatInt(v.ID, 10)},
			},
			// The three questions, behind one press. A chip rather than a
			// button on the card, because it is a thing you say about the note
			// rather than a thing you do to it — the same reason `later` is a
			// chip and DONE is not.
			{
				Label: "something else?", Action: "/pile/more",
				Fields: map[string]string{"id": strconv.FormatInt(v.ID, 10)},
			},
		}, elsewhereFromThePile()...),
	}

	body, err := json.Marshal(sh)
	if err != nil {
		slog.Error("drawing the pile", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: v.Text}
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: squirrel.Say(squirrel.SayingHere, now()), Shown: body}
}

// saidAboutANote carries the way back with the answer, because the card is about
// to be scrollback and scrollback carries no controls.
func saidAboutANote(act, text string, id int64, was string) []squirrel.Turn {
	// The reply varies by the day, the way the slot's own line does. This is
	// the most repeated exchange in the product — four sentences met several
	// times in a single sitting — and a conversation whose every reply is the
	// same word is a conversation with a machine.
	words, reply := "", ""
	switch act {
	case "done":
		words, reply = "done — "+text, squirrel.Say(squirrel.SayingDid, now())
	case "keep":
		words, reply = "keep — "+text, squirrel.Say(squirrel.SayingKept, now())
	case "drop":
		words, reply = "drop — "+text, squirrel.Say(squirrel.SayingDropped, now())
	case "task":
		words, reply = "a task — "+text, squirrel.Say(squirrel.SayingDecided, now())
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

// sayView is a question whose answer is words. Its own box rather than the dock:
// the dock keeps everything you type, and these words replace something.
type sayView struct {
	Action string `json:"action"`
	// Field is what the box is called when posted. The routes do not agree —
	// rewording takes `text`, Buddy `said`, search `q` — and renaming them would
	// break the one URL a person might have typed.
	Field string `json:"field"`
	// Label is what a screen reader calls the box. The question above it is a
	// paragraph, not a label, so without this every box in the product is
	// announced as the one that was written first.
	Label  string            `json:"label,omitempty"`
	Fields map[string]string `json:"fields,omitempty"`
	Was    string            `json:"was,omitempty"`
	Do     string            `json:"do"`
}

// askForWords is the reword question, with what it says now already in the box.
func askForWords(action string, fields map[string]string, was, does string) squirrel.Turn {
	body, err := json.Marshal(drawn{Say: &sayView{
		Action: action, Field: "text", Label: "How it should read",
		Fields: fields, Was: was, Do: does,
	}})
	if err != nil {
		slog.Error("drawing the question", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "Tell me again, in the box below."}
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "How should it read?", Shown: body}
}

// askWhyNot is the three ways a thing can be out of your hands.
//
// Which of the three is a fact about why, not about what happened, so they are
// three answers to one question rather than three verbs on a card.
func askWhyNot(id int64) squirrel.Turn {
	chips := make([]turnChip, 0, len(squirrel.Held))
	for _, state := range squirrel.Held {
		chips = append(chips, turnChip{
			Label: squirrel.HeldWords[state], Action: "/held/act",
			Fields: map[string]string{
				"id": strconv.FormatInt(id, 10), "aside": string(state), "from": "thread",
			},
		})
	}
	body, err := json.Marshal(drawn{Chips: chips})
	if err != nil {
		slog.Error("drawing the whys", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "Why not?"}
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "Why not?", Shown: body}
}

// cutView is a proposal to split a note into pieces, travelling as repeated
// inputs because a card's fields are a map.
//
// Nothing is written when this is drawn, and a proposal in scrollback has lost
// its button by the live edge rule, so a stale one cannot be applied.
type cutView struct {
	Action string   `json:"action"`
	ID     int64    `json:"id"`
	Pieces []string `json:"pieces"`
	Do     string   `json:"do"`
}

// proposeInThread is what Buddy says when a note looks like several things.
func proposeInThread(id int64, pieces []string) squirrel.Turn {
	body, err := json.Marshal(drawn{Cut: &cutView{
		Action: "/pile/split", ID: id, Pieces: pieces, Do: "use these",
	}})
	if err != nil {
		slog.Error("drawing the pieces", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "I could not read it as more than one thing."}
	}
	// What you actually typed is kept beside the pieces, and Buddy says so
	// before you press rather than after: dropping it would make a machine's
	// reading of your words the only surviving version of them.
	return squirrel.Turn{
		Who:   squirrel.SpeakerBuddy,
		Words: "Is this what you meant? The note itself is kept either way.",
		Shown: body,
	}
}

// searchTurn is one search over both kinds of thing, in every state. A result
// says which state it is in, because without that an open task reported itself as
// being in the pile and was offered the pile's verbs.
func searchTurn(ctx context.Context, s Store, personID int64, q string) squirrel.Turn {
	items, more, err := s.SearchItems(ctx, personID, q, searchLimit)
	if err != nil {
		slog.Error("searching", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "I cannot look just now."}
	}
	chores, err := s.SearchChores(ctx, personID, q, choreHits)
	if err != nil {
		slog.Error("searching the chores", "error", err)
	}
	if len(items) == 0 && len(chores) == 0 {
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "Nothing with that word in it."}
	}

	// Results are not cards. A hit is something you are finding, not something you
	// act on; one opens into a real card with real buttons, which is the same
	// one-thing-is-live rule the conversation runs on.
	sh := drawn{}
	for _, c := range chores {
		v := toChoreView(c)
		sh.Hits = append(sh.Hits, hitView{
			Title: v.Name, Meta: "chore · " + choreMeta(v),
			// A form, because hitView has no href — and /open redirects into
			// the room, so this lands where the rail would take you.
			Action: "/open", Fields: map[string]string{"where": "chores"},
		})
	}
	for _, it := range items {
		v := toView(it)
		sh.Hits = append(sh.Hits, hitView{
			Title: v.Text, Meta: whereItIs(v),
			Action: "/find/open",
			Fields: map[string]string{"id": strconv.FormatInt(v.ID, 10)},
		})
	}
	if more {
		// That there is more, and not how much. Narrowing the words is the honest offer:
		// there is no second page of search.
		sh.Chips = []turnChip{{Label: "say it more exactly", Action: "/find/ask"}}
	}

	body, err := json.Marshal(sh)
	if err != nil {
		slog.Error("drawing the results", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: "I cannot draw what I found."}
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: foundLead(len(sh.Hits)), Shown: body}
}

// whereItIs says which of the seven a result is in, because a result read
// without it is a note whose verbs would be wrong.
func whereItIs(v noteView) string {
	where := v.State
	if v.Task {
		where = "a task"
	}
	return v.When + " · " + where
}

func foundLead(n int) string {
	if n == 1 {
		return "One thing with that word in it."
	}
	return fmt.Sprintf("%d things with that word in it.", n)
}

// findHandler is a POST like every other thing you say: a search goes into the
// record, and what is in the turn is what was found when you asked.
func findHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, backToTheRoom(r), http.StatusSeeOther)
			return
		}
		q := strings.TrimSpace(r.FormValue("q"))
		if q == "" {
			http.Redirect(w, r, backToTheRoom(r), http.StatusSeeOther)
			return
		}
		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: q},
			searchTurn(r.Context(), s, personID, q),
		}), backToTheRoom(r))
	}
}

// findOpenHandler turns one search result into a card you can act on, built from
// the note's real state. Nothing about search gets its own vocabulary.
func findOpenHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, backToTheRoom(r), http.StatusSeeOther)
			return
		}
		id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
		if err != nil || id < 1 {
			http.Redirect(w, r, backToTheRoom(r), http.StatusSeeOther)
			return
		}
		it, found, err := s.ItemByID(r.Context(), personID, id)
		if err != nil {
			fail(w, err)
			return
		}
		if !found {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		v := toView(it)
		row := map[string]string{
			"id": strconv.FormatInt(v.ID, 10), "was": v.State, "from": "thread",
		}
		card := cardView{Title: v.Text, Photo: v.Photo, Meta: whereItIs(v)}
		if v.Task {
			card.Kind = "task"
		}
		card.Acts = []actView{
			{Label: "DONE", Action: "/pile/act", Style: "did", Fields: with(row, "act", "done")},
			{Label: "KEEP", Action: "/pile/act", Style: "go", Fields: with(row, "act", "keep")},
			{Label: "DROP", Action: "/pile/act", Style: "stop", Fields: with(row, "act", "drop")},
		}
		body, err := json.Marshal(drawn{Cards: []cardView{card}})
		if err != nil {
			slog.Error("drawing a result", "error", err)
			http.Redirect(w, r, backToTheRoom(r), http.StatusSeeOther)
			return
		}
		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: v.Text},
			{Who: squirrel.SpeakerBuddy, Words: "That one.", Shown: body},
		}), backToTheRoom(r))
	}
}

// atTheEdge draws what is true now.
//
// The same turn machinery the record uses, with two differences that matter.
// Nothing here has an id, because nothing here is a row anybody can point at —
// and two live turns both called turn-0 would be one element as far as the
// script is concerned. Everything here is live, because the whole of what it
// is for is being the thing you can act on.
func atTheEdge(ctx context.Context, where string, turns []squirrel.Turn) []turnView {
	out := turnViews(ctx, turns)
	for i := range out {
		out[i].ID = 0
		out[i].Live = true
		out[i].Day, out[i].When = "", ""
		// Said here, because nothing said it. turnView.Room comes off the
		// stored turn and an edge turn was never stored, so every form it
		// draws would post with no room on it and every press would land in
		// everything — the defect #221 was about, arriving by a new road.
		out[i].Room = where
	}
	return out
}

// saidAndDone takes the controls off the record when there is an edge below it.
//
// Only the newest thing may be acted on — that rule is older than the edge and
// is why a question from this morning is a thing that was asked rather than a
// thing to answer. With the room's list drawn below the conversation, the
// newest thing is down there, and a live turn in the scrollback would be a
// second set of buttons for the same room.
func saidAndDone(said []turnView, hasEdge bool) []turnView {
	if !hasEdge {
		return said
	}
	for i := range said {
		said[i].Live = false
	}
	return said
}

// edgeOnly answers with the edge and nothing around it.
//
// The same template the page uses, so there is one description of a list
// rather than two that can disagree — the argument every fragment on this
// screen is written under.
func edgeOnly(w http.ResponseWriter, r *http.Request, edge []turnView) {
	t := pages["thread"]
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	// Said back, so the script can tell this answer from a page. A handler that
	// did not know about X-Edge would answer with the whole screen, and the
	// script would paste a page into the element it asked about — which is what
	// happened on the front door before this header existed.
	w.Header().Set("X-Edge", "1")
	for _, v := range edge {
		if err := t.ExecuteTemplate(w, "turn", v); err != nil {
			slog.Error("drawing the edge", "error", err)
			return
		}
	}
}
