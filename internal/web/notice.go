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

// One line about what is actually there.
//
// Buddy has only ever handled rows. He can say three come back and cannot say
// that two of them are about the car, which is the difference between somebody
// who has read your notes and a query that has counted them.
//
// This is the smallest thing that closes it: when a door draws its cards, one
// model call may add one sentence about the set. Everything about it is
// bounded on purpose.

// noticeMax is how long that sentence may be.
//
// A door's reply is read on the way to pressing something on it. A paragraph
// there is a paragraph between you and the thing you came for, and the whole
// case for the line is that it is faster to read than the cards.
const noticeMax = 90

// noticeAsk is what the model is asked. Deliberately narrow: it is not being
// invited to coach, and a door is not a moment anybody asked for advice.
const noticeAsk = "Here is what is in front of me. In at most fifteen words, " +
	"say one thing you notice about them as a set — something they share, or " +
	"one that stands out. Do not give advice, do not suggest what to do, and " +
	"do not count them. If there is nothing worth saying, answer with nothing.\n\n"

// noticed caches what was said about a set, so opening a door twice costs one
// call rather than two.
//
// In process, and lost on deploy, which is the right trade for this: the thing
// being protected is a person pressing the same door four times in a minute,
// not the monthly bill — the budget already covers that, and it covers it in
// the one place that can enforce it.
//
// Keyed by the set rather than by the door, so a set that has changed is a set
// worth looking at again, and one that has not is not.
//
// It had no bottom when it shipped: a sync.Map nothing ever evicted, growing by
// one entry per distinct set anybody ever looked at. At one person on a pod
// that restarts every deploy that is slow and invisible, which is the shape of
// leak that is discovered by a pod being OOM-killed on a quiet Tuesday months
// later. Bounded now, and the bound is small on purpose — see noticeKeep.
var noticed = &remembers{said: map[string]string{}}

// noticeKeep is how many answers are held.
//
// Sixty-four, which is far more than the doors a person opens in a sitting and
// far less than a number worth thinking about. The cache exists to stop a
// repeated press repeating a call, and a press you made a hundred sets ago is
// not a press you are about to repeat.
const noticeKeep = 64

// remembers is a bounded map that forgets the oldest thing when it is full.
//
// A ring of keys beside the map rather than a real LRU: this is protecting
// against a repeated press within a minute, and reordering on read would buy
// nothing a queue does not already give. Fewer moving parts is the point.
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

// noticeAbout is that sentence, or nothing at all.
//
// Nothing at all is the common case and the expected one: no key, nothing in
// front of you, a model that would rather not, a model that has been asked for
// this exact set already, or an answer that broke one of the bounds. Every one
// of those paths returns silence rather than an apology, because a door that
// explains why it has nothing to add is worse than a door that has nothing to
// add.
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

	answer, err := opts.Ask(ctx, personID, "door", noticeAsk+strings.Join(of, "\n"), "")
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

// keepIfItIsALine refuses anything that is not one short sentence.
//
// The bounds are the product's, not the model's, and they are checked here
// rather than asked for in the prompt — a prompt is a request and this is a
// rule. A model that ignores "at most fifteen words" costs a dropped line and
// nothing else.
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

// noticeKey identifies a person's view of one set.
//
// The person goes in as digits. It went in as `string(rune(personID))`, which
// is a rune conversion rather than a number: every id above U+10FFFF and every
// negative one becomes the same replacement character, so all of them shared a
// cache entry. Unreachable at one person, and unreachable is not a property
// worth relying on in a key.
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
