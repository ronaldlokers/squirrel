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

func TestTypingAChoreAndPressingEnterAsksForTheRhythm(t *testing.T) {
	f := aBoardStore()
	sp := &fakeSpool{}
	m := mountedSpooling(t, f, sp)

	form := blankStripIn(t, m.call(t, "GET", "/?bay=chores", nil).Body.String(), "bay=chores")
	sent := whatTheBrowserWouldSend(form)
	require.Equal(t, "chores", sent.Get("bay"), "the blank strip does not say which rack it is")
	sent.Set("words", "defrost the freezer")

	res := m.call(t, "POST", "/board/new", strings.NewReader(sent.Encode()))

	require.Empty(t, f.reinterval.name, "the screen guessed a rhythm nobody typed")
	require.Empty(t, sp.written)
	require.Equal(t, 303, res.Code)
	require.Equal(t, "/?bay=chores&rhythm=defrost+the+freezer", res.Header().Get("Location"),
		"the words were not carried back to the question")
}

func TestTheRhythmCountShowsSevenWithoutSendingIt(t *testing.T) {
	form := blankStripIn(t, mounted(t, aBoardStore()).call(t, "GET", "/?bay=chores", nil).Body.String(), "bay=chores")

	require.Contains(t, form, `placeholder="7"`)
	require.Empty(t, whatTheBrowserWouldSend(form).Get("every"), "the count sends a rhythm on its own")
}
