package squirrel_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A comment naming a test names a test that exists.
//
// This codebase argues with itself in its comments — "see X for why", "X above
// proves nothing about Y" — and those pointers are load-bearing: they are how a
// reader learns which of two similar tests is the one that can fail. A pointer
// to a name nothing answers to sends them looking for a file that is not there.
//
// It has already drifted once: a test renamed to say what it actually pinned
// left its own doc pointing at the old name, in the same commit.
//
// Every package, from the repository root, because the pointers cross package
// boundaries as often as not.
func TestEveryTestNamedInACommentExists(t *testing.T) {
	root := repoRoot(t)
	declared := map[string]bool{}
	var comments []citation

	require.NoError(t, filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "dist" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Every build tag: the browser and integration files are where most of
		// the cross-references live, and go/parser reads them regardless.
		set := token.NewFileSet()
		file, err := parser.ParseFile(set, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && strings.HasPrefix(fn.Name.Name, "Test") {
				declared[fn.Name.Name] = true
			}
		}
		for _, group := range file.Comments {
			for _, c := range group.List {
				for _, name := range names.FindAllString(c.Text, -1) {
					comments = append(comments, citation{
						where: set.Position(c.Pos()).String(),
						name:  name,
					})
				}
			}
		}
		return nil
	}))

	require.NotEmpty(t, declared, "no tests were found at all, so this measured nothing")
	require.NotEmpty(t, comments, "no comment names a test, so this measured nothing")

	for _, c := range comments {
		// A `-run` argument is a prefix rather than a whole name: the regenerate
		// instruction in appearance_test.go is written that way on purpose.
		if strings.Contains(c.where, "-run") {
			continue
		}
		if declared[c.name] {
			continue
		}
		matched := false
		for have := range declared {
			if strings.HasPrefix(have, c.name) {
				matched = true
				break
			}
		}
		require.True(t, matched, "%s names %s, and no test is called that", c.where, c.name)
	}
}

type citation struct{ where, name string }

// Long enough not to match the word "Test" on its own, and anchored on the
// capital that follows it.
var names = regexp.MustCompile(`\bTest[A-Z][A-Za-z0-9]{6,}\b`)

// repoRoot walks up from this package to the directory holding go.mod, so the
// walk covers every package rather than only this one.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for range 6 {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("no go.mod above this package")
	return ""
}
