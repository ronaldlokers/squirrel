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

// The four things Buddy can ask for are one object with four contents.
//
// They were four rules repeating one block of stock — same fill, same outline,
// same radius, same lift — which is four chances for a question to drift into
// looking like a different kind of thing. They share one rule now, and this is
// what says the sharing survives: computed in a browser, because the hazard is
// a later rule of equal weight rather than anything visible in the source.
//
// One fixture each, because only the newest Buddy turn draws its controls: a
// page can hold one of these at a time.
func TestBrowserEveryQuestionIsCutFromTheSameStock(t *testing.T) {
	drawn := []struct{ sel, shown string }{
		{".pick", `{"pick":{"action":"/chores/act","do":"that's it",` +
			`"rows":[{"lead":"every","name":"count","options":["1","2"]}]}}`},
		{".calbox", `{"cal":{"action":"/at/make","month":"August","pad":0,` +
			`"days":[{"day":1,"date":"2026-08-01"}],"times":["14:30"],"do":"that's it"}}`},
		{".wordbox", `{"say":{"action":"/pile/fix","fields":{"id":"9"},` +
			`"was":"the boiler","do":"say it this way"}}`},
		{".cut", `{"cut":{"action":"/pile/split","id":9,"pieces":["a","b"],"do":"use these"}}`},
	}

	var first, firstSel string
	for _, d := range drawn {
		f := aPile()
		f.checkin = fresh()
		f.turns = []squirrel.Turn{{ID: 1, Who: squirrel.SpeakerBuddy, Words: "?", Shown: []byte(d.shown)}}
		c := browserAt(t, screen(t, f), "/")

		stock := c.eval(t, `
			const el = document.querySelector("`+d.sel+`");
			if (!el) return "MISSING";
			const cs = getComputedStyle(el);
			return [cs.backgroundColor, cs.borderTopWidth, cs.borderTopColor, cs.borderRadius,
			        cs.padding, cs.boxShadow, cs.display, cs.flexDirection].join(" | ");`)
		require.NotEqual(t, "MISSING", stock, "%s never rendered, so this measured nothing", d.sel)

		if first == "" {
			first, firstSel = stock.(string), d.sel
			continue
		}
		require.Equal(t, first, stock,
			"%s is not cut from the same stock as %s", d.sel, firstSel)
	}
}
