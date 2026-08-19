package squirrel

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// How you are right now, and what Squirrel is allowed to do with it.
//
// This is the sharpest thing in the product. A record of how someone felt is
// data about the person, and the rule here is that nothing accrues that can be
// destroyed — a fortnight of your own bad days rendered as a row of faces is
// the counter this project refuses, wearing a face.
//
// The owner's decision, made deliberately: keep the history, never show it.
// The rows accumulate because a nudge that knows you have been flat can be
// gentler, and that is the entire reason for asking. Nothing may render more
// than the latest one — not a series, not a count, not a trend, and never a
// sentence about the person. That is enforced by what this file exports rather
// than by a comment: LatestCheckin returns one, and there is no function here
// that returns many.

// Mood is one of the five drawn answers. They are not a scale and must never be
// numbered: "low" and "frazzled" are different states wanting different
// answers, which a 1-to-5 row cannot say.
type Mood string

const (
	MoodGood     Mood = "good"
	MoodCalm     Mood = "calm"
	MoodLow      Mood = "low"
	MoodFrazzled Mood = "frazzled"
	MoodWiped    Mood = "wiped"
)

// Moods is the order they are offered in, everywhere. One order, so the two
// surfaces cannot disagree about which face sits where — muscle memory is the
// point of a fixed order, and it is the only reason there is one.
var Moods = []Mood{MoodGood, MoodCalm, MoodLow, MoodFrazzled, MoodWiped}

// Words is what each one is called, for anything that reads the page aloud and
// for the chat, which has no pictures.
var Words = map[Mood]string{
	MoodGood:     "good",
	MoodCalm:     "calm",
	MoodLow:      "low",
	MoodFrazzled: "frazzled",
	MoodWiped:    "wiped",
}

// ParseMood reads what someone typed or tapped. Unknown is not a mood: this
// arrives from a form and from a chat message, so it is read the way a
// stranger's typing is read.
func ParseMood(s string) (Mood, bool) {
	m := Mood(strings.ToLower(strings.TrimSpace(s)))
	if _, ok := Words[m]; ok {
		return m, true
	}
	return "", false
}

// Checkin is one answer, at one moment.
type Checkin struct {
	Mood   Mood
	SaidAt time.Time
}

// Fresh reports whether this reading still describes now.
//
// Six hours, because the question is "how is right now" and this morning is not
// now. A stale reading is not a bad one — it is simply not an answer to the
// question being asked, and treating it as one would let a rough Tuesday
// quietly govern a fine Thursday.
func (c Checkin) Fresh(now time.Time) bool {
	return now.Sub(c.SaidAt) < 6*time.Hour
}

// RecordCheckin stores one answer. Every answer, not the latest — see the
// migration for what that buys and what it costs.
func (s *Store) RecordCheckin(ctx context.Context, personID int64, m Mood, source string, at time.Time) error {
	if _, err := s.pool.Exec(ctx,
		`insert into checkins (person_id, mood, source, said_at) values ($1, $2, $3, $4)`,
		personID, string(m), source, at); err != nil {
		return fmt.Errorf("recording checkin: %w", err)
	}
	return nil
}

// LatestCheckin is the only way to read this table, and it returns one row on
// purpose. There is deliberately no LatestCheckins, no CheckinsSince and no
// count: a caller cannot render a series it cannot obtain, which is a stronger
// guarantee than a rule someone has to remember.
func (s *Store) LatestCheckin(ctx context.Context, personID int64) (Checkin, bool, error) {
	var c Checkin
	var mood string
	err := s.pool.QueryRow(ctx, `
		select mood, said_at from checkins
		 where person_id = $1
		 order by said_at desc, id desc
		 limit 1`, personID).Scan(&mood, &c.SaidAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Checkin{}, false, nil
		}
		return Checkin{}, false, fmt.Errorf("reading checkin: %w", err)
	}
	c.Mood = Mood(mood)
	return c, true, nil
}
