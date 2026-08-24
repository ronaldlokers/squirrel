package web

import (
	"regexp"
	"strings"
	"testing"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Every URL the templates write, matched against the routes that exist.
//
// This is here because the move to the root broke one of them and every test
// passed. The chores screen's form posted to /pile/chores/act, which is not a
// route — and the chores tests never saw it, because they post to the handler
// directly rather than to whatever the page told the browser to post to. A test
// that exercises a route by name cannot notice that the page names a different
// one.
//
// So this reads the pages instead: a href is a GET, an action is a POST, and
// both have to land somewhere.
func TestEveryLinkOnEveryPageGoesSomewhere(t *testing.T) {
	m := mounted(t, &fakeStore{
		items: []squirrel.Item{
			note(1, "the boiler makes a noise", squirrel.ItemOpen),
			note(2, "buy milk", squirrel.ItemOpen),
		},
		chores: []squirrel.Chore{{ID: 1, PersonID: 1, Name: "bins out", Active: true}},
	})

	// The deck, its results and its bottom were three pages; they are one
	// conversation now, and the pages left are the ones still reached by name.
	pages := map[string]string{
		"the conversation": "/",
		"the shelf":        "/kept",
		"the set-aside":    "/held",
		"the readings":     "/moods",
	}
	// The empty pile is its own page, and it is the one with the way back on it.
	empty := mounted(t, &fakeStore{})

	href := regexp.MustCompile(`href="(/[^"]*)"`)
	// A form's method decides which table it is asking of, and the search form
	// is a GET to the same path the deck answers.
	form := regexp.MustCompile(`<form[^>]*>`)
	action := regexp.MustCompile(`\baction="(/[^"]*)"`)
	formaction := regexp.MustCompile(`formaction="(/[^"]*)"`)

	for name, url := range pages {
		body := m.call(t, "GET", url, nil).Body.String()
		if name == "bottom" {
			body = empty.call(t, "GET", "/pile?after=1", nil).Body.String()
		}

		for _, found := range href.FindAllStringSubmatch(body, -1) {
			requireRouted(t, m, "GET", pathOf(found[1]), name)
		}
		for _, tag := range form.FindAllString(body, -1) {
			found := action.FindStringSubmatch(tag)
			if found == nil {
				continue
			}
			method := "POST"
			if strings.Contains(tag, `method="get"`) {
				method = "GET"
			}
			requireRouted(t, m, method, pathOf(found[1]), name)
		}
		// A button may override its form's action, which is how the deck asks
		// a different question with the same fields.
		for _, found := range formaction.FindAllStringSubmatch(body, -1) {
			requireRouted(t, m, "POST", pathOf(found[1]), name)
		}
	}
}

// pathOf drops the query. The stamp is not part of a path, and neither is a
// note's id.
func pathOf(target string) string {
	path, _, _ := strings.Cut(target, "?")
	return path
}

// requireRouted answers the question ServeMux would: is there a pattern that
// takes this? Prefix patterns end in a slash; everything else is exact, except
// `/{$}`, which is the root and only the root.
func requireRouted(t *testing.T, m *testMux, method, path, page string) {
	t.Helper()
	for pattern := range m.routes {
		want, registered, _ := strings.Cut(pattern, " ")
		if want != method {
			continue
		}
		switch {
		case registered == "/{$}":
			if path == "/" {
				return
			}
		case strings.HasSuffix(registered, "/"):
			if strings.HasPrefix(path, registered) {
				return
			}
		case registered == path:
			return
		}
	}
	t.Fatalf("the %s page writes %s %s, and nothing answers it", page, method, path)
}
