package web

import (
	"errors"
	htmlpkg "html"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Buddy, in the conversation.

// asked is what Buddy said back, as the record kept it: the words and whatever
// the turn drew.
func asked(t *testing.T, m *testMux, f *fakeStore, body string) string {
	t.Helper()
	f.appended = nil
	m.call(t, "POST", "/buddy/say", strings.NewReader(body))
	require.NotEmpty(t, f.appended, "nothing was said back")
	last := f.appended[len(f.appended)-1]
	return last.Words + " " + string(last.Shown)
}

// The way in is the rail, which is on every screen and never closes.
//
// It was a chip on the live edge, then a menu entry, and it is furniture now:
// Buddy is a room, and going to a room is a link. Looking something up is not
// a room — it is a thing you do — so it sits below the rule with the way out.
func TestTheWayToBuddyIsOnTheRail(t *testing.T) {
	f := &fakeStore{
		items:   []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)},
		checkin: &squirrel.Checkin{Mood: squirrel.MoodGood},
	}
	body := mounted(t, f).call(t, "GET", "/r/everything", nil).Body.String()

	require.Contains(t, body, `href="/r/everything"`)
	require.Contains(t, body, "look something up")
	require.Contains(t, body, `action="/find/ask"`)
}

// Even with nothing said yet.
//
// This was the argument for chips at the foot rather than on the live edge:
// the live edge is the newest Buddy turn, and a brand new conversation has
// none, so a way to Buddy that appears once Buddy has spoken is a way to Buddy
// you cannot use to start. The rail settles it — it is furniture, so it is
// there before anything is.
func TestTheWayToBuddyIsThereBeforeAnythingIsSaid(t *testing.T) {
	f := &fakeStore{checkin: fresh()}
	body := thread(t, f)

	require.Contains(t, body, `href="/r/everything"`)
	require.Contains(t, body, "look something up")
}

// Asking costs nothing. The question is painted from the picker, which is six
// rules and no model — idle asks have to be free, or the way in becomes a
// thing you think about before pressing.
func TestAskingBuddyCallsNoModel(t *testing.T) {
	f := &fakeStore{offer: &squirrel.Offer{
		Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet",
	}}
	c := &fakeCoach{reply: "should not be called"}

	f.appended = nil
	mountedWith(t, f, c).call(t, "POST", "/buddy/ask", nil)

	require.Contains(t, f.appended[1].Words, "ring the vet",
		"the question is about nothing")
	require.Empty(t, c.asked, "asking called a model")
}

// The narrower half of the same rule, and the one that was missed once: it may
// consult a decision that was already paid for, and may never cause one.
func TestAskingBuddyNeverPaysForADecision(t *testing.T) {
	f := withOffer(&squirrel.Offer{
		Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet", Because: "you decided this one",
	})
	c := &fakeCoach{decision: &fakeDecision{
		kind: "chore", refID: 3, text: "put the bins out", because: "they go out tonight",
	}}

	f.appended = nil
	mountedWith(t, f, c).call(t, "POST", "/buddy/ask", nil)

	require.Equal(t, 1, c.peeked, "asking did not go through the cache")
	require.Contains(t, f.appended[1].Words, "ring the vet")
	require.NotContains(t, f.appended[1].Words, "put the bins out")
}

// The conversation is the screen the decision is for, so it may pay. Stated as
// its own test because the rule is asymmetric, and an asymmetry nobody pinned
// is one that quietly becomes symmetric.
func TestTheConversationMayPayForADecision(t *testing.T) {
	f := withOffer(&squirrel.Offer{
		Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet", Because: "you decided this one",
	})
	c := &fakeCoach{decision: &fakeDecision{
		kind: "chore", refID: 3, text: "put the bins out", because: "they go out tonight",
	}}

	body := mountedWith(t, f, c).call(t, "GET", "/r/everything", nil).Body.String()

	require.Zero(t, c.peeked, "the conversation went through the cache instead of asking")
	require.Contains(t, body, "put the bins out")
}

// A low day does not empty it. Someone who has pressed *ask Buddy* has already
// overridden the quiet by asking.
func TestBuddyAnswersOnALowDay(t *testing.T) {
	f := &fakeStore{
		offer: &squirrel.Offer{Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet"},
		gated: true,
	}
	f.appended = nil
	mounted(t, f).call(t, "POST", "/buddy/ask", nil)

	require.Contains(t, f.appended[1].Words, "ring the vet")
}

// Nothing to be handed is a normal state. The question is asked anyway —
// having nothing offered is not a reason to have nothing to say — and it does
// not invent a subject.
func TestWithNothingToHandBuddyStillAsks(t *testing.T) {
	f := &fakeStore{}
	f.appended = nil
	mounted(t, f).call(t, "POST", "/buddy/ask", nil)

	require.Equal(t, "What is going on?", f.appended[1].Words)
	require.Contains(t, string(f.appended[1].Shown), `"action":"/buddy/say"`)
}

// Typing is never required. Four chips, one press, and the words the ladder
// already uses.
func TestBuddyOffersTheFourWithoutTyping(t *testing.T) {
	f := &fakeStore{}
	m := mounted(t, f)
	drew := asked(t, m, f, "said=everything+at+once")

	for _, b := range squirrel.Blockers {
		require.Contains(t, drew, string(b), "no chip for %q", b)
		require.Contains(t, drew, squirrel.BlockerWords[b])
	}
}

func TestSayingSomethingAsksAboutWhatIsOnScreen(t *testing.T) {
	f := &fakeStore{offer: &squirrel.Offer{
		Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet",
	}}
	c := &fakeCoach{reply: "Ring them. It is two minutes."}
	m := mountedWith(t, f, c)

	f.appended = nil
	w := m.call(t, "POST", "/buddy/say", strings.NewReader("said=I+can%27t+face+it"))

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/r/everything", w.Header().Get("Location"))

	require.Len(t, c.asked, 1)
	require.Equal(t, "thread", c.asked[0].kind)
	require.Equal(t, "I can't face it", c.asked[0].said)
	require.Equal(t, "ring the vet", c.asked[0].subject)

	// Both voices, in the record, in order.
	require.Equal(t, "I can't face it", f.appended[0].Words)
	require.Contains(t, f.appended[1].Words, "Ring them. It is two minutes.")
}

// The floor. Words that reached the server do not evaporate because a model
// was unreachable — the record is what the box used to be.
func TestACoachThatCannotAnswerKeepsTheWords(t *testing.T) {
	f := &fakeStore{}
	c := &fakeCoach{err: errors.New("no coach available")}
	m := mountedWith(t, f, c)
	drew := asked(t, m, f, "said=everything+at+once")

	require.Equal(t, "everything at once", f.appended[0].Words, "the words were eaten")
	require.Contains(t, drew, "Which of these is it?", "nothing was offered instead")
}

// With no key at all it still works: the words are kept and the four chips are
// the answer. The ladder cannot read a paragraph, but it can be asked which of
// four things is in the way.
func TestBuddyWorksWithNoCoachAtAll(t *testing.T) {
	f := &fakeStore{}
	m := mounted(t, f)
	drew := asked(t, m, f, "said=everything+at+once")

	require.Equal(t, "everything at once", f.appended[0].Words)
	require.Contains(t, drew, "Which of these is it?")
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

	f.appended = nil
	w := m.call(t, "POST", "/buddy/say", strings.NewReader("why=big"))

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/r/everything", w.Header().Get("Location"))
	require.Empty(t, c.asked, "a chip called a model")
	require.Equal(t, squirrel.BlockerWords[squirrel.BlockerBig], f.appended[0].Words)
	require.Contains(t, f.appended[1].Words, squirrel.UnstuckFor(squirrel.BlockerBig).Line)
}

// "Not today" is not an obstacle, it is a no — and it is the same no that "not
// now" writes, arrived at from another direction.
func TestNotTodayTurnsTheThingDownAndEndsTheConversation(t *testing.T) {
	f := &fakeStore{offer: &squirrel.Offer{
		Kind: squirrel.OfferTask, RefID: 7, Text: "ring the vet",
	}}
	c := &fakeCoach{}
	m := mountedWith(t, f, c)

	w := m.call(t, "POST", "/buddy/say", strings.NewReader("why=not+today"))

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/r/everything", w.Header().Get("Location"))
	require.Equal(t, []int64{7}, f.refused)
	require.Equal(t, 1, c.forgot, "turning something down left the conversation open")
}

func TestSayingNothingAsksNothing(t *testing.T) {
	c := &fakeCoach{reply: "should not be called"}
	f := &fakeStore{}
	f.appended = nil
	mountedWith(t, f, c).call(t, "POST", "/buddy/say", strings.NewReader("said=+++"))

	require.Empty(t, c.asked)
	require.Empty(t, f.appended, "an empty press said something")
}

// Buddy is a room now, and is still not a place that holds things.
//
// This read "not a fifth place" until 28 August, when he became the first of
// seven. What it was guarding survives the change and is what is asserted
// here: the other six draw a list on arrival, and his draws none. He is where
// you talk; they are where things live.
func TestBuddysRoomHoldsNothing(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	m := newTestMux()
	require.NoError(t, Mount(m, f, signedInOptions()))

	f.appended = nil
	m.call(t, "GET", "/r/everything", nil)

	for _, turn := range f.appended {
		require.NotContains(t, string(turn.Shown), `"place":`,
			"Buddy's room drew a place")
	}

	// And the notes' does, so this measured the difference rather than an empty
	// store. Drawn rather than kept since 31 August 2026 — see view.Edge — so
	// the difference is read where a room's list now lives.
	require.NotEmpty(t, drewIn(t, f, "notes"), "the notes drew no place")
}

// Nothing here renders a total, in either direction. The conversation is a
// conversation, not a tally of how often you needed one.
func TestBuddyNeverEmitsACount(t *testing.T) {
	f := &fakeStore{}
	m := mounted(t, f)
	drew := onlyWords(asked(t, m, f, "said=one"))

	for _, n := range []string{"3", "three messages"} {
		require.NotContains(t, drew, n, "a number reached the reply")
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

// The sheet carried this in its own lid, and the rule it protects is that a
// running cost must never be a number you meet before you have asked for
// anything. The sheet is gone; the reply is the one place left that satisfies
// it, because you have asked by the time you can read it.
func TestTheReplySaysWhatItHasCost(t *testing.T) {
	f := withOffer(nil)
	c := &fakeCoach{reply: "Ring them.", spent: "€2.61", ceiling: "€10.00"}
	m := mountedWith(t, f, c)
	drew := asked(t, m, f, "said=what+now")

	require.Contains(t, drew, "€2.61")
	require.Contains(t, drew, "€10.00")
}

// Nowhere else, and that is the point.
func TestNoScreenSaysWhatItHasCostBeforeYouAsk(t *testing.T) {
	c := &fakeCoach{spent: "€2.61", ceiling: "€10.00"}
	m := mountedWith(t, withOffer(nil), c)

	for _, path := range []string{"/r/everything", "/r/chores"} {
		require.NotContains(t, m.call(t, "GET", path, nil).Body.String(), "€2.61",
			"the spend leaked onto %s", path)
	}
}

// A figure that cannot be trusted is a figure not drawn.
func TestASpendThatCannotBeReadIsNotShown(t *testing.T) {
	f := withOffer(nil)
	m := mountedWith(t, f, &fakeCoach{reply: "Ring them."})

	require.NotContains(t, asked(t, m, f, "said=what+now"), "€")
}

func TestWithNoCoachThereIsNoSpendLine(t *testing.T) {
	f := withOffer(nil)
	m := mounted(t, f)

	require.NotContains(t, asked(t, m, f, "said=what+now"), "€")
}

// No ceiling is a supported choice, and "of €0.00" would read as one that had
// been reached.
func TestWithNoCeilingOnlyTheSpendIsShown(t *testing.T) {
	f := withOffer(nil)
	c := &fakeCoach{reply: "Ring them.", spent: "€2.61"}
	m := mountedWith(t, f, c)
	drew := asked(t, m, f, "said=what+now")

	require.Contains(t, drew, "€2.61")
	require.NotContains(t, drew, " of ")
}

// A fixed sentence cannot land badly in a way worth recording: it is the same
// sentence every time, and it came from the core rather than from a model.
func TestOnlyAModelsWordsOfferTheWayToSayTheyLandedBadly(t *testing.T) {
	f := &fakeStore{}
	c := &fakeCoach{reply: "Ring them."}
	withModel := asked(t, mountedWith(t, f, c), f, "said=what+now")
	require.Contains(t, withModel, "that went badly")

	bare := &fakeStore{}
	require.NotContains(t, asked(t, mounted(t, bare), bare, "said=what+now"),
		"that went badly")
}

func TestTheOldCoachURLsRedirect(t *testing.T) {
	m := mounted(t, &fakeStore{})

	for _, gone := range []string{"/coach", "/buddy"} {
		w := m.call(t, "GET", gone, nil)
		require.Equal(t, 301, w.Code, "%s died quietly", gone)
		require.Equal(t, "/r/everything", w.Header().Get("Location"))
	}
}

// Closing was a route, and is not one. The sheet was a thing that could be
// open; a conversation is not.
func TestThereIsNoWayToCloseAConversation(t *testing.T) {
	m := mounted(t, &fakeStore{})

	require.NotContains(t, m.routes, "POST /buddy/close")
}

// Asking Buddy to show you a place opens it, as cards, in his own turn.
//
// He could not before: Guard refuses a list and the brief is two sentences, so
// the one thing he could honestly say was that he could not — while the menu
// beside him did it in one press.
func TestBuddyOpensAPlaceWhenAskedFor(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		{ID: 9, RawText: "ring the vet", Kind: squirrel.ItemTask, State: squirrel.ItemOpen},
	}}
	c := &fakeCoach{reply: "Here they are.", opens: "tasks"}
	m := mountedWith(t, f, c)
	post(t, m, "/buddy/say", url.Values{"said": {"can you show me the tasks"}})

	require.Len(t, f.appended, 3)
	require.Equal(t, "can you show me the tasks", f.appended[0].Words)
	require.Equal(t, "Here they are.", f.appended[1].Words)
	// His turn, not yours: the record must not invent a sentence you never
	// typed.
	require.Equal(t, squirrel.SpeakerBuddy, f.appended[2].Who)
	require.Contains(t, string(f.appended[2].Shown), "ring the vet")
}

// And says nothing extra when he did not ask for one, which is nearly every
// turn.
func TestAnOrdinaryReplyOpensNothing(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		{ID: 9, RawText: "ring the vet", Kind: squirrel.ItemTask, State: squirrel.ItemOpen},
	}}
	c := &fakeCoach{reply: "That one first."}
	m := mountedWith(t, f, c)
	post(t, m, "/buddy/say", url.Values{"said": {"what should I do"}})

	require.Len(t, f.appended, 2)
}

// A place that does not exist draws nothing rather than an empty turn. The
// screen holds the vocabulary and the lookup miss is the refusal.
func TestAPlaceBuddyInventedDrawsNothing(t *testing.T) {
	f := &fakeStore{}
	c := &fakeCoach{reply: "Here you go.", opens: "inbox"}
	m := mountedWith(t, f, c)
	post(t, m, "/buddy/say", url.Values{"said": {"show me the inbox"}})

	require.Len(t, f.appended, 2)
}
