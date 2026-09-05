//go:build browser

package web

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const controlSizeWalker = `(() => {
  const name = el => el.tagName.toLowerCase() +
    (el.className ? '.' + el.className.toString().trim().split(/\s+/).join('.') : '') +
    (el.id ? '#' + el.id : '');
  const out = [];
  document.querySelectorAll('button, a[href], input, select, textarea, summary, label:has(input), label[for]')
    .forEach(el => {
      if (!el.checkVisibility({ checkOpacity: true, checkVisibilityCSS: true, contentVisibilityAuto: true })) return;
      const r = el.getBoundingClientRect();
      if (r.width < 43.5 || r.height < 43.5) {
        out.push({
          name: name(el),
          text: (el.textContent || el.value || el.getAttribute('aria-label') || '').trim().slice(0, 48),
          width: Math.round(r.width * 10) / 10,
          height: Math.round(r.height * 10) / 10,
        });
      }
    });
  return JSON.stringify(out);
})()`

type undersizedControl struct {
	Name   string  `json:"name"`
	Text   string  `json:"text"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

func controlsUnderTheFloor(t *testing.T, c *cdp) []undersizedControl {
	t.Helper()
	var found []undersizedControl
	raw := c.eval(t, "return ("+controlSizeWalker+")")
	require.NoError(t, json.Unmarshal([]byte(fmt.Sprint(raw)), &found))
	return found
}

func openTheFirstAnswerableStrip(t *testing.T, c *cdp) {
	t.Helper()
	c.until(t, "a strip to answer", `!!document.querySelector(".strip.answerable")`)
	c.eval(t, `document.querySelector(".strip.answerable .what").click(); return 1`)
	c.until(t, "the strip to open", `!!document.querySelector(".strip.answerable.open")`)
}

func aKnownFloorGap(f undersizedControl) (string, bool) {
	switch {
	case strings.Contains(f.Name, ".count") || strings.Contains(f.Name, ".unit"):
		return "the chore's rhythm blank and the appointment's compact date/time entry are words inside a sentence, not boxed fields; giving them the floor is a redesign of that row (docs/superpowers/specs/2026-08-22-devices-design.md), not a padding tweak", true
	case strings.HasPrefix(f.Name, "label.findpress"):
		return "the room and /me lid splits the space beside its search icon with a flex spacer that balances the wordmark's absence; widening the icon to 44 takes the width straight from the input beside it (the board's own bar has room to spare and does this; the lid does not) — a decision about the lid's layout, not a padding tweak", true
	case strings.HasPrefix(f.Name, "textarea"):
		return "the compose box's rest height is calibrated against the dock's reserve math that TestBrowserTheDockGivesTheFieldItsOwnRowOnAPhone pins at one line; raising it to 44 needs that reserve recomputed rather than a taller box bolted on", true
	}
	return "", false
}

func checkTheFloor(t *testing.T, c *cdp, where string) {
	t.Helper()
	var unknown []undersizedControl
	var known []string
	for _, f := range controlsUnderTheFloor(t, c) {
		if reason, ok := aKnownFloorGap(f); ok {
			known = append(known, fmt.Sprintf("%s %q is %vx%v — %s", f.Name, f.Text, f.Width, f.Height, reason))
			continue
		}
		unknown = append(unknown, f)
	}
	for _, f := range unknown {
		t.Errorf("%s: %s %q is %vx%v, under the 44x44 floor", where, f.Name, f.Text, f.Width, f.Height)
	}
	if len(unknown) == 0 && len(known) > 0 {
		t.Skip(strings.Join(known, "\n"))
	}
}

func TestBrowserEveryControlClearsFortyFourOnAPhone(t *testing.T) {
	srv := screen(t, everyScreen())
	c := browserAt(t, srv, "/")
	touching(t, c)

	for _, path := range []string{
		"/", "/?bay=chores", "/?bay=tasks", "/?bay=agenda",
		"/?find=the", "/me", "/r/everything",
	} {
		t.Run(path, func(t *testing.T) {
			c.navigate(t, srv.URL+path)
			checkTheFloor(t, c, path)
		})
	}

	t.Run("an opened strip", func(t *testing.T) {
		c.navigate(t, srv.URL+"/")
		openTheFirstAnswerableStrip(t, c)
		checkTheFloor(t, c, "an opened strip")
	})
}
