package web

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// One step on the screen, and never the sequence.

func breaksInto(c *fakeCoach, steps ...string) *fakeCoach {
	c.steps = steps
	return c
}

func TestPressingTooBigShowsOneStepAndNeverTheList(t *testing.T) {
	f := withOffer(&squirrel.Offer{
		Kind: squirrel.OfferTask, RefID: 7, Text: "the tax thing",
	})
	c := breaksInto(&fakeCoach{}, "open the letter", "find the reference", "ring the number")
	m := mountedWith(t, f, c)

	w := m.call(t, "POST", "/now/stuck", strings.NewReader("why=big&kind=task&id=7"))
	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/?stuck=big", w.Header().Get("Location"))

	body := m.call(t, "GET", "/?stuck=big", nil).Body.String()
	require.Contains(t, body, "open the letter")
	require.NotContains(t, body, "find the reference")
	require.NotContains(t, body, "ring the number")

	// The fixed line stays underneath it. It is the floor this stands on, not
	// a draft it replaces.
	require.Contains(t, body, squirrel.UnstuckFor(squirrel.BlockerBig).Line)
}

// A model that is slow, absent or wrong costs nothing anyone can see, because
// the line was always what rendered.
func TestTooBigFallsBackToTheLineOnTheScreen(t *testing.T) {
	f := withOffer(&squirrel.Offer{Kind: squirrel.OfferTask, RefID: 7, Text: "the tax thing"})
	m := mountedWith(t, f, &fakeCoach{})

	m.call(t, "POST", "/now/stuck", strings.NewReader("why=big&kind=task&id=7"))
	body := m.call(t, "GET", "/?stuck=big", nil).Body.String()

	require.Contains(t, body, squirrel.UnstuckFor(squirrel.BlockerBig).Line)
	require.NotContains(t, body, `class="step"`)
}

// The other three have answers that are not a sequence.
func TestOnlyTooBigAsksForABreakdownOnTheScreen(t *testing.T) {
	for _, why := range []string{"how", "boring"} {
		f := withOffer(&squirrel.Offer{Kind: squirrel.OfferTask, RefID: 7, Text: "the tax thing"})
		c := breaksInto(&fakeCoach{}, "one", "two")
		m := mountedWith(t, f, c)

		m.call(t, "POST", "/now/stuck", strings.NewReader("why="+why+"&kind=task&id=7"))
		require.Zero(t, c.broke, "%q asked for a breakdown", why)
	}
}

// Coming back an hour later and finding the step you were on is the whole
// reason the sequence is stored rather than held in a reply.
func TestTheStepIsStillThereOnTheSheet(t *testing.T) {
	f := withOffer(nil)
	f.steps = []squirrel.Step{{ID: 1, Label: "the tax thing", Body: "open the letter"}}

	body := mounted(t, f).call(t, "GET", "/buddy", nil).Body.String()
	require.Contains(t, body, "open the letter")
}

func TestFinishingAStepMovesToTheNext(t *testing.T) {
	f := withOffer(nil)
	f.steps = []squirrel.Step{
		{ID: 1, Body: "open the letter"},
		{ID: 2, Body: "ring the number", Last: true},
	}
	m := mounted(t, f)

	w := m.call(t, "POST", "/steps", strings.NewReader("act=done&id=1&from=%2Fbuddy"))
	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/buddy", w.Header().Get("Location"))
	require.Equal(t, []int64{1}, f.finished)

	body := m.call(t, "GET", "/buddy", nil).Body.String()
	require.Contains(t, body, "ring the number")
	require.Contains(t, body, "that is the last one")
}

// One press, no consequence, nothing asked back — the same shape "not now"
// already has.
func TestForgettingTheStepsCostsOnePress(t *testing.T) {
	f := withOffer(nil)
	f.steps = []squirrel.Step{{ID: 1, Body: "open the letter"}}
	m := mounted(t, f)

	m.call(t, "POST", "/steps", strings.NewReader("act=clear&from=%2Fbuddy"))
	require.Equal(t, 1, f.cleared)

	body := m.call(t, "GET", "/buddy", nil).Body.String()
	require.NotContains(t, body, "open the letter")
}

// The value arrives from a form field, and a form field is a place a stranger
// can type.
func TestFinishingAStepWillNotSendYouSomewhereElse(t *testing.T) {
	f := withOffer(nil)
	f.steps = []squirrel.Step{{ID: 1, Body: "open the letter"}}

	w := mounted(t, f).call(t, "POST", "/steps",
		strings.NewReader("act=done&id=1&from=https%3A%2F%2Fexample.com%2F"))
	require.Equal(t, "/", w.Header().Get("Location"))
}

// A step is never a count of what is left.
func TestAStepOnTheScreenNeverSaysHowManyAreLeft(t *testing.T) {
	f := withOffer(nil)
	f.steps = []squirrel.Step{{ID: 1, Body: "open the letter"}}

	body := mounted(t, f).call(t, "GET", "/buddy", nil).Body.String()
	for _, count := range []string{"of 3", "1/3", "step 1", "1 of"} {
		require.NotContains(t, body, count)
	}
}

// A sequence that cannot be read must not take down a page that rendered
// without one for the product's whole life.
func TestAnUnreadableSequenceIsNoStep(t *testing.T) {
	f := withOffer(nil)
	f.steps = []squirrel.Step{{ID: 1, Body: "open the letter"}}
	f.err = errTest

	require.NotPanics(t, func() {
		_ = mounted(t, f).call(t, "GET", "/buddy", nil)
	})
}

// The breakdown is asked for with what is on screen, and stored against the
// task it belongs to.
func TestTheBreakdownIsAboutWhatIsOnScreen(t *testing.T) {
	f := withOffer(&squirrel.Offer{Kind: squirrel.OfferTask, RefID: 7, Text: "the tax thing"})
	c := breaksInto(&fakeCoach{}, "open the letter", "ring the number")
	m := mountedWith(t, f, c)

	m.call(t, "POST", "/buddy/say", strings.NewReader("why=big"))

	require.Equal(t, "the tax thing", c.brokeDown)
	require.Equal(t, "too big", c.brokeBlocker)
	require.NotNil(t, f.stepItem)
	require.Equal(t, int64(7), *f.stepItem)
}
