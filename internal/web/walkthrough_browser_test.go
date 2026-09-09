//go:build browser

package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A day's worth of the board, driven the way a person drives it. Every one of
// these walked a form the redesign moved: a form nothing posts to is the way
// this product breaks without any test noticing, because every part of it
// still renders.

func aDay() *fakeStore {
	return &fakeStore{
		checkin: &squirrel.Checkin{Mood: squirrel.MoodCalm, SaidAt: time.Now()},
		usually: map[int64]squirrel.Usually{1: {Part: squirrel.Morning}},
		chores: []squirrel.Chore{{
			ID: 1, Name: "water the plants", Active: true, EverDone: true,
			Every: 24 * time.Hour, EveryDays: 1, SinceDays: 1,
		}},
		items: []squirrel.Item{
			task(2, "book the MOT", squirrel.ItemOpen),
			note(3, "the boiler code is 4471", squirrel.ItemOpen),
		},
	}
}

func typeInto(t *testing.T, c *cdp, sel, words string) {
	t.Helper()
	c.until(t, sel, `!!document.querySelector(`+quoted(sel)+`)`)
	c.eval(t, `const el = document.querySelector(`+quoted(sel)+`);
		el.focus(); el.value = `+quoted(words)+`; return 1`)
}

func quoted(s string) string { return `"` + s + `"` }

func TestBrowserTheAddBarKeepsAThought(t *testing.T) {
	f := aDay()
	srv := screen(t, f)
	c := browserAt(t, srv, "/")

	typeInto(t, c, ".addbar .words", "the bin men come on tuesdays")
	c.eval(t, `document.querySelector(".addbar").requestSubmit(); return 1`)

	require.Eventually(t, func() bool {
		for _, w := range f.inserted {
			if w == "the bin men come on tuesdays" {
				return true
			}
		}
		return false
	}, 4*time.Second, 50*time.Millisecond, "typing a thought and pressing enter kept nothing")
}

func TestBrowserThePlusCarriesTheWordsIntoTheWriter(t *testing.T) {
	srv := screen(t, aDay())
	c := browserAt(t, srv, "/")

	typeInto(t, c, ".addbar .words", "put the bins out")
	c.eval(t, `document.querySelector(".addbar .plus").click(); return 1`)

	c.until(t, "the writer", `!!document.querySelector(".adder.open")`)
	require.Equal(t, "put the bins out", c.eval(t, `return document.querySelector(".addform .words").value`),
		"the words did not travel, so they have to be typed twice")
}

func TestBrowserTheWriterMakesAChoreThatComesBack(t *testing.T) {
	f := aDay()
	srv := screen(t, f)
	c := browserAt(t, srv, "/?words=put+the+bins+out")
	c.until(t, "the writer", `!!document.querySelector(".adder.open")`)

	c.eval(t, `document.querySelector('.kinds input[value="daily"]').click(); return 1`)
	c.eval(t, `document.querySelector('.picks input[value="14"]').click(); return 1`)
	c.eval(t, `document.querySelector(".addfoot .stamp.go").click(); return 1`)

	require.Eventually(t, func() bool { return f.reinterval.name == "put the bins out" },
		4*time.Second, 50*time.Millisecond, "keeping it made no chore")
	require.Equal(t, 14*24*time.Hour, f.reinterval.every, "the rhythm you picked was not the one kept")
}

func TestBrowserTheWriterMakesAFixedPoint(t *testing.T) {
	f := aDay()
	srv := screen(t, f)
	c := browserAt(t, srv, "/?words=the+physio")
	c.until(t, "the writer", `!!document.querySelector(".adder.open")`)

	c.eval(t, `document.querySelector('.kinds input[value="agenda"]').click(); return 1`)
	c.eval(t, `document.querySelector(".whenasks .dd").value = "05";
		document.querySelector(".whenasks .mo").value = "09";
		document.querySelector(".whenasks .hh").value = "14";
		document.querySelector(".whenasks .mm").value = "30";
		return 1`)
	c.eval(t, `document.querySelector('.whenasks input[name="weeks"][value="2"]').click(); return 1`)
	c.eval(t, `document.querySelector(".addfoot .stamp.go").click(); return 1`)

	require.Eventually(t, func() bool { return len(f.moments) == 1 },
		4*time.Second, 50*time.Millisecond, "keeping it made no fixed point")
	require.Equal(t, "the physio", f.moments[0].Label)
	require.Equal(t, 2, f.moments[0].EveryWeeks, "the recurrence you picked was not kept")
}

func TestBrowserTheWriterMakesAThingYouDoOnce(t *testing.T) {
	f := aDay()
	srv := screen(t, f)
	c := browserAt(t, srv, "/?words=ring+the+dentist")
	c.until(t, "the writer", `!!document.querySelector(".adder.open")`)

	c.eval(t, `document.querySelector('.kinds input[value="tasks"]').click(); return 1`)
	c.eval(t, `document.querySelector(".addfoot .stamp.go").click(); return 1`)

	require.Eventually(t, func() bool {
		for _, it := range f.items {
			if it.RawText == "ring the dentist" && it.Kind == squirrel.ItemTask {
				return true
			}
		}
		return false
	}, 4*time.Second, 50*time.Millisecond, "keeping it made no thing to do once")
}

func TestBrowserTheHeadOfTheRailCanBeAnswered(t *testing.T) {
	f := aDay()
	srv := screen(t, f)
	c := browserAt(t, srv, "/")
	touching(t, c)
	c.navigate(t, srv.URL+"/")
	c.until(t, "the head", `!!document.querySelector(".headcard .bigstamp")`)

	c.eval(t, `document.querySelector(".bigstamp.did").click(); return 1`)

	require.Eventually(t, func() bool { return len(f.completed) == 1 },
		4*time.Second, 50*time.Millisecond, "the head of the rail cannot be answered")
}

func TestBrowserTheWallDropsANote(t *testing.T) {
	f := aDay()
	srv := screen(t, f)
	c := browserAt(t, srv, "/notes")
	c.until(t, "a pin", `!!document.querySelector('.pin .stamp[value="drop"]')`)

	c.eval(t, `document.querySelector('.pin .stamp[value="drop"]').click(); return 1`)

	require.Eventually(t, func() bool { return f.states[3] == squirrel.ItemDropped },
		4*time.Second, 50*time.Millisecond, "a note on the wall cannot be dropped")
}
