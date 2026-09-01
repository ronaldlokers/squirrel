//go:build browser

// The chores' own picker, and the keys it was not given.
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

// Looking something up had a keyboard path and keeping a thought did not, so
// the first thing home asked of a keyboard user was to reach for the mouse —
// for the one act this product calls sacred.
func TestBrowserAKeyReachesTheSlotOnHome(t *testing.T) {
	f := aPile()
	srv := screen(t, f)
	c := browserAt(t, srv, "/r/everything")

	c.key(t, "t")

	require.Equal(t, "TEXTAREA", c.eval(t, `return document.activeElement.tagName`))
	require.Equal(t, true, c.eval(t, `return !!document.activeElement.closest(".slot")`),
		"t on home did not reach the slot")
}

// The question is a turn, so the carve-out is about the picker wherever it is:
// a digit is a count, a letter is a unit by its own first letter, and Enter
// answers.
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
