package web

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Stopping, as a place.
//
// The one screen here that exists to be an ending, and the one most at risk of
// growing a number: a page about stopping is exactly where a well-meaning
// change would add "you did three". These tests are the fence around that.

func TestStoppingIsAPlaceYouCanGet(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/enough", nil).Body.String()

	// The wording varies by the day, so this asks for today's rather than for
	// the one that shipped. What it is checking is that the screen says the
	// thing at all, which is the whole of Principle 3 on a page.
	require.Contains(t, body, squirrel.Say(squirrel.SayingEnough, time.Now()))
	require.Contains(t, body, "the pile keeps")
}

// The pile's own sentence is how you reach it. It carried the third principle
// and was not a control, so the only way to act on it was to close the tab.
func TestThePileOffersTheWayToStop(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{
		note(1, "the boiler makes a noise", squirrel.ItemOpen),
	}}
	body := mounted(t, f).call(t, "GET", "/pile", nil).Body.String()

	require.Contains(t, body, `href="/enough"`)
	require.Contains(t, body, squirrel.Say(squirrel.SayingStop, time.Now()))
}

// Nothing about how much you did. Not a count, not a word for one, not a
// judgement about the size of it.
func TestStoppingSaysNothingAboutHowMuchYouDid(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/enough", nil).Body.String()

	// Only the part this page authored: the layout carries a search field and
	// a mark, and neither is about the session.
	said := body[strings.Index(body, `class="empty enough"`):]
	// Praise and measurement, which are the two ways this page could go wrong.
	// Not a blunt "you did": the copy says "what you did not get to", and a
	// test that cannot tell that from "you did three" would be a test that
	// forces worse writing rather than better behaviour.
	for _, judgement := range []string{
		"well done", "good work", "good job", "nice work", "great",
		"session", "cleared", "progress", "streak", "so far", "keep going",
		"you did well", "you got through", "that many",
	} {
		require.NotContains(t, strings.ToLower(said), judgement,
			"the stopping screen had an opinion about how much you did")
	}
	for _, digit := range []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"} {
		require.NotContains(t, said, ">"+digit, "a number reached the stopping screen")
	}
}

// It reads nothing, so it cannot start counting. The handler takes no store at
// all, and this is the test that says so: a store that fails every call still
// renders the page.
func TestStoppingNeedsNothingFromTheStore(t *testing.T) {
	res := mounted(t, &fakeStore{err: errTest}).call(t, "GET", "/enough", nil)

	require.Equal(t, http.StatusOK, res.Code,
		"the stopping screen asked the store for something")
	require.Contains(t, res.Body.String(), squirrel.Say(squirrel.SayingEnough, time.Now()))
}

// And a way back in, because pressing it and then finding you have another one
// in you is a thing that happens.
func TestStoppingLetsYouChangeYourMind(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/enough", nil).Body.String()

	require.Contains(t, body, "actually, one more")
	require.Contains(t, body, `href="/pile"`)
}
