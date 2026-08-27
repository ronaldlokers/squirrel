package coach

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Rule 10 is a property of the type system now, not of anyone's memory.
//
// It held before, and it held because six methods each carried an identical
// four-line check of the month's ceiling. All six were correct. Nothing
// enforced the seventh — so the guard was one plausible feature away from
// being forgotten, and forgetting it means either spending past a ceiling
// that was set on purpose or failing to fall back to the answers that
// shipped first.
//
// A paid call cannot be made without a Permit, and a Permit cannot be had
// without asking the budget. This test is the belt to that braces: it fails if
// anyone gives Permit a constructor, which is the one way to get one without
// the question having been put.
func TestOnlyTheBudgetCanIssueAPermit(t *testing.T) {
	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	require.NoError(t, err)

	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		require.NoError(t, err)
		f, err := parser.ParseFile(fset, name, src, 0)
		require.NoError(t, err)

		ast.Inspect(f, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Type.Results == nil {
				return true
			}
			for _, out := range fn.Type.Results.List {
				id, ok := out.Type.(*ast.Ident)
				if !ok || id.Name != "Permit" {
					continue
				}
				require.Equal(t, "Ask", fn.Name.Name,
					"%s returns a Permit; only Budget.Ask may, or the ceiling can be "+
						"walked around without being read", fn.Name.Name)
			}
			return true
		})
	}
}

// And the model is only ever reached through something that demands one.
func TestNoPaidCallIsMadeWithoutAPermit(t *testing.T) {
	src, err := os.ReadFile("openai.go")
	require.NoError(t, err)

	require.Contains(t, string(src), "completionWithTools(ctx context.Context, _ Permit,",
		"the one function that talks to the provider no longer requires a permit")
}

// One paid call at a time, and the gate always comes back.
func TestOnlyOneCallHoldsTheGate(t *testing.T) {
	freshGate(t)
	b := Budget{Log: gateLog{}}

	first, err := b.Ask(t.Context(), 1, time.Now(), "the picker chooses")
	require.NoError(t, err)

	// The second caller cannot get in while the first holds it. A short
	// deadline rather than the real wait: what is under test is that it blocks
	// at all, not how patient it is.
	ctx, cancel := context.WithTimeout(t.Context(), 150*time.Millisecond)
	defer cancel()
	_, err = b.Ask(ctx, 1, time.Now(), "the picker chooses")
	require.ErrorIs(t, err, context.DeadlineExceeded,
		"two calls held the gate at once, which is the race this closes")

	first.Release()

	third, err := b.Ask(t.Context(), 1, time.Now(), "the picker chooses")
	require.NoError(t, err, "the gate was never given back")
	third.Release()
}

func TestReleasingTwiceFreesOneTurn(t *testing.T) {
	freshGate(t)
	b := Budget{Log: gateLog{}}

	one, err := b.Ask(t.Context(), 1, time.Now(), "x")
	require.NoError(t, err)
	one.Release()
	one.Release()

	two, err := b.Ask(t.Context(), 1, time.Now(), "x")
	require.NoError(t, err)
	defer two.Release()

	// If the double release had freed a second turn, this would get in too.
	ctx, cancel := context.WithTimeout(t.Context(), 150*time.Millisecond)
	defer cancel()
	_, err = b.Ask(ctx, 1, time.Now(), "x")
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

// Over the ceiling, the gate is given back before the refusal is returned —
// or a spent month would wedge every later call behind a permit nobody holds.
func TestARefusedCallGivesTheGateBack(t *testing.T) {
	freshGate(t)
	b := Budget{Log: gateLog{spent: 1_000_000}, CeilingFor: FlatCeiling(1)}

	_, err := b.Ask(t.Context(), 1, time.Now(), "the picker chooses")
	require.ErrorIs(t, err, ErrUnavailable)

	ctx, cancel := context.WithTimeout(t.Context(), 150*time.Millisecond)
	defer cancel()
	after, err := b.Ask(ctx, 1, time.Now(), "the picker chooses")
	require.ErrorIs(t, err, ErrUnavailable,
		"a month over its ceiling wedged the gate shut")
	after.Release()
}

// freshGate empties the gate before and after a test.
//
// The gate is process-wide on purpose — one pod, one person, one paid call at
// a time — so tests share it, and a test that deliberately never releases
// would otherwise hand its held gate to whatever runs next. That is not a flaw
// in the design being tested; it is the design, and tests of shared state have
// to say so.
func freshGate(t *testing.T) {
	t.Helper()
	drain := func() {
		for {
			select {
			case <-spending:
			default:
				return
			}
		}
	}
	drain()
	t.Cleanup(drain)
}

// A log that answers the one question the gate asks, and records nothing.
type gateLog struct{ spent int64 }

func (gateLog) RecordCoachAnswer(context.Context, int64, Answer) error { return nil }
func (l gateLog) CoachSpentSince(context.Context, int64, time.Time) (int64, error) {
	return l.spent, nil
}

// A permit that is never released does not hang the next call.
//
// This is the whole reason the gate is a channel with a deadline rather than a
// mutex. Every design that closes the race properly puts a second obligation
// on six call sites — reserve here, settle there — and the failure mode of a
// forgotten settle is that every future call waits forever. That is worse than
// the overshoot it prevents, so this one gives up instead: it says so, and
// goes.
func TestAForgottenReleaseDoesNotHang(t *testing.T) {
	freshGate(t)
	was := spendWait
	spendWait = 120 * time.Millisecond
	t.Cleanup(func() { spendWait = was })

	b := Budget{Log: gateLog{}}

	// Taken and deliberately never released.
	_, err := b.Ask(t.Context(), 1, time.Now(), "x")
	require.NoError(t, err)

	start := time.Now()
	after, err := b.Ask(t.Context(), 1, time.Now(), "x")
	require.NoError(t, err, "a forgotten release turned into a refusal")
	require.Greater(t, time.Since(start), 100*time.Millisecond,
		"it did not wait at all, so nothing was gating")
	require.Less(t, time.Since(start), time.Second,
		"a forgotten release hung the next call")

	// And the permit it hands back holds nothing, so releasing it cannot free
	// somebody else's turn.
	after.Release()
	require.False(t, after.held)
}

// The ordinary path is prompt: a released gate is available immediately, not
// after the deadline.
func TestAReleasedGateIsAvailableAtOnce(t *testing.T) {
	freshGate(t)
	was := spendWait
	spendWait = 5 * time.Second
	t.Cleanup(func() { spendWait = was })

	b := Budget{Log: gateLog{}}

	first, err := b.Ask(t.Context(), 1, time.Now(), "x")
	require.NoError(t, err)
	first.Release()

	start := time.Now()
	second, err := b.Ask(t.Context(), 1, time.Now(), "x")
	require.NoError(t, err)
	defer second.Release()

	require.Less(t, time.Since(start), 500*time.Millisecond,
		"the gate was given back and the next call waited for the deadline anyway")
	require.True(t, second.held, "the next call did not actually take the gate")
}
