package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// This is the only thing in the product that holds an opinion about somebody
// rather than something they said. A product that quietly builds a picture of
// you that you cannot read is not this product.

func knows(said ...string) *fakeStore {
	return &fakeStore{knowing: said}
}

func TestYouCanReadWhatItKnows(t *testing.T) {
	f := knows(
		"Phone calls get done; forms get put off.",
		"Things started before lunch get finished.",
	)
	f.appended = nil
	routed(t, f).call(t, "POST", "/knowing", nil)

	require.Len(t, f.appended, 2)
	drew := string(f.appended[1].Shown)
	require.Contains(t, drew, "Phone calls get done")
	require.Contains(t, drew, "Things started before lunch")
	require.Contains(t, drew, "forget all of it",
		"there is no way to throw it away")
}

// In its own words, not a summary of them. A paraphrase of an opinion about
// you is a second opinion about you.
func TestItIsShownInItsOwnWords(t *testing.T) {
	f := knows("Phone calls get done; forms get put off.")
	f.appended = nil
	routed(t, f).call(t, "POST", "/knowing", nil)

	require.Contains(t, string(f.appended[1].Shown),
		"Phone calls get done; forms get put off.")
}

func TestItSaysItCouldBeWrong(t *testing.T) {
	f := knows("Phone calls get done.")
	f.appended = nil
	routed(t, f).call(t, "POST", "/knowing", nil)

	require.Contains(t, strings.ToLower(f.appended[1].Words), "could be wrong")
}

// Nothing yet is the ordinary state for a first week. It says what would
// change that rather than apologising.
func TestNothingKnownYetSaysWhatWouldChangeThat(t *testing.T) {
	f := &fakeStore{}
	f.appended = nil
	routed(t, f).call(t, "POST", "/knowing", nil)

	require.Contains(t, f.appended[1].Words, "Nothing yet")
	require.Contains(t, f.appended[1].Words, "about once a week")
	for _, apology := range []string{"sorry", "unfortunately", "not enough"} {
		require.NotContains(t, strings.ToLower(f.appended[1].Words), apology)
	}
}

// One press to throw it away, and the answer says what that costs.
func TestYouCanMakeItForget(t *testing.T) {
	f := knows("Phone calls get done.")
	f.appended = nil
	routed(t, f).call(t, "POST", "/knowing/forget", nil)

	require.True(t, f.forgot, "it did not forget")
	require.Empty(t, f.knowing)
	require.Contains(t, f.appended[1].Words, "Forgotten")
	require.Contains(t, f.appended[1].Words, "from here",
		"it did not say what forgetting costs")
}

// The way to it is on a reply a model wrote, and only there: what Squirrel
// knows shapes those and nothing else, so beside a fixed sentence from the
// core it would point at something that had no part in it.
func TestTheWayToItIsOnAModelsReply(t *testing.T) {
	f := &fakeStore{}
	c := &fakeCoach{reply: "Ring them."}
	require.Contains(t, asked(t, mountedWith(t, f, c), f, "said=what+now"),
		"what do you know about me")

	bare := &fakeStore{}
	require.NotContains(t, asked(t, mounted(t, bare), bare, "said=what+now"),
		"what do you know about me")
}

// It is not a door and not at the foot of the thread. It is a thing you check
// when you wonder, and wondering about Buddy happens in Buddy's own turn.
func TestItIsNotAPlaceYouGo(t *testing.T) {
	f := knows("Phone calls get done.")
	f.checkin = fresh()
	body := thread(t, f)

	require.NotContains(t, body, `value="knowing"`)
	require.NotContains(t, body, "what do you know about me")
}

// A store that cannot be read says so rather than answering with an error
// page. Nothing on the screen depends on this.
func TestAStoreThatCannotBeReadSaysSo(t *testing.T) {
	f := &fakeStore{err: errTest}
	f.appended = nil
	w := routed(t, f).call(t, "POST", "/knowing", nil)

	require.Equal(t, 303, w.Code)
}

// That the model is told never to say any of this back is pinned in
// internal/coach, where the sentence lives — see TestTheModelIsToldNotToSayIt
// Back in knowing_test.go there.
