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

// And it never shadows the deck's own `t`, which is A TASK.
func TestBrowserTheSlotsKeyDoesNotTakeTheDecksTask(t *testing.T) {
	c, _ := open(t, aPile())

	// A slot put on the deck on purpose. No screen has both today, which is
	// why the guard is otherwise unreachable — and an unreachable guard that
	// no test can tell from its absence is exactly the kind this project does
	// not keep. This makes the condition real so the guard has something to be
	// checked against.
	c.eval(t, `
		const box = document.createElement("textarea");
		const slot = document.createElement("form");
		slot.className = "slot";
		slot.appendChild(box);
		document.getElementById("stage").appendChild(slot);`)

	c.key(t, "t")
	c.until(t, "the card to be stamped", `document.getElementById("card").classList.contains("stamped")`)

	require.Equal(t, false, c.eval(t, `return !!document.activeElement.closest(".slot")`),
		"t reached a slot on a screen that also has a deck, where it means A TASK")

	require.Equal(t, "A TASK", c.eval(t, `return document.getElementById("stampText").textContent`),
		"t on the deck stopped meaning A TASK")
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
