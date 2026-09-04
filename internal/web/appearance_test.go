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
		".ops .rail", ".ops .rail .find", ".ops .chip.face",
		".baysign", ".baysign .n", ".channel",
		".strip", ".strip .holder", ".strip .words", ".strip .what", ".strip .mark",
		".strip.blank", ".strip.blank .words", ".stamp", ".stamp .k",
		".blankstrip .inline", ".blankstrip .count", ".seam", ".strip.resting", ".pulled", ".pulled .why b", ".pulled .said",
		".ticking .left", ".tray", ".tray .strip.out .words",
	},

	// The thread, which is the whole app. Only the newest Buddy turn carries
	// controls, so one load can record one interactive shape and no more: the
	// card is the one worth having, and the picker, the word box and the split
	// are covered by the contrast walk, which renders them one at a time.
	//
	// Buddy's words are `.said` and yours are `.bub`, so there is no
	// `.frombuddy .bub` to record.
	"/r/everything": {
		// The mark, the wordmark and the rail's two controls went with the
		// lid's contents on 3 September 2026: his room carries the board's bar,
		// and the chips are what is in it.
		".lid", ".lid .chip", ".lid .find input",
		// The rail, which is furniture on every screen and the largest thing
		// this snapshot could miss.
		//
		// `:not(.in)` and not the bare `.rail .room`: the first room on this
		// page is Buddy's, which is the room you are in, so the bare selector
		// recorded the current shape twice and the resting one never. The
		// difference between them is the whole of how this rail says where you
		// are — a recessed well against a solid one.
		// One link where five rooms were, and the two controls that always sat
		// below the rule.
		// The control that names the room you are in is deliberately NOT here.
		// This snapshot visits one viewport and it is a desktop one, where the
		// control is inside a display:none parent — getComputedStyle still
		// returns its own styles, so it would record a full set of values for
		// something nobody can see and pass whatever happened to the phone.
		// Its markup is held by TestNothingNeedsAScriptToBeReached; its appearance on a
		// phone is held by nothing, and saying so is better than a line that
		// looks like cover.
		".thread", ".turn", ".frombuddy .said", ".fromyou .bub",
		// When it was said and which day it was. Both are quiet text on the
		// field, which is exactly the kind of thing a colour change elsewhere
		// takes with it without anybody noticing.
		".whensaid", ".whenday",
		".turncard", ".turnname", ".turnmeta", ".abtn", ".abtn.later", ".abtn.why",
		".dock", ".slot", ".slot textarea", ".slot .post",
	},
	// The check-in as a question. It is a turn like any other, and the faces
	// are worth their own visit: they are the control the capacity gate
	// depends on.
	"/r/everything?ask=1": {".faces", ".face", ".face img", ".face span"},
	// `.empty img` is here and nowhere else because /enough is the one screen
	// that overrides it. The size is the difference between a different drawing
	// and the same one shrunk, and the HTML attribute cannot hold it — the
	// shared rule would win.
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
	f.chores = []squirrel.Chore{{
		ID: 1, Name: "bins out", Every: 7 * 24 * time.Hour,
		EveryDays: 7, SinceDays: 6, Active: true, EverDone: true,
	}}
	f.checkin = &squirrel.Checkin{Mood: squirrel.MoodCalm, SaidAt: time.Now()}
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
	c := browserAt(t, srv, "/r/everything")
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
