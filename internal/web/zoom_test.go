package web

import (
	"regexp"
	"strconv"
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

	// The two fields a thumb can put a caret in: the dock's slot, and the box
	// Buddy draws when he asks for words. There are no others — a list here
	// that names a field no screen has is a floor holding nothing up.
	for _, sel := range []string{".slot textarea", ".wordbox textarea"} {
		size := phoneSizeOf(t, string(css), sel)
		require.GreaterOrEqual(t, size, 16.0,
			"%s is %gpx on a phone; under 16px, focusing it zooms the page", sel, size)
	}
}

// phoneSizeOf is the size sel renders at on a phone: its size inside the phone
// breakpoint, or its base size when it does not restate one. A field is allowed
// to have no phone rule — it has to clear the floor, not to be listed twice.
func phoneSizeOf(t *testing.T, css, sel string) float64 {
	t.Helper()
	base, phone := split620(t, css)
	if size := fontSizeFor(phone, sel); size > 0 {
		return size
	}
	size := fontSizeFor(base, sel)
	require.NotZero(t, size, "%s declares no size anywhere, so nothing here is measuring it", sel)
	return size
}

// split620 is the stylesheet either side of its phone breakpoint. Phone rules
// win at 620px and under, so a field's size there is the phone rule when it has
// one and the base rule when it has not.
func split620(t *testing.T, css string) (base, phone string) {
	t.Helper()
	parts := strings.Split(css, "@media (max-width: 620px)")
	require.Greater(t, len(parts), 1, "the stylesheet has no phone breakpoint")
	return parts[0], strings.Join(parts[1:], "")
}

// fontSizeFor reads the last size declared for sel, in px, from either
// `font-size:` or the `font:` shorthand — both are in this stylesheet, and a
// reader that saw only one of them measured half the fields it was given.
//
// Last, because a later rule of equal weight wins: that is the whole reason the
// search field spent a release at 15px while a 16px rule for it sat in the file.
func fontSizeFor(css, sel string) float64 {
	rule := regexp.MustCompile(regexp.QuoteMeta(sel) + `\s*\{([^}]*)\}`)
	longhand := regexp.MustCompile(`font-size:\s*([0-9.]+)px`)
	shorthand := regexp.MustCompile(`(?:^|[;\s])font:[^;]*?([0-9.]+)px`)

	var found float64
	for _, block := range rule.FindAllStringSubmatch(css, -1) {
		for _, at := range []*regexp.Regexp{longhand, shorthand} {
			if m := at.FindStringSubmatch(block[1]); m != nil {
				size, err := strconv.ParseFloat(m[1], 64)
				if err == nil {
					found = size
				}
			}
		}
	}
	return found
}
