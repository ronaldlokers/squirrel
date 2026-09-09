package web

import (
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	aTag     = regexp.MustCompile(`<(input|select)\b[^>]*>`)
	anAttr   = regexp.MustCompile(`(\w[\w-]*)="([^"]*)"`)
	anOption = regexp.MustCompile(`<option value="([^"]*)"`)
)

func whatTheBrowserWouldSend(form string) url.Values {
	sent := url.Values{}
	for _, tag := range aTag.FindAllString(form, -1) {
		attrs := map[string]string{}
		for _, a := range anAttr.FindAllStringSubmatch(tag, -1) {
			attrs[a[1]] = a[2]
		}
		name := attrs["name"]
		if name == "" || attrs["type"] == "file" {
			continue
		}
		if strings.HasPrefix(tag, "<select") {
			rest := form[strings.Index(form, tag):]
			if o := anOption.FindStringSubmatch(rest); o != nil {
				sent.Set(name, o[1])
			}
			continue
		}
		if attrs["type"] == "radio" && !strings.Contains(tag, " checked") {
			continue
		}
		sent.Set(name, attrs["value"])
	}
	return sent
}

func blankStripIn(t *testing.T, page, bay string) string {
	t.Helper()
	rack := theRackIn(t, page, bay)
	from := strings.Index(rack, `<form class="blankstrip"`)
	require.GreaterOrEqual(t, from, 0, "the rack has no blank strip")
	to := strings.Index(rack[from:], "</form>")
	require.GreaterOrEqual(t, to, 0, "the blank strip does not close")
	return rack[from : from+to]
}

// One writer for the whole board since 9 September 2026: a modal that asks what
// the thing is and then only what that kind needs.
func theChoreWriter(t *testing.T, page string) string {
	t.Helper()
	from := strings.Index(page, `<form class="addform`)
	require.GreaterOrEqual(t, from, 0, "the board has nowhere to put anything")
	to := strings.Index(page[from:], "</form>")
	require.GreaterOrEqual(t, to, 0, "the writer does not close")
	return page[from : from+to]
}

func theTaskWriter(t *testing.T, page string) string {
	t.Helper()
	from := strings.Index(page, `<form class="newtask`)
	require.GreaterOrEqual(t, from, 0, "the board has nowhere to put a thing you do one time")
	to := strings.Index(page[from:], "</form>")
	require.GreaterOrEqual(t, to, 0, "the task writer does not close")
	return page[from : from+to]
}

func TestTypingAChoreAndPressingEnterAsksForTheRhythm(t *testing.T) {
	f := aBoardStore()
	sp := &fakeSpool{}
	m := mountedSpooling(t, f, sp)

	sent := whatTheBrowserWouldSend(theChoreWriter(t, m.call(t, "GET", "/", nil).Body.String()))
	sent.Set("words", "defrost the freezer")
	sent.Set("bay", "daily")
	sent.Del("every")

	res := m.call(t, "POST", "/board/new", strings.NewReader(sent.Encode()))

	require.Empty(t, f.reinterval.name, "the screen guessed a rhythm nobody typed")
	require.Empty(t, sp.written)
	require.Equal(t, 303, res.Code)
	require.Equal(t, "/?bay=daily&rhythm=defrost+the+freezer", res.Header().Get("Location"),
		"the words were not carried back to the question")
}

// How often is four presets rather than a number and a unit: the answer is
// nearly always one of four, and two decisions where one will do is the tax
// this product exists to stop charging.
func TestTheRhythmIsFourPresetsAndAWeekIsTheOneStanding(t *testing.T) {
	form := theChoreWriter(t, mounted(t, aBoardStore()).call(t, "GET", "/", nil).Body.String())

	for _, want := range []string{"every day", "every week", "2 weeks", "a month"} {
		require.Contains(t, form, want, "the writer does not offer %q", want)
	}
	require.NotContains(t, form, `name="unit"`, "the writer still asks for a unit")
	require.Equal(t, "7", whatTheBrowserWouldSend(form).Get("every"),
		"a week is not what an unanswered rhythm means")
}
