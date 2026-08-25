package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// captureLimit is longer than any thought anyone types on a phone and short
// enough that a stuck key cannot fill a table. It is a guard rather than a
// rule about how much you may say.
const captureLimit = 4000

// The slot in the lid of the box: you post a thought in without opening it.
//
// This screen refused to capture for its whole life, and the reasoning is
// worth keeping rather than deleting: two capture surfaces means two places to
// look for a thought, which is the problem this product exists to solve. The
// owner overruled it on 20 August 2026, choosing a direct write over a relay
// through Campfire. What makes that survivable is that both surfaces write the
// same row to the same table, so there is one pile with two doors into it.
//
// What it cost was real, and for one release it was not paid: there was no
// spool behind this write. The chat's 👀 means the words reached disk before
// anything else could go wrong, and here there was no such stage — so a live
// network and an unhealthy database was a note that was never taken. The
// screen said so loudly and gave the words back, which is honest and is not
// the same as durable, because a page is one reload from empty.
//
// It goes through the same spool the room's captures do now. Written, fsynced
// and renamed before anything says it was kept; the drain moves it on, and the
// drain has always known how to wait for a database. One durability mechanism
// for both doors rather than two that have to be kept in step.
//
// What that costs, stated: a note is in the pile a moment later rather than
// instantly — the drain runs every second by default. The slot is on home and
// the pile is a different screen, so the gap is invisible in practice; and the
// room has always worked this way.
//
// The Campfire room still stops being the complete record. That part of the
// original bargain stands.
func captureHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Still checked, and still refuses: the owner not being known yet means
		// the drain cannot resolve this capture to anybody either, so
		// accepting it would spool a note nobody owns.
		if _, ok := personOf(r); !ok {
			// The same 503 the rest of the screen gives when nobody knows
			// whose pile this is: a redirect here would look like the words
			// went somewhere.
			fail(w, errNoOwner)
			return
		}
		// Every capture that arrives says so, and this is not debugging left in
		// by accident.
		//
		// A capture that failed used to log one line and a capture that worked
		// logged none, so "nothing happened" and "the request never got here"
		// looked identical from the outside — and the difference between them
		// is the entire diagnosis. Twice now the answer has come from noticing
		// an absence rather than reading a message.
		//
		// Metadata only. The shape of the request, never a word of what was
		// said: the whole product exists to be a place thoughts are safe, and a
		// log that quotes them is a second copy nobody asked for.
		slog.Info("a capture arrived",
			"content_type", r.Header.Get("Content-Type"),
			"bytes", r.ContentLength,
			"transport", "screen")

		// The photograph goes to disk before the capture that references it,
		// and is fsynced there. So a spool entry never points at a file that
		// is not on the volume; the other order would give a note that renders
		// a broken picture, which is a worse thing to find than a missing one.
		//
		// An entry that fails to write after the bytes landed leaves a file
		// nothing references — litter on a volume, not a lost thought.
		text, photo, kind, err := readCapture(r, opts)
		if err != nil {
			// Said out loud, because until this was written every way a
			// capture could be refused was silent: the screen said "not kept"
			// and the logs said nothing at all, so the only account of what
			// went wrong was the sentence the person reading it could not
			// act on.
			slog.Warn("a capture was refused", "error", err)
			answerWith(w, r, saidInThread(r, s, opts, text, refusalOf(err)), "/")
			return
		}

		// Nothing said and nothing photographed is nothing to keep. A
		// photograph on its own is a capture — that is most of the point of
		// having one — so it is only the pair being empty that does nothing.
		if text == "" && photo == "" {
			// A real no-op — an empty box, pressed. It is also exactly what a
			// photograph lost on the way in looks like, so it says which it
			// was rather than redirecting in silence.
			slog.Info("a capture had nothing in it",
				"content_type", r.Header.Get("Content-Type"), "bytes", r.ContentLength)
			if wantsFragment(r) {
				// Nothing back, and it must be nothing rather than a
				// redirect: the script follows a redirect, and what it would
				// have appended is the whole page.
				w.WriteHeader(http.StatusNoContent)
				return
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Whose it is, said in the transport's own vocabulary rather than as a
		// person id: the drain resolves every capture's owner from its sender,
		// and this one is no different for being typed on the screen. boot
		// seeds the matching identity.
		sender := subOf(r)

		slog.Info("a capture is being kept",
			"photograph", photo != "", "kind", kind, "words", len(text) > 0)

		if _, err := opts.Spool.Write(squirrel.Capture{
			Transport:  squirrel.ScreenTransport,
			SenderID:   &sender,
			Text:       text,
			Payload:    []byte(squirrel.ScreenCapture),
			ReceivedAt: time.Now(),
			PhotoName:  photo,
			PhotoType:  kind,
		}); err != nil {
			// The words go back to the page rather than into a log. A capture
			// box that clears on failure is a capture box that eats thoughts.
			//
			// This means the disk is unwritable, which is a different and much
			// louder problem than a database being briefly unreachable.
			slog.Warn("a capture could not be spooled", "error", err)
			answerWith(w, r, saidInThread(r, s, opts, text, refusalOf(err)), "/")
			return
		}
		// Says which it was, so the script knows whether to empty the box. The
		// turns alone cannot answer that: a failure is two turns as well, and
		// clearing on one of them is a capture box that eats thoughts.
		w.Header().Set("X-Kept", "1")
		answerWith(w, r, saidInThread(r, s, opts, text,
			whatBuddyMakesOfIt(r, s, opts, text, photo != "")), "/")
	}
}

// errNotAPhotograph is a photograph this will not keep — the wrong kind, too
// big, empty. Its own error because it is the one refusal that is about what
// was sent rather than about the machine, and the screen has to say so: "I
// cannot reach my memory" is a lie when the truth is "that photo is too big",
// and it is a lie that makes you press the same button again.
var errNotAPhotograph = errors.New("not a photograph this keeps")

// refusalOf is which sentence Buddy says back.
//
// Two sentences and not one, because they send you to do different things: the
// first is worth pressing again in a moment, the second never will be.
func refusalOf(err error) string {
	if errors.Is(err, errNotAPhotograph) {
		return "That photograph was not kept — too big, or a kind Squirrel does not take. Your words are still here; keep them without it, or try another picture."
	}
	return "Not kept — Squirrel cannot reach its memory. Your words are still here; try again in a moment."
}

// saidInThread puts a capture and its answer into the conversation.
//
// The screen used to carry this back in the address bar and render it inside
// the slot — which worked while there was a home screen to come back to. The
// thread says it the way it says everything else, and Buddy's line must never
// claim more than happened: a "kept" over a failed write is the two views
// disagreeing about the pile.
//
// A capture with no words is a photograph, and it says so rather than putting
// an empty bubble in a record that is never rewritten.
func saidInThread(r *http.Request, s Store, opts Options, text, reply string) []squirrel.Turn {
	ctx := r.Context()
	personID, ok := personOf(r)
	if !ok {
		return nil
	}
	yours := text
	if yours == "" {
		yours = "a photograph"
	}
	return keepSaid(ctx, s, personID, []squirrel.Turn{
		{Who: squirrel.SpeakerYou, Words: yours},
		answerable(opts, text, reply),
	})
}

// answerable is Buddy's reply, with the way to say he read it wrong.
//
// The judgement that got here was made from one word by a small model, or by a
// rule that only says yes when the sentence is doing nothing else. Both are
// deliberately biased towards keeping — a thought dropped out of the pile is
// the one failure this product does not have, so the bias costs a question
// answered as a note.
//
// This is the way out of that, and it is one press: the words are handed to
// Buddy properly. The rule stays free and you keep the escape hatch, which is
// the shape Ronald asked for on 25 August 2026.
//
// Only on an acknowledgement, and only when there is somebody to ask. A chip
// offering to answer something that has just been answered is furniture, and
// one that cannot work is worse.
func answerable(opts Options, text, reply string) squirrel.Turn {
	said := squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: reply}
	if opts.Reads == nil || strings.TrimSpace(text) == "" {
		return said
	}
	if !isKeptWording(reply) {
		return said
	}
	body, err := json.Marshal(drawn{Chips: []turnChip{{
		Label: "answer this", Action: "/buddy/say",
		Fields: map[string]string{"said": text},
	}}})
	if err != nil {
		slog.Error("drawing the way to be answered", "error", err)
		return said
	}
	said.Shown = body
	return said
}

// isKeptWording is whether Buddy said one of the acknowledgements rather than
// something he wrote.
//
// Asked of the pool rather than tracked alongside, because the pool is the
// definition: an acknowledgement is one of these sentences, and anything else
// is an answer. A second flag threaded through the call would be a second
// place for the two to disagree.
func isKeptWording(reply string) bool {
	for _, one := range squirrel.Sayings(squirrel.SayingKept) {
		if one == reply {
			return true
		}
	}
	return false
}

// readCapture pulls the words and the photograph off the request, streaming
// the photograph straight onto the volume it is going to live on.
//
// Not ParseMultipartForm, and that is the whole of the bug this replaced. That
// call holds the first megabyte in memory and spills the rest to a temporary
// file — and this pod runs with a read-only root filesystem and no writable
// /tmp, because everything it writes has its own volume. So every photograph
// over a megabyte, which is every photograph a phone takes, failed in the
// parser before the handler ever saw it, and failed with the one message that
// was not true: that Squirrel could not reach its memory.
//
// It was invisible to every test because the way to test an upload is with a
// small file, and a small file is exactly the one that never touches the disk.
//
// Streaming is also simply the right shape: the bytes have a durable home to
// go to and Keep already writes and fsyncs them there, so a copy through a
// temporary file was doing nothing except needing somewhere to happen.
func readCapture(r *http.Request, opts Options) (text, photo, kind string, err error) {
	parts, err := r.MultipartReader()
	if errors.Is(err, http.ErrNotMultipart) {
		// Words alone, which is what the service worker's flush sends and what
		// a browser sends from a form with no file on it.
		if err := r.ParseForm(); err != nil {
			return "", "", "", fmt.Errorf("reading what you said: %w", err)
		}
		return said(r.FormValue("text")), "", "", nil
	}
	if err != nil {
		return "", "", "", fmt.Errorf("reading what you sent: %w", err)
	}

	// By name rather than by position: the order of the parts is the order of
	// the fields in the markup, and a capture must not break because someone
	// moved the camera above the box.
	for {
		part, err := parts.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return text, photo, kind, fmt.Errorf("reading what you sent: %w", err)
		}

		switch part.FormName() {
		case "text":
			// Bounded here rather than after the fact, so a stuck key cannot
			// be read into memory before it is cut.
			raw, err := io.ReadAll(io.LimitReader(part, captureLimit+1))
			_ = part.Close()
			if err != nil {
				return text, photo, kind, fmt.Errorf("reading what you said: %w", err)
			}
			text = said(string(raw))
		case "photo":
			// An empty file part is what a form with nothing chosen sends, and
			// it is the ordinary case rather than a failure.
			if part.FileName() == "" || opts.Photos == nil {
				_ = part.Close()
				continue
			}
			// What the browser said it is, checked against the handful this
			// keeps. The type stored is the one from that list rather than the
			// browser's own string, so nothing it claimed is ever handed back
			// as a content type.
			declared := part.Header.Get("Content-Type")
			if _, ok := squirrel.KnownKind(declared); !ok {
				_ = part.Close()
				return text, "", "", fmt.Errorf("%w: %q", errNotAPhotograph, declared)
			}
			photo, err = opts.Photos.Keep(part, declared)
			_ = part.Close()
			if err != nil {
				return text, "", "", fmt.Errorf("%w: %w", errNotAPhotograph, err)
			}
			kind = squirrel.PhotoKind(declared)
		default:
			_ = part.Close()
		}
	}
	return text, photo, kind, nil
}

// said is the one thing done to what you typed: the whitespace a keyboard adds
// at the ends, and nothing else. Never trimmed of its meaning.
func said(raw string) string {
	text := strings.TrimSpace(raw)
	if len(text) > captureLimit {
		text = text[:captureLimit]
	}
	return text
}

// whatBuddyMakesOfIt is the answer to what you typed.
//
// The box was a capture slot and said "Kept.", which is what a filing cabinet
// says. Ronald asked on 25 August 2026 for typing to be talking, and chose the
// version where Buddy decides what the words were — knowing, because it was
// said at the time, that a model between you and the capture promise can be
// wrong.
//
// The order is where that risk is managed, and it is the whole design. The
// words are already spooled and already a note by the time this is called.
// Nothing here can stop that. What this can do is drop a note afterwards,
// which is a state the product already has, which the pile can reverse, and
// which leaves the words in the database either way.
//
// So the failures line up in the safe direction. No coach, a spent budget, an
// unreachable model, a reply that fails its shape, a wrong judgement — every
// one of them costs a note sitting in the pile that you did not want there.
// None of them costs a thought.
//
// A photograph is always kept and never read. It is not words, there is
// nothing to answer, and a model asked to judge a picture it cannot see would
// be guessing about the one capture that is hardest to make again.
func whatBuddyMakesOfIt(r *http.Request, s Store, opts Options, text string, photo bool) string {
	ctx := r.Context()
	kept := squirrel.Say(squirrel.SayingKept, now())
	if photo || strings.TrimSpace(text) == "" {
		return kept
	}
	personID, ok := personOf(r)
	if !ok {
		return kept
	}

	// Was that a question? Three tiers, cheapest first.
	//
	// The rule needs nothing running and is the floor. The house is a small
	// model on the cluster and is asked only to improve on the rule — it costs
	// electricity in a cupboard rather than money abroad, so it may run on
	// everything typed. Neither of them writes an answer; they only decide
	// whether one is worth paying for.
	asking := squirrel.LooksLikeAQuestion(text)
	if opts.AskedAQuestion != nil {
		if said, answered := opts.AskedAQuestion(ctx, text); answered {
			asking = said
		}
	}
	if !asking {
		// A thought, which is almost everything typed into this box. Kept, and
		// answered in the product's own words — no call, no network, and the
		// same reply it gave for a month.
		return kept
	}

	if opts.Reads == nil {
		// It reads as a question and there is nobody to answer it. Keeping it
		// is the honest outcome: a question nobody answered is a note you will
		// want to see again.
		return kept
	}

	say, keep, err := opts.Reads(ctx, personID, text)
	if err != nil {
		// The floor, and it is the box exactly as it was before any of this:
		// kept, and said so.
		return kept
	}
	if keep {
		// It read the whole sentence and disagrees with the one-word
		// judgement that got it here. Its answer wins.
		return say
	}

	// A question, answered. The note is dropped rather than deleted — the
	// words stay in the database, the pile can put it back, and "drop" is what
	// this product already calls a note it does not want.
	if err := dropWhatWasAQuestion(ctx, s, personID, text); err != nil {
		// It could not be dropped, so it is still in the pile. Say the answer
		// anyway: the answer is the useful half, and a note you did not want
		// is a smaller problem than an answer you did not get.
		slog.Warn("a question stayed in the pile", "error", err)
	}
	return say
}

// dropWhatWasAQuestion finds the note that was just made and drops it.
//
// By the words rather than by an id, because there is no id: the capture went
// through the spool, which is what makes it durable, and the row it becomes is
// written by the drain rather than returned from here. Reading the newest open
// note back and checking it says what was typed is how this stays honest — if
// the drain has not caught up, nothing matches and nothing is dropped, which
// leaves a question in the pile and loses nothing.
func dropWhatWasAQuestion(ctx context.Context, s Store, personID int64, text string) error {
	items, _, err := s.OpenItems(ctx, personID, 1)
	if err != nil {
		return err
	}
	if len(items) == 0 || strings.TrimSpace(items[0].RawText) != strings.TrimSpace(text) {
		return nil
	}
	_, err = s.MoveItemState(ctx, items[0].ID, squirrel.ItemOpen, squirrel.ItemDropped, now())
	return err
}
