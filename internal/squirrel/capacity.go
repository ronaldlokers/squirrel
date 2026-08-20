package squirrel

import (
	"context"
	"time"
)

// How much of you there is, right now.
//
// This is the whole of the mood check-in's effect on anything, and it lives in
// its own file so there is exactly one definition of "a low day". Two surfaces
// and the nudge all read it, and three copies of a three-line rule is three
// copies that drift.
//
// It changes what is *offered*. It never changes what is *true*: a chore is
// exactly as due on a low day as on any other, the pile holds what it holds,
// and nothing anywhere is hidden from a person who goes looking. Capacity is
// the difference between what Squirrel raises unasked and what it will show
// you when you ask.

// Capacity is two values, not five, and deliberately not a number. The five
// faces are a vocabulary for how you are; this is the only question the
// machinery asks of them, and the answer it needs is "should I be putting
// things in front of you".
type Capacity string

const (
	CapacityOK  Capacity = "ok"
	CapacityLow Capacity = "low"
)

// lowMoods are the two answers that mean "not right now".
//
// `low` is deliberately not among them, and that distinction is the reason the
// five are not a scale. Low is how you feel; wiped and frazzled are how much
// you have. Someone flat but functional is not helped by a day with nothing in
// it — that is the shape where an empty screen reads as the product agreeing
// you are finished.
var lowMoods = map[Mood]bool{
	MoodWiped:    true,
	MoodFrazzled: true,
}

// CapacityOf reads the one check-in the store will hand out.
//
// A stale reading is not a low reading. Fresh() is six hours, because the
// question asked is "how is right now" and this morning is not now — and
// letting a rough Tuesday quietly govern a fine Thursday is exactly the
// accumulating shape this product exists without.
//
// No reading at all is CapacityOK. The absence of an answer is not an answer,
// and a person who has never once used the check-in must not silently get a
// smaller product than one who has.
func CapacityOf(c Checkin, found bool, now time.Time) Capacity {
	if !found || !c.Fresh(now) {
		return CapacityOK
	}
	if lowMoods[c.Mood] {
		return CapacityLow
	}
	return CapacityOK
}

// Capacity reads the store and answers the same question. The error is
// swallowed on purpose and answered as CapacityOK: a database that cannot say
// how you are must not be able to decide that you have nothing in you.
func (s *Store) Capacity(ctx context.Context, personID int64, now time.Time) Capacity {
	c, found, err := s.LatestCheckin(ctx, personID)
	if err != nil {
		return CapacityOK
	}
	return CapacityOf(c, found, now)
}
