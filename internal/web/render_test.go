package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestEveryActionIsAFormSubmissionNotAScriptHook(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	body := opened(t, f, "notes")

	require.Contains(t, body, `method="post"`)
	require.Contains(t, body, `action="/board/act"`)
	require.NotContains(t, body, "onclick=",
		"behaviour lives in pile.js; a page that needs inline handlers is a page that fails without them")
	// Every answer travels as a button's own value inside a form rather than on
	// a script hook, so a press is a form submission whatever the script is
	// doing — and every one of them names the thing it acts on.
	require.Equal(t,
		strings.Count(body, `name="answer" value=`),
		strings.Count(body, `<button class="stamp`)-strings.Count(body, `name="every" value=`)-strings.Count(body, `name="chore" value=`),
		"an answer that is not a submit button is an answer only a script can make")
}
