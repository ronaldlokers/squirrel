//go:build browser

package web

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The page about you, with both readings on it.

func aboutYouStore() *fakeStore {
	f := aFortnight()
	f.knowing = []string{
		"Phone calls get done; forms get put off.",
		"Things started before lunch get finished.",
	}
	return f
}

// Both of these were a press that answered somewhere else. On arrival now, and
// on one screen: this is the page you go to when you wonder about yourself.
func TestBrowserBothReadingsAreOnThePageWithNothingPressed(t *testing.T) {
	c := browserAt(t, screen(t, aboutYouStore()), "/me")
	c.until(t, "the page", `!!document.querySelector('.youpage')`)

	require.Equal(t, float64(6), c.eval(t, `return document.querySelectorAll('.weekrow').length`))
	require.Equal(t, float64(2), c.eval(t, `return document.querySelectorAll('.knows li').length`))
	require.Equal(t, float64(0), c.eval(t,
		`return document.querySelectorAll('.youpage form button[type="submit"]').length
			- document.querySelectorAll('form[action="/me/forget"] button, form[action="/auth/out"] button').length`),
		"something on this page still has to be asked for")
}

// It is a phone screen first, and a grid plus a list of sentences is the
// widest thing this page has ever carried.
func TestBrowserNothingOnThePageAboutYouRunsOffTheSide(t *testing.T) {
	c := browserAt(t, screen(t, aboutYouStore()), "/me")
	c.until(t, "the page", `!!document.querySelector('.knows li')`)

	over := c.eval(t, `return [...document.querySelectorAll('.youpage *')]
		.filter(e => e.getBoundingClientRect().right > innerWidth + 0.5).length`)
	require.Equal(t, float64(0), over, "something on the page is wider than the screen")

	require.GreaterOrEqual(t, c.eval(t,
		`return document.querySelector('form[action="/me/forget"] button')
			.getBoundingClientRect().height`), float64(44),
		"the way to throw it away is smaller than a thumb")
}

// Nothing said, and nothing worked out: two sentences rather than an empty
// page or a grid of forty-two days you did not answer.
func TestBrowserAnEmptyPageSaysWhatWouldFillIt(t *testing.T) {
	c := browserAt(t, screen(t, &fakeStore{}), "/me")
	c.until(t, "the page", `!!document.querySelector('.youpage')`)

	require.Equal(t, float64(0), c.eval(t, `return document.querySelectorAll('.weekrow').length`))
	said := c.eval(t, `return document.body.innerText`)
	require.Contains(t, said, "not said how you are lately")
	require.Contains(t, said, "Nothing yet")
	require.Equal(t, float64(0), c.eval(t,
		`return document.querySelectorAll('form[action="/me/forget"]').length`),
		"a way to forget nothing")
}
