package web

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	cssComment      = regexp.MustCompile(`(?s)/\*.*?\*/`)
	propertyUsed    = regexp.MustCompile(`var\(\s*(--[A-Za-z0-9_-]+)`)
	propertyDefined = regexp.MustCompile(`(--[A-Za-z0-9_-]+)\s*:`)
)

func TestEveryCustomPropertyIsDefinedSomewhere(t *testing.T) {
	sheets, err := filepath.Glob(filepath.Join("static", "*.css"))
	require.NoError(t, err)
	require.NotEmpty(t, sheets)
	pages, err := filepath.Glob(filepath.Join("templates", "*.html"))
	require.NoError(t, err)

	defined := map[string]bool{}
	used := map[string][]string{}

	for _, page := range pages {
		body, err := os.ReadFile(page)
		require.NoError(t, err)
		for _, m := range propertyDefined.FindAllStringSubmatch(string(body), -1) {
			defined[m[1]] = true
		}
	}
	for _, sheet := range sheets {
		body, err := os.ReadFile(sheet)
		require.NoError(t, err)
		bare := cssComment.ReplaceAllString(string(body), " ")
		for _, m := range propertyDefined.FindAllStringSubmatch(bare, -1) {
			defined[m[1]] = true
		}
		for _, m := range propertyUsed.FindAllStringSubmatch(bare, -1) {
			used[m[1]] = append(used[m[1]], filepath.Base(sheet))
		}
	}

	var missing []string
	for name, where := range used {
		if !defined[name] {
			missing = append(missing, name+" ("+strings.Join(where, ", ")+")")
		}
	}
	sort.Strings(missing)

	require.Empty(t, missing, "a property nothing defines falls back to the inherited value, "+
		"which is how what Squirrel knows about you came to be drawn in cream on cream")
}
