//go:build browser

// Every word on every screen, measured against what is behind it.
//
// This exists because the two ways a colour goes wrong here are both invisible
// to a person writing the rule that causes them. The first is a scoped rule
// reaching further than its author meant: `.checkin .lead` was written for a
// question standing on the purple field and it also caught every small label
// inside the offer card, so the head of the one thing home hands you rendered
// tail cream on cream stock at 1.18:1 — present in the markup, gone from the
// screen. `.results .lead` did the same to HOW OFTEN? on a search result. Both
// shipped, both were read past for weeks, and neither is visible in a diff.
//
// The second is a fill and a type colour that were each chosen well and never
// measured together. Paper on Acorn Orange is 3.1:1, which failed on all six
// orange controls at once, on five screens, for the whole life of the screen.
//
// DESIGN.md's own instruction is "measure a colour against the type that will
// sit on it, and change the number until it passes". A person doing that by
// eye is the thing that failed. This does it by arithmetic, on the computed
// styles, on every screen, at both widths.
//
// It is a browser test rather than a stylesheet test on purpose: what a piece
// of text ends up coloured is a cascade result, and the two bugs above were
// both rules that were present, correct, and outvoted.
package web

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// contrastWalker returns every visible run of text whose measured contrast
// against its own background is below what WCAG asks for its size and weight.
//
// It composites: an element with no background of its own inherits its
// parent's, translucent fills are laid over what is under them, and an
// element's own `opacity` is folded into its ink — which is what caught the
// separator dot in a chore's meta line sitting at .45 on cream, at 2.19:1.
//
// The field's own base purple is the floor of the chain rather than white,
// because the body's gradient is not a colour a script can read back. It is
// the darkest the field ever is, so cream on it measures at its most
// favourable here; the lit centre of the radial is the case DESIGN.md already
// measured by hand and holds at .35 for.
const contrastWalker = `(() => {
  const lum = (r,g,b) => { const f=v=>{v/=255;return v<=0.03928?v/12.92:Math.pow((v+0.055)/1.055,2.4)};
    return 0.2126*f(r)+0.7152*f(g)+0.0722*f(b) };
  const parse = c => { const m=c.match(/rgba?\(([^)]+)\)/); if(!m) return null;
    const p=m[1].split(',').map(s=>parseFloat(s)); return {r:p[0],g:p[1],b:p[2],a:p.length>3?p[3]:1} };
  const over = (fg,bg) => ({ r: fg.r*fg.a+bg.r*(1-fg.a), g: fg.g*fg.a+bg.g*(1-fg.a), b: fg.b*fg.a+bg.b*(1-fg.a), a:1 });
  const bgOf = el => {
    let n = el; const chain = [];
    while (n && n.nodeType === 1) {
      const c = parse(getComputedStyle(n).backgroundColor);
      if (c && c.a > 0) chain.push(c);
      if (c && c.a === 1) break;
      n = n.parentElement;
    }
    chain.push({ r: 88, g: 56, b: 138, a: 1 });
    let out = chain[chain.length - 1];
    for (let i = chain.length - 2; i >= 0; i--) out = over(chain[i], out);
    return out;
  };
  const out = [];
  const name = el => el.tagName.toLowerCase() + (el.className ? '.' + el.className.toString().trim().split(/\s+/).join('.') : '');
  const measure = (el, cs, text, where) => {
    let fg = parse(cs.color);
    if (!fg) return;
    const bg = bgOf(el);
    const op = parseFloat(getComputedStyle(el).opacity);
    if (op < 1) fg = { r: fg.r, g: fg.g, b: fg.b, a: fg.a * op };
    const eff = over(fg, bg);
    const l1 = lum(eff.r, eff.g, eff.b), l2 = lum(bg.r, bg.g, bg.b);
    const ratio = (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05);
    const size = parseFloat(cs.fontSize);
    const weight = parseInt(cs.fontWeight) || 400;
    const large = size >= 24 || (size >= 18.66 && weight >= 700);
    const need = large ? 3 : 4.5;
    if (ratio >= need) return;
    out.push({
      where: where,
      text: text.slice(0, 48),
      ratio: Math.round(ratio * 100) / 100,
      need: need,
      size: size,
      color: cs.color,
      bg: 'rgb(' + Math.round(bg.r) + ',' + Math.round(bg.g) + ',' + Math.round(bg.b) + ')'
    });
  };
  const drawn = el => {
    const cs = getComputedStyle(el);
    if (cs.visibility === 'hidden' || cs.display === 'none' || parseFloat(cs.opacity) === 0) return false;
    const rect = el.getBoundingClientRect();
    return rect.width > 0 && rect.height > 0;
  };
  document.querySelectorAll('*').forEach(el => {
    const text = Array.from(el.childNodes).filter(n => n.nodeType === 3)
      .map(n => n.textContent.trim()).join(' ').trim();
    if (!text || !drawn(el)) return;
    measure(el, getComputedStyle(el), text, name(el));
  });
  // What a field says before you have typed in it is text you have to read to
  // know what the field is for, and it lives on a pseudo-element the loop
  // above cannot see. Every one of them was under 4.5:1 and nothing said so.
  document.querySelectorAll('[placeholder]').forEach(el => {
    if (!drawn(el)) return;
    measure(el, getComputedStyle(el, '::placeholder'), el.getAttribute('placeholder'), name(el) + '::placeholder');
  });
  return JSON.stringify(out);
})()`

type unreadable struct {
	Where string  `json:"where"`
	Text  string  `json:"text"`
	Ratio float64 `json:"ratio"`
	Need  float64 `json:"need"`
	Size  float64 `json:"size"`
	Color string  `json:"color"`
	Bg    string  `json:"bg"`
}

// everyScreen is a store with something on every one of them. A screen that
// renders empty measures nothing, so a fixture that forgets a state is a check
// that silently covers less than it says.
func everyScreen() *fakeStore {
	now := time.Now()
	return &fakeStore{
		items: []squirrel.Item{
			note(9, "the boiler makes a noise when the heating comes on in the morning", squirrel.ItemOpen),
			note(8, "meter reading 48213", squirrel.ItemOpen),
			note(7, "ask about the bins", squirrel.ItemOpen),
			note(6, "kaas", squirrel.ItemKept),
			note(5, "call the vet back about the appointment", squirrel.ItemDone),
			task(4, "book the car in for its service", squirrel.ItemOpen),
			task(3, "email the landlord about the leak", squirrel.ItemDone),
			note(2, "old thing that stopped mattering", squirrel.ItemDropped),
		},
		chores: []squirrel.Chore{
			{ID: 1, Name: "water the plants", Every: 7 * 24 * time.Hour, Active: true, SinceDays: 9, EveryDays: 7, EverDone: true},
			{ID: 2, Name: "descale the kettle", Every: 30 * 24 * time.Hour, Active: true, SinceDays: 2, EveryDays: 30, EverDone: true},
			{ID: 3, Name: "change the filter", Every: 90 * 24 * time.Hour, Active: true, SinceDays: 4, EveryDays: 90},
		},
		checkin: &squirrel.Checkin{Mood: squirrel.MoodCalm, SaidAt: now.Add(-time.Hour)},
		// The conversation, carrying every shape a turn can hold. The chores
		// and the tasks stopped being pages on 24 August 2026, so the walk
		// reaches them by rendering the turns that draw them rather than by
		// navigating to a URL that no longer exists.
		turns: []squirrel.Turn{
			{ID: 1, Who: squirrel.SpeakerYou, Words: "the chores"},
			{ID: 2, Who: squirrel.SpeakerBuddy, Words: "Two come back.",
				Shown: []byte(`{"place":"the chores","cards":[{"title":"water the plants","meta":"EVERY WEEK · LAST DONE a while back","acts":[{"label":"DID IT","action":"/chores/act","style":"did","fields":{"id":"1","act":"done"}},{"label":"HOW OFTEN","action":"/chores/often","style":"go","fields":{"id":"1"}},{"label":"STOP ASKING","action":"/chores/act","style":"stop","fields":{"id":"1","act":"retire"}}]}]}`)},
			{ID: 3, Who: squirrel.SpeakerYou, Words: "the tasks"},
			{ID: 4, Who: squirrel.SpeakerBuddy, Words: "One thing you decided.",
				Shown: []byte(`{"place":"the tasks","cards":[{"title":"book the car in for its service","meta":"decided this morning","acts":[{"label":"did it","action":"/tasks/act","style":"did","fields":{"id":"4","act":"done"}},{"label":"not a task","action":"/tasks/act","style":"back","fields":{"id":"4","act":"untask"}}]}],"chips":[{"label":"what you cannot act on","href":"/held"}]}`)},
			{ID: 6, Who: squirrel.SpeakerBuddy, Words: "Is this what you meant?",
				Shown: []byte(`{"cut":{"action":"/pile/split","id":9,"pieces":["ring the vet","put the bins out"],"do":"use these"}}`)},
			{ID: 7, Who: squirrel.SpeakerYou, Words: "the boiler"},
			{ID: 8, Who: squirrel.SpeakerBuddy, Words: "How should it read?",
				Shown: []byte(`{"say":{"action":"/pile/fix","fields":{"id":"9"},"was":"the boiler makes a noise","do":"say it this way"}}`)},
			{ID: 5, Who: squirrel.SpeakerYou, Words: "how often — water the plants"},
			{ID: 6, Who: squirrel.SpeakerBuddy, Words: "How often should it come back?",
				Shown: []byte(`{"pick":{"action":"/chores/act","fields":{"id":"1"},"do":"that's it","rows":[{"lead":"every","name":"count","options":["1","2","3","4","6","8"],"chosen":"1"},{"lead":"of these","name":"unit","options":["days","weeks","months"],"chosen":"weeks"}]}}`)},
		},
		readings: []squirrel.Checkin{
			{Mood: squirrel.MoodCalm, SaidAt: now.Add(-time.Hour)},
			{Mood: squirrel.MoodLow, SaidAt: now.Add(-26 * time.Hour)},
		},
		offer: &squirrel.Offer{
			Kind: squirrel.OfferChore, RefID: 1, Text: "water the plants",
			Because: "it is due and you are about",
		},
		moment: &squirrel.Moment{
			ID: 4, Label: "dentist", Starts: now.Add(3 * time.Hour),
			Travel: 15 * time.Minute, Ready: 10 * time.Minute, Bring: "keys, wallet",
		},
		upcoming: []squirrel.Moment{{
			ID: 4, Label: "dentist", Starts: now.Add(3 * time.Hour),
			Travel: 15 * time.Minute, Ready: 10 * time.Minute,
		}},
		attached: []squirrel.Item{
			{ID: 30, RawText: "the referral letter", ReceivedAt: now.Add(-time.Hour)},
		},
		aside: []squirrel.HeldItem{
			{ID: 20, Text: "chase the vet", State: squirrel.ItemWaiting, Because: "the vet to ring back", Kind: squirrel.ItemTask},
			{ID: 21, Text: "learn to solder", State: squirrel.ItemSomeday, Kind: squirrel.ItemNote},
			{ID: 22, Text: "fit the new tap", State: squirrel.ItemBlocked, Because: "the part to arrive", Kind: squirrel.ItemTask},
		},
	}
}

// Both widths, because the Step-Up Rule changes six roles below 620px and a
// size change moves what counts as large text.
func TestEveryWordCanBeRead(t *testing.T) {
	screens := []string{
		// "/" is several screens now: the rail, the transcript with a chore
		// card, a task card and the interval picker on it, and the dock.
		// "/" is the whole app: the rail, the transcript with a chore card, a
		// task card, a note being triaged and the interval picker on it, and
		// the dock.
		"/",
		"/kept", "/held", "/moods", "/enough", "/buddy",
		"/at/4",
	}

	srv := screen(t, everyScreen())
	c := browserAt(t, srv, "/")

	for _, width := range []int{1280, 390} {
		c.send(t, "Emulation.setDeviceMetricsOverride", map[string]any{
			"width": width, "height": 844, "deviceScaleFactor": 1, "mobile": width < 620,
		})
		for _, path := range screens {
			c.navigate(t, srv.URL+path)

			var found []unreadable
			raw := c.eval(t, "return ("+contrastWalker+")")
			require.NoError(t, json.Unmarshal([]byte(fmt.Sprint(raw)), &found))

			for _, f := range found {
				t.Errorf("%s at %dpx: %s %q is %.2f:1, needs %.1f:1 — %s at %.1fpx on %s",
					path, width, f.Where, f.Text, f.Ratio, f.Need, f.Color, f.Size, f.Bg)
			}
		}
	}
}
