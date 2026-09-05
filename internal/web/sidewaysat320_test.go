//go:build browser

package web

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

const sidewaysOverflowWalker = `(() => {
  const doc = document.body;
  const over = doc.scrollWidth - doc.clientWidth;
  if (over <= 0) return JSON.stringify({ over: 0 });
  const cw = doc.clientWidth;
  const name = el => el.tagName.toLowerCase() +
    (el.className ? '.' + el.className.toString().trim().split(/\s+/).join('.') : '') +
    (el.id ? '#' + el.id : '');
  let worst = null;
  document.querySelectorAll('body *').forEach(el => {
    const r = el.getBoundingClientRect();
    const past = r.right - cw;
    if (past > 0.5 && (!worst || past > worst.past)) {
      worst = { name: name(el), past: Math.round(past * 10) / 10, text: (el.textContent || '').trim().slice(0, 48) };
    }
  });
  return JSON.stringify({ over: over, worst: worst });
})()`

type sidewaysOverflow struct {
	Over  float64 `json:"over"`
	Worst *struct {
		Name string  `json:"name"`
		Past float64 `json:"past"`
		Text string  `json:"text"`
	} `json:"worst"`
}

func sidewaysOverflowOf(t *testing.T, c *cdp) sidewaysOverflow {
	t.Helper()
	var found sidewaysOverflow
	raw := c.eval(t, "return ("+sidewaysOverflowWalker+")")
	require.NoError(t, json.Unmarshal([]byte(fmt.Sprint(raw)), &found))
	return found
}

func at320(t *testing.T, c *cdp) {
	t.Helper()
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 320, "height": 844, "deviceScaleFactor": 2, "mobile": true,
	})
	c.send(t, "Emulation.setTouchEmulationEnabled", map[string]any{"enabled": true, "maxTouchPoints": 1})
	c.send(t, "Emulation.setEmitTouchEventsForMouse", map[string]any{"enabled": true, "configuration": "mobile"})
}

func requireNoSidewaysScroll(t *testing.T, c *cdp, where string) {
	t.Helper()
	found := sidewaysOverflowOf(t, c)
	if found.Over <= 0 {
		return
	}
	if found.Worst != nil {
		t.Errorf("%s scrolls sideways by %v CSS px at 320px: %s %q sits %v past the edge",
			where, found.Over, found.Worst.Name, found.Worst.Text, found.Worst.Past)
		return
	}
	t.Errorf("%s scrolls sideways by %v CSS px at 320px", where, found.Over)
}

func TestBrowserNothingScrollsSidewaysAtThreeTwenty(t *testing.T) {
	srv := screen(t, everyScreen())
	c := browserAt(t, srv, "/")
	at320(t, c)

	for _, path := range []string{
		"/", "/?bay=chores", "/?bay=tasks", "/?bay=agenda",
		"/?find=the", "/me", "/r/everything",
	} {
		c.navigate(t, srv.URL+path)
		requireNoSidewaysScroll(t, c, path)
	}

	c.navigate(t, srv.URL+"/")
	openTheFirstAnswerableStrip(t, c)
	requireNoSidewaysScroll(t, c, "an opened strip")
}
