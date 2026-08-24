package web

import (
	"errors"
	htmlpkg "html"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Buddy is on every screen, and it is a real link to a real page — which is
// what makes the sheet an upgrade rather than a requirement.
//
// It was a button floating over the content and it is an icon in the lid now,
// so the class changed with it. What the test is about did not: one tap from
// anywhere, and it knows where to come back to.
func TestTheAcornIsOnEveryScreenAndLinksBack(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	m := mounted(t, f)

	href := regexp.MustCompile(`<a class="lidbtn tobuddy" href="([^"]+)"`)
	for _, path := range []string{"/", "/pile", "/kept"} {
		body := m.call(t, "GET", path, nil).Body.String()
		found := href.FindStringSubmatch(body)
		require.Len(t, found, 2, "no way to Buddy on %s", path)

		// Parsed rather than string-matched: the template escapes the path
		// into the query, and a test that pins the escaping is a test about
		// html/template rather than about the link.
		u, err := url.Parse(found[1])
		require.NoError(t, err)
		require.Equal(t, "/buddy", u.Path)
		require.Equal(t, path, u.Query().Get("from"),
			"Buddy on %s does not know where to come back to", path)
	}
}

// Not on the coach page itself. A link to the page you are on is furniture.
func TestTheAcornIsNotOnTheCoachPage(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/buddy", nil).Body.String()
	require.NotContains(t, body, `class="lidbtn tobuddy"`)
}

// The button and the card's drawn badge are two different things with two
// different names, and this is here because for one release they were not:
// taking `.acorn` for the button turned every note's badge into a 62px circle
// in the corner of the screen, and made pile.js wire the badge instead of the
// button, so the sheet never opened on the pile at all.
func TestTheAcornButtonDoesNotTakeTheCardsBadgeName(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/pile", nil).Body.String()

	require.Contains(t, body, `<svg class="acorn"`, "the card lost its badge")
	require.Contains(t, body, `class="lidbtn tobuddy"`, "the button lost its own name")
	require.NotContains(t, body, `<a class="acorn"`)
}

// Opening costs nothing: the sheet paints with what the picker would hand you,
// and no model is called. Idle opens have to be free or the acorn becomes a
// thing you think about before pressing.
func TestOpeningTheCoachCallsNoModel(t *testing.T) {
	f := &fakeStore{offer: &squirrel.Offer{
		Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet",
		Because: "you decided this one",
	}}
	c := &fakeCoach{reply: "should not be called"}

	body := mountedWith(t, f, c).call(t, "GET", "/buddy", nil).Body.String()

	require.Contains(t, body, "ring the vet")
	require.Contains(t, body, "you decided this one")
	require.Empty(t, c.asked, "opening the sheet asked a model")
}

// The narrower half of the same rule, and the one that was missed: the sheet
// may consult a decision that was already paid for, and may never cause one.
//
// This test exists because the first version of it only checked the
// conversational seam, which stayed quiet while the picker's seam paid for a
// tool loop on every acorn press.
func TestOpeningTheCoachNeverPaysForADecision(t *testing.T) {
	f := withOffer(&squirrel.Offer{
		Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet", Because: "you decided this one",
	})
	c := &fakeCoach{decision: &fakeDecision{
		kind: "chore", refID: 3, text: "put the bins out", because: "they go out tonight",
	}}

	body := mountedWith(t, f, c).call(t, "GET", "/buddy", nil).Body.String()

	require.Equal(t, 1, c.peeked, "the sheet did not go through the cache")
	require.Contains(t, body, "ring the vet", "the sheet did not fall back to the picker")
	require.NotContains(t, body, "put the bins out")
}

// Home is the screen the decision is for, so it may pay. Stated as its own
// test because the rule is asymmetric and an asymmetry nobody pinned is an
// asymmetry that quietly becomes symmetric.
func TestHomeMayPayForADecision(t *testing.T) {
	f := withOffer(&squirrel.Offer{
		Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet", Because: "you decided this one",
	})
	c := &fakeCoach{decision: &fakeDecision{
		kind: "chore", refID: 3, text: "put the bins out", because: "they go out tonight",
	}}

	body := mountedWith(t, f, c).call(t, "GET", "/", nil).Body.String()

	require.Zero(t, c.peeked, "home went through the cache instead of asking")
	require.Contains(t, body, "put the bins out")
}

// A low day does not empty the sheet. Someone who has opened the coach has
// already overridden the quiet by asking.
func TestTheCoachPaintsOnALowDay(t *testing.T) {
	f := &fakeStore{
		offer: &squirrel.Offer{Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet"},
		gated: true,
	}
	body := mounted(t, f).call(t, "GET", "/buddy", nil).Body.String()
	require.Contains(t, body, "ring the vet")
}

// Nothing to be handed is a normal state. It renders as an absence rather than
// as a reassuring sentence, which would be the product deciding you ought to
// be busy.
func TestTheCoachWithNothingToHandSaysNothingAboutIt(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/buddy", nil).Body.String()
	require.NotContains(t, body, "ON SCREEN")
	// The way in is still there. Having nothing offered is not a reason to
	// have nothing to say.
	require.Contains(t, body, `name="said"`)
}

// Typing is never required. Four chips, one press, and the words the ladder
// already uses.
func TestTheCoachOffersTheFourWithoutTyping(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/buddy", nil).Body.String()
	for _, b := range squirrel.Blockers {
		require.Contains(t, body, `value="`+string(b)+`"`, "no chip for %q", b)
		// The words, escaped the way the template escapes them — "don't know
		// how" carries an apostrophe and the page is HTML.
		require.Contains(t, body, template.HTMLEscapeString(squirrel.BlockerWords[b]))
	}
}

func TestSayingSomethingAsksTheCoachAboutWhatIsOnScreen(t *testing.T) {
	f := &fakeStore{offer: &squirrel.Offer{
		Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet",
	}}
	c := &fakeCoach{reply: "Ring them. It is two minutes."}
	m := mountedWith(t, f, c)

	w := m.call(t, "POST", "/buddy/say", strings.NewReader("said=I+can%27t+face+it"))

	// A redirect, so the answer survives a reload and back does not offer to
	// ask again. The conversation is in the window; the page is a view of it.
	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/buddy", w.Header().Get("Location"))

	require.Len(t, c.asked, 1)
	require.Equal(t, "sheet", c.asked[0].kind)
	require.Equal(t, "I can't face it", c.asked[0].said)
	require.Equal(t, "ring the vet", c.asked[0].subject)

	require.Contains(t, m.call(t, "GET", "/buddy", nil).Body.String(),
		"Ring them. It is two minutes.")
}

// The conversation, oldest first, both voices.
func TestTheCoachShowsWhatWasSaidAndAnswered(t *testing.T) {
	c := &fakeCoach{talk: []Exchange{
		{Said: "what now", Replied: "The envelope."},
		{Said: "no, something else", Replied: "The bins, then."},
	}}
	body := mountedWith(t, &fakeStore{}, c).call(t, "GET", "/buddy", nil).Body.String()

	require.Contains(t, body, "what now")
	require.Contains(t, body, "The envelope.")
	require.Less(t, strings.Index(body, "what now"), strings.Index(body, "no, something else"),
		"the conversation is oldest first")
}

// The floor, and the rule the slot already has: a box that clears on failure
// is a box that eats thoughts.
func TestACoachThatCannotAnswerKeepsTheWords(t *testing.T) {
	c := &fakeCoach{err: errors.New("no coach available")}
	body := mountedWith(t, &fakeStore{}, c).
		call(t, "POST", "/buddy/say", strings.NewReader("said=everything+at+once")).Body.String()

	require.Contains(t, body, "everything at once", "the words were eaten")
	require.Contains(t, body, "WHICH OF THESE IS IT", "nothing was offered instead")
}

// With no key at all the sheet still works: the words are kept and the four
// chips are the answer. The ladder cannot read a paragraph, but it can be
// asked which of four things is in the way.
func TestTheSheetWorksWithNoCoachAtAll(t *testing.T) {
	body := mounted(t, &fakeStore{}).
		call(t, "POST", "/buddy/say", strings.NewReader("said=everything+at+once")).Body.String()

	require.Contains(t, body, "everything at once")
	require.Contains(t, body, "WHICH OF THESE IS IT")
}

// A chip is answered by the ladder and stays answered by the ladder. The worst
// case of having a coach at all is that you press one and read the same fixed
// sentence you would have read anyway.
func TestAChipIsAnsweredByTheLadderAndNotByAModel(t *testing.T) {
	f := &fakeStore{offer: &squirrel.Offer{
		Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet",
	}}
	c := &fakeCoach{reply: "should not be called"}
	m := mountedWith(t, f, c)

	w := m.call(t, "POST", "/buddy/say", strings.NewReader("why=big"))
	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/buddy?stuck=big", w.Header().Get("Location"))
	require.Empty(t, c.asked, "a chip called a model")

	body := m.call(t, "GET", "/buddy?stuck=big", nil).Body.String()
	require.Contains(t, body, squirrel.UnstuckFor(squirrel.BlockerBig).Line)
	// One line and at most one control. There is nowhere here for a second
	// step, which is the whole point.
	require.Contains(t, body, "5 MIN")
}

// "Not today" is not an obstacle, it is a no — and it is the same no that "not
// now" writes on the home screen, arrived at from another direction.
func TestNotTodayTurnsTheThingDownAndEndsTheConversation(t *testing.T) {
	f := &fakeStore{offer: &squirrel.Offer{
		Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet",
	}}
	c := &fakeCoach{}
	m := mountedWith(t, f, c)

	w := m.call(t, "POST", "/buddy/say", strings.NewReader("why=not+today"))

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/", w.Header().Get("Location"))
	require.Equal(t, []int64{7}, f.refused)
	require.Equal(t, 1, c.forgot, "turning something down left the conversation open")
}

// Closing means the conversation is over, and it means nothing else.
func TestClosingForgetsTheConversationAndGoesBack(t *testing.T) {
	c := &fakeCoach{talk: []Exchange{{Said: "what now", Replied: "The envelope."}}}
	m := mountedWith(t, &fakeStore{}, c)

	w := m.call(t, "POST", "/buddy/close", strings.NewReader("from=%2Ftasks"))

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/tasks", w.Header().Get("Location"))
	require.Equal(t, 1, c.forgot)
}

// The value arrives from a form field, and a form field is a place a stranger
// can type. A page behind forward-auth is still a page.
func TestClosingWillNotBeSentSomewhereElse(t *testing.T) {
	for _, from := range []string{
		"https://example.com/", "//example.com/", "javascript:alert(1)", "",
	} {
		w := mounted(t, &fakeStore{}).call(t, "POST", "/buddy/close",
			strings.NewReader("from="+from))
		require.Equal(t, "/", w.Header().Get("Location"), "sent away by %q", from)
	}
}

// Saying nothing does nothing, and costs nothing.
func TestSayingNothingAsksNothing(t *testing.T) {
	c := &fakeCoach{reply: "should not be called"}
	w := mountedWith(t, &fakeStore{}, c).
		call(t, "POST", "/buddy/say", strings.NewReader("said=+++"))

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Empty(t, c.asked)
}

// The sheet is chrome, not a destination: the lid still offers the two places
// you are not, and home still has three doors.
func TestTheCoachDidNotBecomeAFourthDoor(t *testing.T) {
	home := mounted(t, &fakeStore{}).call(t, "GET", "/", nil).Body.String()
	require.NotContains(t, home, `href="/coach"`,
		"the acorn carries a from parameter; a bare link would be a fourth door")
	// Four doors on the rail, and the sheet is not one of them.
	require.Equal(t, 4, strings.Count(home, `class="rdoor`))
}

// Nothing here renders a total, in either direction. The conversation is a
// conversation, not a tally of how often you needed one.
func TestTheCoachNeverEmitsACount(t *testing.T) {
	c := &fakeCoach{talk: []Exchange{
		{Said: "one", Replied: "a"}, {Said: "two", Replied: "b"}, {Said: "three", Replied: "c"},
	}}
	body := mountedWith(t, &fakeStore{}, c).call(t, "GET", "/buddy", nil).Body.String()

	// The words, not the markup. This grepped the raw response until the lid
	// grew drawn icons and the hamburger's own path — "M3 5.5h14M3 10h14" —
	// matched a search for "3 ". A count is a thing a person reads, so the
	// test reads what a person would.
	for _, n := range []string{"3", "three messages"} {
		require.NotContains(t, onlyWords(body), n,
			"a number reached the sheet")
	}
}

// onlyWords is the response with every tag taken out and every entity turned
// back into the character it stands for, so an assertion about what is on the
// screen cannot be satisfied or broken by markup.
//
// Both halves earned their place. Stripping tags came first, because the
// hamburger's own path — "M3 5.5h14M3 10h14" — answered a search for a "3".
// Unescaping came second, because "don&#39;t know how" answered the next one.
func onlyWords(markup string) string {
	var out strings.Builder
	depth := 0
	for _, r := range markup {
		switch {
		case r == '<':
			depth++
		case r == '>':
			if depth > 0 {
				depth--
			}
		case depth == 0:
			out.WriteRune(r)
		}
	}
	return htmlpkg.UnescapeString(out.String())
}

// What the coach has cost, in the sheet's own lid and nowhere else.
func TestTheSheetSaysWhatItHasCost(t *testing.T) {
	c := &fakeCoach{spent: "€2.61", ceiling: "€10.00"}
	body := mountedWith(t, withOffer(nil), c).call(t, "GET", "/buddy", nil).Body.String()

	require.Contains(t, body, "€2.61")
	require.Contains(t, body, "€10.00")
}

// Nowhere else, and that is the point. A running cost on the home screen would
// be a number you meet before you have asked for anything.
func TestNoOtherScreenSaysWhatItHasCost(t *testing.T) {
	c := &fakeCoach{spent: "€2.61", ceiling: "€10.00"}
	m := mountedWith(t, withOffer(nil), c)

	for _, path := range []string{"/", "/pile", "/kept"} {
		require.NotContains(t, m.call(t, "GET", path, nil).Body.String(), "€2.61",
			"the spend leaked onto %s", path)
	}
}

// A figure that cannot be trusted is a figure not drawn.
func TestASpendThatCannotBeReadIsNotShown(t *testing.T) {
	body := mountedWith(t, withOffer(nil), &fakeCoach{}).
		call(t, "GET", "/buddy", nil).Body.String()
	require.NotContains(t, body, "€")
}

// With no coach there is nothing to report on, so nothing reports.
func TestWithNoCoachThereIsNoSpendLine(t *testing.T) {
	require.NotContains(t, mounted(t, withOffer(nil)).call(t, "GET", "/buddy", nil).Body.String(), "€")
}

// No ceiling is a supported choice, and "of €0.00" would read as one that had
// been reached.
func TestWithNoCeilingOnlyTheSpendIsShown(t *testing.T) {
	c := &fakeCoach{spent: "€2.61"}
	body := mountedWith(t, withOffer(nil), c).call(t, "GET", "/buddy", nil).Body.String()

	require.Contains(t, body, "€2.61")
	require.NotContains(t, body, " of ")
}
