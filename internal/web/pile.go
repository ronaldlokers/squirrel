package web

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// pileLimit is one card. The second row is read but never rendered, so "is there
// more" can be answered without a count — the same device OpenItems uses.
const pileLimit = 1

// Mux is the routing surface the screen needs from the shared server.
type Mux interface {
	Get(pattern string, h http.HandlerFunc)
	Post(pattern string, h http.HandlerFunc)
}

// Mount registers the screen, or refuses. A missing gate is not something to
// warn about and continue past: the pile is every thought you have ever had at
// this bot.
//
// Every write below carries both checks. The identity says who is asking;
// sameOrigin says which page asked.
// posting is what every press goes through: signed in, same origin, and
// carrying the room it was made in. Composed once rather than spelled out per
// route, because the room is invisible when it is forgotten — see
// inTheRoomItCameFrom.
func posting(opts Options, h http.HandlerFunc) http.HandlerFunc {
	return guard(opts, sameOrigin(inTheRoomItCameFrom(h)))
}

func Mount(m Mux, s Store, opts Options) error {
	// Refused rather than defaulted, and the only value here where a default would be
	// dangerous: everything else missing degrades to less product, and an empty
	// required group would degrade to more access.
	if opts.RequiredGroup == "" {
		return fmt.Errorf("refusing to mount the pile: WEB_REQUIRED_GROUP is empty")
	}
	// Every handler can find out who is asking, without fifty routes each
	// remembering to arrange it. See knowsYou.
	m = knowsYou(m, s, opts.Location)
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
	m.Get("/{$}", guard(opts, boardHandler(s, opts)))
	// Buddy's room, by its own name. The same handler as "/": the worked
	// example, the check-in and the offer all live on the thread, and a second
	// handler for the same room would be a second set of them.
	//
	// Go's ServeMux prefers the more specific pattern, so this wins over
	// /r/{room} without any ordering care here.
	m.Get("/r/everything", guard(opts, withRoom("everything", threadHandler(s, opts))))
	// Any other room. A GET, because entering a place is navigation — see
	// roomHandler.
	m.Get("/r/{room}", guard(opts, roomRoute(s, opts)))
	// The dock, in whichever room it was typed in. See fromTheDock.
	m.Post("/capture", posting(opts, captureHandler(s, opts)))
	// The door, as it was until 28 August 2026. See openHandler.
	m.Post("/open", posting(opts, openHandler(s, opts)))
	// A photograph, behind the same guard as everything else: a picture of a
	// letter is at least as private as the note beside it.
	// Your own face is not one of them: it arrives with the identity rather
	// than with a note, so it is mounted whether or not photographs are.
	m.Get("/board", guard(opts, boardHandler(s, opts)))
	m.Post("/board/act", posting(opts, boardActHandler(s, opts)))
	m.Post("/board/undo", posting(opts, boardUndoHandler(s, opts)))
	m.Post("/board/new", posting(opts, boardNewHandler(s, opts)))
	m.Post("/board/now", posting(opts, boardNowHandler(s, opts)))
	m.Post("/board/buddy", posting(opts, boardBuddyHandler(s, opts)))
	m.Post("/board/badly", posting(opts, boardBadlyHandler(s, opts)))
	m.Post("/board/capture", posting(opts, boardCaptureHandler(s, opts)))
	m.Post("/board/chore", posting(opts, boardChoreHandler(s, opts)))
	m.Get("/me/face", guard(opts, faceHandler(s)))
	if opts.Photos != nil {
		m.Get("/photo/{id}", guard(opts, photoHandler(s, opts)))
		// The card asks for this one. See thumbHandler.
		m.Get("/photo/{id}/thumb", guard(opts, thumbHandler(s, opts)))
	}
	m.Post("/mood", posting(opts, threadMoodHandler(s, opts)))
	// The one thing's three answers.
	m.Post("/now/act", posting(opts, nowActHandler(s, opts)))
	// I can't start. Its own route rather than a fourth act, because it is the
	// one answer that is about you rather than about the thing.
	m.Post("/now/stuck", posting(opts, nowStuckHandler(s, opts)))
	// Where to reach this browser. Only mounted when there is a key to
	// subscribe with — a route that always answers 400 is a route that teaches
	// the client to stop asking.
	if opts.PushKey != "" {
		m.Post("/push/subscribe", guard(opts, sameOrigin(pushSubscribeHandler(s, opts))))
		// And the way off. Not `posting`: this is a setting rather than
		// something said, so it answers no turn and belongs in no room.
		m.Post("/push/forget", guard(opts, sameOrigin(pushForgetHandler(s, opts))))
	}
	m.Post("/pile/act", posting(opts, actHandler(s, opts)))
	// Starting fresh, when Buddy offers you back a run you were part way
	// through. Its other answer — carry on — is an ordinary door press and
	// needs no route of its own.
	m.Post("/place/fresh", posting(opts, freshHandler(s, opts)))
	// Triage, in the conversation: skipping one, and changing your mind.
	m.Post("/pile/later", posting(opts, laterHandler(s, opts)))
	m.Post("/pile/undo", posting(opts, undoHandler(s, opts)))
	// The three questions a note can be asked, rather than the three verbs that
	// end it. Each reuses the shape the chores already have, and each arrives
	// behind /pile/more — see moreHandler for why they are a turn rather than a
	// panel.
	m.Post("/pile/often", posting(opts, askAbout(s, opts, func(it squirrel.Item) squirrel.Turn {
		return askHowOften("/pile/chore",
			map[string]string{"id": strconv.FormatInt(it.ID, 10), "from": "thread"}, "", "", "")
	})))
	m.Post("/pile/reword", posting(opts, askAbout(s, opts, func(it squirrel.Item) squirrel.Turn {
		return askForWords("/pile/fix",
			map[string]string{"id": strconv.FormatInt(it.ID, 10), "from": "thread"},
			it.RawText, "say it this way")
	})))
	m.Post("/find", posting(opts, findHandler(s, opts)))
	// One result, opened into a card you can act on. A hit is quiet because it
	// is a thing you are finding; this is the moment it becomes a thing you are
	// deciding about. See findOpenHandler.
	m.Post("/find/open", posting(opts, findOpenHandler(s, opts)))
	m.Post("/pile/more", posting(opts, moreHandler(s, opts)))
	m.Post("/pile/why", posting(opts, askAbout(s, opts, func(it squirrel.Item) squirrel.Turn {
		return askWhyNot(it.ID)
	})))
	m.Post("/pile/chore", posting(opts, choreHandler(s, opts)))
	m.Post("/pile/fix", posting(opts, fixHandler(s, opts)))
	// Four thoughts captured as one note, offered as four. Both halves —
	// asking, and keeping what was asked for — because they are the only two
	// things you can do to a proposal.
	m.Post("/pile/split", posting(opts, splitHandler(s, opts)))
	// Buddy, as turns rather than a page of his own: there is a conversation to
	// join now, so closing went with it. You stop talking.
	m.Post("/buddy/ask", posting(opts, coachAskHandler(s, opts)))
	// Looking something up. A chip rather than a field in the lid — see
	// findAskHandler.
	m.Post("/find/ask", posting(opts, findAskHandler(s, opts)))
	// What Squirrel thinks it knows about you, and the way to throw it away.
	// A POST for both: reading it is something you asked, and it goes into the
	// record like anything else you say. See knowing.go.
	m.Post("/knowing", posting(opts, knowingHandler(s, opts)))
	m.Post("/knowing/forget", posting(opts, forgetKnowingHandler(s, opts)))
	// A new one, at every door. Each asks for words; the routes underneath are
	// the ones the screens posted to. See newone.go.
	m.Post("/chores/ask", posting(opts, askNameHandler(s, opts,
		"a new chore", "What should come back?", "/chores/name", "name", "next")))
	m.Post("/chores/name", posting(opts, choreNameHandler(s, opts)))
	// The agenda had no ask step and its chip posted straight to /at/new,
	// which needs a label before it can offer a day. With none it redirected,
	// and fetch follows a redirect without telling the script — so the whole
	// page came back and was pasted into the room. Every other room asks
	// first; this one does now too.
	m.Post("/at/ask", posting(opts, askNameHandler(s, opts,
		"a new appointment", "What is it?", "/at/new", "label", "next")))
	m.Post("/tasks/ask", posting(opts, askNameHandler(s, opts,
		"a new task", "What did you decide to do?", "/tasks/new", "text", "keep it")))
	// The two shelves. Doors on the rail until 31 August 2026, and a press
	// inside the notes since. See shelfHandler.
	m.Post("/notes/shelf", posting(opts, shelfHandler(s, opts)))
	m.Post("/pile/ask", posting(opts, askNameHandler(s, opts,
		"put something down", "What is it?", "/capture", "text", "keep it")))
	m.Post("/buddy/say", posting(opts, coachSayHandler(s, opts)))
	// "That landed badly." One press, about the thing you just read.
	m.Post("/buddy/badly", posting(opts, coachBadlyHandler(s, opts)))
	// A proposal, applied because it was pressed. Four things and no more —
	// see coachDoHandler for why it is a switch rather than a dispatcher.
	m.Post("/buddy/do", posting(opts, coachDoHandler(s, opts)))
	// It was /coach, then /buddy, and now it is the conversation. A bookmark that
	// dies quietly is worse than a redirect nobody notices. The query string is
	// dropped: nothing at the other end reads one any more.
	// And the four rooms that stopped being rooms on 31 August 2026. A room
	// was a URL you could put on a home screen, so these are the same promise
	// /coach and /buddy were given: a bookmark that dies quietly is worse than
	// a redirect nobody notices. The two shelves land in the notes, which is
	// where they are now — a press away rather than a door.
	for from, to := range map[string]string{
		"/r/buddy": "/r/everything", "/r/pile": "/r/notes", "/r/held": "/r/notes", "/r/kept": "/r/notes",
	} {
		where := to
		m.Get(from, guard(opts, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, where, http.StatusMovedPermanently)
		}))
	}
	for _, gone := range []string{"/coach", "/buddy"} {
		m.Get(gone, guard(opts, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/r/everything", http.StatusMovedPermanently)
		}))
	}
	// A step finished, or a sequence thrown away. One route because they are
	// the only two things you can do to a breakdown.
	m.Post("/steps", posting(opts, stepsHandler(s, opts)))
	// One fixed point, as a real page: a notification sent yesterday is still
	// on a lock screen, and tapping it has to land somewhere. See sw.js.
	m.Get("/at/{id}", guard(opts, atOneHandler(s, opts)))
	// One fixed point, drawn into the conversation. See atOpenHandler.
	m.Post("/at/open", posting(opts, atOpenHandler(s, opts)))
	// Which day, and what time. See askForADay.
	m.Post("/at/new", posting(opts, atNewHandler(s, opts)))
	m.Post("/at/make", posting(opts, atMakeHandler(s, opts)))
	m.Post("/at/{id}/note", posting(opts, atNoteHandler(s, opts)))
	m.Post("/at/{id}/detach", posting(opts, atDetachHandler(s, opts)))
	// Setting something aside and picking it back up. What you set aside is a
	// message now — see elsewhere.go.
	m.Post("/held/act", posting(opts, heldActHandler(s, opts)))
	// How you have been, asked for by name from the settings panel or from the
	// chip beside the answer you just gave. A press rather than a page since
	// 31 August 2026: it was the last screen in this product that was not a
	// conversation, and asking for it took you out of the room you were in.
	m.Post("/me/moods", posting(opts, moodsHandler(s, opts)))
	// The page it was. A bookmark that dies quietly is worse than a redirect
	// nobody notices.
	m.Get("/moods", guard(opts, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/r/everything", http.StatusMovedPermanently)
	}))
	m.Post("/tasks/act", posting(opts, taskActHandler(s, opts)))
	m.Post("/tasks/new", posting(opts, newTaskHandler(s, opts)))
	m.Post("/chores/act", posting(opts, choreActHandler(s, opts)))
	// How often, as a number and a unit. See askHowOften.
	m.Post("/chores/often", posting(opts, oftenHandler(s, opts)))
	m.Post("/chores/new", posting(opts, newChoreHandler(s, opts)))
	m.Post("/timer", posting(opts, timerHandler(s, opts)))
	// The chores screen lived here for its whole life. A bookmark that dies
	// quietly is worse than a redirect nobody notices.
	m.Get("/pile/chores", guard(opts, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/r/everything", http.StatusMovedPermanently)
	}))
	// The way in, and the three routes that work it. Outside the guard
	// necessarily: a person with no session has to be able to get one.
	//
	// Both writes still carry the origin check. Weaker here — there is no
	// session to ride on — but a cross-site POST to /auth/out signs you out of
	// your own notes from a page you were only reading.
	m.Get("/auth", gateHandler())
	m.Post("/auth/in", sameOrigin(beginHandler(opts)))
	m.Get("/auth/callback", backHandler(opts))
	m.Post("/auth/out", sameOrigin(outHandler(opts)))
	// Outside the guard, like the worker below: a browser fetches a manifest
	// without the cookies that carry the identity, and one that answers 403
	// leaves an installed app with no icon. There is nothing in it to protect.
	m.Get("/manifest.webmanifest", manifestHandler())
	// Not behind the guard: a browser fetches the worker without the cookies
	// that carry the identity, and a worker that 302s to a login page is a
	// worker that never installs. It contains no notes — only which files to
	// keep and what to say when the network is gone.
	m.Get("/sw.js", swHandler())
	m.Get("/static/", staticHandler())
	return nil
}

// splittable reports whether the note is worth asking about. A free check, and it
// is what keeps a model off every note in the pile.
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
