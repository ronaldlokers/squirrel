package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestEveryActionIsAFormSubmissionNotAScriptHook(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	body := opened(t, f, "pile")

	require.Contains(t, body, `method="post"`)
	require.Contains(t, body, `action="/pile/act"`)
	require.NotContains(t, body, "onclick=",
		"behaviour lives in pile.js; a page that needs inline handlers is a page that fails without them")
	// Every act travels as a hidden field in a form rather than on the button,
	// so a press is a form submission whatever the script is doing — and there
	// are as many acts as there are forms that carry one.
	require.Equal(t,
		strings.Count(body, `<input type="hidden" name="act"`),
		strings.Count(body, `action="/pile/act"`),
		"an act that is not in a form is an act only a script can make")
}

// Three went with the deck's card and its disclosures. The rule they served —
// an action is a form submission, not a script hook — is still pinned by
// TestEveryActionIsAFormSubmissionNotAScriptHook, which now walks the
// conversation.
