package web

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The confirmation surface, on the sheet. Nothing here is applied by anything
// except the press, and everything arrives back through a form — so everything
// is read the way a stranger's typing is read.

func proposes(p *Proposal) *fakeCoach {
	return &fakeCoach{reply: "Shall I?", propose: p}
}

func TestAProposalIsRenderedAndNothingIsWritten(t *testing.T) {
	f := withOffer(nil)
	c := proposes(&Proposal{
		Do: "moment", Said: "Shall I keep 14:30 for the dentist?",
		Text: "dentist", At: "14:30",
	})

	body := mountedWith(t, f, c).
		call(t, "POST", "/buddy/say", strings.NewReader("said=dentist+at+half+two")).Body.String()

	require.Contains(t, body, "Shall I keep 14:30 for the dentist?")
	require.Contains(t, body, `value="moment"`)
	require.Contains(t, body, "KEEP IT")
	require.Empty(t, f.moments, "a proposal created something")
}

// It travels in the form that renders it. Nothing stores it, so it cannot be
// applied by anything but the press — and a reload asks again rather than
// doing it.
func TestAProposalIsStoredNowhere(t *testing.T) {
	f := withOffer(nil)
	c := proposes(&Proposal{Do: "chore", Said: "Shall I?", Text: "bins", Every: "every 2 weeks"})
	m := mountedWith(t, f, c)

	m.call(t, "POST", "/buddy/say", strings.NewReader("said=the+bins+keep+piling+up"))

	body := m.call(t, "GET", "/buddy", nil).Body.String()
	require.NotContains(t, body, "KEEP IT", "a proposal survived the page it was on")
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

	m.call(t, "POST", "/buddy/say", strings.NewReader("said=I+did+the+bins"))

	body := m.call(t, "GET", "/buddy", nil).Body.String()
	require.Contains(t, body, "Done.")
	require.Contains(t, body, "put the bins out is done")
}
