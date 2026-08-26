package web

import (
	"strconv"

	"fmt"
	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"net/http"
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

// Mount registers the screen, or refuses. A missing gate is not a
// misconfiguration to warn about and continue past: the pile is every thought
// you have ever had at this bot.
func Mount(m Mux, s Store, opts Options) error {
	// Refused rather than defaulted, and it is the only value here that would
	// be dangerous to default. Everything else missing degrades to less
	// product — no coach, no camera, no push. An empty required group would
	// degrade to more access, which is the one direction a default must never
	// go.
	if opts.RequiredGroup == "" {
		return fmt.Errorf("refusing to mount the pile: WEB_REQUIRED_GROUP is empty")
	}
	if opts.Gate == nil {
		return fmt.Errorf("refusing to mount the pile: no way in")
	}
	if opts.Sessions == nil {
		// guard would refuse every request forever, which looks from the
		// outside like an outage nobody can explain.
		return fmt.Errorf("refusing to mount the pile: no sessions")
	}
	if opts.Login == nil {
		return fmt.Errorf("refusing to mount the pile: nothing turns a login into a person")
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
	m.Get("/{$}", guard(opts, threadHandler(s, opts)))
	// The slot. Behind the origin check like every other write here: the
	// identity says who is asking, sameOrigin says which page asked.
	m.Post("/capture", guard(opts, sameOrigin(captureHandler(s, opts))))
	// A door being pressed. A POST rather than a link — see openHandler.
	m.Post("/open", guard(opts, sameOrigin(openHandler(s, opts))))
	// A photograph, behind the same guard as everything else: a picture of a
	// letter is at least as private as the note beside it.
	if opts.Photos != nil {
		m.Get("/photo/{id}", guard(opts, photoHandler(s, opts)))
		// The card asks for this one. See thumbHandler.
		m.Get("/photo/{id}/thumb", guard(opts, thumbHandler(s, opts)))
	}
	m.Post("/mood", guard(opts, sameOrigin(threadMoodHandler(s, opts))))
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
	// Starting fresh, when Buddy offers you back a run you were part way
	// through. Its other answer — carry on — is an ordinary door press and
	// needs no route of its own.
	m.Post("/place/fresh", guard(opts, sameOrigin(freshHandler(s, opts))))
	// Triage, in the conversation: skipping one, and changing your mind.
	m.Post("/pile/later", guard(opts, sameOrigin(laterHandler(s, opts))))
	m.Post("/pile/undo", guard(opts, sameOrigin(undoHandler(s, opts))))
	// The three questions a note can be asked, rather than the three verbs
	// that end it. Each reuses the shape the chores already have.
	m.Post("/pile/often", guard(opts, sameOrigin(askAbout(s, opts, func(it squirrel.Item) squirrel.Turn {
		return askHowOften("/pile/chore",
			map[string]string{"id": strconv.FormatInt(it.ID, 10), "from": "thread"}, "", "", "")
	}))))
	m.Post("/pile/reword", guard(opts, sameOrigin(askAbout(s, opts, func(it squirrel.Item) squirrel.Turn {
		return askForWords("/pile/fix",
			map[string]string{"id": strconv.FormatInt(it.ID, 10), "from": "thread"},
			it.RawText, "say it this way")
	}))))
	m.Post("/find", guard(opts, sameOrigin(findHandler(s, opts))))
	// One result, opened into a card you can act on. A hit is quiet because it
	// is a thing you are finding; this is the moment it becomes a thing you are
	// deciding about. See findOpenHandler.
	m.Post("/find/open", guard(opts, sameOrigin(findOpenHandler(s, opts))))
	// The three questions a note can be asked, behind one press. See
	// moreHandler for why they are a turn rather than a panel.
	m.Post("/pile/more", guard(opts, sameOrigin(moreHandler(s, opts))))
	m.Post("/pile/why", guard(opts, sameOrigin(askAbout(s, opts, func(it squirrel.Item) squirrel.Turn {
		return askWhyNot(it.ID)
	}))))
	m.Post("/pile/chore", guard(opts, sameOrigin(choreHandler(s, opts))))
	m.Post("/pile/fix", guard(opts, sameOrigin(fixHandler(s, opts))))
	// Four thoughts captured as one note, offered as four. Both halves —
	// asking, and keeping what was asked for — because they are the only two
	// things you can do to a proposal.
	m.Post("/pile/split", guard(opts, sameOrigin(splitHandler(s, opts))))
	// Buddy. A page until 25 August 2026, and turns since: the sheet brought a
	// conversation with it because there was not one to join, and there is
	// one now. Closing went with it — you stop talking, the way you stop
	// talking to anyone.
	m.Post("/buddy/ask", guard(opts, sameOrigin(coachAskHandler(s, opts))))
	// Looking something up. A chip rather than a field in the lid — see
	// findAskHandler.
	m.Post("/find/ask", guard(opts, sameOrigin(findAskHandler(s, opts))))
	// What Squirrel thinks it knows about you, and the way to throw it away.
	// A POST for both: reading it is something you asked, and it goes into the
	// record like anything else you say. See knowing.go.
	m.Post("/knowing", guard(opts, sameOrigin(knowingHandler(s, opts))))
	m.Post("/knowing/forget", guard(opts, sameOrigin(forgetKnowingHandler(s, opts))))
	// A new one, at every door. Each asks for words; the routes underneath are
	// the ones the screens posted to. See newone.go.
	m.Post("/chores/ask", guard(opts, sameOrigin(askNameHandler(s, opts,
		"a new chore", "What should come back?", "/chores/name", "name", "next"))))
	m.Post("/chores/name", guard(opts, sameOrigin(choreNameHandler(s, opts))))
	m.Post("/tasks/ask", guard(opts, sameOrigin(askNameHandler(s, opts,
		"a new task", "What did you decide to do?", "/tasks/new", "text", "keep it"))))
	m.Post("/pile/ask", guard(opts, sameOrigin(askNameHandler(s, opts,
		"put something down", "What is it?", "/capture", "text", "keep it"))))
	m.Post("/buddy/say", guard(opts, sameOrigin(coachSayHandler(s, opts))))
	// "That landed badly." One press, about the thing you just read.
	m.Post("/buddy/badly", guard(opts, sameOrigin(coachBadlyHandler(s, opts))))
	// A proposal, applied because it was pressed. Four things and no more —
	// see coachDoHandler for why it is a switch rather than a dispatcher.
	m.Post("/buddy/do", guard(opts, sameOrigin(coachDoHandler(s, opts))))
	// It was /coach, then /buddy, and now it is the conversation. A bookmark
	// that dies quietly is worse than a redirect nobody notices — the same
	// reasoning /pile/chores already gets, and the same status. The query
	// string is dropped rather than carried: nothing at the other end reads
	// one any more.
	for _, gone := range []string{"/coach", "/buddy"} {
		m.Get(gone, guard(opts, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/", http.StatusMovedPermanently)
		}))
	}
	// A step finished, or a sequence thrown away. One route because they are
	// the only two things you can do to a breakdown.
	m.Post("/steps", guard(opts, sameOrigin(stepsHandler(s, opts))))
	// What you cannot act on. Its own page rather than a fourth door: see
	// held.go for why home does not carry it.
	// What is coming, and one of them. The notification lands on the second:
	// see sw.js, and DESIGN.md for what that replaced.
	m.Get("/at/{id}", guard(opts, atOneHandler(s, opts)))
	// One fixed point, drawn into the conversation. See atOpenHandler.
	m.Post("/at/open", guard(opts, sameOrigin(atOpenHandler(s, opts))))
	// Which day, and what time. See askForADay.
	m.Post("/at/new", guard(opts, sameOrigin(atNewHandler(s, opts))))
	m.Post("/at/make", guard(opts, sameOrigin(atMakeHandler(s, opts))))
	m.Post("/at/{id}/note", guard(opts, sameOrigin(atNoteHandler(s, opts))))
	m.Post("/at/{id}/detach", guard(opts, sameOrigin(atDetachHandler(s, opts))))
	// The page went on 25 August 2026 — what you set aside is a message now,
	// reached from the pile's turn. Setting one aside and picking it back up
	// stayed, and both answer with a turn. See elsewhere.go.
	m.Post("/held/act", guard(opts, sameOrigin(heldActHandler(s, opts))))
	// How you have been, and only when asked for by name. Nothing links here
	// except the check-in you just answered.
	m.Get("/moods", guard(opts, moodsHandler(s, opts)))
	// Stopping. No store, on purpose: a route that cannot read cannot start
	// keeping score of how much you did before you pressed it.
	m.Get("/enough", guard(opts, enoughHandler(s, opts)))
	m.Post("/tasks/act", guard(opts, sameOrigin(taskActHandler(s, opts))))
	m.Post("/tasks/new", guard(opts, sameOrigin(newTaskHandler(s, opts))))
	m.Post("/chores/act", guard(opts, sameOrigin(choreActHandler(s, opts))))
	// How often, as a number and a unit. See askHowOften.
	m.Post("/chores/often", guard(opts, sameOrigin(oftenHandler(s, opts))))
	m.Post("/chores/new", guard(opts, sameOrigin(newChoreHandler(s, opts))))
	m.Post("/timer", guard(opts, sameOrigin(timerHandler(s, opts))))
	// The chores screen lived here for its whole life. A bookmark that dies
	// quietly is worse than a redirect nobody notices.
	m.Get("/pile/chores", guard(opts, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
	}))
	// Outside the guard, like the worker below and for the same reason: a
	// browser fetches a manifest without the cookies that carry the identity,
	// and one that answers 403 leaves an installed app with no icon and no
	// explanation. It names the app and lists four PNGs — there is nothing in
	// it to protect.
	// The way in, and the three routes that work it. Outside the guard on
	// purpose and necessarily: a person with no session has to be able to get
	// one.
	//
	// Both writes still carry the origin check. It is a weaker claim here than
	// elsewhere — there is no session yet to ride on — but a cross-site POST
	// to /auth/in is a login started by somebody else's page, and a cross-site
	// POST to /auth/out signs you out of your own notes from a page you were
	// only reading.
	m.Get("/auth", gateHandler())
	m.Post("/auth/in", sameOrigin(beginHandler(opts)))
	m.Get("/auth/callback", backHandler(opts))
	m.Post("/auth/out", sameOrigin(outHandler(opts)))
	m.Get("/manifest.webmanifest", manifestHandler())
	// Not behind the guard: a browser fetches the worker without the cookies
	// that carry the identity, and a worker that 302s to a login page is a
	// worker that never installs. It contains no notes — only which files to
	// keep and what to say when the network is gone.
	m.Get("/sw.js", swHandler())
	m.Get("/static/", staticHandler())
	return nil
}

// splittable reports whether the note on the card is worth asking about.
//
// A free check, and it is what keeps a model off every note in the pile: the
// press is only drawn on the ones that look like several things, so the
// expensive part never happens for the ordinary ones.
func splittable(opts Options, text string) bool {
	return opts.Split != nil && opts.Splittable != nil && opts.Splittable(text)
}

// searchLimit caps the result list. The cap is what makes "there is more"
// truthful; without it the line would appear over a complete list, which is a
// false claim in the one place the counting rule is most likely to leak.
const searchLimit = 6

// choreHits caps the chores a search answers with. Short on purpose: a chore
// list is short, and this is a way to reach one rather than a second list to
// read.
const choreHits = 3
