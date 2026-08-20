package coach

import (
	"sync"
	"time"
)

// The offer cache.
//
// Two reasons, and both were already written down in this codebase before a
// model existed. The first is money: opening home is the most repeated action
// in the product and most opens change nothing, so calling the deep model on
// each of them pays over and over for the same answer. The second is
// pick.go's own argument — an offer that changes on every reload "reads as the
// product changing its mind", and a model asked the same question twice will
// not answer it identically.

// StaleAfter is the floor. Nothing is held longer, however unchanged the day
// looks: half an hour is long enough that a run of opens costs one call, and
// short enough that an answer never outlives the afternoon it was about.
const StaleAfter = 30 * time.Minute

// Offers holds the last decision per person.
//
// In memory, like the conversation window and for the same reason: a restart
// costing one extra model call is not worth a table, a migration and a
// retention rule.
type Offers struct {
	mu sync.Mutex
	by map[int64]held
}

type held struct {
	// basis is the picker's own answer at the time the decision was made,
	// rendered as kind:id.
	//
	// This is what invalidation is built on, and it is worth saying why rather
	// than the list of events the design first proposed. PickNow already
	// reflects every one of them — a check-in changes capacity, a timer
	// changes rules 2 and 3, a completion or a refusal removes the row, a
	// moment entering its leave-by window outranks everything else — so
	// comparing its answer catches all five without a single hook at a single
	// write site. Hooks are the version of this that gets forgotten when a
	// seventh write path is added.
	//
	// It costs one PickNow per open, which every open already did.
	basis string
	d     Decision
	at    time.Time
}

func NewOffers() *Offers { return &Offers{by: make(map[int64]held)} }

// Get is the cached decision, when the day has not moved under it.
func (o *Offers) Get(personID int64, basis string, now time.Time) (Decision, bool) {
	if o == nil {
		return Decision{}, false
	}
	o.mu.Lock()
	defer o.mu.Unlock()

	h, ok := o.by[personID]
	if !ok || h.basis != basis || now.Sub(h.at) >= StaleAfter {
		return Decision{}, false
	}
	return h.d, true
}

// Put holds a decision against the picker's answer at the moment it was made.
func (o *Offers) Put(personID int64, basis string, d Decision, now time.Time) {
	if o == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.by[personID] = held{basis: basis, d: d, at: now}
}

// There is no Forget, and the absence is deliberate. Answering an offer — done,
// not now, started — changes what the picker says next, so the basis stops
// matching and the entry is dead the moment it is looked at. A second way to
// invalidate would be a second thing to keep in step with the first.
