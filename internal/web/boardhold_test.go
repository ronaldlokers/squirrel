//go:build browser

package web

import (
	"testing"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/stretchr/testify/require"
)

func TestBrowserAnAnsweredStripStaysWhileTheUndoCouldStillBeWanted(t *testing.T) {
	f := &fakeStore{
		items: []squirrel.Item{
			{ID: 1, RawText: "boiler service code is 4471", State: squirrel.ItemOpen, Kind: squirrel.ItemNote, ReceivedAt: time.Now()},
		},
	}
	srv := screen(t, f)
	c := browserAt(t, srv, "/board")

	c.until(t, "a strip with its answers", `!!document.querySelector('.strip.answerable form.stamps')`)
	c.eval(t, `const f = document.querySelector('.strip.answerable form.stamps');
		f.requestSubmit(f.querySelector('button[value="keep"]')); return 1`)

	c.until(t, "the strike", `!!document.querySelector('.strip.struck')`)
	struckAt := time.Now()
	c.until(t, "the board again", `!document.querySelector('.strip.struck')`)

	require.GreaterOrEqual(t, time.Since(struckAt), 1100*time.Millisecond,
		"the strip left before the undo had anywhere to be")
}
