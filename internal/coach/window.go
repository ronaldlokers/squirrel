package coach

import (
	"sync"
	"time"
)

// The rolling window, held in memory.
//
// In memory rather than in Postgres, deliberately. What this holds is three
// exchanges and half an hour of them; a restart forgetting the last thing you
// said to the coach costs a sentence of continuity and nothing else. Putting it
// in the database would mean a table, a migration, a retention rule, and a
// second permanent record of bad moments — coach_answers is already that record
// and already keeps everything.
//
// Bounded per person and per room: at most WindowSize exchanges in each, and
// Trim drops anything past WindowAge on every read.
//
// Per room, because two rooms are two conversations — carrying what you said
// in the chores into the pile is this surface remembering across a boundary
// drawn on purpose. Somebody who talks in all seven holds seven times three
// exchanges, which is the same order as the three this held before rooms.
//
// Not bounded by people. An entry is dropped when that person next reads and
// finds nothing fresh, so somebody who stops talking to Buddy and never comes
// back leaves three exchanges behind until the process restarts. That is
// bounded by who can sign in at all, which the required group decides — the
// door cache and the session cache are both keyed by things a stranger can
// invent, and both needed a real bottom for exactly that reason.
type Conversations struct {
	mu sync.Mutex
	by map[int64]map[string][]Exchange
}

func NewConversations() *Conversations {
	return &Conversations{by: make(map[int64]map[string][]Exchange)}
}

// Recent is what to carry into the next turn: the newest few exchanges that are
// still about now.
//
// Trimming on read rather than only on write is what makes the age bound real.
// A conversation that stopped an hour ago is dropped when it is next asked for,
// not left waiting for a write that may never come.
func (c *Conversations) Recent(personID int64, room string, now time.Time) []Exchange {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	fresh := Trim(c.by[personID][room], now)
	if len(fresh) == 0 {
		// Both levels, or the outer map only ever grows: a person whose last
		// room went stale would keep an empty map of rooms forever.
		delete(c.by[personID], room)
		if len(c.by[personID]) == 0 {
			delete(c.by, personID)
		}
		return nil
	}
	c.by[personID][room] = fresh
	return fresh
}

// Add records an exchange that actually happened. Only usable replies get here
// — a guard rejection left nothing for the next turn to refer back to.
func (c *Conversations) Add(personID int64, room, said, replied string, at time.Time) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.by[personID] == nil {
		c.by[personID] = make(map[string][]Exchange)
	}
	next := append(c.by[personID][room], Exchange{Said: said, Replied: replied, At: at})
	c.by[personID][room] = Trim(next, at)
}

// Forget drops one room's conversation.
//
// Ending one has to mean something. A surface that remembers what you said last
// time is a surface you think about before opening, and thinking about it is
// the cost this product spends everything else avoiding.
//
// One room, not all seven: they are separate conversations, and stopping
// talking in the chores is not stopping talking.
func (c *Conversations) Forget(personID int64, room string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.by[personID], room)
	if len(c.by[personID]) == 0 {
		delete(c.by, personID)
	}
}
