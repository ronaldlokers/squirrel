package squirrel

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The boundary is the point of the split, so something has to hold it.
//
// A file move is only worth doing if the two halves stay apart. Nothing stops
// the next tap-handling function being written back into `apply.go` beside the
// command parsers, and if that happens the split has bought a commit message
// and nothing else.
//
// So: everything that resolves a tap lives in `taps.go`, and `taps.go` is only
// that. It is a rule about where code goes rather than about what it does,
// which is exactly the kind that rots without a test.
func TestTapHandlingLivesInOnePlace(t *testing.T) {
	fset := token.NewFileSet()

	taps, err := parser.ParseFile(fset, "taps.go", nil, parser.ParseComments)
	require.NoError(t, err)
	apply, err := parser.ParseFile(fset, "apply.go", nil, parser.ParseComments)
	require.NoError(t, err)

	// What resolving a tap is called, in this package's own words.
	const tapWork = "applyAction applyItemAction isActionPayload isTap"

	inTaps := map[string]bool{}
	for _, d := range taps.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			inTaps[fn.Name.Name] = true
		}
	}
	for _, name := range strings.Fields(tapWork) {
		require.True(t, inTaps[name],
			"%s resolves a tap and is not in taps.go — the split has come apart", name)
	}

	// And nothing in apply.go went back to doing it.
	for _, d := range apply.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		for _, name := range strings.Fields(tapWork) {
			require.NotEqual(t, name, fn.Name.Name,
				"%s is in apply.go, which is the room's half", fn.Name.Name)
		}
		// A function whose name says "action" or "tap" belongs on the other
		// side of the line, whatever else it does.
		lower := strings.ToLower(fn.Name.Name)
		require.False(t, strings.Contains(lower, "tap"),
			"%s handles taps and lives in the room's half", fn.Name.Name)
	}
}

// taps.go stays the size of what it is for. Nothing here parses a sentence,
// which is the one way this file could quietly become apply.go again.
func TestTapsKnowsNothingAboutSentences(t *testing.T) {
	fset := token.NewFileSet()
	taps, err := parser.ParseFile(fset, "taps.go", nil, parser.ParseComments)
	require.NoError(t, err)

	for _, d := range taps.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		require.NotContains(t, strings.ToLower(fn.Name.Name), "reply",
			"%s composes a reply; a tap earns no reply, the boost is the receipt", fn.Name.Name)
	}
}
