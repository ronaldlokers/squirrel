//go:build browser

// What the screens look like, in a form a diff can read.
//
// Both critique passes over this product found visual drift by eye — a rule
// redefined somewhere global, four screens quietly repainted, nobody the
// wiser until a person went looking. The browser suite covers the hard part,
// which is behaviour: the stamp, the keys, search answering as you type. It
// has never covered what any of it looks like.
//
// The obvious answer is a screenshot diff, and it is the wrong one here. Font
// rasterisation differs between this machine and the runner, so a committed
// PNG fails on the first CI run for a reason that has nothing to do with the
// change — and a check that cries wolf teaches you to re-run the job instead
// of reading it. That lesson is fresh: a genuine flake in this suite failed a
// branch that had not touched it, an hour before this file was written.
//
// So this records the *computed* values instead. They are what the cascade
// actually settled on, they do not depend on how a glyph is rasterised, and
// they diff as text — you can read what changed in the pull request rather
// than squinting at two images.
package web

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The properties worth pinning: the ones a global rule can move without anyone
// meaning to. Not every property — a snapshot of everything is a snapshot
// nobody reads, and it fails on changes that are the point of the commit.
var appearanceProps = []string{
	"font-size", "font-variation-settings", "letter-spacing", "text-transform",
	"color", "background-color", "border-width", "border-radius", "outline-color",
	"padding", "margin", "display", "width", "min-height",
	// The mark that says what pressing something does. Underline means you are
	// leaving and means nothing else, and the rule was broken by a rule that
	// never mentioned it: `.ends a` outranks `.quietpick`, so one link written
	// once rendered as a plain pill on the deck and an underlined pill on nine
	// other screens. Nothing recorded here could see that.
	"text-decoration-line",
}

// One selector per thing that has a shape of its own. Where a screen has many
// of something, the first is enough: what this catches is a rule moving, and a
// rule moves all of them at once.
var appearanceScreens = map[string][]string{
	// The thread. The doors are a rail now and the offer is an ordinary card
	// in it, so what is recorded here is the conversation's own shapes.
	"/": {
		".lid", ".brand img", ".wordmark", ".lidbtn",
		".railwrap", ".rail", ".rdoor", ".rname",
		".thread", ".turn", ".frombuddy .bub", ".fromyou .bub",
		".turncard", ".turnname", ".turnmeta", ".abtn", ".abtn.later", ".abtn.why",
		".pick", ".pick .lead", ".pickrow", ".pick .make", ".wordbox", ".wordbox textarea", ".cut", ".cut .piece",
		".dock", ".slot", ".slot textarea", ".slot .post",
	},
	// The check-in as a question. It is a turn like any other, and the faces
	// are worth their own visit: they are the control the capacity gate
	// depends on.
	"/?ask=1": {".faces", ".face", ".face img", ".face span"},
	// `.ends .quietpick` as well as `.ends`: the underline is on the link, and
	// the paragraph around it cannot report what the link is wearing.
	"/moods": {".deckhead", ".aday", ".aday .when", ".areading", ".ends"},
	// `.empty img` is here and nowhere else because /enough is the one screen
	// that overrides it. The size is the difference between a different drawing
	// and the same one shrunk, and the HTML attribute cannot hold it — the
	// shared rule would win.
	"/enough": {".empty h1", ".empty p", ".empty img", ".leavehere", ".onemore a"},
}

const appearanceFile = "testdata/appearance.json"

// TestTheScreensLookLikeThemselves compares every screen's computed shape with
// what is recorded, and fails with the property that moved.
//
// Regenerate deliberately, never automatically:
//
//	APPEARANCE=rewrite go test -tags=browser -run TestTheScreensLookLike ./internal/web/
//
// and read the diff before committing it. A snapshot that rewrites itself on
// failure is a snapshot that records whatever happened, which is the opposite
// of a fence.
func TestTheScreensLookLikeThemselves(t *testing.T) {
	f := aPile()
	f.chores = []squirrel.Chore{{
		ID: 1, Name: "bins out", Every: 7 * 24 * time.Hour,
		EveryDays: 7, SinceDays: 6, Active: true, EverDone: true,
	}}
	f.checkin = &squirrel.Checkin{Mood: squirrel.MoodCalm, SaidAt: time.Now()}
	f.readings = []squirrel.Checkin{{Mood: squirrel.MoodCalm, SaidAt: time.Now()}}
	f.items = append(f.items,
		note(91, "the bike rack", squirrel.ItemKept),
		task(92, "ring the vet", squirrel.ItemOpen),
	)
	f.aside = []squirrel.HeldItem{{
		ID: 93, Text: "chase the landlord", State: squirrel.ItemWaiting,
		Because: "waiting on him", Kind: squirrel.ItemNote,
	}}
	f.offer = &squirrel.Offer{
		Kind: squirrel.OfferChore, RefID: 1, Text: "bins out", Because: "it is bin day",
	}

	// The clock is frozen, and that is load-bearing rather than tidy: four of
	// the sentences on these screens are chosen from the date, so a snapshot
	// taken on a Tuesday would fail on a Wednesday for a reason that has
	// nothing to do with anybody's change. A record that expires is a record
	// that teaches you to regenerate it without reading it.
	was := now
	now = func() time.Time { return time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { now = was })

	srv := screenWith(t, f, &fakeCoach{reply: "one thing at a time."})
	c := browserAt(t, srv, "/")
	c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 1280, "height": 900, "deviceScaleFactor": 1, "mobile": false,
	})

	got := map[string]map[string]map[string]string{}
	paths := make([]string, 0, len(appearanceScreens))
	for path := range appearanceScreens {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		c.navigate(t, srv.URL+path)
		got[path] = map[string]map[string]string{}
		for _, sel := range appearanceScreens[path] {
			got[path][sel] = shapeOf(t, c, sel)
		}
	}

	fresh, err := json.MarshalIndent(got, "", "  ")
	require.NoError(t, err)
	fresh = append(fresh, '\n')

	if os.Getenv("APPEARANCE") == "rewrite" {
		require.NoError(t, os.MkdirAll(filepath.Dir(appearanceFile), 0o755))
		require.NoError(t, os.WriteFile(appearanceFile, fresh, 0o644))
		t.Log("rewrote " + appearanceFile + " — read the diff before committing it")
		return
	}

	want, err := os.ReadFile(appearanceFile)
	require.NoError(t, err, "no recorded appearance; regenerate with APPEARANCE=rewrite")

	if string(want) == string(fresh) {
		return
	}
	t.Fatalf("the screens changed shape:\n%s\n\nIf that was the point of the change, "+
		"regenerate with APPEARANCE=rewrite and read the diff before committing it.",
		strings.Join(whatMoved(t, want, fresh), "\n"))
}

// shapeOf is one element's recorded properties, or a marker that it is not on
// the screen — an element disappearing is a change worth failing on.
func shapeOf(t *testing.T, c *cdp, sel string) map[string]string {
	t.Helper()
	got := c.eval(t, fmt.Sprintf(`
		const el = document.querySelector(%q);
		if (!el) return {missing: "yes"};
		const cs = getComputedStyle(el);
		const out = {};
		for (const p of %s) out[p] = cs.getPropertyValue(p);
		return out;`, sel, mustJSON(t, appearanceProps)))
	raw, ok := got.(map[string]any)
	require.True(t, ok, "%s answered %#v", sel, got)
	out := map[string]string{}
	for k, v := range raw {
		out[k], _ = v.(string)
	}
	return out
}

// whatMoved is the readable half of the failure: which screen, which element,
// which property, from what to what. A JSON diff of the whole file says "these
// two blobs differ", which is true and useless at three in the morning.
func whatMoved(t *testing.T, want, got []byte) []string {
	t.Helper()
	var a, b map[string]map[string]map[string]string
	require.NoError(t, json.Unmarshal(want, &a))
	require.NoError(t, json.Unmarshal(got, &b))

	var moved []string
	for _, path := range sortedKeys(b) {
		for _, sel := range sortedKeys(b[path]) {
			for _, prop := range sortedKeys(b[path][sel]) {
				was, had := a[path][sel][prop]
				now := b[path][sel][prop]
				if !had {
					moved = append(moved, fmt.Sprintf("  %s  %s  %s is new: %s", path, sel, prop, now))
					continue
				}
				if was != now {
					moved = append(moved, fmt.Sprintf("  %s  %s  %s: %s → %s", path, sel, prop, was, now))
				}
			}
		}
	}
	for _, path := range sortedKeys(a) {
		for _, sel := range sortedKeys(a[path]) {
			if _, still := b[path][sel]; !still {
				moved = append(moved, fmt.Sprintf("  %s  %s is gone", path, sel))
			}
		}
	}
	if len(moved) == 0 {
		moved = append(moved, "  (only formatting)")
	}
	return moved
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}
