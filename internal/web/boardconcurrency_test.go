package web

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestBoardReadsGoConcurrently(t *testing.T) {
	probe := &concurrencyProbe{sleep: 20 * time.Millisecond}
	f := aBoardOfFourBays()
	f.probe = probe

	start := time.Now()
	mounted(t, f).call(t, "GET", "/", nil)
	elapsed := time.Since(start)

	require.GreaterOrEqual(t, probe.peak, 8,
		"only %d of the board's store reads overlapped — they are being made one after another", probe.peak)
	require.Less(t, elapsed, 150*time.Millisecond,
		"the board took %s to draw with a 20ms store — that is close to the serial total, not the concurrent one", elapsed)
}

func TestOnlyOneBaySkipsTheShelves(t *testing.T) {
	developing(t)
	f := &fakeStore{items: []squirrel.Item{note(1, "the boiler makes a noise", squirrel.ItemOpen)}}

	mounted(t, f).call(t, "GET", "/?only=chores", nil)

	require.Equal(t, int32(0), atomic.LoadInt32(&f.shelfReads),
		"the shelves were read for a single-bay draw that cannot show them")
}

func TestAShippedBoardStillReadsTheShelves(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "the boiler makes a noise", squirrel.ItemOpen)}}

	mounted(t, f).call(t, "GET", "/?only=chores", nil)

	require.Greater(t, atomic.LoadInt32(&f.shelfReads), int32(0),
		"a shipped board skipped the shelves outside development, where ?only= has no effect")
}
