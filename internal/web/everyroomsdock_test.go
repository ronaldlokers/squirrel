//go:build browser

package web

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func settle(t *testing.T, c *cdp) {
	t.Helper()
	c.eval(t, `return new Promise(r => setTimeout(() => r(1), 1200))`)
}

// The dock, in every where, answering in the where it was pressed in.
//
// pile.js bound to `.slot[action="/capture"]`, which is the dock in Buddy's
// where, the pile and the two shelves — and not the dock in the agenda, the
// chores or the tasks, which post to their own routes. Nothing intercepted
// those three: the browser posted the form itself, the server answered a 303
// to "/", and you arrived in Buddy having filed something in the where you had
// been standing in.
func TestBrowserEveryRoomsDockStaysPut(t *testing.T) {
	// The dock is his room's. The board has no dock — it has a blank strip at
	// the head of every rack — so this is one screen now rather than five.
	for _, where := range []string{"/r/everything"} {
		t.Run(where, func(t *testing.T) {
			srv := screen(t, aPile())
			c := browserAt(t, srv, where)
			c.navigate(t, srv.URL+where)
			c.until(t, "a dock", `!!document.querySelector(".dock form .post")`)

			was := c.eval(t, `return document.querySelectorAll("#thread .turn").length`)
			c.eval(t, `const f = document.querySelector(".dock form");
				f.querySelector("textarea").value = "dentist";
				f.requestSubmit(f.querySelector(".post"));
				return 1`)
			settle(t, c)

			require.Equal(t, where, c.eval(t, `return location.pathname`),
				"the press left the room it was made in")
			require.Greater(t, c.eval(t, `return document.querySelectorAll("#thread .turn").length`), was,
				"the press did nothing at all")
			require.Equal(t, false, c.eval(t, `return !!document.querySelector("#thread .rail, #thread .dock")`),
				"a whole page was pasted into the conversation")
		})
	}
}

// The day picker's two browser tests went with the agenda room on 2 September
// 2026. Choosing a day out of a month was reached by pressing "a new
// appointment" in that room; the agenda rack takes the sentence chat parses,
// which reaches today and tomorrow, and anything further out is Buddy's own
// picker in his room. That the picker still works is his room's test, and that
// the board says where words with no time in them went is the board's.

// One face, one markup. The rail drew a bare <img class="youface">, which the
// rule that rounds a picture cannot reach, so the same face was a circle in
// the conversation and a square in the rooms.
func TestBrowserYourFaceIsRoundEverywhere(t *testing.T) {
	f := &fakeStore{whoName: "Ronald Lokers", whoFace: []byte("\x89PNG\r\n\x1a\n")}
	f.turns = []squirrel.Turn{{ID: 1, Who: squirrel.SpeakerYou, Words: "the tasks"}}
	srv := screen(t, f)
	c := browserAt(t, srv, "/r/everything")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 1280, "height": 900, "deviceScaleFactor": 1, "mobile": false,
	})
	// Three places now: your own turns, the settings page, and the chip in the
	// bar that opens it. The rail's copy went with the rail.
	round := `return [...document.querySelectorAll(".youface img, .chip.face img")].every(i => {
			const r = getComputedStyle(i).borderRadius;
			const own = r === "999px" || parseFloat(r) >= i.getBoundingClientRect().width / 2;
			const p = getComputedStyle(i.parentElement).borderRadius;
			return own || p === "999px" || parseFloat(p) >= i.getBoundingClientRect().width / 2;
		})`

	c.navigate(t, srv.URL+"/r/everything")
	c.until(t, "the faces", `document.querySelectorAll(".youface img, .chip.face img").length === 2`)
	require.Equal(t, true, c.eval(t, round), "a picture of you is square in his room")

	c.navigate(t, srv.URL+"/me")
	c.until(t, "the faces", `document.querySelectorAll(".youface img, .chip.face img").length === 2`)
	require.Equal(t, true, c.eval(t, round), "a picture of you is square on the settings page")
}
