//go:build browser

package web

import (
	"encoding/json"
	"fmt"
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
      // A control clipped for screen readers only is not one anybody taps, and
      // the thing it duplicates carries the floor. checkVisibility calls it
      // visible because it is not hidden — it is one pixel and clipped.
      if (el.closest('.offscreen')) return;
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

func checkTheFloor(t *testing.T, c *cdp, where string) {
	t.Helper()
	for _, f := range controlsUnderTheFloor(t, c) {
		t.Errorf("%s: %s %q is %vx%v, under the 44x44 floor", where, f.Name, f.Text, f.Width, f.Height)
	}
}

func TestBrowserEveryControlClearsFortyFourOnAPhone(t *testing.T) {
	srv := screen(t, everyScreen())
	c := browserAt(t, srv, "/")
	touching(t, c)

	for _, path := range []string{
		"/", "/?bay=weekly", "/?bay=tasks", "/?bay=agenda",
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
