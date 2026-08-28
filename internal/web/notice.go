package web

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strconv"
	"strings"
	"sync"
)

// One line about what is actually there. Buddy can say three chores come back and
// cannot say that two of them are about the car.
//
// When a door draws its cards, one model call may add one sentence about the set.
// Everything about it is bounded on purpose.

// noticeMax is how long that sentence may be. A door's reply is read on the way
// to pressing something on it, and the whole case for the line is that it is
// faster to read than the cards.
const noticeMax = 90

// noticeAsk is what the model is asked. Deliberately narrow: it is not being
// invited to coach, and a door is not a moment anybody asked for advice.
const noticeAsk = "Here is what is in front of me. In at most fifteen words, " +
	"say one thing you notice about them as a set — something they share, or " +
	"one that stands out. Do not give advice, do not suggest what to do, and " +
	"do not count them. If there is nothing worth saying, answer with nothing.\n\n"

// noticed caches what was said about a set, so opening a door twice costs one
// call. In process and lost on deploy: what is being protected is a person
// pressing the same door four times in a minute, not the monthly bill.
//
// Keyed by the set rather than the door, so a set that has changed is worth
// looking at again.
//
// It had no bound when it shipped — a sync.Map nothing evicted, growing by one
// entry per distinct set anybody looked at.
var noticed = &remembers{said: map[string]string{}}

// noticeKeep is how many answers are held. Far more than the doors a person opens
// in a sitting: a press you made a hundred sets ago is not one you are about to
// repeat.
const noticeKeep = 64

// remembers is a bounded map that forgets the oldest when full. A ring of keys
// rather than a real LRU: reordering on read would buy nothing here.
type remembers struct {
	mu    sync.Mutex
	said  map[string]string
	order []string
}

func (r *remembers) Load(key string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	said, ok := r.said[key]
	return said, ok
}

func (r *remembers) Store(key, said string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, already := r.said[key]; already {
		return
	}
	r.said[key] = said
	r.order = append(r.order, key)
	for len(r.order) > noticeKeep {
		delete(r.said, r.order[0])
		r.order = r.order[1:]
	}
}

// Clear empties it. For tests, which would otherwise share one cache across
// every case in the package and pass or fail by the order they ran in.
func (r *remembers) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.said, r.order = map[string]string{}, nil
}

// noticeAbout is that sentence, or nothing at all — the common case: no key,
// nothing in front of you, a set already asked about, or an answer that broke a
// bound. Every path returns silence rather than an apology.
func noticeAbout(ctx context.Context, opts Options, personID int64, place string, of []string) string {
	if opts.Ask == nil || len(of) < 2 {
		// Below two there is no set to notice anything about, and "one thing
		// you notice about this one thing" is a description.
		return ""
	}
	key := noticeKey(personID, place, of)
	if said, ok := noticed.Load(key); ok {
		return said
	}

	// The room comes from the context rather than from `place`, which is the
	// display name: "the tasks" is not a room key, and asking as one would
	// hand this turn Buddy's whole toolset under a name that looks right.
	answer, err := opts.Ask(ctx, personID, "door", roomOf(ctx), noticeAsk+strings.Join(of, "\n"), "")
	if err != nil {
		// The floor. A door that cannot reach a model is a door, and the
		// cards are what you came for.
		slog.Info("no line about what is behind a door", "place", place, "error", err)
		noticed.Store(key, "")
		return ""
	}
	said := keepIfItIsALine(answer.Text)
	noticed.Store(key, said)
	return said
}

// keepIfItIsALine refuses anything that is not one short sentence. Checked here
// rather than asked for in the prompt: a prompt is a request and this is a rule.
func keepIfItIsALine(said string) string {
	said = strings.TrimSpace(said)
	if said == "" || len(said) > noticeMax {
		return ""
	}
	// One sentence. Two is a paragraph starting.
	if strings.Count(said, ". ") > 0 {
		return ""
	}
	// Never advice, whatever was asked for. These are the shapes that make a
	// remark into an instruction, and an instruction on a door is the product
	// telling you what to do with your own afternoon.
	for _, telling := range []string{
		"you should", "you could", "try ", "why not", "maybe start",
		"i suggest", "consider ", "don't forget", "remember to",
	} {
		if strings.Contains(strings.ToLower(said), telling) {
			return ""
		}
	}
	return said
}

// noticeKey identifies a person's view of one set, with the person as digits. It
// went in as `string(rune(personID))`, so every id above U+10FFFF and every
// negative one became the same replacement character and shared a cache entry.
func noticeKey(personID int64, place string, of []string) string {
	h := sha256.New()
	_, _ = h.Write([]byte(place))
	for _, s := range of {
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(s))
	}
	return strconv.FormatInt(personID, 10) + ":" + hex.EncodeToString(h.Sum(nil))
}

// titlesOf is what is on the cards, for the model to look at. Only the words:
// a card carries an id and a photograph and neither is something to notice.
func titlesOf(cards []cardView) []string {
	out := make([]string, 0, len(cards))
	for _, c := range cards {
		if t := strings.TrimSpace(c.Title); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// withNotice is the door's own sentence, and then the model's, when there is
// one. Two sentences, and the product's own comes first — Principle 8 draws
// the line at authorship, and the count is Squirrel's own fact.
func withNotice(ctx context.Context, opts Options, personID int64, place, lead string, cards []cardView) string {
	if said := noticeAbout(ctx, opts, personID, place, titlesOf(cards)); said != "" {
		return lead + " " + said
	}
	return lead
}
