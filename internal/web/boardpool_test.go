package web

import (
	"testing"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/stretchr/testify/require"
)

func TestOneBoardRenderNeverAsksForMoreConnectionsThanThePoolHas(t *testing.T) {
	probe := &concurrencyProbe{sleep: 20 * time.Millisecond}
	f := aBoardOfSevenPlaces()
	f.probe = probe

	mounted(t, f).call(t, "GET", "/", nil)

	require.LessOrEqual(t, probe.peak, squirrel.MaxConns,
		"one render wanted %d connections at once from a pool of %d, so part of every "+
			"board load queues inside pgxpool where nothing can see it",
		probe.peak, squirrel.MaxConns)
	require.GreaterOrEqual(t, probe.peak, 8,
		"only %d reads overlapped — bounding the fan-out has serialised the board", probe.peak)
}
