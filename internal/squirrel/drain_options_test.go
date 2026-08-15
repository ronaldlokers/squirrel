package squirrel_test

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Run's backoff starts at Interval and only ever doubles it, so a zero
// Interval never grows past zero: every tick fires immediately, spinning as
// fast as the CPU allows rather than waiting between passes. This never
// touches Postgres — it points the spool at a directory that has been
// removed, so every pass defers via a List error and OnError fires once per
// pass, giving an observable count of how many times Run actually looped.
// MaxBackoff is set low and explicitly (unaffected by this fix) so that a
// correctly-defaulted Interval still only produces a handful of passes in
// the test's short window; an undefaulted zero Interval produces thousands.
func TestNewDrainDefaultsAZeroInterval(t *testing.T) {
	dir := t.TempDir()
	sp, err := squirrel.OpenSpool(dir)
	require.NoError(t, err)
	require.NoError(t, os.RemoveAll(dir))

	var passes int32
	drain := squirrel.NewDrain(squirrel.DrainOptions{
		Spool:      sp,
		MaxBackoff: 50 * time.Millisecond,
		OnError:    func(error) { atomic.AddInt32(&passes, 1) },
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	drain.Run(ctx)

	require.Less(t, int(atomic.LoadInt32(&passes)), 20,
		"a defaulted Interval should keep passes rare; a zero Interval spins as fast as the CPU allows")
}
