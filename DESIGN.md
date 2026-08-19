---
name: Squirrel
description: A shoebox of cards you take one from — the mascot's own sticker language turned into a working tool.
colors:
  purple: "#6c4da9"
  purple-deep: "#58388a"
  purple-dark: "#3b2560"
  violet: "#5e23b1"
  violet-ink: "#56209f"
  orange: "#e66d0d"
  orange-lit: "#ff8a2b"
  outline: "#1c110b"
  tail-cream: "#fed6a7"
  card: "#fdecd4"
  card-deep: "#f6d5ab"
  paper: "#fffbf3"
  paper-deep: "#f7e9d4"
  headphone-brown: "#58413d"
  placeholder-ink: "#8a7361"
  field-lift: "rgba(150, 110, 220, .5)"
  shadow-cast: "rgba(0, 0, 0, .75)"
  shadow-cast-soft: "rgba(0, 0, 0, .7)"
  state-done: "#529414"
  state-done-ink: "#3c6d0e"
  state-kept: "#ffb300"
  state-kept-ink: "#8a5c00"
  state-dropped: "#8a6a55"
  state-dropped-ink: "#6a4f3e"
  state-chore-ink: "#b0530a"
  # Type on the orange selection band, and the only white in the system. The
  # No Neutral Rule governs surfaces; this is ink, on a surface that is
  # already orange.
  selection-ink: "#fefefe"
  # Lifted fills: a state colour raised toward white until dark ink reads on
  # it. See The Lifted Fill Rule. Base and hover for each.
  state-done-lifted: "#71a73e"
  state-done-lifted-hover: "#89b65f"
  state-dropped-lifted: "#9f8574"
  state-dropped-lifted-hover: "#af9a8b"
  state-kept-lifted-hover: "#ffc12e"
typography:
  display:
    fontFamily: "Inter, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "26px"
    fontWeight: 900
    letterSpacing: "-0.028em"
  headline:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "clamp(27px, 3.4vw, 36px)"
    lineHeight: "1.15"
    letterSpacing: "-0.01em"
    fontVariation: "'MONO' 0, 'CASL' 1, 'wght' 800"
  body:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "clamp(21px, 2.2vw, 27px)"
    lineHeight: "1.4"
    letterSpacing: "-0.008em"
    fontVariation: "'MONO' 0, 'CASL' 1, 'wght' 520"
  label:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "13px"
    letterSpacing: "0.08em"
    fontVariation: "'MONO' 0, 'CASL' 0, 'wght' 830"
  title:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "12.5px"
    letterSpacing: "0.09em"
    fontVariation: "'MONO' 0, 'CASL' 0, 'wght' 750"
  voice:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "17px"
    fontVariation: "'MONO' 0, 'CASL' 1, 'wght' 450"
  result:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "18px"
    lineHeight: "1.4"
    fontVariation: "'MONO' 0, 'CASL' 1, 'wght' 480"
  meta:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "11.5px"
    letterSpacing: "0.1em"
    fontVariation: "'MONO' 0, 'CASL' 0, 'wght' 750"
  stamp:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "15px"
    letterSpacing: "0.1em"
    fontVariation: "'MONO' 0, 'CASL' 0, 'wght' 850"
  keycap:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "11px"
    fontVariation: "'MONO' 0, 'CASL' 0, 'wght' 800"
  field:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "14.5px"
    fontVariation: "'MONO' 0, 'CASL' 0, 'wght' 600"
  # The Step-Up Rule's own sizes, below 620px. The same roles at a second
  # size, not new roles — written here as well as in prose because a reader
  # that only sees this ramp otherwise reads the rule as a violation of it.
  body-phone:
    fontSize: "23px"
  headline-phone:
    fontSize: "31px"
  voice-phone:
    fontSize: "18px"
  result-phone:
    fontSize: "18.5px"
  meta-phone:
    fontSize: "12px"
  field-phone:
    fontSize: "16px"
rounded:
  chip: "999px"
  card: "14px"
  card-inner: "11px"
  stamp: "10px"
  tab: "8px"
  key: "5px"
  dot: "3px"
spacing:
  stack-offset: "8px"
  outline: "3px"
  card-inset: "32px"
  tray: "22px"
components:
  button-action:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.outline}"
    typography: "{typography.label}"
    rounded: "{rounded.chip}"
    padding: "12px 18px"
  button-action-hover:
    backgroundColor: "{colors.paper-deep}"
  button-make:
    backgroundColor: "{colors.orange}"
    textColor: "{colors.paper}"
    typography: "{typography.label}"
    rounded: "{rounded.chip}"
    padding: "12px 20px"
  button-make-hover:
    backgroundColor: "{colors.orange-lit}"
  card-note:
    backgroundColor: "{colors.card}"
    textColor: "{colors.outline}"
    rounded: "{rounded.card}"
  card-behind:
    backgroundColor: "{colors.card-deep}"
    rounded: "{rounded.card}"
  titlebar:
    backgroundColor: "{colors.purple}"
    textColor: "{colors.card}"
    typography: "{typography.title}"
    padding: "11px 16px"
  button-did:
    backgroundColor: "{colors.state-done-lifted}"
    textColor: "{colors.outline}"
    typography: "{typography.label}"
    rounded: "{rounded.chip}"
    padding: "11px 16px"
  button-did-hover:
    backgroundColor: "{colors.state-done-lifted-hover}"
  button-often:
    backgroundColor: "{colors.state-kept}"
    textColor: "{colors.outline}"
    typography: "{typography.label}"
    rounded: "{rounded.chip}"
    padding: "11px 16px"
  button-often-hover:
    backgroundColor: "{colors.state-kept-lifted-hover}"
  button-stop:
    backgroundColor: "{colors.state-dropped-lifted}"
    textColor: "{colors.outline}"
    typography: "{typography.label}"
    rounded: "{rounded.chip}"
    padding: "11px 16px"
  button-stop-hover:
    backgroundColor: "{colors.state-dropped-lifted-hover}"
  input-search:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.outline}"
    rounded: "{rounded.chip}"
    padding: "9px 14px"
---

# Design System: Squirrel

## Overview

**Creative North Star: "The Shoebox"**

A physical box of index cards kept somewhere you can reach it. You lift the lid,
take the card off the top, decide what it is, and it leaves. The box never tells
you how many are left, because a box can't count and neither can this product.
Every structural decision follows from that image: the header is the lid, the
notes are cards, the stack beneath the top card is the only thing that says
there is more, and the whole surface has exactly one thing on it at a time.

The material is the mascot's own drawing language, taken directly off
`assets/logo.png` rather than chosen: a 3px near-black outline around flat
saturated fills, soft rounded geometry, no gradients on anything you can press.
Cards are cream stock, not white — a white surface is not Squirrel. Purple is
the room, orange is reserved for the single action that makes something instead
of disposing of it.

The system holds two tempers at once and must not average them. The content is
the owner's own words and is set warm, in a casual brush-drawn axis, close to
handwriting. Every machine part — buttons, keycaps, dates, state names — is set
in the linear axis of the same family: tight, uppercase, tracked out, precise.
One family, two voices, no compromise between them.

**Key Characteristics:**

- One card at a time; the stack is texture, never a number
- 3px `#1c110b` outline on every object, without exception
- Flat saturated fills; gradients only on the field, never on an object
- Cream card stock (`#fdecd4`), never white
- Orange means *makes something*; the three disposal actions never take it
- Four states, four colours, taken from the mascot's notebook tabs; orange is
  an action, never a state
- Keyboard letters are actions; movement never takes a letter

## Colors

The palette is the mark's, not a preference: purple and orange are the product's
own colours, and any screen that reads as neutral has failed to be Squirrel.

### Primary

- **Cap Purple** (`#6c4da9`): the card's title bar, and the pressed state of the
  make-a-chore button. The mid purple of the mascot's cap.
- **Field Purple** (`#58388a`): the base colour of the room the cards sit in.
- **Lid Purple** (`#3b2560`): the header bar and its brim, and the scrollbar
  track. The deepest purple in the system; nothing goes darker except the
  outline.

### Secondary

- **Acorn Orange** (`#e66d0d`): reserved. It marks the one action per note that
  creates something rather than ending it — *make a chore* — plus the search
  match underline, the caret, the selection highlight and the scrollbar thumb.
- **Lit Orange** (`#ff8a2b`): the hover of anything already orange, and the
  focus ring on purple.

### Tertiary

The four states, lifted from the coloured page tabs of the notebook the mascot
holds. Each has a fill for the tab and a darker sibling for type, because the
fills are picked to be recognised at a glance and type has to be read.

- **Notebook Violet** (`#5e23b1`, ink `#56209f`): `open`. Still in the pile.
- **Done Green** (`#529414`, ink `#3c6d0e`): it was a task and it happened.
- **Kept Amber** (`#ffb300`, ink `#8a5c00`): not a task. Reference.
- **Dropped Brown** (`#8a6a55`, ink `#6a4f3e`): it stopped mattering. Muted on
  purpose — dropping is not a failure and must not be coloured like one.

**Acorn Orange with ink `#b0530a` is not a fifth state.** A note promoted to a
chore is recorded as `done` — the spec says so and `items.state` has no value
to hold anything else. The orange CHORE stamp is the moment of promotion, shown
once on the card as it leaves. Nothing in a search result may ever wear it.

### Neutral

- **Outline** (`#1c110b`): every stroke in the system, at 3px. Also all body
  type on cream.
- **Card Stock** (`#fdecd4`): the tail's cream, lightened. The surface notes are
  written on.
- **Stock Shade** (`#f6d5ab`): the cards underneath the top one.
- **Paper** (`#fffbf3`): everything that sits *on* the stock — buttons, chips,
  the stamp, the search field — and all type on purple.
- **Tail Cream** (`#fed6a7`): the mascot's tail. Secondary type on purple, at
  80–85% opacity.
- **Headphone Brown** (`#58413d`): meta type on cream. Never grey.

### Named Rules

**The Orange Reserve Rule.** Orange marks creation, never disposal. *Done*,
*keep* and *drop* are paper; *make a chore* is orange. If a second orange
element appears on a screen, one of them is wrong.

**A chore at rest is not orange.** The orange belongs to the moment a note
*becomes* a chore, shown once, on the card where it happened. A chore living
its life afterwards — on the chores screen, in a list, being adjusted or
stopped — is cream stock like everything else the owner owns. A column of
orange tabs would put a dozen creations on one screen and cost the one that
means something.

**The No Neutral Rule.** No surface in this system is white, grey or
near-grey. (`#fefefe` exists for exactly one thing: type inside the orange
selection band. The rule is about surfaces, and that surface is orange.) Cards are cream, the field is purple, secondary type on purple is
tinted from the tail cream and secondary type on cream is tinted from the
headphones. A grey would be the first sign the design has drifted off the mark.

**The Fill-and-Ink Rule.** Every state colour ships as a pair. The fill goes on
shapes — tabs, dots, pressed buttons. The ink goes on type. Never set type in a
fill value; three of the four fail contrast at label size.

**The Lifted Fill Rule.** A control that acts on something wears that thing's
colour at rest, not only under the thumb — but the state fills are chosen to be
seen on a page edge rather than read through, so the fill is *lifted* toward
white until the card's own dark ink reads on it. 18% for the resting state, 32%
for hover, and pressing drops back to the base colour, so the press still reads
as a change:

| Control | Rest | Hover | Press |
| --- | --- | --- | --- |
| *did it* | `#71a73e` | `#89b65f` | `#529414` |
| *how often* | `#ffb300` | `#ffc12e` | `#8a5c00` |
| *stop asking* | `#9f8574` | `#af9a8b` | `#8a6a55` |

Type on a lifted fill is always the outline ink, never cream — the lift exists
precisely so the ink can stay dark, which is the Fill-and-Ink Rule holding
rather than bending. Measured, not judged: 6.4:1, 11.4:1 and 5.4:1 against
`#1c110b`. The values arrived after two attempts that passed contrast and made
the screen *darker* than the cards, which is how you can tell brightness is a
requirement and not a preference.

Amber here is the *kept* fill doing a second job on a surface that has no note
states. On a screen that ever shows a note and a chore together, it would be
saying two things at once, and one of them would have to move.

## Typography

**Display Font:** Inter Black (900), for the wordmark only
**Body Font:** Recursive, self-hosted variable, `MONO` and `CASL` axes
**Label Font:** Recursive, same family, `CASL` 0

**Character:** One variable family carries both halves of the brief instead of
splitting the difference. Recursive's casual axis is brush-drawn and sits close
to handwriting; its linear axis is even and mechanical. The owner's words get
the first, the product's machinery gets the second. Inter Black appears exactly
once, on the wordmark, where it plays the printed label on the side of the box
against the drawn mascot beside it.

### Hierarchy

- **Wordmark** (Inter 900, 26px, `-0.028em`, 2px outline stroke via
  `paint-order: stroke fill`): the product name in the lid. Nowhere else.
- **Headline** (`CASL` 1, `wght` 800, `clamp(27px, 3.4vw, 36px)`): the empty
  state, and any moment where the product speaks in its own voice.
- **Note** (`CASL` 1, `wght` 520, `clamp(21px, 2.2vw, 27px)`, `1.4`, max 34ch):
  the owner's captured text. The only casual-axis element that is his words
  rather than the product's.
- **Label** (`CASL` 0, `wght` 830, 13px, `0.08em`, uppercase): every button.
- **Title** (`CASL` 0, `wght` 750, 12.5px, `0.09em`, uppercase): the card's
  title bar, result meta lines, state names.
- **Keycap** (`CASL` 0, `wght` 800, 11px, 2px `currentColor` border, 5px
  radius, 72% opacity): the letter that performs each action, inside its
  button. One size wherever it appears — a key on a chip is the same key.
- **Stamp** (`CASL` 0, `wght` 850, 15px, `0.1em`, uppercase): the state word on
  an actioned card.
- **Result** (`CASL` 1, `wght` 480, 18px, `1.4`, max 46ch): a note's text in
  search results. Smaller than the deck's note because it is being scanned
  rather than decided on.
- **Meta** (`CASL` 0, `wght` 750, 11.5px, `0.1em`, uppercase): the date and
  state line above a result.
- **Voice** (`CASL` 1, `wght` 450, 17px): the product speaking in full
  sentences — the empty state's second line, an empty search.
- **Field** (`CASL` 0, `wght` 600, 14.5px): the search input's own text.

### Named Rules

**The Two Voices Rule.** `CASL` 1 is his words. `CASL` 0 is the product's.
Nothing is set in the casual axis unless the owner typed it *or the product is
speaking in full sentences* — the Headline and Voice roles, which are the two
places Squirrel talks rather than labels. No control is ever set in it: a
button is machinery, and machinery does not have a voice.

**The Soft Elapsed Rule.** A chore reports when it was last done in words and
never in numbers: *today*, *yesterday*, *this week*, *last week*, *this month*,
*a while back*. The buckets stop there — there is no bucket for a long time,
because that sentence is about the person rather than the chore. A chore that
has never been done says nothing at all about it; what has not happened is not
reported. An exact day count is a deadline wearing a different hat, and it
grows while nobody is looking, which is the shape this product exists without.

**The No Mono Rule.** `MONO` is pinned to 0 on both voices. Recursive ships a
monospace axis and this is a warm tool, not a terminal.

## Layout

One column, centred, `min(720px, 100%)`. No navigation, no sidebar, no second
region — the shoebox has one opening.

The lid is a full-width flex bar: mark and wordmark left, search field right,
`11px 22px 13px` padding. Below it hangs the brim, a full-width SVG whose
outline stroke is held at 3px by `vector-effect="non-scaling-stroke"` regardless
of viewport width.

The card sits vertically centred in the remaining space with its note block
centred inside it, so a two-word note and a forty-word note both sit on the same
optical line. Minimum card height 296px desktop, 284px mobile.

Search replaces the deck in place — same column, same width — and switches the
main region from centred to top-aligned. Results scroll; they never paginate,
because a page count is a total in disguise.

**Breakpoint: 620px.** Below it the three disposal actions become an equal
three-column grid with *make a chore* full-width beneath them, and the interval
chips become a two-column grid with the lead and *never mind* spanning both.
The mark grows to 72px. Touch targets are 48px in the action row, 44px
everywhere else.

**The Step-Up Rule.** A phone is read at arm's length in worse light, so roles
step *up* below the breakpoint rather than down: the note goes to a flat 23px
rather than the clamp's 21px floor, the headline to 31px, Voice to 18px, Result
to 18.5px, Meta to 12px, and the search field to 16px — the last of those
because anything smaller makes iOS zoom the page on focus. These are the same
roles at a second size, not new roles.

**Keycaps and key hints do not survive `pointer: coarse`.** A key legend on a
thumb-only surface spends attention on something the reader cannot press. The
hint's last clause stays, because *stop whenever you like* is the product's
line rather than the keyboard's.

## Elevation & Depth

Sticker depth. Every raised object carries two shadows at once: a solid offset
in the outline colour, which is the sticker's own thickness, and a real cast
shadow with offset and blur, which puts it in the room. Neither alone is the
system — the offset without the cast shadow reads as flat neobrutalism, and the
cast shadow without the offset reads as any other card UI.

Objects are flat-filled. The one gradient in the system is the field, and the
field is not an object.

### Shadow Vocabulary

- **Card** (`box-shadow: 0 6px 0 0 #1c110b, 0 26px 40px -18px rgba(0,0,0,.75)`):
  the note card in the deck.
- **Result card** (`box-shadow: 0 5px 0 0 #1c110b, 0 16px 26px -16px rgba(0,0,0,.7)`):
  smaller, so a shallower offset.
- **Control** (`box-shadow: 0 4px 0 0 #1c110b`): buttons, chips, the stamp.
  Offset only — a control is a sticker on the card, not floating above it.
- **Field texture** (`radial-gradient(120% 90% at 15% -10%, rgba(150,110,220,.5), transparent 62%)`
  over `#58388a`, plus a 22px dot grid in `rgba(254,214,167,.07)`): the room's
  light, fixed to the viewport.

### Named Rules

**The Press Rule.** Pressing a control moves it down by exactly its offset and
removes the offset, so it lands flush with the surface. `transform:
translateY(4px)` against `0 4px 0 0`. The numbers must always match; a press
that doesn't land is the most obvious tell in this world.

**The Field Exception.** The no-gradient rule governs objects. The field carries
a radial lift and a dot grid because a full-screen flat purple reads as a dead
swatch, and because habituation is a documented risk for this user. Nothing that
can be pressed may take a gradient.

## Shapes

Soft rounded geometry, matching the mascot's own construction. Cards take a 14px
radius; anything you press is a full pill (`999px`); the stamp takes 10px
because it is stuck on rather than pressed; keycaps take 5px.

Every shape carries the 3px `#1c110b` outline. There are no borderless objects
and no hairlines — a 1px rule anywhere is a defect.

The deck's stack is three cards translated by multiples of an 8px offset and
rotated between 0.7° and 1.8°, randomised on every card change. Nothing in this
world sits perfectly square, because nothing handled ever does.

Icons are authored SVG in the same language: the acorn from the cap badge, the
search glyph at 2.6px stroke. No icon fonts, no emoji standing in for an icon.

## Components

### Buttons

- **Shape:** full pill (`999px`), 3px outline, `0 4px 0 0` offset.
- **Action** (*done*, *keep*, *drop*): paper fill, outline type, 12px/18px
  padding, 44px minimum height. Each carries its keycap.
- **Make** (*make a chore*): orange fill, paper type, pushed to the far end of
  the row with `margin-left: auto`. The only orange control on the card.
- **Hover:** action buttons deepen to `#f7e9d4`; make brightens to `#ff8a2b`.
- **Press:** translate down 4px, offset removed, fill swaps to the button's own
  state colour (`done` green, `keep` amber, `drop` brown, `make` purple).
- **Focus:** 3px `#ff8a2b` ring at 3px offset on purple; `#5e23b1` on cream.

- **Chore controls** (*did it*, *how often*, *stop asking*): lifted state fills
  with outline type, 11px/16px padding, 44px minimum height, no keycaps — the
  chores screen is read on a phone as often as not. *Stop asking* is pushed to
  the far end with `margin-left: auto` the way *make a chore* is on the deck:
  the action that ends something is separated from the two that do not. On a
  phone the two routine controls sit side by side and *stop asking* takes the
  full width beneath them.

### Chips

Interval chips in the chore picker share the button shape at 13.5px/11px
padding. The escape chip (*never mind*) drops its offset and takes a dashed
border — it is the only dashed object in the system, and it means "this does
nothing".

### The Lid Link

A quiet cross-link in the lid — *the pile*, *chores* — at 13px, `wght` 650,
`.8` opacity, underlined on hover or focus, 44px minimum height. It is a place
to go rather than a thing to do, so it takes no pill, no outline and no offset;
it is the one control-shaped thing in the system that is deliberately not a
button.

### Cards / Containers

- **Corner:** 14px.
- **Background:** `#fdecd4` stock; the three cards behind take `#f6d5ab`.
- **Title bar:** cap purple, paper type, 3px bottom outline, `11px 11px 0 0`
  radius so it sits inside the card's corner, with the acorn badge at its left.
- **Shadow:** see Elevation.
- **Padding:** 30px/32px for the note, 22px/32px for the action tray.

### Inputs / Fields

The search field is a paper pill with the 3px outline and a 3px offset,
containing an authored search glyph and a borderless input with an orange caret.
Focus adds a 3px `#ff8a2b` ring outside the existing offset rather than
replacing it.

### The Stamp

The signature component. When a note is actioned the card does not leave
immediately: a stamp lands on it — the state word in that state's ink, on paper,
in a 3px border of the state's fill, with a matching offset, rotated `-7deg`,
scaling from 1.7 with no rebound. The card then holds still for 1150ms with an
undo button on the card itself before flipping away over 470ms.

The hold is not decoration. Undo has to live on the thing it undoes, and if the
card leaves at once there is nowhere for it to be.

### Page Tabs

Search results carry a small coloured tab protruding from the card's right edge
— 26×34px, the state's fill, 3px outline, rounded on the outer corners only.
Straight off the notebook the mascot is holding. It replaces the coloured left
border that a card UI would reach for by default.

## Do's and Don'ts

### Do:

- **Do** put the 3px `#1c110b` outline on every object. No exceptions, no
  hairlines.
- **Do** pair the block offset with a real cast shadow on cards, and match the
  press translation to the offset exactly.
- **Do** set the owner's captured text in `CASL` 1 and every control in `CASL` 0.
- **Do** use a state's ink for type and its fill for shapes — and its *lifted*
  fill when a control wears that state at rest, with the outline ink on top.
- **Do** say *that* there is more — the stack, a line of copy — and let the
  scroll carry the rest.
- **Do** theme the browser's own surfaces: selection, caret, scrollbar track and
  thumb, focus ring, underline offset.
- **Do** randomise the stack's rotation on every card change. Habituation is a
  documented risk for this user, and a surface identical every time stops being
  seen inside a week.

### Don't:

- **Don't** emit a count, a total, a badge, a percentage or a page number, in
  any form, on any surface. This is the product's single hardest rule and the
  one most likely to be broken by accident.
- **Don't** use white or grey for any surface or any secondary type. Tint from
  the tail cream on purple, from the headphone brown on cream.
- **Don't** put orange on a disposal action. Orange means something was made.
- **Don't** dress a chore at rest in orange either, or in the page-tab grammar
  that says what a note ended up as. A chore has a rhythm, not an outcome.
- **Don't** print how long it has been since a chore was done. Say it in words,
  softly, or say nothing.
- **Don't** put a gradient on anything that can be pressed.
- **Don't** open a modal for the chore interval, or for anything else that needs
  neither interruption nor protected focus. The picker replaces the action row
  in place, on the card.
- **Don't** take a letter key for navigation. Letters are actions; `k` is keep.
  Movement is space and the arrow keys.
- **Don't** let a control leave the surface it belongs to — undo lives on the
  card it undoes, not in a corner of the screen.
- **Don't** celebrate an empty pile. It is a normal ending, not an achievement,
  and a reward here is a counter wearing a different hat.
