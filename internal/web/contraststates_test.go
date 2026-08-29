//go:build browser

package web

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func forceEveryPseudo(t *testing.T, c *cdp, state string) {
	t.Helper()
	c.eval(t, `
		const st = document.createElement("style");
		st.textContent = "*, *::before, *::after { transition: none !important; }";
		document.head.append(st); return 1`)

	c.send(t, "DOM.enable", map[string]any{})
	c.send(t, "CSS.enable", map[string]any{})

	doc := c.send(t, "DOM.getDocument", map[string]any{"depth": -1})
	root := doc["root"].(map[string]any)["nodeId"]

	found := c.send(t, "DOM.querySelectorAll", map[string]any{
		"nodeId":   root,
		"selector": "button, a, label.chip, label.day, summary, textarea",
	})
	ids, ok := found["nodeIds"].([]any)
	require.True(t, ok, "no interactive nodes found to force %s on", state)
	require.NotEmpty(t, ids, "no interactive nodes found to force %s on", state)

	for _, id := range ids {
		c.send(t, "CSS.forcePseudoState", map[string]any{
			"nodeId":              id,
			"forcedPseudoClasses": []string{state},
		})
	}
}

func TestEveryWordCanBeReadWhilePressed(t *testing.T) {
	srv := screen(t, everyScreen())
	c := browserAt(t, srv, "/")

	for _, width := range []int{1280, 390} {
		c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
			"width": width, "height": 844, "deviceScaleFactor": 1, "mobile": width < 620,
		})
		for _, path := range []string{"/", "/enough", "/at/4"} {
			for _, state := range []string{"hover", "active", "focus-visible"} {
				c.navigate(t, srv.URL+path)
				forceEveryPseudo(t, c, state)

				var found []unreadable
				raw := c.eval(t, "return ("+contrastWalker+")")
				require.NoError(t, json.Unmarshal([]byte(fmt.Sprint(raw)), &found))

				for _, f := range found {
					t.Errorf("%s at %dpx in :%s: %s %q is %.2f:1, needs %.1f:1 — %s at %.1fpx on %s",
						path, width, state, f.Where, f.Text, f.Ratio, f.Need, f.Color, f.Size, f.Bg)
				}
			}
		}
	}
}
