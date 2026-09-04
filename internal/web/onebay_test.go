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

func aBoardOfFourBays() *fakeStore {
	return &fakeStore{items: []squirrel.Item{note(1, "the boiler makes a noise", squirrel.ItemOpen)}}
}

func TestAShippedBinaryDrawsEveryBayWhateverIsAsked(t *testing.T) {
	if got := racksIn(t, aBoardOfFourBays(), "/?only=chores"); got != 4 {
		t.Fatalf("a shipped board drew %d racks for ?only=chores, and the query is not its business", got)
	}
}

func TestTheDevelopmentBoardDrawsTheOneBayItWasAskedFor(t *testing.T) {
	developing(t)
	f := aBoardOfFourBays()

	body := mounted(t, f).call(t, "GET", "/?only=chores", nil).Body.String()

	if got := strings.Count(body, `data-bay="`); got != 1 {
		t.Fatalf("drew %d racks, so a picked element is still %d things on screen", got, got)
	}
	if !strings.Contains(body, `data-bay="chores"`) {
		t.Fatal("it drew a rack, and not the one that was asked for")
	}
	if got := strings.Count(body, `class="blankstrip"`); got != 1 {
		t.Fatalf("drew %d blank strips", got)
	}
}

func TestTheOneBayIsTheOneYouAreStandingIn(t *testing.T) {
	developing(t)

	body := mounted(t, aBoardOfFourBays()).call(t, "GET", "/?only=agenda", nil).Body.String()

	if !strings.Contains(body, `class="rack in" data-bay="agenda"`) {
		t.Fatal("the only rack on the page is not lit, so a phone width shows nothing")
	}
}

func TestABayNobodyHasDrawsThemAll(t *testing.T) {
	developing(t)

	if got := racksIn(t, aBoardOfFourBays(), "/?only=nonsense"); got != 4 {
		t.Fatalf("asking for a bay that does not exist drew %d racks", got)
	}
}

func TestTheDevelopmentBoardIsStillTheWholeBoardWhenNothingIsAsked(t *testing.T) {
	developing(t)

	if got := racksIn(t, aBoardOfFourBays(), "/"); got != 4 {
		t.Fatalf("development mode drew %d racks on its own", got)
	}
}
