# One vocabulary

*21 August 2026. Written after "the app feels super messy, weird placements,
things are not hidden when they have no use, actions don't work" and "on the
stack, there is buttons, but also links. It's a mess."*

The material is fine. That is the first finding and it changes the shape of the
work: 22 of the radii are `999px`, 19 are `var(--r)`, 17 of the shadows are
`0 4px 0 0 var(--outline)`, and 29 of the tap targets are exactly 44px. The
palette, the two type axes and the sticker world are consistent and were
consistently applied.

What is inconsistent is **what things mean**. Three specific places, in the
order they are worth fixing.

---

## 1. Three kinds of control wearing one costume

Pressing something here does one of three things, and the screen does not say
which:

| | what happens | how it looks now |
| --- | --- | --- |
| **a button** | something changes, on the server | filled, bordered, hard shadow |
| **a link** | you go to another screen | underlined text |
| **a disclosure** | something opens, here, and nothing changes | *underlined text* — or a filled pill, or a `+` |

That third row is the whole complaint. On the card, `LATER →` goes somewhere,
`fix the words` opens in place, and `i can't act on this` opens in place — and
all three are underlined text. Meanwhile `HOW OFTEN` is a filled yellow pill
and `a new chore` has a plus sign, so disclosures are not even consistent with
each other. Four appearances for one idea.

**The rule.** Underline means you are leaving. A disclosure carries a chevron
that turns when it opens, and nothing else does. A disclosure that sits in a
row of buttons keeps the row's shape — a chip in a row of chips must look like
its neighbours or the row breaks — but it carries the chevron too, so the mark
is what distinguishes it rather than the weight.

That is one drawn glyph and a rule about underlines, and it makes every screen
readable without changing a single layout.

## 2. Five heading treatments, and eight screens with no heading at all

| treatment | screens |
| --- | --- |
| `<h1 class="deckhead">` | held, moods |
| bare `<h1>` | bottom, empty, enough |
| `<p class="head">` | archive, coach, home, tasks |
| `<p class="chorehead">` | chores |
| `<p class="resultsHead">` | kept, results |

Ten of thirteen templates have no `<h1>`, which is why heading navigation does
not work anywhere in this product. It is one treatment and one element, applied
everywhere: `<h1 class="deckhead">`. The three paragraph classes go.

This is the cheapest change here and the one with the largest effect, because a
screen that opens with the same shaped sentence in the same place is most of
what "consistent" means when you are looking rather than measuring.

## 3. Fourteen class names for five roles

`.btn` `.abtn` `.tbtn` `.rbtn` `.obtn` `.decide` `.post` `.make` `.why` `.chip`
`.undo` `.shut` `.askSplit` `.rewordSave`.

Some of those are genuinely different — `.undo` and `.shut` are one-offs that
earn their names. Most are the same control on a different screen, which is how
`.tbtn` came to be declared twice three hundred lines apart and paint the chores
screen's timer buttons in done-green for a release.

Collapsing them changes nothing a person can see. It is worth doing anyway,
because every one of the visual bugs this week lived in the gap between two
names for one thing — but it is worth doing **last**, and separately, so that
a refactor with no visible outcome is never mixed into a change that has one.

---

## What this is not

**Not a redesign.** The world stays: the shoebox, the two voices, the sticker
shadows, the four state colours, no counts anywhere. Nothing in the deck's
composition moves.

**Not a component library.** Five roles, named once. This binary has no build
step and does not want one.

**Not the screens themselves.** Whether the chores screen should exist now the
picker does, whether the card should ask one question or five, whether the
check-in belongs on home — all open, all separate, none of them this.

---

## Order, and why

1. **One heading, everywhere.** Thirteen templates, one rule. Fixes the
   accessibility gap at the same time. Nothing else depends on it.
2. **One affordance rule.** The chevron, and underline reserved for leaving.
   This is the one a person will actually feel.
3. **One page skeleton.** Heading, content, ends — the same widths and the same
   vertical rhythm on every screen. Small after (1).
4. **One control surface.** The rename. No visible change; guarded by
   before-and-after screenshots of all thirteen screens.

Each of those is a release of its own, to staging, looked at on a phone and a
tablet before it goes further. Written down because the alternative was tried
on 20 August: four screens' worth of changes went to production together, none
of them had been used, and the app came back unusable.

---

## The thing this week actually taught

Three separate defects hid behind the same mistake — a test that asked the DOM
about an attribute while the element sat visible on the screen. A photograph
preview that stayed after it was kept. A dialog that closed and did not go
away, for three releases and two confident wrong diagnoses. A lid checked with
a bounding rect that Chrome reports for subtrees it is not painting.

`[hidden]` is enforced globally now and the tests ask `checkVisibility`. But
the rule worth keeping is the smaller one: **when something is meant to
disappear, assert that it is invisible, not that it is unmarked.**
