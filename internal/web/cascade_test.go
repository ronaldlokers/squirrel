//go:build browser

// Two rules that were being outvoted.
//
// Both of the defects these cover looked right in the source and were wrong on
// the screen, because a later rule of equal or greater weight was quietly
// winning. That is the same shape as the dialog that closed and stayed visible
// and the photograph preview that was hidden and wasn't: the file said one
// thing and the render said another.
//
// So these ask the browser for the computed value rather than reading the
// stylesheet, and they compare against a sibling rather than against a number
// wherever the point is that two things must match.
package web

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// styleOf is one resolved property of the first element matching sel. Resolved
// rather than declared: the whole point is what the cascade settled on.
const styleOf = `(sel, prop) => {
	const el = document.querySelector(sel);
	if (!el) throw new Error("no such element: " + sel);
	return getComputedStyle(el).getPropertyValue(prop);
}`

func style(t *testing.T, c *cdp, sel, prop string) string {
	t.Helper()
	got := c.eval(t, fmt.Sprintf("return (%s)(%q, %q)", styleOf, sel, prop))
	s, ok := got.(string)
	require.True(t, ok, "%s of %s came back as %#v", prop, sel, got)
	return s
}

// A field under 16px makes iOS zoom the page the moment it takes focus, which
// on a phone throws the lid, the panel and whatever you were reading around.
//
// Every other field in this product clears the floor by being large for the
// Step-Up Rule's own reasons. The search field is small on purpose — it sits
// in a 320px panel — so the floor is the only thing holding it up, and it
// stopped applying without anything failing: the rule was written for
// `.find input`, and when search moved behind an icon the field became
// `.findbox .find input`, which outranks it.

// The offer's three controls are one shape, and only colour separates them.
// That is the whole argument for the row: it used to offer four choices in
// four different kinds of object, so choosing meant comparing the objects
// before you could compare the options.
//
// `not now` was still not one of them. `.later` gives every quiet pill a 2px
// currentColor border — right for a pill standing on the purple field with
// nothing behind it, wrong in a row of two controls wearing the system's 3px
// outline. Equal specificity, later in the file, so it won.
func TestBrowserTheOffersThirdControlIsTheSameShapeAsTheOtherTwo(t *testing.T) {
	f := aPile()
	f.offer = &squirrel.Offer{
		Kind:    squirrel.OfferTask,
		RefID:   3,
		Text:    "ring the boiler people",
		Because: "you decided this on tuesday",
	}
	// Home draws the offer inside `{{if .Mood}}`, so an offer with no reading
	// behind it renders the check-in question instead and this test finds
	// nothing. A minute ago rather than time.Now() so the reading is never
	// fractionally in the future, and a duration rather than a date because
	// freshness here is six hours from the reading and has no calendar in it.
	f.checkin = &squirrel.Checkin{
		Mood: squirrel.MoodGood, SaidAt: time.Now().Add(-time.Minute),
	}

	srv := cameraScreen(t, f, &fakeSpool{}, &fakePhotos{})
	c := browserAt(t, srv, "/")

	for _, prop := range []string{"border-top-width", "border-top-color"} {
		// .abtn now: the offer is an ordinary card in the thread, wearing the
		// same buttons every other card wears, which is most of the point.
		did := style(t, c, ".abtn.did", prop)
		later := style(t, c, ".abtn.later", prop)
		require.Equal(t, did, later,
			"the offer's %s differs: `did it` is %s, `not now` is %s", prop, did, later)
	}
}
