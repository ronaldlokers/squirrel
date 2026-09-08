//go:build browser

// What the screens look like, in a form a diff can read.
//
// Both critique passes found visual drift by eye — a global rule redefined, four
// screens quietly repainted, nobody the wiser. The browser suite covers
// behaviour and has never covered what any of it looks like.
//
// Not a screenshot diff: font rasterisation differs between this machine and the
// runner, so a committed PNG fails on the first CI run for a reason unrelated to
// the change, and a check that cries wolf teaches you to re-run the job.
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
	// leaving and means nothing else, and the way it goes wrong is a rule that
	// never mentions it: a descendant selector outranking the one that set it,
	// so the same link renders underlined on one screen and not on another.
	// Nothing else recorded here can see that.
	"text-decoration-line",
	// How this system says pressed: down onto its own shadow, never a
	// different colour. Added 28 August 2026 with the rail, whose current room
	// is drawn exactly that way — without these two, `.rail .room.in` recorded
	// byte-identically to `.rail .room` and the one mark saying which room you
	// are in was pinned by nothing.
	//
	// It is also the sticker offset, which is the depth rule of the whole
	// system, and this is the first thing that has ever watched it.
	"transform", "box-shadow",
}

// One selector per thing that has a shape of its own. Where a screen has many
// of something, the first is enough: what this catches is a rule moving, and a
// rule moves all of them at once.
var appearanceScreens = map[string][]string{
	// The board, which is the front door now. Its furniture and its one
	// content object: a strip is the only shape here that carries words, so
	// recording it and its parts records most of the world.
	"/": {
		".ops", ".ops .wordmark", ".ops .clock .t", ".ops .clock .d", ".ops .chip",
		".ops .rail", ".ops .rail .find", ".ops .chip.face", ".ops .chip.mood",
		".baysign", ".baysign .n", ".channel",
		".strip", ".strip .holder", ".strip .words", ".strip .what", ".strip .mark",
		".strip .why", ".strip.wants", ".rack .seam",
		".strip.blank", ".strip.blank .words", ".stamp", ".stamp .k",
		".blankstrip .inline", ".blankstrip .count", ".newchore",
		".dial", ".dial .ring", ".dial .today", ".dial .checkin", ".dial .checkin .face",
		".dial .record", ".doors", ".door", ".door .through",
		".coming", ".comingsign", ".attime", ".atlabel", ".leaveby",
		".pulled", ".pulled .why b", ".pulled .said",
		".ticking .left", ".tray", ".tray .strip.out .words",
	},

	// What a door opens onto. The notes are the only place the seam and the
	// settled strip are drawn, and they left the board when the doors did — a
	// selector recorded against a screen that no longer draws it pins nothing.
	"/?bay=notes": {".seam", ".strip.resting", ".strip.back", ".blankstrip"},

	"/me": {".youface", ".youhead", ".weekrow"},
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
// appearanceFixture is a store with something in every shape the screens can
// draw. A selector whose element the fixture never renders records only that it
// is missing, which pins nothing.
func appearanceFixture() *fakeStore {
	f := aPile()
	// Two, so both of a rack row's shapes are drawn: one whose rhythm came
	// round and says so, and one whose turn is not today and says instead when
	// you usually do it.
	f.chores = []squirrel.Chore{{
		ID: 1, Name: "bins out", Every: 7 * 24 * time.Hour,
		EveryDays: 7, SinceDays: 7, Active: true, EverDone: true,
	}, {
		ID: 2, Name: "water the plants", Every: 7 * 24 * time.Hour,
		EveryDays: 7, SinceDays: 2, Active: true, EverDone: true,
	}}
	f.usually = map[int64]squirrel.Usually{
		2: {Weekday: time.Sunday, OnADay: true, Part: squirrel.Morning},
	}
	f.checkin = &squirrel.Checkin{Mood: squirrel.MoodCalm, SaidAt: time.Now()}
	// Two fixed points, so the diary in the sidebar draws both a time today
	// and a day further out. Neither is inside its leave-by window: the hoist
	// is a state, and a record of one state cannot hold two.
	f.upcoming = []squirrel.Moment{
		{ID: 21, Label: "dentist", Starts: now().Add(3 * time.Hour), Travel: 15 * time.Minute},
		{ID: 22, Label: "the school run", Starts: now().Add(30 * time.Hour)},
	}
	// A timer and a tray, so the board's two bands that only exist when
	// something is happening are recorded rather than silently absent.
	f.timer = &squirrel.Timer{Label: "the kitchen", Started: now(), Ends: now().Add(11 * time.Minute)}
	f.triaged = []squirrel.Item{
		{ID: 91, RawText: "the washing machine one", State: squirrel.ItemDone, Kind: squirrel.ItemNote},
	}
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
	// Plain scrollback, so both speakers' words have a shape to record. The
	// offer stays the live edge and draws the card.
	// Said at a time, so the timestamp under a run of turns is recorded rather
	// than silently absent: a turn with no SaidAt draws no time at all.
	f.turns = []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerYou, Words: "the chores", SaidAt: now().Add(-time.Hour)},
		{ID: 2, Who: squirrel.SpeakerBuddy, Words: "Two come back round.", SaidAt: now().Add(-time.Hour)},
	}
	return f
}

func TestTheScreensLookLikeThemselves(t *testing.T) {
	f := appearanceFixture()

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
