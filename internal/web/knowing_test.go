package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// This is the only thing in the product that holds an opinion about somebody
// rather than something they said. A product that quietly builds a picture of
// you that you cannot read is not this product.
//
// It was a press that answered in the conversation until 3 September 2026. It
// is drawn on the page about you now: the conversation is being retired, and
// an opinion about you belongs where the rest of you is.

func knows(said ...string) *fakeStore {
	return &fakeStore{knowing: said}
}

func aboutYou(t *testing.T, f *fakeStore) string {
	t.Helper()
	return mounted(t, f).call(t, "GET", "/me", nil).Body.String()
}

func TestYouCanReadWhatItKnows(t *testing.T) {
	f := knows(
		"Phone calls get done; forms get put off.",
		"Things started before lunch get finished.",
	)
	f.appended = nil
	body := aboutYou(t, f)

	require.Contains(t, body, "Phone calls get done")
	require.Contains(t, body, "Things started before lunch")
	require.Contains(t, body, `action="/me/forget"`, "there is no way to throw it away")
	require.Empty(t, f.appended, "reading the page put something in the record")
}

// In its own words, not a summary of them. A paraphrase of an opinion about
// you is a second opinion about you.
func TestItIsShownInItsOwnWords(t *testing.T) {
	f := knows("Phone calls get done; forms get put off.")

	require.Contains(t, aboutYou(t, f), "Phone calls get done; forms get put off.")
}

func TestItSaysItCouldBeWrong(t *testing.T) {
	require.Contains(t, strings.ToLower(aboutYou(t, knows("Phone calls get done."))),
		"could be wrong")
}

// Nothing yet is the ordinary state for a first week. It says what would
// change that rather than apologising.
func TestNothingKnownYetSaysWhatWouldChangeThat(t *testing.T) {
	body := aboutYou(t, &fakeStore{})

	require.Contains(t, body, "Nothing yet")
	require.Contains(t, body, "about once a week")
	require.NotContains(t, body, `action="/me/forget"`,
		"a way to forget nothing")
	for _, apology := range []string{"sorry", "unfortunately", "not enough"} {
		require.NotContains(t, strings.ToLower(body), apology)
	}
}

// One press to throw it away, and it lands back on the page that showed it —
// where the empty state is now what says what forgetting cost.
func TestYouCanMakeItForget(t *testing.T) {
	f := knows("Phone calls get done.")
	f.appended = nil
	res := mounted(t, f).call(t, "POST", "/me/forget", nil)

	require.True(t, f.forgot, "it did not forget")
	require.Empty(t, f.knowing)
	require.Equal(t, 303, res.Code)
	require.Equal(t, "/me", res.Header().Get("Location"))
	require.Empty(t, f.appended, "forgetting was written into the record as something said")
}

// The way to it is on a reply a model wrote, and only there: what Squirrel
// knows shapes those and nothing else, so beside a fixed sentence from the
// core it would point at something that had no part in it. A link now, since
// what it points at is a page.
func TestTheWayToItIsOnAModelsReply(t *testing.T) {
	f := &fakeStore{}
	c := &fakeCoach{reply: "Ring them."}
	said := asked(t, mountedWith(t, f, c), f, "said=what+now")
	require.Contains(t, said, "what do you know about me")
	require.Contains(t, said, `"href":"/me"`, "it still answers in the room")

	bare := &fakeStore{}
	require.NotContains(t, asked(t, mounted(t, bare), bare, "said=what+now"),
		"what do you know about me")
}

// Not at the foot of the thread and not a door off it.
func TestItIsNotAPlaceYouGoFromTheConversation(t *testing.T) {
	f := knows("Phone calls get done.")
	f.checkin = fresh()
	body := thread(t, f)

	require.NotContains(t, body, `value="knowing"`)
	require.NotContains(t, body, "what do you know about me")
}

// A store that cannot be read says so on the page rather than answering with
// an error. Nothing else on the page depends on this one read.
func TestAStoreThatCannotBeReadSaysSo(t *testing.T) {
	body := aboutYou(t, &fakeStore{knowingErr: errTest})

	require.Contains(t, body, "cannot reach that just now")
	require.Contains(t, body, "Who you are", "the rest of the page went with it")
}

// The address it had lands on the page that draws it. It may be on a home
// screen.
func TestTheOldAddressForWhatItKnowsStillLands(t *testing.T) {
	res := mounted(t, &fakeStore{}).call(t, "GET", "/knowing", nil)

	require.Equal(t, 301, res.Code)
	require.Equal(t, "/me", res.Header().Get("Location"))
}
