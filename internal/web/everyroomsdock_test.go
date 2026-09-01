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

// The dock, in every room, answering in the room it was pressed in.
//
// pile.js bound to `.slot[action="/capture"]`, which is the dock in Buddy's
// room, the pile and the two shelves — and not the dock in the agenda, the
// chores or the tasks, which post to their own routes. Nothing intercepted
// those three: the browser posted the form itself, the server answered a 303
// to "/", and you arrived in Buddy having filed something in the room you had
// been standing in.
func TestBrowserEveryRoomsDockStaysPut(t *testing.T) {
	for _, room := range []string{"at", "chores", "tasks", "notes", "everything"} {
		t.Run(room, func(t *testing.T) {
			srv := screen(t, aPile())
			c := browserAt(t, srv, "/r/"+room)
			c.navigate(t, srv.URL+"/r/"+room)
			c.until(t, "a dock", `!!document.querySelector(".dock form .post")`)

			was := c.eval(t, `return document.querySelectorAll("#thread .turn").length`)
			c.eval(t, `const f = document.querySelector(".dock form");
				f.querySelector("textarea").value = "dentist";
				f.requestSubmit(f.querySelector(".post"));
				return 1`)
			settle(t, c)

			require.Equal(t, "/r/"+room, c.eval(t, `return location.pathname`),
				"the press left the room it was made in")
			require.Greater(t, c.eval(t, `return document.querySelectorAll("#thread .turn").length`), was,
				"the press did nothing at all")
			require.Equal(t, false, c.eval(t, `return !!document.querySelector("#thread .rail, #thread .dock")`),
				"a whole page was pasted into the conversation")
		})
	}
}

// Turning the month puts the new one where the old one was.
func TestBrowserTurningTheMonthDoesNotSayItAgain(t *testing.T) {
	srv := screen(t, aPile())
	c := browserAt(t, srv, "/r/at")
	c.navigate(t, srv.URL+"/r/at")
	c.until(t, "the agenda", `!!document.querySelector("#edge .turn")`)

	c.eval(t, `const b = [...document.querySelectorAll("#edge form button")]
		.find(x => x.textContent.includes("appointment"));
		b.closest("form").requestSubmit(b); return 1`)
	settle(t, c)
	c.eval(t, `const f = [...document.querySelectorAll("#thread form.wordbox")].pop();
		f.querySelector("textarea").value = "dentist";
		f.requestSubmit(f.querySelector("button")); return 1`)
	settle(t, c)
	c.until(t, "a calendar", `!!document.querySelector(".calbox")`)

	was := c.eval(t, `return document.querySelectorAll("#thread .turn").length`)
	month := c.eval(t, `return document.querySelector(".calhead b").textContent`)

	c.eval(t, `const b = [...document.querySelectorAll(".calhead form button")].pop();
		b.closest("form").requestSubmit(b); return 1`)
	settle(t, c)

	require.Equal(t, was, c.eval(t, `return document.querySelectorAll("#thread .turn").length`),
		"paging the calendar said the same question again")
	require.Equal(t, float64(1), c.eval(t, `return document.querySelectorAll(".calbox").length`),
		"there are two calendars on the screen")
	require.NotEqual(t, month, c.eval(t, `return document.querySelector(".calhead b").textContent`),
		"the month did not turn")
}

// Any time on the clock, and the three are a shortcut into the field.
func TestBrowserTheTimeIsAFieldAndNotThreeAnswers(t *testing.T) {
	srv := screen(t, aPile())
	c := browserAt(t, srv, "/r/at")
	c.navigate(t, srv.URL+"/r/at")
	c.until(t, "the agenda", `!!document.querySelector("#edge .turn")`)

	c.eval(t, `const b = [...document.querySelectorAll("#edge form button")]
		.find(x => x.textContent.includes("appointment"));
		b.closest("form").requestSubmit(b); return 1`)
	settle(t, c)
	c.eval(t, `const f = [...document.querySelectorAll("#thread form.wordbox")].pop();
		f.querySelector("textarea").value = "dentist";
		f.requestSubmit(f.querySelector("button")); return 1`)
	settle(t, c)
	c.until(t, "a calendar", `!!document.querySelector(".attime")`)

	c.eval(t, `document.querySelector("[data-at]").click(); return 1`)
	require.Equal(t, "09:00", c.eval(t, `return document.querySelector(".attime").value`),
		"the shortcut did not fill the field")

	c.eval(t, `const d = [...document.querySelectorAll(".cal input[name=day]")].find(x => !x.disabled);
		d.checked = true;
		const f = d.closest("form");
		f.querySelector(".attime").value = "11:15";
		f.requestSubmit(f.querySelector(".make"));
		return 1`)
	settle(t, c)

	require.Contains(t, c.eval(t, `return document.querySelector("#thread .turn:last-child").textContent`),
		"11:15", "a time no chip offers was not kept")
}

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
	c.navigate(t, srv.URL+"/r/everything")
	c.until(t, "both faces", `document.querySelectorAll(".youface img").length === 2`)

	require.Equal(t, true, c.eval(t, `
		return [...document.querySelectorAll(".youface img")].every(i => {
			const r = getComputedStyle(i).borderRadius;
			return r === "999px" || parseFloat(r) >= i.getBoundingClientRect().width / 2;
		})`), "a picture of you is square somewhere")
}
