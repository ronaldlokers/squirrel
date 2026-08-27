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
// So this reads the pages instead: a href and a src are GETs, an action is a
// POST, and all of them have to land somewhere. src as well as href because a
// card's photograph is an <img> — renaming the route it points at leaves every
// card on the screen drawing a broken image, and a walk that only reads links
// would not see it.
func TestEveryLinkOnEveryPageGoesSomewhere(t *testing.T) {
	// With a camera, because the routes that take a note's id in the path are
	// only mounted when there is somewhere to put a photograph — and a card
	// drawing one is the only thing on any page that writes such a URL.
	m := mountedWithCamera(t, &fakeStore{
		items: []squirrel.Item{
			note(1, "the boiler makes a noise", squirrel.ItemOpen),
			note(2, "buy milk", squirrel.ItemOpen),
		},
		chores: []squirrel.Chore{{ID: 1, PersonID: 1, Name: "bins out", Active: true}},
		// A card carrying a photograph, so the two routes that take a note's
		// id in the path are among the URLs this walks. Without one the only
		// links on the page are fixed, and a wildcard route could be written
		// wrong without anything here noticing.
		turns: []squirrel.Turn{{ID: 1, Who: squirrel.SpeakerBuddy, Words: "This one.",
			Shown: []byte(`{"cards":[{"title":"the tax letter","photo":"/photo/9"}]}`)}},
	}, &fakeSpool{}, &fakePhotos{})

	pages := map[string]string{
		"the conversation": "/",
		"the readings":     "/moods",
	}

	href := regexp.MustCompile(`href="(/[^"]*)"`)
	src := regexp.MustCompile(`src="(/[^"]*)"`)
	// A form's method decides which table it is asking of, and the search form
	// is a GET to the same path the deck answers.
	form := regexp.MustCompile(`<form[^>]*>`)
	action := regexp.MustCompile(`\baction="(/[^"]*)"`)
	formaction := regexp.MustCompile(`formaction="(/[^"]*)"`)

	for name, url := range pages {
		body := m.call(t, "GET", url, nil).Body.String()
		for _, at := range []*regexp.Regexp{href, src} {
			for _, found := range at.FindAllStringSubmatch(body, -1) {
				requireRouted(t, m, "GET", pathOf(found[1]), name)
			}
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
// `/{$}`, which is the root and only the root, and `{id}`, which takes one
// segment.
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
		case strings.Contains(registered, "{"):
			if takesSegment(registered, path) {
				return
			}
		case registered == path:
			return
		}
	}
	t.Fatalf("%s writes %s %s, and nothing answers it", page, method, path)
}

// takesSegment is a wildcard pattern against a real path, segment by segment.
// A `{name}` takes exactly one and says nothing about what is in it, which is
// all ServeMux promises and all this needs to know.
func takesSegment(pattern, path string) bool {
	want := strings.Split(strings.Trim(pattern, "/"), "/")
	got := strings.Split(strings.Trim(path, "/"), "/")
	if len(want) != len(got) {
		return false
	}
	for i, segment := range want {
		if strings.HasPrefix(segment, "{") {
			continue
		}
		if segment != got[i] {
			return false
		}
	}
	return true
}
