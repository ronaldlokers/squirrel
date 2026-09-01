package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// thread.js appends turns to #thread and says out loud what arrived, so the
// region it says it into has to sit outside the container being appended to: a
// live region that is itself replaced announces nothing.
func TestTheLiveRegionSitsOutsideTheConversation(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	body := mounted(t, f).call(t, "GET", "/r/everything", nil).Body.String()

	require.Contains(t, body, `id="threadsay"`)
	require.Contains(t, body, `aria-live="polite"`)

	// Not a descendant of #thread: walk from the container to the region and
	// require the nesting to come back to zero on the way.
	from := strings.Index(body, `id="thread"`)
	to := strings.Index(body, `id="threadsay"`)
	require.Positive(t, from)
	require.Greater(t, to, from, "the region is drawn before the conversation")

	// At least one more close than open, which is #thread's own: anything less
	// and the region is still inside it.
	between := body[from:to]
	require.Greater(t, strings.Count(between, "</div>"), strings.Count(between, "<div"),
		"the live region is inside the conversation, so appending a turn would replace it")
}
