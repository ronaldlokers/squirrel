package coach

import (
	"sync"
	"time"
)

// The offer cache. Two reasons, both written down here before a model existed:
// opening home is the most repeated action in the product and most opens change
// nothing, and an offer that changes on every reload reads as the product
// changing its mind.

// StaleAfter is the floor. Nothing is held longer, however unchanged the day
// looks: half an hour is long enough that a run of opens costs one call, and
// short enough that an answer never outlives the afternoon it was about.
const StaleAfter = 30 * time.Minute

// Offers holds the last decision per person, in memory like the conversation
// window: a restart costing one extra model call is not worth a table, a
// migration and a retention rule.
type Offers struct {
	mu sync.Mutex
	by map[int64]held
}

type held struct {
	// basis is the picker's own answer when the decision was made, as kind:id, and it
	// is what invalidation is built on.
	//
	// PickNow already reflects every invalidating event — a check-in changes
	// capacity, a timer changes rules 2 and 3, a completion or refusal removes the
	// row — so comparing its answer catches all of them without a hook at a single
	// write site. It costs one PickNow per open, which every open already did.
	basis string
	d     Decision
	at    time.Time
}

func NewOffers() *Offers { return &Offers{by: make(map[int64]held)} }

// Forget is the answer to that: one call, from the one handler that answers an
// offer, and the entry is gone whether or not the picker noticed.
func (o *Offers) Forget(personID int64) {
	if o == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.by, personID)
}

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

// Forget throws a person's decision away, whatever the picker says next.
//
// It was absent, on the argument that answering an offer changes what the picker
// says next so the basis stops matching. That holds only while the model agreed
// with the picker. `judged` lets the model replace the picker's answer with a
// different row, so "not now" records a refusal against that row while the
// picker's suppression is keyed on its own — the picker goes on saying what it
// said, and the same card comes back for up to StaleAfter. From outside, a button
// that reloads the page. "Did it" was worse: the row really is done.
//
// Not covered, deliberately: marking the same row done from the tasks or chores
// screen. Closing that means the cache checking whether what it showed is still
// offerable, which is a bigger change than this bug earns.
