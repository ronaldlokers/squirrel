package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Answering the offer throws away the decision behind it.
//
// The offer on the card is not always the one the picker chose: when a coach is
// configured, `judged` lets the model replace the picker's answer with a
// different row, and the card carries the model's row in its hidden fields. So
// "not now" refuses that row, while the picker's suppression set is keyed on
// the row the picker chose — the picker goes on saying the same thing, the
// decision cache's basis never moves, and the identical card is served back for
// half an hour. From the outside, a button that reloads the page.
//
// The cache cannot see any of that, so the handler says it: whatever was
// answered, and whichever of the four ways it was answered, the held decision
// is gone before the redirect.
//
// Not asserted through a rendering. What matters is that the hook was called
// with this person, because the next render is a different request and the
// thing being invalidated lives in memory between the two.
func forgetful(t *testing.T, f *fakeStore) (*testMux, *[]int64) {
	t.Helper()
	forgot := []int64{}
	m := newTestMux()
	opts := Options{
		RequiredGroup: "squirrel-users", Gate: &Gate{},
		Sessions:    newSessions(alwaysSignedIn{}, cacheFor, cacheMost),
		Login:       aTestLogin,
		ForgetOffer: func(personID int64) { forgot = append(forgot, personID) },
	}
	require.NoError(t, Mount(m, f, opts))
	return m, &forgot
}

func TestAnsweringTheOfferForgetsTheDecisionBehindIt(t *testing.T) {
	for _, act := range []string{"later", "did", "start"} {
		t.Run(act, func(t *testing.T) {
			m, forgot := forgetful(t, answered(aTask))

			form := url.Values{
				"kind": {"task"}, "id": {"7"},
				"label": {"ring the vet"}, "minutes": {"10"},
				"act": {act},
			}
			res := m.call(t, "POST", "/now/act", strings.NewReader(form.Encode()))
			require.Equal(t, 303, res.Code)

			require.Equal(t, []int64{1}, *forgot,
				"the decision was made against the picker's row, and this answer may not have moved it")
		})
	}
}

// "Not today" is the same no, arrived at from the ladder rather than from the
// row of buttons, and it writes the same refusal — so it has to drop the same
// decision.
func TestNotTodayForgetsTheDecisionToo(t *testing.T) {
	m, forgot := forgetful(t, answered(aTask))

	form := url.Values{"kind": {"task"}, "id": {"7"}, "why": {"not today"}}
	res := m.call(t, "POST", "/now/stuck", strings.NewReader(form.Encode()))
	require.Equal(t, 303, res.Code)

	require.Equal(t, []int64{1}, *forgot)
}

// A blocker that is not a refusal changes nothing about what is offered, so it
// leaves the decision alone: the ladder answers underneath the same card, and
// throwing the decision away there would swap the thing you just said you could
// not start.
func TestABlockerThatIsNotARefusalKeepsTheDecision(t *testing.T) {
	m, forgot := forgetful(t, answered(aTask))

	form := url.Values{"kind": {"task"}, "id": {"7"}, "why": {"too big"}}
	m.call(t, "POST", "/now/stuck", strings.NewReader(form.Encode()))

	require.Empty(t, *forgot)
}

// The no-coach build has no decision to forget, and the hook is nil there.
func TestAnsweringTheOfferWithoutACoachIsSafe(t *testing.T) {
	m := mounted(t, answered(aTask))

	form := url.Values{"kind": {"task"}, "id": {"7"}, "act": {"later"}}
	require.NotPanics(t, func() {
		m.call(t, "POST", "/now/act", strings.NewReader(form.Encode()))
	})
}
