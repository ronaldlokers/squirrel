package web

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// No zooming on a phone, and the two halves of it that reach different places.
//
// The viewport meta holds in the installed app and is ignored by iOS in a
// browser tab; `touch-action: manipulation` is honoured in both. Neither of
// them stops a field zooming the page when it takes focus, which is what the
// 16px floor is for — so the fence that matters most here is the last test,
// which stops someone deleting that floor on the strength of this feature.

func TestTheViewportRefusesToScale(t *testing.T) {
	body := templates(t)["layout.html"]

	meta := regexp.MustCompile(`<meta name="viewport" content="([^"]+)">`).FindStringSubmatch(body)
	require.Len(t, meta, 2, "the layout has no viewport meta at all")

	for _, want := range []string{
		"width=device-width",
		"initial-scale=1",
		// The notch. Losing this is how the lid ends up under the status bar.
		"viewport-fit=cover",
		"user-scalable=no",
		"maximum-scale=1",
	} {
		require.Contains(t, meta[1], want, "the viewport dropped %s", want)
	}
}

// A double tap that lands slightly fast is the normal gesture, not a mistake.
func TestDoubleTapDoesNotMagnify(t *testing.T) {
	css, err := staticFS.ReadFile("static/pile.css")
	require.NoError(t, err)

	require.Contains(t, string(css), "touch-action: manipulation",
		"double-tap zoom is back, and the viewport meta does not stop it in a browser tab")
	require.NotContains(t, string(css), "touch-action: none",
		"`none` would take panning with it; `manipulation` is the one that only drops double-tap")
}

// The floor stays, and this is the test that says why.
//
// `user-scalable=no` looks like it makes the 16px rule redundant. It does not:
// iOS ignores it in a browser tab, and the zoom that happens when a field
// takes focus is a different mechanism from the one that meta tag governs.
// Every field in this product has to clear 16px on a phone whatever the
// viewport says.
func TestEveryFieldStillClearsTheZoomFloorOnAPhone(t *testing.T) {
	css, err := staticFS.ReadFile("static/pile.css")
	require.NoError(t, err)

	phone := phoneBlock(t, string(css))

	// Each of these is a field a thumb puts a caret in.
	for _, sel := range []string{
		".slot textarea",
		".reword textarea",
		".newtask textarea",
		".findbox .find input",
	} {
		size := fontSizeFor(phone, sel)
		require.NotEmpty(t, size, "%s has no phone size, so it renders at its desktop one", sel)
		require.GreaterOrEqual(t, size, 16.0,
			"%s is %gpx on a phone; under 16px, focusing it zooms the page", sel, size)
	}
}

// phoneBlock is everything inside a max-width: 620px media query, which is
// where this stylesheet keeps its phone sizes.
func phoneBlock(t *testing.T, css string) string {
	t.Helper()

	var out strings.Builder
	for _, chunk := range strings.Split(css, "@media (max-width: 620px)")[1:] {
		out.WriteString(chunk)
	}
	require.NotEmpty(t, out.String(), "the stylesheet has no phone breakpoint")
	return out.String()
}

// fontSizeFor reads the last font-size declared for sel, in px. Last, because
// a later rule of equal weight wins — which is the whole reason the search
// field spent a release at 15px while a 16px rule for it sat in the file.
func fontSizeFor(css, sel string) float64 {
	rule := regexp.MustCompile(regexp.QuoteMeta(sel) + `\s*\{([^}]*)\}`)
	size := regexp.MustCompile(`font-size:\s*([0-9.]+)px`)

	var found float64
	for _, block := range rule.FindAllStringSubmatch(css, -1) {
		if m := size.FindStringSubmatch(block[1]); m != nil {
			found = atof(m[1])
		}
	}
	return found
}

func atof(s string) float64 {
	var f float64
	var frac float64 = 0
	for _, r := range s {
		switch {
		case r == '.':
			frac = 1
		case frac > 0:
			frac /= 10
			f += float64(r-'0') * frac
		default:
			f = f*10 + float64(r-'0')
		}
	}
	return f
}
