package squirrel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestApplierApplyRecoversFromAPanicInMatch guards the drain crash-loop fix:
// a panic anywhere inside Apply — Match, in the case that actually happened —
// must be reported as an error and not left to propagate through Drain.one,
// out of Drain.Run, and out of the process. The real regression (a
// byte-length mismatch on case-folded runes in ParseEvery) is fixed and
// covered directly in intent_duration_test.go, so Match itself no longer has
// a reachable panic; matchFn is swapped for a stand-in that always panics so
// this test exercises the recover itself, as a safety net against whatever
// the next bug in a derived view turns out to be.
//
// This is an internal (package squirrel, not squirrel_test) test file
// specifically so it can reach the unexported matchFn seam.
func TestApplierApplyRecoversFromAPanicInMatch(t *testing.T) {
	prev := matchFn
	matchFn = func(string) Intent { panic("boom") }
	t.Cleanup(func() { matchFn = prev })

	a := NewApplier(nil, nil, nil)
	personID := int64(1)
	convo := "7"

	require.NotPanics(t, func() {
		err := a.Apply(context.Background(), Item{
			RawText:        "anything",
			ConversationID: &convo,
		}, &personID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "panicked")
	})
}
