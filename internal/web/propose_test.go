package web

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The confirmation surface. Nothing here is applied by anything except the
// press, and everything arrives back through a form — so everything is read
// the way a stranger's typing is read.

func proposes(p *Proposal) *fakeCoach {
	return &fakeCoach{reply: "Shall I?", propose: p}
}

func TestAProposalIsRenderedAndNothingIsWritten(t *testing.T) {
	f := withOffer(nil)
	c := proposes(&Proposal{
		Do: "moment", Said: "Shall I keep 14:30 for the dentist?",
		Text: "dentist", At: "14:30",
	})

	m := mountedWith(t, f, c)
	drew := asked(t, m, f, "said=dentist+at+half+two")

	require.Contains(t, drew, "Shall I keep 14:30 for the dentist?")
	require.Contains(t, drew, `"moment"`)
	require.Contains(t, drew, "KEEP IT")
	require.Empty(t, f.moments, "a proposal created something")
}

// A proposal is answerable while it is the newest thing Buddy said, and not
// after. It travels in the form that renders it and nothing else holds it, so
// a proposal that has scrolled up is a card with its words and no press —
// there is no stored intent for a later reload to act on.
func TestAProposalStopsBeingAnswerableOnceSomethingElseIsSaid(t *testing.T) {
	f := withOffer(nil)
	c := proposes(&Proposal{Do: "chore", Said: "Shall I?", Text: "bins", Every: "every 2 weeks"})
	m := mountedWith(t, f, c)

	m.call(t, "POST", "/buddy/say", strings.NewReader("said=the+bins+keep+piling+up"))
	require.Contains(t, string(f.appended[len(f.appended)-1].Shown), "KEEP IT",
		"the proposal never rendered, so this measures nothing")

	// Said, and then said over.
	f.turns = append(append(f.turns, f.appended...),
		squirrel.Turn{ID: 99, Who: squirrel.SpeakerBuddy, Words: "Something else."})
	f.appended = nil
	f.checkin = fresh()

	require.NotContains(t, thread(t, f), "KEEP IT",
		"a proposal in scrollback still offers a press")
}

func TestKeepingAMomentCreatesIt(t *testing.T) {
	f := withOffer(nil)

	w := mounted(t, f).call(t, "POST", "/buddy/do",
		strings.NewReader("do=moment&text=dentist&at=14%3A30"))

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Len(t, f.moments, 1)
	require.Equal(t, "dentist", f.moments[0].Label)
}

// The time goes through the core's own parser rather than being trusted from
// the form. A guessed time is a missed appointment, and the bar that keeps a
// note from silently becoming something that interrupts you is not lowered
// because a model was the one asking.
func TestAMomentThatDoesNotParseCreatesNothing(t *testing.T) {
	f := withOffer(nil)

	w := mounted(t, f).call(t, "POST", "/buddy/do",
		strings.NewReader("do=moment&text=dentist&at=sometime+soon"))

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Empty(t, f.moments)
}

func TestKeepingAChoreCreatesItWithItsRhythm(t *testing.T) {
	f := withOffer(nil)

	mounted(t, f).call(t, "POST", "/buddy/do",
		strings.NewReader("do=chore&text=put+the+bins+out&every=every+2+weeks"))

	require.Equal(t, "put the bins out", f.reinterval.name)
	require.Equal(t, 14*24*time.Hour, f.reinterval.every)
}

func TestAChoreWithNoRhythmCreatesNothing(t *testing.T) {
	f := withOffer(nil)

	mounted(t, f).call(t, "POST", "/buddy/do",
		strings.NewReader("do=chore&text=put+the+bins+out&every=whenever"))

	require.Empty(t, f.reinterval.name)
}

// Only ever a chore that is yours, and the active list is what says so.
func TestRetiringOnlyEverTouchesYourOwnChore(t *testing.T) {
	f := withOffer(nil)
	f.chores = []squirrel.Chore{{ID: 3, Name: "put the bins out"}}
	m := mounted(t, f)

	m.call(t, "POST", "/buddy/do", strings.NewReader("do=retire&id=99"))
	require.Empty(t, f.retired, "it retired something that is not yours")

	m.call(t, "POST", "/buddy/do", strings.NewReader("do=retire&id=3"))
	require.Equal(t, []int64{3}, f.retired)
}

func TestDroppingOnlyEverTouchesYourOwnNote(t *testing.T) {
	f := withOffer(nil)
	f.items = []squirrel.Item{note(1, "the boiler makes a noise", squirrel.ItemOpen)}
	m := mounted(t, f)

	m.call(t, "POST", "/buddy/do", strings.NewReader("do=drop&id=99"))
	require.Empty(t, f.states)

	m.call(t, "POST", "/buddy/do", strings.NewReader("do=drop&id=1"))
	require.Equal(t, map[int64]squirrel.ItemState{1: squirrel.ItemDropped}, f.states)
}

// Four things and no more. A kind that is not one of them does nothing at all,
// because the route is a switch rather than a dispatcher.
func TestAnUnknownProposalDoesNothing(t *testing.T) {
	f := withOffer(nil)
	f.items = []squirrel.Item{note(1, "the boiler makes a noise", squirrel.ItemOpen)}

	for _, do := range []string{"reword", "delete", "checkin", ""} {
		w := mounted(t, f).call(t, "POST", "/buddy/do",
			strings.NewReader("do="+do+"&id=1&text=anything"))
		require.Equal(t, http.StatusSeeOther, w.Code)
	}
	require.Empty(t, f.states)
	require.Empty(t, f.moments)
	require.Empty(t, f.reinterval.name)
}

// What actually changed goes into the conversation alongside what was said,
// in the application's words.
func TestWhatChangedIsShownInTheConversation(t *testing.T) {
	f := withOffer(nil)
	c := &fakeCoach{reply: "Done.", did: []string{"put the bins out is done"}}
	m := mountedWith(t, f, c)

	drew := asked(t, m, f, "said=I+did+the+bins")

	require.Contains(t, drew, "Done.")
	require.Contains(t, drew, "put the bins out is done")
}
