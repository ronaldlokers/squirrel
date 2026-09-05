package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestTheOpenedStripOffersToBeReworded(t *testing.T) {
	f := aBoardStore()
	body := mounted(t, f).call(t, "GET", "/?open=1", nil).Body.String()

	require.Contains(t, body, "say it another way")
	require.NotContains(t, body, `action="/board/fix"`)
}

func TestAskingToRewordShowsTheFieldWithWhatItSaysNow(t *testing.T) {
	f := aBoardStore()
	body := mounted(t, f).call(t, "GET", "/?open=1&reword=1", nil).Body.String()

	require.Contains(t, body, `action="/board/fix"`)
	require.Contains(t, body, `value="boiler service code is 4471"`)
}

func TestRewordingChangesOnlyTheWords(t *testing.T) {
	f := aBoardStore()
	m := mounted(t, f)

	w := post(t, m, "/board/fix", url.Values{"id": {"1"}, "text": {"boiler code is 9911 now"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "/?open=1", w.Header().Get("Location"))
	require.Equal(t, "boiler code is 9911 now", f.items[0].RawText)
	require.Equal(t, squirrel.ItemOpen, f.items[0].State)
}

func TestRewordingWithEmptyTextChangesNothing(t *testing.T) {
	f := aBoardStore()
	m := mounted(t, f)

	post(t, m, "/board/fix", url.Values{"id": {"1"}, "text": {"  "}})

	require.Equal(t, "boiler service code is 4471", f.items[0].RawText)
}

func TestTheOpenedStripOffersTheThreeWaysItCannotBeActedOn(t *testing.T) {
	f := aBoardStore()
	body := mounted(t, f).call(t, "GET", "/?open=1", nil).Body.String()

	for _, want := range []string{
		`name="answer" value="waiting"`, `name="answer" value="blocked"`, `name="answer" value="someday"`,
	} {
		require.Contains(t, body, want)
	}
}

func TestPressingWaitingSetsItAside(t *testing.T) {
	f := aBoardStore()
	m := mounted(t, f)

	w := post(t, m, "/board/act", url.Values{"what": {"note"}, "id": {"1"}, "answer": {"waiting"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, squirrel.ItemWaiting, f.items[0].State)
}

func TestAClosedStripOffersNoCorrectionsOrHeldButtons(t *testing.T) {
	f := aBoardStore()
	f.items[0].State = squirrel.ItemDone
	body := mounted(t, f).call(t, "GET", "/?open=1", nil).Body.String()

	require.NotContains(t, body, "say it another way")
	require.NotContains(t, body, `name="answer" value="waiting"`)
}

func TestTheOpenedStripsCorrectionDoesNotLeakOntoRackStrips(t *testing.T) {
	f := aBoardStore()
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.NotContains(t, body, "say it another way")
	require.NotContains(t, body, `name="answer" value="waiting"`)
	require.Equal(t, 0, strings.Count(body, `action="/board/fix"`))
}
