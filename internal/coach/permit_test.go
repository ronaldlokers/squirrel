package coach

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Rule 10 is a property of the type system now, not of anyone's memory.
//
// It held before, and it held because six methods each carried an identical
// four-line check of the month's ceiling. All six were correct. Nothing
// enforced the seventh, and the roadmap has two more model-touching features
// on it — so the guard was one plausible feature away from being forgotten,
// and forgetting it means either spending past a ceiling that was set on
// purpose or failing to fall back to the answers that shipped first.
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
