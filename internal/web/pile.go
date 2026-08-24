package web

import (
	"fmt"
	"net/http"
	"strings"
)

// The deck shows one card. The second row is never rendered; it is read only so
// that "is there more" can be answered without a count, which is the same
// device OpenItems uses and for the same reason.
const pileLimit = 1

// Mux is the routing surface the screen needs from the shared server.
type Mux interface {
	Get(pattern string, h http.HandlerFunc)
	Post(pattern string, h http.HandlerFunc)
}

// Mount registers the screen, or refuses. An empty identity is not a
// misconfiguration to warn about and continue past: the pile is every thought
// you have ever had at this bot.
func Mount(m Mux, s Store, opts Options) error {
	if opts.Identity == "" {
		return fmt.Errorf("refusing to mount the pile: WEB_IDENTITY is empty")
	}
	if opts.Owner == nil {
		return fmt.Errorf("refusing to mount the pile: no owner")
	}
	if opts.Spool == nil {
		// A screen that captures with nowhere durable to put the words is the
		// gap this closes. Refusing at mount rather than at the first capture,
		// because the first capture is the worst moment to find out.
		return fmt.Errorf("refusing to mount the pile: no spool")
	}
	// `{$}` and not `/`: a bare "/" is Go's catch-all, and the home screen would
	// then answer for every URL nobody else claimed — including the typos, which
	// would arrive looking like a working page.
	m.Get("/{$}", guard(opts, homeHandler(s, opts)))
	m.Get("/pile", guard(opts, pileHandler(s, opts)))
	// The slot. Behind the origin check like every other write here: the
	// identity says who is asking, sameOrigin says which page asked.
	m.Post("/capture", guard(opts, sameOrigin(captureHandler(s, opts))))
	// A photograph, behind the same guard as everything else: a picture of a
	// letter is at least as private as the note beside it.
	if opts.Photos != nil {
		m.Get("/photo/{id}", guard(opts, photoHandler(s, opts)))
	}
	m.Post("/mood", guard(opts, sameOrigin(moodHandler(s, opts))))
	// The one thing's three answers. Behind the origin check like every other
	// write here.
	m.Post("/now/act", guard(opts, sameOrigin(nowActHandler(s, opts))))
	// I can't start. Its own route rather than a fourth act, because it is the
	// one answer that is about you rather than about the thing.
	m.Post("/now/stuck", guard(opts, sameOrigin(nowStuckHandler(s, opts))))
	// Where to reach this browser. Only mounted when there is a key to
	// subscribe with — a route that always answers 400 is a route that teaches
	// the client to stop asking.
	if opts.PushKey != "" {
		m.Post("/push/subscribe", guard(opts, sameOrigin(pushSubscribeHandler(s, opts))))
	}
	// Both writes carry the origin check as well as the identity one: the
	// identity says who is asking, sameOrigin says which page asked.
	m.Post("/pile/act", guard(opts, sameOrigin(actHandler(s, opts))))
	m.Post("/pile/chore", guard(opts, sameOrigin(choreHandler(s, opts))))
	m.Post("/pile/fix", guard(opts, sameOrigin(fixHandler(s, opts))))
	// Four thoughts captured as one note, offered as four. Both halves —
	// asking, and keeping what was asked for — because they are the only two
	// things you can do to a proposal.
	m.Post("/pile/split", guard(opts, sameOrigin(splitHandler(s, opts))))
	// Buddy. A real route rather than a JavaScript-only construct, so it works
	// with scripting off, deep-links, and survives a reload — and so pile.js
	// has something real to upgrade into a sheet.
	m.Get("/buddy", guard(opts, coachHandler(s, opts)))
	m.Post("/buddy/say", guard(opts, sameOrigin(coachSayHandler(s, opts))))
	// Closing is a write because it forgets the conversation, and a write here
	// carries the origin check like every other one.
	m.Post("/buddy/close", guard(opts, sameOrigin(coachCloseHandler(opts))))
	// "That landed badly." One press, about the thing you just read.
	m.Post("/buddy/badly", guard(opts, sameOrigin(coachBadlyHandler(s, opts))))
	// A proposal, applied because it was pressed. Four things and no more —
	// see coachDoHandler for why it is a switch rather than a dispatcher.
	m.Post("/buddy/do", guard(opts, sameOrigin(coachDoHandler(s, opts))))
	// It was /coach for the release it shipped in. A bookmark that dies
	// quietly is worse than a redirect nobody notices — the same reasoning
	// /pile/chores already gets, and the same status.
	m.Get("/coach", guard(opts, func(w http.ResponseWriter, r *http.Request) {
		to := "/buddy"
		if q := r.URL.RawQuery; q != "" {
			to += "?" + q
		}
		http.Redirect(w, r, to, http.StatusMovedPermanently)
	}))
	// A step finished, or a sequence thrown away. One route because they are
	// the only two things you can do to a breakdown.
	m.Post("/steps", guard(opts, sameOrigin(stepsHandler(s, opts))))
	// What you cannot act on. Its own page rather than a fourth door: see
	// held.go for why home does not carry it.
	// What is coming, and one of them. The notification lands on the second:
	// see sw.js, and DESIGN.md for what that replaced.
	m.Get("/at", guard(opts, atHandler(s, opts)))
	m.Get("/at/{id}", guard(opts, atOneHandler(s, opts)))
	m.Post("/at/{id}/note", guard(opts, sameOrigin(atNoteHandler(s, opts))))
	m.Post("/at/{id}/detach", guard(opts, sameOrigin(atDetachHandler(s, opts))))
	m.Get("/held", guard(opts, heldHandler(s, opts)))
	m.Post("/held/act", guard(opts, sameOrigin(heldActHandler(s, opts))))
	// How you have been, and only when asked for by name. Nothing links here
	// except the check-in you just answered.
	m.Get("/moods", guard(opts, moodsHandler(s, opts)))
	m.Get("/chores", guard(opts, choresHandler(s, opts)))
	m.Get("/kept", guard(opts, keptHandler(s, opts)))
	// Stopping. No store, on purpose: a route that cannot read cannot start
	// keeping score of how much you did before you pressed it.
	m.Get("/enough", guard(opts, enoughHandler(opts)))
	m.Get("/tasks", guard(opts, tasksHandler(s, opts)))
	m.Get("/tasks/done", guard(opts, archiveHandler(s, opts)))
	m.Post("/tasks/act", guard(opts, sameOrigin(taskActHandler(s, opts))))
	m.Post("/tasks/new", guard(opts, sameOrigin(newTaskHandler(s, opts))))
	m.Post("/chores/act", guard(opts, sameOrigin(choreActHandler(s, opts))))
	m.Post("/chores/new", guard(opts, sameOrigin(newChoreHandler(s, opts))))
	m.Post("/timer", guard(opts, sameOrigin(timerHandler(s, opts))))
	// The chores screen lived here for its whole life. A bookmark that dies
	// quietly is worse than a redirect nobody notices.
	m.Get("/pile/chores", guard(opts, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/chores", http.StatusMovedPermanently)
	}))
	// Outside the guard, like the worker below and for the same reason: a
	// browser fetches a manifest without the cookies that carry the identity,
	// and one that answers 403 leaves an installed app with no icon and no
	// explanation. It names the app and lists four PNGs — there is nothing in
	// it to protect.
	m.Get("/manifest.webmanifest", manifestHandler())
	// Not behind the guard: a browser fetches the worker without the cookies
	// that carry the identity, and a worker that 302s to a login page is a
	// worker that never installs. It contains no notes — only which files to
	// keep and what to say when the network is gone.
	m.Get("/sw.js", swHandler())
	m.Get("/static/", staticHandler())
	return nil
}

func pileHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		undo := undoFrom(r.URL.Query())
		clash := clashFrom(r.URL.Query())
		if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
			searchInto(w, r, s, opts, personID, q, undo, clash)
			return
		}
		// The cursor is skipping, and it lives here rather than in the store's
		// idea of the pile: a note skipped past is untouched, and reloading
		// without the parameter is how you get back to the top.
		after := cursorFrom(r.URL.Query())
		items, more, err := s.OpenItemsAfter(r.Context(), personID, after, pileLimit)
		if err != nil {
			fail(w, err)
			return
		}
		v := view{Here: "pile", More: more, Undo: undo, After: after, Clash: clash}
		if len(items) == 0 {
			// Nothing older is not an empty pile. Everything skipped past is
			// still open, and a page that said "nothing in the pile" here
			// would be describing the cursor rather than the pile.
			if after != 0 {
				renderWith(w, r, s, opts, "bottom", v)
				return
			}
			renderWith(w, r, s, opts, "empty", v)
			return
		}
		n := toView(items[0])
		v.Note = &n
		v.Splittable = splittable(opts, n.Text)
		v.HeldChips = heldChips()
		renderWith(w, r, s, opts, "pile", v)
	}
}

// splittable reports whether the note on the card is worth asking about.
//
// A free check, and it is what keeps a model off every note in the pile: the
// press is only drawn on the ones that look like several things, so the
// expensive part never happens for the ordinary ones.
func splittable(opts Options, text string) bool {
	return opts.Split != nil && opts.Splittable != nil && opts.Splittable(text)
}

// pileInto renders the pile with a proposal attached to the top card.
//
// The same page the pile handler draws, so the proposal appears where the note
// is rather than on a screen of its own — the confirmation surface is the card
// that is already there, which is what the architecture asked for.
func pileInto(w http.ResponseWriter, r *http.Request, s Store, opts Options, personID int64, sp *splitView) {
	items, more, err := s.OpenItemsAfter(r.Context(), personID, 0, pileLimit)
	if err != nil {
		fail(w, err)
		return
	}
	if len(items) == 0 {
		http.Redirect(w, r, "/pile", http.StatusSeeOther)
		return
	}
	n := toView(items[0])
	renderWith(w, r, s, opts, "pile", view{
		Here: "pile", More: more, Note: &n, Split: sp,
		Splittable: splittable(opts, n.Text), HeldChips: heldChips(),
	})
}

// searchLimit caps the result list. The cap is what makes "there is more"
// truthful; without it the line would appear over a complete list, which is a
// false claim in the one place the counting rule is most likely to leak.
const searchLimit = 6

// choreHits caps the chores a search answers with. Short on purpose: a chore
// list is short, and this is a way to reach one rather than a second list to
// read.
const choreHits = 3

func searchInto(w http.ResponseWriter, r *http.Request, s Store, opts Options, personID int64, q string, undo *undoView, clash bool) {
	items, more, err := s.SearchItems(r.Context(), personID, q, searchLimit)
	if err != nil {
		fail(w, err)
		return
	}
	// One search, both kinds of thing. The lid carries one field on every
	// screen, and a person typing a word has not first decided whether the word
	// belongs to a note or to a chore.
	chores, err := s.SearchChores(r.Context(), personID, q, choreHits)
	if err != nil {
		fail(w, err)
		return
	}
	v := view{Here: "pile", Query: q, More: more, Undo: undo, Clash: clash}
	for _, c := range chores {
		v.Chores = append(v.Chores, toChoreView(c))
	}
	for _, it := range items {
		v.Results = append(v.Results, toView(it))
	}
	renderWith(w, r, s, opts, "results", v)
}
