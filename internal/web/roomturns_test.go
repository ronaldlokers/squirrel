package web

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The only places allowed to reach the store directly. Everything else goes
// through keepSaid, which is the one thing that knows what room this is.
//
// keepSaid reading the room from context only works while it is the only
// reader: a handler calling AppendTurn itself would put its turn wherever it
// liked, and nothing would notice, because a turn in the wrong room looks
// exactly like a room that is quiet.
var mayAppendDirectly = map[string]bool{
	// keepSaid is the sanctioned path. It is in this list because the scan
	// cannot tell it apart from the handlers it exists to replace.
	"keepSaid": true,
	// threadHandler appends while building the page it is about to render,
	// and needs the saved turn back in the slice it renders from.
	"threadHandler": true,
}

func TestOnlyKeepSaidPutsTurnsInARoom(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	require.NoError(t, err)
	require.Contains(t, pkgs, "web")

	for _, f := range pkgs["web"].Files {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			ast.Inspect(fn, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "AppendTurn" {
					return true
				}
				require.True(t, mayAppendDirectly[fn.Name.Name],
					"%s calls AppendTurn directly at %s — use keepSaid, or add it "+
						"to mayAppendDirectly and say why",
					fn.Name.Name, fset.Position(sel.Pos()))
				return true
			})
		}
	}
}
