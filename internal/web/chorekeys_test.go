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

// An arrow pressed at an open question means "move between its answers", or it
// means nothing. It does not mean "leave, and go to a different chore".
//
// The deck's own picker was given 1-4 for exactly this reason. This screen has
// the same picker, drawn the same way, and kept the screen-wide arrows — so an
// arrow aimed at the four chips threw the focus onto another chore's buttons,
// and the next letter acted on that one instead.
func TestBrowserAnOpenIntervalQuestionKeepsTheArrows(t *testing.T) {
	srv := screen(t, twoChores())
	c := browserAt(t, srv, "/chores")

	c.eval(t, `document.querySelector("article.chore details.often > summary").focus()`)
	c.key(t, "o")
	c.until(t, "the question", `!!document.querySelector("article.chore details.often[open]")`)

	require.Equal(t, "bins out", c.eval(t, `
		return document.activeElement.closest("article.chore").querySelector(".name").textContent`))

	c.key(t, "ArrowDown")

	require.Equal(t, "bins out", c.eval(t, `
		return document.activeElement.closest("article.chore")?.querySelector(".name").textContent ?? "nowhere"`),
		"the arrow left the question and landed on another chore")
	require.Equal(t, true, c.eval(t, `
		return !!document.querySelector("article.chore details.often[open]")`),
		"and it took the question with it")
}

// And the digits answer it, the way they do on the deck.
func TestBrowserTheChoresPickerTakesTheSameDigitsTheDeckDoes(t *testing.T) {
	f := twoChores()
	srv := screen(t, f)
	c := browserAt(t, srv, "/chores")

	c.eval(t, `document.querySelector("article.chore details.often > summary").focus()`)
	c.key(t, "o")
	c.until(t, "the question", `!!document.querySelector("article.chore details.often[open]")`)

	c.eval(t, `
		window.__chose = null;
		document.querySelectorAll("article.chore .chips .chip").forEach(chip => {
			chip.addEventListener("click", e => { e.preventDefault(); window.__chose = chip.value; });
		});`)
	c.key(t, "2")

	require.Equal(t, "every week", c.eval(t, `return window.__chose`),
		"the second digit chose nothing")
}

// And on the card the three chips behave like the other four: a stamp, the
// hold, and the way back focused.
//
// They live in their own form outside `.actions`, because "i can't act on
// this" is not an answer to what the note is — so nothing ever wired them to
// any of it, and a note set aside left the screen with no stamp, no hold and
// nothing offering it back.
func TestBrowserSettingAsideStampsTheCardLikeEveryOtherAnswer(t *testing.T) {
	c, _ := open(t, aPile())

	c.eval(t, `document.querySelector(".cantact").open = true`)
	c.eval(t, `document.querySelector('.whys button[value="waiting"]').click()`)
	c.until(t, "the card to be stamped", `document.getElementById("card").classList.contains("stamped")`)

	require.Equal(t, "WAITING", c.eval(t, `return document.getElementById("stampText").textContent`))
	require.Equal(t, "waiting on someone", c.eval(t, `return document.getElementById("said").textContent`))
	require.Equal(t, false, c.eval(t, `return document.getElementById("undoRow").hidden`),
		"the way back is not reachable while the card it undoes is still there")
	require.Equal(t, "undo", c.eval(t, `return document.activeElement.id`))
}
