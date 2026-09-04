package web

import (
	"io/fs"
	"strings"
	"testing"
)

func TestNoTemplateCarriesADevelopmentScript(t *testing.T) {
	err := fs.WalkDir(templateFS, "templates", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		body, err := templateFS.ReadFile(p)
		if err != nil {
			return err
		}
		for _, dead := range []string{"impeccable", "localhost:", "127.0.0.1"} {
			if strings.Contains(string(body), dead) {
				t.Errorf("%s carries %q: a script from a development machine would ship with it", p, dead)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
