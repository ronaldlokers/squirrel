package web

import (
	"strings"
	"testing"
)

func theBarIn(t *testing.T, body string) string {
	t.Helper()
	at := strings.Index(body, `<header class="ops">`)
	if at < 0 {
		t.Fatal("the board has no bar")
	}
	return body[at : at+strings.Index(body[at:], "</header>")]
}

func insideTheRail(t *testing.T, bar string) string {
	t.Helper()
	at := strings.Index(bar, `<span class="rail">`)
	if at < 0 {
		t.Fatal("the bar has no cluster")
	}
	rest, depth := bar[at:], 0
	for i := 0; i < len(rest); i++ {
		switch {
		case strings.HasPrefix(rest[i:], "<span"):
			depth++
		case strings.HasPrefix(rest[i:], "</span>"):
			depth--
			if depth == 0 {
				return rest[:i]
			}
		}
	}
	t.Fatal("the cluster never closes")
	return ""
}

func TestEverythingYouOperateIsInOneCluster(t *testing.T) {
	bar := theBarIn(t, mounted(t, aBoardStore()).call(t, "GET", "/", nil).Body.String())

	rail := insideTheRail(t, bar)
	for _, want := range []string{`class="find`, `class="chip buddy"`, `class="chip bell`, `class="chip face"`} {
		if !strings.Contains(rail, want) {
			t.Fatalf("%s is outside the cluster it belongs to", want)
		}
	}
}

func TestTheMarkAndTheWordmarkLeadTheBar(t *testing.T) {
	bar := theBarIn(t, mounted(t, aBoardStore()).call(t, "GET", "/", nil).Body.String())

	brand := strings.Index(bar, `class="brand"`)
	rail := strings.Index(bar, `class="rail"`)
	if brand < 0 || rail < 0 || brand > rail {
		t.Fatalf("the identity does not lead the bar: brand at %d, cluster at %d", brand, rail)
	}
}

func TestTheClockIsNotOneOfTheControls(t *testing.T) {
	bar := theBarIn(t, mounted(t, aBoardStore()).call(t, "GET", "/", nil).Body.String())

	if strings.Contains(insideTheRail(t, bar), `class="clock"`) {
		t.Fatal("the clock is inside the cluster, where it reads as something to press")
	}
	if !strings.Contains(bar, `<span class="clock">`) {
		t.Fatal("the clock left the bar")
	}
}
