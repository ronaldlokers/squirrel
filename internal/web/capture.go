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
// It writes through the same spool the room's captures do — fsynced and renamed
// before anything says it was kept. One durability mechanism for both doors.
//
// The cost, stated: a note reaches the pile a drain tick later.
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
		// Every capture that arrives is logged, because a failure logging one line and a
		// success logging none made "nothing happened" and "the request never arrived"
		// look identical.
		//
		// Metadata only. The shape of the request, never a word of what was said.
		slog.Info("a capture arrived",
			"content_type", r.Header.Get("Content-Type"),
			"bytes", r.ContentLength,
			"transport", "screen")

		// The photograph goes to disk before the capture that references it, and is
		// fsynced there, so a spool entry never points at a file that is not on the
		// volume. An entry that fails after the bytes landed leaves litter, not a lost
		// thought.
		text, photo, kind, err := readCapture(r, opts)
		if err != nil {
			// Said out loud: every way a capture could be refused used to be silent, so the
			// only account of what went wrong was a sentence the reader could not act on.
			slog.Warn("a capture was refused", "error", err)
			answerWith(w, r, saidInThread(r, s, opts, text, refusalOf(err), ""), "/")
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
			// The words go back to the page: a capture box that clears on failure is a
			// capture box that eats thoughts. This means the disk is unwritable, which is
			// louder than a database being briefly unreachable.
			slog.Warn("a capture could not be spooled", "error", err)
			answerWith(w, r, saidInThread(r, s, opts, text, refusalOf(err), ""), "/")
			return
		}
		// Says which it was, so the script knows whether to empty the box. The
		// turns alone cannot answer that: a failure is two turns as well, and
		// clearing on one of them is a capture box that eats thoughts.
		w.Header().Set("X-Kept", "1")
		reply, open := whatBuddyMakesOfIt(r, s, opts, text, photo != "")
		answerWith(w, r, saidInThread(r, s, opts, text, reply, open), "/")
	}
}

// errNotAPhotograph is a photograph this will not keep — wrong kind, too big,
// empty. Its own error because "I cannot reach my memory" is a lie when the truth
// is "that photo is too big", and a lie that makes you press the button again.
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

// saidInThread puts a capture and its answer into the conversation. Buddy's line
// must never claim more than happened: a "kept" over a failed write is the two
// views disagreeing about the pile.
//
// A capture with no words is a photograph, and says so rather than putting an
// empty bubble in a record that is never rewritten.
func saidInThread(r *http.Request, s Store, opts Options, text, reply, open string) []squirrel.Turn {
	ctx := r.Context()
	personID, ok := personOf(r)
	if !ok {
		return nil
	}
	yours := text
	if yours == "" {
		yours = "a photograph"
	}
	turns := []squirrel.Turn{
		{Who: squirrel.SpeakerYou, Words: yours},
		answerable(opts, text, reply),
	}
	// And the place, when what you typed asked to see one. The same draw the
	// menu makes and the same one Buddy makes from the other box — one way of
	// showing a place, reached three ways.
	if place, ok := placeSaid(ctx, s, opts, personID, open, 0); ok {
		turns = append(turns, alsoOffer(place, newChipFor(open)...))
	}
	return keepSaid(ctx, s, personID, turns)
}

// answerable is Buddy's reply, with the way to say he read it wrong.
//
// The judgement that got here is biased towards keeping, so it costs a question
// answered as a note. This is the one press out of that.
//
// Only on an acknowledgement, and only when there is somebody to ask.
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

// isKeptWording asks the pool rather than tracking alongside it, because the pool
// is the definition: an acknowledgement is one of these sentences.
func isKeptWording(reply string) bool {
	for _, one := range squirrel.Sayings(squirrel.SayingKept) {
		if one == reply {
			return true
		}
	}
	return false
}

// readCapture streams the photograph straight onto the volume it will live on.
//
// Not ParseMultipartForm: that holds the first megabyte in memory and spills the
// rest to a temporary file, and this pod has a read-only root and no writable
// /tmp. Every photograph over a megabyte failed in the parser with the one
// message that was untrue — that Squirrel could not reach its memory.
//
// Invisible to every test, because the way to test an upload is with a small
// file and a small file never touches the disk.
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
// The order is the design, and it is what makes a model in this path safe. The
// words are already spooled and already a note by the time this is called;
// nothing here can stop that. All this can do is drop a note afterwards, which
// the pile can reverse and which leaves the words in the database either way.
//
// So every failure — no coach, a spent budget, an unreachable model, a reply
// that fails its shape, a wrong judgement — costs a note you did not want in
// the pile. None of them costs a thought.
//
// A photograph is always kept and never read: a model asked to judge a picture
// it cannot see would be guessing about the capture hardest to make again.
func whatBuddyMakesOfIt(r *http.Request, s Store, opts Options, text string, photo bool) (string, string) {
	ctx := r.Context()
	kept := squirrel.Say(squirrel.SayingKept, now())
	if photo || strings.TrimSpace(text) == "" {
		return kept, ""
	}
	personID, ok := personOf(r)
	if !ok {
		return kept, ""
	}

	// Was that a question? Three tiers, cheapest first: the rule needs nothing
	// running, the house costs electricity rather than money and may run on
	// everything typed, and Reads may not. Neither of the first two writes an answer.
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
		return kept, ""
	}

	if opts.Reads == nil {
		// It reads as a question and there is nobody to answer it. Keeping it
		// is the honest outcome: a question nobody answered is a note you will
		// want to see again.
		return kept, ""
	}

	say, keep, open, err := opts.Reads(ctx, personID, text)
	if err != nil {
		// The floor, and it is the box exactly as it was before any of this:
		// kept, and said so.
		return kept, ""
	}
	if keep {
		// It read the whole sentence and disagrees with the one-word judgement that got
		// it here. Its answer wins, and a thought has no place to draw.
		return say, ""
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
	return say, open
}

// dropWhatWasAQuestion finds the note by its words rather than an id, because
// there is no id: the capture went through the spool and the row is written by
// the drain. If the drain has not caught up, nothing matches and nothing is
// dropped, which leaves a question in the pile and loses nothing.
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
