package web

import (
	"strings"
	"testing"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func racksIn(t *testing.T, f *fakeStore, path string) int {
	t.Helper()
	return strings.Count(mounted(t, f).call(t, "GET", path, nil).Body.String(), `data-bay="`)
}

func developing(t *testing.T) {
	t.Helper()
	was := devDir
	devDir = "."
	t.Cleanup(func() { devDir = was })
}

func aBoardOfSevenPlaces() *fakeStore {
	return &fakeStore{items: []squirrel.Item{note(1, "the boiler makes a noise", squirrel.ItemOpen)}}
}

func TestAShippedBinaryDrawsEveryBayWhateverIsAsked(t *testing.T) {
	if got := racksIn(t, aBoardOfSevenPlaces(), "/?only=weekly"); got != 6 {
		t.Fatalf("a shipped board drew %d places for ?only=weekly, and the query is not its business", got)
	}
}

func TestTheDevelopmentBoardDrawsTheOneBayItWasAskedFor(t *testing.T) {
	developing(t)
	f := aBoardOfSevenPlaces()

	body := mounted(t, f).call(t, "GET", "/?only=weekly", nil).Body.String()

	if got := strings.Count(body, `data-bay="`); got != 1 {
		t.Fatalf("drew %d racks, so a picked element is still %d things on screen", got, got)
	}
	if !strings.Contains(body, `data-bay="weekly"`) {
		t.Fatal("it drew a rack, and not the one that was asked for")
	}
	if got := strings.Count(body, `class="blankstrip"`); got != 0 {
		t.Fatalf("drew %d blank strips beside a rack, which has no writer of its own", got)
	}
}

func TestTheOneBayIsTheOneYouAreStandingIn(t *testing.T) {
	developing(t)

	body := mounted(t, aBoardOfSevenPlaces()).call(t, "GET", "/?only=seldom", nil).Body.String()

	if !strings.Contains(body, `class="rack in" data-bay="seldom"`) {
		t.Fatal("the only rack on the page is not lit, so a phone width shows nothing")
	}
}

func TestABayNobodyHasDrawsThemAll(t *testing.T) {
	developing(t)

	if got := racksIn(t, aBoardOfSevenPlaces(), "/?only=nonsense"); got != 6 {
		t.Fatalf("asking for a place that does not exist drew %d of them", got)
	}
}

func TestTheDevelopmentBoardIsStillTheWholeBoardWhenNothingIsAsked(t *testing.T) {
	developing(t)

	if got := racksIn(t, aBoardOfSevenPlaces(), "/"); got != 6 {
		t.Fatalf("development mode drew %d places on its own", got)
	}
}
