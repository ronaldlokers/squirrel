//go:build browser

// The chores screen's own picker, and the keys it was not given.
package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func twoChores() *fakeStore {
	f := aPile()
	f.chores = []squirrel.Chore{
		{ID: 1, Name: "bins out", Every: 7 * 24 * time.Hour, EveryDays: 7, SinceDays: 6, Active: true, EverDone: true},
		{ID: 2, Name: "water the plants", Every: 4 * 24 * time.Hour, EveryDays: 4, SinceDays: 9, Active: true, EverDone: true},
	}
	return f
}

// Looking something up had a keyboard path and keeping a thought did not.
//
// `/` has focused the lid's search since it was written. Nothing focused the
// slot, so on the deliberate-desktop scene the first thing home asked of a
// keyboard user was to reach for the mouse — for the one act this product
// calls sacred.
func TestBrowserAKeyReachesTheSlotOnHome(t *testing.T) {
	f := aPile()
	srv := screen(t, f)
	c := browserAt(t, srv, "/")

	c.key(t, "t")

	require.Equal(t, "TEXTAREA", c.eval(t, `return document.activeElement.tagName`))
	require.Equal(t, true, c.eval(t, `return !!document.activeElement.closest(".slot")`),
		"t on home did not reach the slot")
}

// Retired on 24 August 2026, with the control they covered.
//
// The interval question was a `details.often` disclosure inside the chore card,
// and these pinned its behaviour: that opening it replaced the row rather than
// opening a modal, that 1-4 answered it, and that the arrows belonged to it
// while it was open. It is a turn of its own now — see askHowOften — with two
// radio rows and one submit, and none of those three facts has anything left to
// be true of.
//
// What went with it and is not replaced: the digit keys that answered the
// question without reaching for the mouse. Recorded in docs/roadmap.md rather
// than quietly dropped.

// Both were about the deck: a stamp on a card that is gone, and `t` meaning
// two things on a screen that had both a slot and a deck. There is one dock and
// no deck, so the collision cannot happen — TestBrowserAKeyInTheDockIsJustA
// Letter is what pins the remaining half.

// The deck's digits went with its disclosure and roadmap v0.24.0 recorded that
// as worth restoring. The question is a turn now, so the carve-out is about the
// picker wherever it is: a digit is a count, a letter is a unit by its own first
// letter, and Enter answers.
func TestBrowserTheIntervalQuestionAnswersToKeys(t *testing.T) {
	f := &fakeStore{
		chores: []squirrel.Chore{{ID: 1, Name: "bins out", Every: 7 * 24 * time.Hour,
			EveryDays: 7, SinceDays: 6, Active: true, EverDone: true}},
		checkin: &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: time.Now()},
	}
	srv := screen(t, f)
	c := atChores(t, srv)

	c.eval(t, `document.querySelector('article.chore form[action="/chores/often"] button').click()`)
	c.until(t, "the question", `!!document.querySelector(".pick")`)

	c.key(t, "3")
	c.key(t, "m")
	require.Equal(t, "3", c.eval(t,
		`return document.querySelector('.pick input[name="count"]:checked').value`))
	require.Equal(t, "months", c.eval(t,
		`return document.querySelector('.pick input[name="unit"]:checked').value`))

	c.key(t, "Enter")
	c.until(t, "it to be answered", `!document.querySelector(".turn:last-child .pick")`)
	require.Equal(t, 90*24*time.Hour, f.reinterval.every, "the keys answered something else")
}

// And a letter aimed at the question does not act on anything behind it.
func TestBrowserAKeyAimedAtTheQuestionStaysInIt(t *testing.T) {
	f := &fakeStore{
		items: []squirrel.Item{note(9, "the boiler", squirrel.ItemOpen)},
		chores: []squirrel.Chore{{ID: 1, Name: "bins out", Every: 7 * 24 * time.Hour,
			EveryDays: 7, Active: true}},
		checkin: &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: time.Now()},
	}
	srv := screen(t, f)
	c := atChores(t, srv)

	c.eval(t, `document.querySelector('article.chore form[action="/chores/often"] button').click()`)
	c.until(t, "the question", `!!document.querySelector(".pick")`)

	// `d` is DONE on a note and "days" in the question. The question owns it.
	c.key(t, "d")
	c.eval(t, `return new Promise(r => setTimeout(r, 250))`)

	require.Equal(t, "days", c.eval(t,
		`return document.querySelector('.pick input[name="unit"]:checked').value`))
	require.Empty(t, f.states, "a letter meant for the question triaged a note")
}
