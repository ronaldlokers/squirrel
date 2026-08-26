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
  # What a field says before you have typed in it. It was `#8a7361`, which
  # measured 4.32:1 on paper and 3.85:1 on card stock — a placeholder is text
  # and both of those fail. Two shades down clears 5:1 on paper and 4.5:1 on
  # stock, and is still plainly quieter than the words it makes room for.
  placeholder-ink: "#7e6857"
  # The room's light. It was `.5` until 21 August 2026; the number is a
  # contrast result, not a taste one. See The Field Exception.
  field-lift: "rgba(150, 110, 220, .35)"
  # The wash under a lid icon on hover, and under a quiet pill on the field.
  cream-wash: "rgba(254, 214, 167, .16)"
  shadow-cast: "rgba(0, 0, 0, .75)"
  shadow-cast-soft: "rgba(0, 0, 0, .7)"
  # Behind Buddy's sheet. The outline colour, not black.
  backdrop: "rgba(28, 17, 11, .55)"
  state-done: "#529414"
  # `#3c6d0e` until 24 August 2026, when it measured 4.44:1 against card-deep
  # — which is the archive's stock, and the one place this ink lands on it.
  state-done-ink: "#37640d"
  state-kept: "#ffb300"
  state-kept-ink: "#8a5c00"
  state-dropped: "#8a6a55"
  state-dropped-ink: "#6a4f3e"
  state-chore-ink: "#b0530a"
  # Retired on 24 August 2026, and with it the last white in the system. It
  # was the ink on the orange selection band, and white on Acorn Orange
  # measures 3.1:1 — a selected sentence you could not read. The band takes
  # the outline like every other orange surface now. See The Orange Ink Rule.
  # The token stays recorded rather than deleted because `#fefefe` is still
  # written in the mark's own palette, and the reason it left the interface is
  # the useful part.
  selection-ink: "#1c110b"
  # Lifted fills: a state colour raised toward white until dark ink reads on
  # it. See The Lifted Fill Rule. Base and hover for each.
  state-done-lifted: "#71a73e"
  state-done-lifted-hover: "#89b65f"
  state-dropped-lifted: "#9f8574"
  state-dropped-lifted-hover: "#af9a8b"
  state-open-lifted: "#9e7bd0"
  state-open-lifted-hover: "#b295da"
typography:
  display:
    fontFamily: "Inter, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "26px"
    fontWeight: 900
    letterSpacing: "-0.028em"
  # Every screen's title, in the wordmark's own face. See The One Title Rule.
  screentitle:
    fontFamily: "Inter, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "21px"
    fontWeight: 900
    letterSpacing: "-0.01em"
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
  # What you type into the slot, and into the reword box. His words, so his
  # axis — the same sentence before and after it is kept.
  slotfield:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "18px"
    lineHeight: "1.45"
    fontVariation: "'MONO' 0, 'CASL' 1, 'wght' 480"
  # What you type into the search field. Also his words.
  field:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "15px"
    lineHeight: "1.3"
    fontVariation: "'MONO' 0, 'CASL' 1, 'wght' 450"
  # A sentence in a conversation. The document's own base size, and the only
  # role that sits at it.
  conversation:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "16px"
    lineHeight: "1.4"
    fontVariation: "'MONO' 0, 'CASL' 1, 'wght' 500"
  # A place in the menu, and a place under a title.
  nav:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "14.5px"
    fontVariation: "'MONO' 0, 'CASL' 0, 'wght' 640"
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
  slotfield-phone:
    fontSize: "18.5px"
  # The one role that steps DOWN on a phone — see The Lid Step-Down Rule.
  display-phone:
    fontSize: "20px"
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
  outline-small: "2px"
  card-inset: "32px"
  tray: "22px"
  frame: "26px"
  ends: "30px"
components:
  button-action:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.outline}"
    typography: "{typography.label}"
    rounded: "{rounded.chip}"
    padding: "12px 18px"
  button-action-hover:
    backgroundColor: "{colors.paper-deep}"
  # The ink on orange is the outline, not paper. See The Orange Ink Rule.
  button-make:
    backgroundColor: "{colors.orange}"
    textColor: "{colors.outline}"
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
  button-stop:
    backgroundColor: "{colors.state-dropped-lifted}"
    textColor: "{colors.outline}"
    typography: "{typography.label}"
    rounded: "{rounded.chip}"
    padding: "11px 16px"
  button-stop-hover:
    backgroundColor: "{colors.state-dropped-lifted-hover}"
  button-back:
    backgroundColor: "{colors.state-open-lifted}"
    textColor: "{colors.outline}"
    typography: "{typography.label}"
    rounded: "{rounded.chip}"
    padding: "10px 14px"
  button-back-hover:
    backgroundColor: "{colors.state-open-lifted-hover}"
  # A disclosure keeps its row's shape and drops its fill. The chevron is what
  # says it is not a button. See The Three Marks Rule.
  disclosure-inrow:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.outline}"
    typography: "{typography.label}"
    rounded: "{rounded.chip}"
    padding: "11px 16px"
  # The three icons in the lid, and the one that closes Buddy.
  lid-icon:
    backgroundColor: "transparent"
    textColor: "{colors.tail-cream}"
    rounded: "{rounded.chip}"
    size: "44px"
  lid-icon-open:
    backgroundColor: "{colors.violet}"
    textColor: "{colors.paper}"
  panel:
    backgroundColor: "{colors.card}"
    textColor: "{colors.outline}"
    rounded: "{rounded.card}"
    padding: "6px"
  panel-item:
    backgroundColor: "transparent"
    textColor: "{colors.outline}"
    typography: "{typography.nav}"
    rounded: "{rounded.chip}"
    padding: "10px 14px"
  panel-item-here:
    backgroundColor: "{colors.violet}"
    textColor: "{colors.paper}"
  # The way into the pile: the one inset object in the system.
  slot:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.outline}"
    typography: "{typography.slotfield}"
    rounded: "{rounded.card}"
    padding: "12px 12px 12px 18px"
  # Same ink as button-make, and for the same measurement.
  button-post:
    backgroundColor: "{colors.orange}"
    textColor: "{colors.outline}"
    typography: "{typography.label}"
    rounded: "{rounded.chip}"
    padding: "11px 18px"
  button-post-hover:
    backgroundColor: "{colors.orange-lit}"
  # The quiet answer: a pill you can plainly press that is plainly not the
  # default. On the field it is outlined in cream; on the card, in brown.
  pill-quiet:
    backgroundColor: "transparent"
    textColor: "{colors.tail-cream}"
    typography: "{typography.title}"
    rounded: "{rounded.chip}"
    padding: "10px 16px"
  pill-quiet-hover:
    backgroundColor: "{colors.cream-wash}"
    textColor: "{colors.paper}"
  # Turning down the offer. Filled with the card's own deeper stock.
  pill-refuse:
    backgroundColor: "{colors.card-deep}"
    textColor: "{colors.headphone-brown}"
    typography: "{typography.label}"
    rounded: "{rounded.chip}"
    padding: "10px 16px"
  view-link:
    backgroundColor: "transparent"
    textColor: "{colors.tail-cream}"
    typography: "{typography.nav}"
    height: "44px"
  door:
    backgroundColor: "{colors.card}"
    textColor: "{colors.outline}"
    rounded: "{rounded.card}"
    padding: "22px 24px"
  door-hover:
    backgroundColor: "{colors.paper}"
  input-search:
    backgroundColor: "{colors.card}"
    textColor: "{colors.outline}"
    typography: "{typography.field}"
    rounded: "{rounded.card}"
    padding: "6px"
  photograph:
    backgroundColor: "{colors.card-deep}"
    rounded: "{rounded.stamp}"
---

# Design System: Squirrel

## Overview

**Creative North Star: "The Shoebox"**

A physical box of index cards kept somewhere you can reach it. You lift the lid,
take the card off the top, decide what it is, and it leaves. The box never tells
you how many are left, because a box can't count and neither can this product.
Every structural decision follows from that image: the header is the lid, the
notes are cards, the stack beneath the top card is the only thing that says
there is more, and the surface has one thing on it at a time.

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

**The box grew a lid you can open.** For most of this surface's life there was
no navigation at all, which was right when there was one screen and then two.
There are thirteen now, and the shape that replaced "no navigation" is not a
menu bar: it is one mark and three icons, and everything a screen can reach is
either behind one press in the lid or written under that screen's own title.
The shoebox still has one opening. You just no longer have to know where it is.

**Key Characteristics:**

- One card at a time; the stack is texture, never a number
- 3px `#1c110b` outline on every object; 2px only on a small control inside a
  card, and never thinner than that
- Flat saturated fills; gradients only on the field, never on an object
- Cream card stock (`#fdecd4`), never white
- Orange means *makes something*; the three disposal actions never take it
- Four states, four colours, taken from the mascot's notebook tabs; orange is
  an action, never a state
- Three kinds of control, three marks: a fill, an underline, a chevron
- Every screen opens the same way — title, then what it is about, then the way
  onward
- Keyboard letters are actions; movement never takes a letter

## How this document stays true

This file has drifted from the code twice. On 20 August 2026 an outside review
found seven places where it described a system the stylesheet no longer had; on
22 August it was still claiming the product resisted zoom in ways the shipped
CSS does not. Both times the drift was found by a person reading the markup line
by line, because nothing else could find it.

Two things now hold it:

- **The appearance snapshot** (`internal/web/appearance_test.go`) records the
  computed shape of every selector that has a shape of its own, across every
  screen. It catches the *code* moving. It cannot catch this document going
  stale, because it does not read this document.
- **The contrast walk** (`internal/web/contrast_test.go`) measures every
  visible run of text and every placeholder on eleven screens, at 1280px and
  390px, against the background it actually composites onto, and fails below
  4.5:1 — 3:1 for large text. Added 24 August 2026, when a review by eye found
  six failures at once and two of them were text nobody could see at all.
  
  It is here rather than in the snapshot because the two ways a colour goes
  wrong are both invisible in a diff. A scoped rule reaches further than its
  author meant — `.checkin .lead` was written for a question standing on the
  field and also caught every label inside the offer card, so the head of the
  one thing home hands you rendered cream on cream stock at 1.18:1, and
  `.results .lead` did the same to *HOW OFTEN?* on a search result. Or a fill
  and a type colour are each chosen well and never measured together, which is
  the whole of The Orange Ink Rule. This document already said *measure a
  colour against the type that will sit on it*; a person doing that by eye is
  the thing that failed.
- **A blocking CI check** (`.github/scripts/design-follows-the-code.sh`). A pull
  request that changes `internal/web/static/pile.css` or any template does not
  merge unless `DESIGN.md` changed with it.

The second one will sometimes be wrong: plenty of stylesheet changes document
nothing here. **Put `no-design-change` in the pull request title or a commit
message** and it stands down, saying in the log that it did. That marker is the
honest way past it. If it starts appearing on most pull requests, the check is
measuring the wrong thing and should be changed rather than routinely overridden
— which is the failure mode a hard gate has, and the reason the override is
named here rather than left to be discovered.

## Colors

The palette is the mark's, not a preference: purple and orange are the product's
own colours, and any screen that reads as neutral has failed to be Squirrel.

### Primary

- **Cap Purple** (`#6c4da9`): the card's title bar, and the interval a chore
  currently has. The mid purple of the mascot's cap.
- **Field Purple** (`#58388a`): the base colour of the room the cards sit in.
- **Lid Purple** (`#3b2560`): the header bar and its brim, and the scrollbar
  track. The deepest purple in the system; nothing goes darker except the
  outline.

### Secondary

- **Acorn Orange** (`#e66d0d`): reserved. It marks the actions that create
  rather than end — *make a chore*, *tell it*, *decide it*, and the door out of
  the ending — plus the search match underline, the caret, the selection
  highlight, the scrollbar thumb and Buddy's context rule.
- **Lit Orange** (`#ff8a2b`): the hover of anything already orange, and the
  focus ring on purple.

**Everything that stands on the orange is the outline ink.** See The Orange
Ink Rule below; it is the one thing about this colour that changed on
24 August 2026, and the fill did not move.

### Tertiary

The four states, lifted from the coloured page tabs of the notebook the mascot
holds. Each has a fill for the tab and a darker sibling for type, because the
fills are picked to be recognised at a glance and type has to be read.

- **Notebook Violet** (`#5e23b1`, ink `#56209f`): `open`. Still in the pile.
  It does a second job as the product's own "this one" — the open lid icon, the
  place you are standing in the menu, the interval a chore has.
- **Done Green** (`#529414`, ink `#3c6d0e`): it was a task and it happened.
- **Kept Amber** (`#ffb300`, ink `#8a5c00`): not a task. Reference.
- **Dropped Brown** (`#8a6a55`, ink `#6a4f3e`): it stopped mattering. Muted on
  purpose — dropping is not a failure and must not be coloured like one.

**Acorn Orange with ink `#b0530a` is not a fifth state.** A note promoted to a
chore is recorded as `done` — the spec says so and `items.state` has no value
to hold anything else. The orange CHORE stamp is the moment of promotion, shown
once on the card as it leaves. Nothing in a search result may ever wear it.

### Neutral

- **Outline** (`#1c110b`): every stroke in the system. Also all body type on
  cream.
- **Card Stock** (`#fdecd4`): the tail's cream, lightened. The surface notes are
  written on, and the surface both panels in the lid are made of.
- **Stock Shade** (`#f6d5ab`): the cards underneath the top one, an archived
  card, and the fill of the one control that means *not this*.
- **Paper** (`#fffbf3`): everything that sits *on* the stock — buttons, chips,
  the stamp, the slot — and all type on purple.
- **Tail Cream** (`#fed6a7`): the mascot's tail. Secondary type on purple, the
  lid's icons, and every quiet pill standing on the field.
- **Headphone Brown** (`#58413d`): meta type on cream. Never grey.

### Named Rules

**The Orange Reserve Rule.** Orange marks creation, never disposal. *Done*,
*keep* and *drop* are paper; *make a chore*, *tell it* and *decide it* are
orange. If a second orange element appears on a screen, one of them is wrong.

**A chore at rest is not orange.** The orange belongs to the moment a note
*becomes* a chore, shown once, on the card where it happened. A chore living
its life afterwards — on the chores screen, in a list, being adjusted or
stopped — is cream stock like everything else the owner owns. A column of
orange tabs would put a dozen creations on one screen and cost the one that
means something.

**The Orange Ink Rule.** Type on Acorn Orange is the outline `#1c110b`, never
paper and never white. Added 24 August 2026, and it is the Fill-and-Ink Rule
arriving at the one fill that had been exempt from it by habit.

Paper on `#e66d0d` measures **3.1:1**. Every orange control in the product is a
13px uppercase label, so every one of them failed, on five screens, for the
whole life of the screen: *tell it*, *make a chore* on the deck and again on
each search result, *make it a chore*, *decide it*, *say it*, and the door out
of `/enough`. The keycap inside *make a chore* was worse at **2.28:1**, because
the keycap sits at 72% of its label's ink — so it stands at full strength on
this one button and nowhere else. The selection band went the same way; white
on that orange was a selected sentence you could not read.

The outline on the same orange measures **5.78:1**, and **7.87:1** on the lit
orange it hovers to. So the fill did not move. That matters more than the
number: `#e66d0d` is the mark's own colour and darkening it to carry white
would have taken the brand off the one control that most carries it.

**The press is the exception, and it is stated at every site.** *tell it*,
*decide it* and *make it a chore* swap their fill for a violet while they are
held, and *make a chore* for a purple; the outline on those measures 2.07:1 and
2.88:1. Paper goes back on for those 90ms. Written out at each of the four
rather than inherited, because a press state that reads correctly is invisible
and a press state that does not is invisible too.

**The No Neutral Rule.** No surface in this system is white, grey or
near-grey. (`#fefefe` was type inside the orange selection band and is not
used any more — see The Orange Ink Rule. The one survivor is the focused slot,
which lifts a shade to say the caret is in it. The rule is about the palette
and that does not introduce a grey.) Cards are cream, the field is purple,
secondary type on purple is tinted from the tail cream and secondary type on
cream is tinted from the headphones. A grey would be the first sign the design
has drifted off the mark.

**The Fill-and-Ink Rule.** Every state colour ships as a pair. The fill goes on
shapes — tabs, dots, pressed buttons. The ink goes on type. Never set type in a
fill value; three of the four fail contrast at label size.

`state-done-ink` moved from `#3c6d0e` to `#37640d` on 24 August 2026. It reads
on card stock either way; on `card-deep`, which is the archive's own stock and
the one place it lands there, the old value measured 4.44:1. An ink that passes
on one of the two stocks in the system is an ink that is right by accident.

**The Lifted Fill Rule.** A control that acts on something wears that thing's
colour at rest, not only under the thumb — but the state fills are chosen to be
seen on a page edge rather than read through, so the fill is *lifted* toward
white until the card's own dark ink reads on it. 18% for the resting state, 32%
for hover, and pressing drops back to the base colour, so the press still reads
as a change:

| Control | Rest | Hover | Press |
| --- | --- | --- | --- |
| *did it* — on a chore, on a task, on the offer | `#71a73e` | `#89b65f` | `#529414` |
| *stop asking* | `#9f8574` | `#af9a8b` | `#8a6a55` |
| *back in the pile* | `#9e7bd0` | `#b295da` | `#5e23b1` |

**The percentages are the outcome, not the rule.** *Back in the pile* returns a
thing to the pile, so it wears the pile's own Notebook Violet — and violet is
much the darkest of the four state colours, so 18% is not enough lift for the
ink to read: it measures 3.17:1, which fails. It takes 40% to reach 5.46:1,
inside the band the other three already sit in. Lift until the ink reads, then
measure; do not lift by a number because the number worked somewhere else.

Type on a lifted fill is always the outline ink, never cream — the lift exists
precisely so the ink can stay dark, which is the Fill-and-Ink Rule holding
rather than bending. Measured, not judged: 6.4:1, 5.4:1 and 5.46:1 against
`#1c110b`. The values arrived after two attempts that passed contrast and made
the screen *darker* than the cards, which is how you can tell brightness is a
requirement and not a preference.

**Amber left this table on 21 August 2026, and its leaving is the rule working.**
*How often* was a filled amber pill, which made the loudest control on a chore
card the one that changes nothing on the server. It is a disclosure, so it is
outlined paper now and the chevron says the rest — see The Three Marks Rule.
Amber is a state fill again and only a state fill, which is what it always
should have been: on a screen that showed a note and a chore together it would
have been saying two things at once.

**Three fills is now the whole lifted set**, and each of the three names a
transition that exists in `items.state`. A fourth would mean a fourth state.

## Typography

**Display Font:** Inter Black (900), for the wordmark and for every screen title
**Body Font:** Recursive, self-hosted variable, `MONO` and `CASL` axes
**Label Font:** Recursive, same family, `CASL` 0

**Character:** One variable family carries both halves of the brief instead of
splitting the difference. Recursive's casual axis is brush-drawn and sits close
to handwriting; its linear axis is even and mechanical. The owner's words get
the first, the product's machinery gets the second. Inter Black is the printed
label on the side of the box, played against the drawn mascot beside it.

### Hierarchy

- **Wordmark** (Inter 900, 26px, `-0.028em`, 2px outline stroke via
  `paint-order: stroke fill`): the product name in the lid.
- **Screen title** (Inter 900, 21px, `-0.01em`, sentence case, paper on the
  field): the name of the screen you are on. One element, one class, thirteen
  screens.
- **Headline** (`CASL` 1, `wght` 800, `clamp(27px, 3.4vw, 36px)`, 2px outline
  stroke): an empty state, and any moment where the product speaks in its own
  voice at full size.
- **Note** (`CASL` 1, `wght` 520, `clamp(21px, 2.2vw, 27px)`, `1.4`, max 34ch):
  the owner's captured text. The only casual-axis element that is his words
  rather than the product's.
- **Label** (`CASL` 0, `wght` 830, 13px, `0.08em`, uppercase): every button.
- **Title** (`CASL` 0, `wght` 750, 12.5px, `0.09em`, uppercase): the card's
  title bar, result meta lines, state names, and the quiet pills.
- **Keycap** (`CASL` 0, `wght` 800, 11px, 2px `currentColor` border, 5px
  radius, 72% opacity): the letter that performs each action, inside its
  button. One size wherever it appears — a key on a chip is the same key.
- **Stamp** (`CASL` 0, `wght` 850, 15px, `0.1em`, uppercase): the state word on
  an actioned card.
- **Result** (`CASL` 1, `wght` 480, 18px, `1.4`, max 46ch): a note's text in
  search results, and a task's text on the tasks screen. Smaller than the
  deck's note because it is being scanned rather than decided on.
- **Meta** (`CASL` 0, `wght` 750, 11.5px, `0.1em`, uppercase): the date and
  state line above a result, a chore's rhythm, an offer's head. It sits 6px
  above what it labels and takes no margin over it — a label belongs to the
  thing under it, and unstated it took the browser's own 1em on both sides and
  floated equidistant between the card's edge and the words it names. The
  `·` between two halves of a meta line stands at 80% of the ink, not 45%:
  at 45% it measured 2.19:1, which is not a quiet character, it is an absent
  one.
- **Group** (`CASL` 0, `wght` 760, 12.5px, `0.1em`, uppercase, cream): the
  label over a run of cards saying what the run is — *WAITING ON* on the
  set-aside, *CHORES* and *EVERYTHING ELSE* in search. A step above Meta
  because it stands on the field rather than inside a card, and it is read on
  the way past rather than once you have stopped.
- **Voice** (`CASL` 1, `wght` 450, 17px): the product speaking in full
  sentences — an empty search, the answer to *I can't start*.
- **Slot field** (`CASL` 1, `wght` 480, 18px, `1.45`): what you type into the
  slot, the reword box, and the new-task line.
- **Search field** (`CASL` 1, `wght` 450, 15px): what you type into the lid's
  search.
- **Conversation** (`CASL` 1, `wght` 480–520, 16px, `1.4`): a sentence being
  said rather than labelled — both voices in Buddy's sheet, a piece of a split
  proposal, a thing the coach is asking permission for. It sits at the
  document's own base size and is the only role that does. It is not the Voice
  role: Voice is the product narrating on a screen, and this is one turn of a
  conversation, which wants to look like the turn above it.
- **Nav** (`CASL` 0, `wght` 640, 14.5px): a place in the menu. Its sibling under
  a screen title runs one step quieter at 13.5px `wght` 600.

### Named Rules

**The Two Voices Rule.** `CASL` 1 is his words. `CASL` 0 is the product's.
Nothing is set in the casual axis unless the owner typed it, is typing it, *or
the product is speaking in full sentences* — the Headline and Voice roles, which
are the two places Squirrel talks rather than labels. No control is ever set in
it: a button is machinery, and machinery does not have a voice.

**The One Title Rule.** Every screen that is a screen opens with one `<h1>`
carrying one class, in Inter, sentence case, in the same place. There were five
treatments and ten of thirteen templates had no `<h1>` at all, which meant
heading navigation did not work anywhere in a product built for someone who
might well use it. ~~A title never counts anything — this is the one place a
count would look most reasonable, and *3 things you kept* is still a count.~~
**The no-count half was retired on 24 August 2026** with Principle 2; a title
still does not count anything, now as a matter of taste rather than of rule.

**The thread has no `<h1>`**, on home's own exemption, and a turn that opens a
place carries that place's name as an `<h2>` instead. Heading navigation walks
the conversation rather than a list of pages.

**Inter is the box's own label, so a title is set in it.** The lid says
Squirrel; each screen says its own name in the same face, one size down. That is
the second and last place Inter appears, and the pairing is the point: the drawn
world is Recursive, and the two things that name a place are printed.

**The empty states keep a treatment of their own** — the Headline role, larger,
in the casual axis, with the mascot above it. A screen that is an absence is a
different composition, not a different size of the same one. Two headings with a
reason, rather than five without one.

**Stopping is not an empty state, and it has its own drawing to say so.** The
empty pile, the empty chores, the empty tasks and the empty archive share the
mark. `/enough` does not: it has a resting pose — leaning on a closed lid, cap
forward, tail settled — at `resting.png`, drawn at 300×207 rather than the
shared 186×140.

Both halves of that are load-bearing. For its whole life this screen used the
same drawing at the same size as the four true empty states, so **choosing to
stop looked exactly like having nothing left**, which is the one equivalence
this product exists to break. And the size is not decoration: the pose is a
whole figure where the mark is a head, so at 186px the face — which is the
character — landed at about a third of the width it has on every other screen,
and the screen read as the same drawing shrunk rather than as a different one.

*Resting*, and the reject list is the brief: not asleep, not relieved, not
pleased. Stopping is a normal ending, so a drawing that looked rescued would
make it something you needed rescuing from, and one that looked proud would
make it an achievement.

**It carries soft shading, and the mark does not.** Recorded rather than
smoothed over: the fur has modelled light and the box has a shaded face, where
`logo.png` is flat fill throughout. The no-gradient rule governs *objects you
can press* — see The Field Exception, which the mood tiles' gloss already sits
under — and a drawing is not one. It is still the only illustration in the
system that is modelled rather than flat, and a second one is a decision rather
than a precedent.

**The Soft Elapsed Rule.** A chore reports when it was last done in words and
never in numbers: *today*, *yesterday*, *this week*, *last week*, *this month*,
*a while back*. The buckets stop there — there is no bucket for a long time,
because that sentence is about the person rather than the chore. A chore that
has never been done says nothing at all about it; what has not happened is not
reported. An exact day count is a deadline wearing a different hat, and it
grows while nobody is looking, which is the shape this product exists without.

**The No Mono Rule.** `MONO` is pinned to 0 on both voices. Recursive ships a
monospace axis and this is a warm tool, not a terminal.

**Case is applied, never typed.** A word is written in the vocabulary as a
person would say it — *this week* — and `text-transform` shouts it. What is
read aloud stays a sentence.

## Layout

One column, `min(720px, 100%)`, on every screen. No sidebar and no second
region. Every screen is the same three slots in the same order:

**Title → what the screen is about → the way onward.**

The third slot is `.ends`: the same distance below the content everywhere
(30px), the same size, the same place to look. It was a footer under a list on
some screens, a pair of links on others, and absent on three, which is what
made thirteen screens read as thirteen things.

**The column does not move between screens.** `html` carries
`overflow-y: scroll`, so the scrollbar is reserved on every screen rather than
appearing only on the ones long enough to need it. Without it the 720px column
— every title, every card, every button — sat five pixels further right on
`/moods` and `/kept` than on `/chores` and the deck, and walking between two
screens slid the whole page sideways.

`overflow-y: scroll` rather than `scrollbar-gutter: stable`, and the difference
is what stands in the reserved strip. A stable gutter reserves the space and
paints nothing in it, so ten pixels of field showed beside the lid — which is
the one full-bleed object in the product — and the lid stopped short of the
edge. Kept as a scrollbar, the strip is the track this system already themes,
and that track is the lid's own Lid Purple. There is no seam because there is
nothing there that was not already there on every screen long enough to scroll.

**The Top Edge Rule.** Content starts at the top of the field, 40px down. The
deck is the one exception and keeps the vertical centring, because one card
floating in the middle of the room is the shoebox's composition rather than a
layout default. Everything centred was right when the pile was the only screen;
with thirteen it left 230px of empty field above a heading and spent a 900px
viewport on one row of moods. The deck's centring is `safe`, so a card taller
than the window top-aligns instead — a centred flex child that overflows puts
its own head above the scroll origin, where nothing can reach it.

**Every screen states its width.** `held` and `moods` had no width rule at all,
so as flex children of a centred `main` they shrank to their content: measured
at 1280px, `held` rendered 440px and `moods` 227px, against everyone else's
720px. Not merely narrow — not even the same width as each other, because the
width was whatever the words happened to need.

### The lid

A full-width flex bar: the mark and wordmark on the left, then a spacer, then
the timer strip if one is running, then **three 44px icons** — Buddy, search,
and the map. Padding `11px 22px 13px`. Below it hangs the brim, a full-width SVG
whose outline stroke is held at 3px by `vector-effect="non-scaling-stroke"`
regardless of viewport width.

Both panels — the menu and the search field — hang absolutely from the right
edge of their own control, 8px below it. Absolute, so opening one never moves
the page underneath it; right-aligned, because both controls sit at the right
end of the lid and a panel hung from the left ran off the side of a phone.

### The thread

**Home became a conversation on 24 August 2026.** What follows replaces its
fixed order; everything home carried is a turn now, and the record of what it
was is in `docs/superpowers/specs/2026-08-24-the-thread-design.md`.

**The thread is the one screen with no title**, and it keeps that for home's own
reason: every other screen is a place you navigated to and its title answers
*where am I*; this is where you start, and nobody arrives there wondering. What
replaces the title for heading navigation is **an `<h2>` on any turn that opens
a place** — walking the headings walks the conversation, which is a better map
of this app than a list of pages was.

Three bands, top to bottom:

1. **The rail** — the four doors, pinned under the lid and never scrolled away.
2. **The transcript** — what has been said, oldest first, ending at the live
   edge.
3. **The dock** — the slot, pinned to the bottom, on every view because there is
   only one view.

`main` is a column of full-width bands here and a centred flex row everywhere
else. That is a real difference and it is load-bearing: as a row it put the
three bands side by side and shrank the conversation to 240px of a 390px phone.

**The four doors are equals**: four cells of one grid, the same stock, the same
depth, rendering identically in every state — art beside name above 620px, art
over name below it, four across at every width. What changed with them is that
they carry a number, and the number is what is *waiting* behind the door rather
than what it holds. Zero renders no number at all: a door reading `0` is a
scoreboard, and that is what Principle 2 was protecting against when it was
retired.

### Triage, in the conversation

One note at a time, exactly as the deck does it: the pile is a thing you decide
about, and a list of things to decide about is a list you are behind on. What
changes is that the next one arrives underneath the last rather than replacing
it, and that having decided is still the moment you are most able to decide
again.

**The card carries two rows, and the second is quieter.** DONE, KEEP, DROP and
A TASK end the note or move it; *make a chore*, *say it another way*, *i can't
act on this* and *this is more than one thing* ask a question instead. The
second row wears `.abtn.why` — transparent, brown type, no fill and no
uppercase — because a way out should not look like a way on. The split press is
drawn only on a note that looks like several things, which is a free check and
what keeps the model off every note in the pile.

**The next note is its own turn**, not part of the answer, so pressing DONE on
it cannot read as pressing DONE on what was just decided.

### The four shapes a question takes

Each is a card of the same stock, and what makes it a question rather than a
thing is what is inside it.

- **A row of choices** (`.pick`) — the interval question. Two radio rows and one
  submit; see Chips.
- **A month** (`.calbox`) — the day question; see Chips.
- **A box of words** (`.wordbox`) — rewording. Its own box and deliberately not
  the dock: the dock keeps everything you type, and these words replace
  something. A dock that sometimes captured and sometimes edited would be a dock
  you could not trust with a thought. `.wordbox` and not `.say`, because the
  coach sheet's slot already wears that word.
- **A proposal** (`.cut`) — the split. The pieces are drawn as they would be
  kept, on paper stock inside the card, so what you are agreeing to is the thing
  in front of you rather than a description of it. Nothing is written until the
  press.

**A notification lands in the conversation.** `/at/{id}` is the URL a leave-by
warning carries and it keeps working forever — one sent last week is still on a
lock screen, and a link that 404s is worse than one that lands somewhere true.
It writes the appointment as a turn and sends you to the thread, so tapping a
warning puts the thing you are about to leave for at the live edge, with what to
take on it.

It is the one GET in the product that writes. That is deliberate: the tap *was*
the press, and the redirect means reloading where you land does not write again.

**A refusal is answered with where the switch is.** A browser told no will not
ask again and this site cannot make it, so there is no button to draw — one
would be a control that cannot work. What is shown instead, once and only to
somebody who has actually refused, is the sentence that says where to turn them
back on.

**An empty place is a sentence, not a drawing.** The empty states were their
own treatment — the mascot, a headline in the casual axis, a screen that is an
absence. A place that is a message has no screen to be absent: Buddy says
*nothing to decide about* and the conversation carries on. `/enough` is the last
screen in the product that is an absence, and it keeps the treatment.

What that costs, stated rather than discovered: the drawing was doing work a
sentence does not, which is to make an empty pile feel like a place rather than
a failure. If the sentence ever starts reading as a shrug, the drawing is what
to put back — inside the turn, not around it.

**A search is a turn, and a result carries no verbs.** One field in the lid,
both kinds of thing, and every state — a result says which of the seven it is
in, because without that an open task reported itself as being in the pile and
was offered the pile's verbs. It is a thing you went looking for rather than a
thing you are deciding about, which is the distinction the deck's results screen
already made when it drew a found chore as a link.

Search-as-you-type belongs to the deck, where a query swaps the stage for
results in place. On the thread the form posts and the answer arrives as a turn,
so that enhancement stands aside rather than fetching a page nobody asked for
and pasting it over the conversation.

**A question in progress owns the keys.** While the interval question is on the
table a digit is a count, a letter is a unit by the word's own first letter, and
Enter answers — so `d` means *days* rather than DONE, and a letter aimed at the
question cannot act on the note behind it. The deck earned this carve-out and
then lost the digits when its disclosure went; they are back.

Only that question. A digit inside the day picker would be a day of the month,
and guessing which of the two was meant would book the wrong one.

**Buddy may speak first, and only when there is something true to say.** The
bar is meaningfulness, not a budget: silence on a quiet afternoon is the
correct answer, and a line that arrives whether or not anything is happening is
wallpaper by the third day. One thing, never a summary — and never the same
sentence twice in a row.

**A reply hands you a few, not all of them.** Five cards and one press for the
rest. A reply that arrives as a screen of list is the work the conversation was
supposed to replace.

**A model may remark; only the product may instruct.** One short sentence about
what a set has in common, after the product's own. Advice is refused in code
rather than discouraged in the prompt.

**Nothing is drawn around the face.** The artwork carries the same heavy
outline the system draws on every object, so a disc behind it stacks two — and
two of them is the heaviest thing on a surface made of light cream bubbles. It
is also this world's own language: the material is described as the mascot's
sticker drawing, and a sticker has no plate under it.

The gutter is 40px rather than the 34 a disc needed. A circle crops to the
face; a silhouette has to fit the ears and the headphones inside the same
width, so at the same size the face itself comes out about a fifth smaller.

**The artwork is head-only, and that is what makes it work.** An avatar is a
portrait, and artwork whose centre of mass is the head crops to a portrait by
itself. The version with a tail needed the crop to fight its own composition:
the face landed low and left in every framing, because the frame's centre was
never the head's.

**Only Buddy has a face, and it appears once per run.** There is one person
using this and he knows which words are his; a profile picture on your own
messages is furniture, and the alignment and the fill already say it. What the
conversation was missing was not two avatars but one presence — the product is
built around a mascot who appeared nowhere in the conversation he was having.

It is the mascot's own head, drawn. The acorn was tried first and shipped for
an afternoon — DESIGN.md's rule that the acorn exists for exactly this size was
written when the only alternative was the whole character, tail included, which
is mush. A head-only portrait is a case that rule did not consider, and it was
measured rather than argued: at 40px the cap, the acorn badge, both eyes and
the mouth all read.

Once per run, so consecutive turns read as one utterance. An acorn on every
bubble is wallpaper by the third day, and habituation is a documented risk for
this user.

**A turn is a gutter and a column.** Buddy's face on the left, everything he
said to the right of it. Every turn of his reserves the gutter whether or not
it draws a face, so a run stays on one left edge instead of stepping in and
out — and the place's name sits in that column too, because a heading outside
the column it titles belongs to nothing.

**A full-width control spans the gutter.** A question with its answers drawn on
it is a control strip rather than speech. The check-in's five labels stopped
fitting a 390px phone the moment the gutter took 44px off them.

**Tight within a speaker's run, generous between them.** A conversation where
every gap is the same size is a list of sentences. The reply sits closer to
what it answers than to the next thing entirely.

**A bubble is card stock, never paper.** `--paper` is the raised things sat *on*
the stock; a bubble on the purple field is the stock. Buddy's was `--paper` for
a day, which is near enough to white — and the second line of this system's key
characteristics is that a white surface is not Squirrel. What let it drift is
that the fill was doing the work of saying who spoke, and the face does that
now.

**What is always there is quieter than what just arrived.** The two chips at
the foot of the conversation are outlined rather than filled, and they sit on
the thread's own left edge rather than centred. At identical weight, directly
under chips that had just arrived, they read as a footer belonging to the page.

**The page reserves the dock's measured height, and the reserve is on the whole
column.** The dock is fixed to the bottom and covers whatever is under it. The
clearance sat on `.thread` until 25 August 2026, and `.thread` is not the last
thing on the page — the two chips and the way out are below it, so on a phone
they were behind the box with no way to scroll them clear. Measured rather than
guessed, because the slot grows as you type: at four lines a static reserve
leaves the last thing you said underneath the box you said it in. The
stylesheet keeps a fallback for the height at rest, so the page is right before
the script runs.

**What must be reachable from anywhere goes at the foot of the thread, not on
the live edge.** The live edge is the newest Buddy turn, and there is not always
one — a conversation that has not started has no turns at all. A control that
appears once Buddy has spoken is a control you cannot use to start.

**A turn with nothing on it can still take you somewhere.** A chip that hangs
off a drawn card disappears with the card, and the branch that draws no card is
usually the branch where going elsewhere is the point. The shelf was reachable
from nowhere for a day because of exactly this.

**A photograph on a card is bounded by height, and it is the small copy.**
A card in a conversation is read at a glance and scrolled past, so the failure
to prevent is a portrait photograph pushing the buttons off the screen —
`max-height`, and `object-fit: contain` rather than `cover`, because a note's
photograph is usually *of words* and cropping them is losing the note. The card
asks for a card-sized copy and links to the whole picture; tapping it is how
you read something photographed at arm's length.

A kind the standard library cannot decode has no small copy and never will.
The card draws the original rather than a broken image, and that is a larger
download for WebP and HEIC.

**A chip that acts is a form, not a link.** Chips in a turn carry either an href
or an action, never both: a GET that writes writes again on every reload, which
is the same reason a door is a press.

**The live edge.** Only the newest turn Buddy has spoken carries controls. Older
turns keep their words and lose their buttons — a card from a conversation three
days old acts on a state nobody is looking at, and history is not rewritten.
The script applies the same rule to what is already on the screen when new turns
arrive.

**Buddy does not talk over himself.** Nothing is put on the table while
something Buddy already put there is unanswered. Without that rule the offer was
appended on every page load: a reload wrote a second copy into a record that is
never rewritten, and it stole the live edge from whatever had just been said.

**The script is required here.** ~~Every upgrade `pile.js` makes works with the
script off.~~ **Retired 24 August 2026**, with Principle 2: without JavaScript
the thread is the lid, the rail and nothing else. What is *not* retired is the
single rendering path — handlers return HTML from the same templates whether the
browser asked for a page or for the turns a press produced, so there is one
description of a card rather than two that can drift. Capture stays in
`pile.js`, which owns the camera, the stash and the offline path; `thread.js`
owns everything else and exposes the two entry points capture answers through.

### The deck

The card sits vertically centred in the remaining space with its note block
centred inside it, so a two-word note and a forty-word note both sit on the same
optical line. Minimum card height 296px desktop, 284px mobile. Search replaces
the deck in place — same column, same width — and switches the main region from
centred to top-aligned. Results scroll; they never paginate, because a page
count is a total in disguise.

### Breakpoint: 620px

Below it the four answers become a two-by-two grid with *make a chore* on its
own row beneath them; the chore's three actions become a two-column grid; the
interval chips become two columns with the lead and *never mind* spanning both;
the five faces share one row exactly rather than each taking its own width. The
mark drops to 56×42. Touch targets are 48px in the action row, 44px everywhere
else.

***Make a chore* has its own row and not the whole of it.** This document said
full-width until 24 August 2026 and the stylesheet has never done it: the
summary spans the grid and then sits at the end of its own row, with the reason
written beside the rule. It is the rarest of the five answers on the card and
spanning made it the widest, loudest object under the thumb — the same argument
that refuses *stop asking* the full width eighteen lines further down in the
same file. The code was right and the record was stale, which is the third time
that has been true here and the reason the two checks below exist.

**The Step-Up Rule.** A phone is read at arm's length in worse light, so roles
step *up* below the breakpoint rather than down: the note goes to a flat 23px
rather than the clamp's 21px floor, the headline to 31px, Result to 18.5px, Meta
to 12px, and the slot, the reword box and the new-task line to 18.5px — the
last of those because **anything under 16px makes iOS zoom the page on focus**,
which is a hard floor for every field and not a preference. These are the same
roles at a second size, not new roles.

**The lid's search field spent a release under that floor**, at 15px: the rule
inside the breakpoint that set 16px was written for `.find input`, and when
search moved behind an icon the field became `.findbox .find input`, which
outranks it. Fixed in v0.19.0, and the fix had to be placed *after* the rule it
corrects — an override at equal specificity earlier in the file loses the same
way the original did. The episode is kept because the failure mode is the
point: a rule can be present, correct, and outvoted.

**Nothing on a phone zooms.** The viewport refuses to scale
(`user-scalable=no, maximum-scale=1`) and `touch-action: manipulation` drops
double-tap magnification. This is a screen you glance at with one thumb, and a
stray pinch that leaves it magnified is a screen you have to repair before you
can use it. Panning survives.

**A deliberate pinch does not survive, and this document said it did.**
Corrected 22 August 2026. `user-scalable=no` with `maximum-scale=1` takes
enlarging-to-read along with the accidental magnification it was aimed at —
wherever it takes effect at all, which is the installed app, which is the
scenario this product is built for. `touch-action: manipulation` is the part
that only drops double-tap; the viewport line is broader than that and the two
were described as though they were one.

So this fails WCAG 1.4.4 (Resize Text) in the installed app, and nothing here
substitutes for what it takes: every size in this system is a fixed pixel
value, so the OS text-size setting does not reach them either.

Accepted on one ground, and it is worth stating rather than implying: the only
user has no vision need this takes away from, and a screen that has to be
repaired before it can be used is a cost he does have. It is not accepted on
the ground that the criterion does not apply — it does, and this is a screen
that would fail an audit. If the ground ever changes, the answer is not to
restore pinch-zoom, which breaks the one-thumb layout this protects. It is a
text-scale the person sets, persisted per device, multiplying a type scale that
would first have to stop being fixed pixels.

**Neither of those retires the 16px floor, and the floor is the load-bearing
one.** iOS ignores `user-scalable=no` in a browser tab and has since iOS 10, so
the no-zoom rules hold in the installed app and not there — while the zoom that
happens when a *field takes focus* is a different mechanism from the one that
meta tag governs, and no viewport setting prevents it. Every field clears 16px
on a phone whatever the viewport says. A test asserts this against the
stylesheet so the floor cannot be deleted on the strength of the meta tag.

**The Lid Step-Down Rule.** The lid is the one thing that gets *smaller* on a
phone: the mark to 56×42 and the wordmark to 20px, against the Step-Up Rule
above and for the reason that rule gives. Roles grow because a phone is read at
arm's length — and the lid is not read. It is what you look past on the way to
the card, and the card is what the stepping up was for. At the deck's own sizes
the lid spent about a fifth of a small screen saying a name you already know.

**Three icons is what made the lid fit.** Before them it was a row of words and
an open search field that ran to three rows on a phone — about a hundred and
eighty pixels of lid before anything you came for. The Step-Down Rule was the
first answer to that and it was not enough on its own.

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
  smaller, so a shallower offset. Also the door, the chore, the task, the offer
  and Buddy's sheet.
- **Panel** (`box-shadow: 0 5px 0 0 #1c110b, 0 18px 30px -18px rgba(0,0,0,.8)`):
  the menu and the search field, hanging under the lid. A deeper cast than the
  result card because it floats over the page rather than sitting on the field.
- **Control** (`box-shadow: 0 4px 0 0 #1c110b`): buttons, chips, the stamp.
  Offset only — a control is a sticker on the card, not floating above it.
- **Small control** (`box-shadow: 0 3px 0 0 #1c110b`): a control inside a card
  that is being scanned rather than decided on — the search result's actions.
  Paired with the 2px outline; see Shapes.
- **The slot, inset** (`box-shadow: inset 0 4px 0 0 rgba(28,17,11,.16), 0 2px 0 0 rgba(28,17,11,.35)`):
  the one object in the system with its shadow on the inside.
- **Field texture** (`radial-gradient(120% 90% at 15% -10%, rgba(150,110,220,.35), transparent 62%)`
  over `#58388a`, plus a 22px dot grid in `rgba(254,214,167,.07)`): the room's
  light, fixed to the viewport.

### Named Rules

**The Press Rule.** Pressing a control moves it down by exactly its offset and
removes the offset, so it lands flush with the surface. `transform:
translateY(4px)` against `0 4px 0 0`; 5px against 5, 3px against 3. The numbers
must always match; a press that doesn't land is the most obvious tell in this
world.

**The Inset Exception.** Everything in this system sits *on* a surface. The slot
is the way *into* one, so it is the one object whose shadow is on the inside and
whose fill is the paper the surface is made of rather than a card stuck to it.
That difference is the point rather than an inconsistency: a card is a note, and
the slot is the opening a note goes through. The new-task line takes the same
grammar for the same reason.

**The Field Exception.** The no-gradient rule governs objects. The field carries
a radial lift and a dot grid because a full-screen flat purple reads as a dead
swatch, and because habituation is a documented risk for this user. Nothing that
can be pressed may take a gradient.

**The lift is `.35`, and the number is a contrast result.** It was `.5`, and
cream on the lit centre of that radial measured 4.19:1 — a fail — which quietly
made every piece of secondary type on the field illegible in the one corner of
the screen where the field is brightest. At `.35` the lit centre carries cream
at 4.8:1 and the room is still a room. The Field Exception buys atmosphere; it
does not buy an unreadable corner.

**The Day Rule.** Three things in this system are chosen from the date rather
than fixed: the sentences you meet most often, the angle the stamp lands at
(between -11° and -3°, from -7° fixed), and where the field's light falls
(between 8% and 26% across, from 15% fixed). The card stack was randomised from
the day it was drawn because PRODUCT.md's own premise is that a surface which
looks identical every time stops being seen within about a week, and "one
frame" — which made every screen deliberately more uniform — started a clock on
the rest.

Four constraints hold it to art and phrasing, which is all the roadmap
sanctions:

- **Chosen from the day, never random.** A phone and a desktop are one product,
  so a line or an angle that differs between them is a bug rather than a
  variation, and a reload is not a slot machine. The server hands the two
  numbers down as `--tilt` and `--light` on the body; `pile.css` carries what
  shipped as the fallback, so a page rendered without them is the original.
- **Never a control.** Every label, every position, every target stays exactly
  where it was. Muscle memory is what "the same every time" protects, and a
  button that renamed or moved itself is a button you have to read again.
- **Never brighter.** The light slides sideways only. Its height and its `.35`
  are fixed, because that number is the contrast measurement above and a
  highlight that wandered vertically would quietly undo it.
- **Never far.** The stamp leans one way, always negative, inside eight
  degrees. Past about a dozen it stops reading as slapped on and starts reading
  as crooked, and the word inside it gets harder to read at a glance — which is
  the one job it has.

## Shapes

Soft rounded geometry, matching the mascot's own construction. Cards take a 14px
radius; anything you press is a full pill (`999px`); the stamp and every
photograph take 10px because they are stuck on rather than pressed; keycaps take
5px.

**The outline scales with the object, and never goes thin.** 3px is the mark's
own weight and the default everywhere. 2px is the one step down, and it is
reserved for a small control living inside a card — a search result's actions, a
keycap, a quiet pill — where a 3px stroke at 12.5px type reads as a blob. There
is no third step. A 1px rule anywhere is a defect, and there are no borderless
objects.

The deck's stack is three cards translated by multiples of an 8px offset and
rotated between 0.7° and 1.8°, randomised on every card change. Nothing in this
world sits perfectly square, because nothing handled ever does.

**Dashed means this does nothing.** The escape chip out of the interval picker,
the disclosure that has opened and become its own way back, the new-chore form
when shut, the rule above a chore's timer: all dashed, and nothing else is.

Icons are authored SVG in the same language, all at a 2.4px round-capped stroke:
the acorn from the cap badge, the search glyph, the hamburger, the cross, the
camera, the chevron. No icon fonts, no emoji standing in for an icon, and no
icon built out of CSS borders — the chevron is a masked path precisely so it can
take `currentColor` and sit at the same weight as its five siblings.

## Components

### The three marks

**The Three Marks Rule.** Pressing something here does one of three things, and
the screen says which before you press it:

| | what happens | the mark |
| --- | --- | --- |
| **a button** | something changes, on the server | a fill, an outline, and the offset |
| **a link** | you go to another screen | an underline, and underline means nothing else |
| **a disclosure** | something opens here; nothing changes | a chevron that turns 180° when it opens |

All three were underlined text in places. On the card, `LATER` went to another
note, *fix the words* opened a box in place, and *i can't act on this* opened a
box in place — three different consequences, one appearance. Meanwhile the
disclosures did not agree with each other either: a filled amber pill on chores,
a `+` on the new-chore form, plain underlined text on the card. Four appearances
for one idea.

**A disclosure in a row of buttons keeps the row's shape.** A chip among chips
must look like its neighbours or the row breaks. It carries the chevron too, so
the mark is what distinguishes it rather than the weight — which is also why it
loses its fill: weight says how loud a control is, and the mark says what kind
of thing it is.

**Underline now means one thing, everywhere: you are leaving.**

### Buttons

- **Shape:** full pill (`999px`), 3px outline, `0 4px 0 0` offset.
- **Action** (*done*, *keep*, *drop*, *a task*): paper fill, outline type,
  12px/18px padding, 44px minimum height. Each carries its keycap.
- **Make** (*make a chore*): orange fill, **outline type**, pushed to the far
  end of the row with `margin-left: auto`. The only orange control on the card,
  and its keycap stands at full strength rather than 72% — see The Orange Ink
  Rule.
- **Hover:** action buttons deepen to `#f7e9d4`; make brightens to `#ff8a2b`.
- **Press:** translate down 4px, offset removed, fill swaps to the button's own
  state colour (`done` green, `keep` amber, `drop` brown, `a task` violet,
  `make` purple). Make's type returns to paper for the press, because the
  outline does not read on the purple it lands on.
- **Focus:** 3px `#ff8a2b` ring at 3px offset on purple; `#5e23b1` on cream.

**A task presses violet, not green.** Deciding sends a note to the tasks; the
pile's own colour is what it wears, because nothing ended there.

- **Chore controls** (*did it*, *stop asking*): lifted state fills with outline
  type, 11px/16px padding, 44px minimum height, no keycaps at rest — the chores
  screen is read on a phone as often as not, and the keycaps appear only on the
  chore you are focused in. *Stop asking* is pushed to the far end with
  `margin-left: auto` the way *make a chore* is on the deck: the action that
  ends something is separated from the two that do not. On a phone it does
  **not** span the full width — it is the one press here that ends something,
  and spanning the card made it the largest thing on it.

### Chips

Interval chips share the button shape at 13.5px/11px padding. The escape chip
(*never mind*) drops its offset and takes a dashed border. The chip that carries
the interval a chore already has is filled Cap Purple: the product's own colour,
not a state and not a reward. A chip is a `<label>` wrapping its radio, so the
whole chip is the target and the choice survives with no script at all — the
radio itself is never drawn, because the press that stayed down is what says
which one is chosen.

**Which day is a month, since 24 August 2026.** The day picker is seven columns
with Monday first, cells 44px tall and about 43 wide at 390px — one pixel under
the touch floor, accepted deliberately: seven columns cannot do better without
cropping the month to weekdays or scrolling it sideways, and both are worse than
the pixel. A day already gone is drawn and not offered, because a month with
holes in it is harder to read than one where some days cannot be pressed. There
is no way back past this month.

Turning to another month is the one control in either picker that posts on its
own. It is not an answer, it is turning a page, so it re-asks rather than
writing your half of a question into a record that is never rewritten.

**How often is a number and a unit, since 24 August 2026.** ~~Four fixed chips —
every day, every week, every 2 weeks, every month.~~ Two rows now: `every`
1/2/3/4/6/8, and `of these` days/weeks/months, in one form with one submit.

The picker asks once and is answered once. Pressing a number is half an answer,
and a control that posted on each press would write a turn per fiddle into a
record that is never rewritten — the sound of somebody deciding, kept forever.

It composes the sentence `every 3 weeks` and hands it to the same parser a typed
sentence goes through, so the two lanes cannot disagree about what a rhythm
means. Six numbers cover what anyone reaches for; `every 9 weeks` is a sentence.
A rhythm the picker cannot say — a fortnight, ten days — opens with **both rows
unmarked** rather than rounding to the nearest offered thing: marking the wrong
one would say the chore is something it is not.

### The quiet pill

The answer you take when you are *not* doing the thing: `LATER` on the card,
*again* on the check-in, *not now* on the offer, the escapes on a proposal.

- **Shape:** full pill, **2px** `currentColor` border, no fill, no offset,
  10px/16px padding, 44px minimum height, Title role uppercase.
- **On the field or the card's purple title bar:** tail cream, hovering to a
  `rgba(254,214,167,.16)` wash and paper type.
- **Inside a cream card:** headphone brown, hovering to paper-deep and outline
  type.

**A quiet pill inside `.ends` keeps its own mark and takes no underline.**
*that is enough whenever you say* is one link, written once, rendered in two
places: bare on the deck's hint line and inside `.ends` on nine other screens.
`.ends a` outranks `.quietpick`, so the same words were a plain pill on the
deck and an underlined pill everywhere else — one control with two
appearances, which is the exact fault the Three Marks Rule exists to prevent,
committed by the rule that was enforcing it. The pill carries its own mark; the
underline belongs to the bare links beside it. `text-decoration-line` is
recorded in the appearance snapshot now, because nothing there could see this.

**Why it exists, which is the clearest evidence the old vocabulary was wrong.**
*not now* on the offer is a `<button>`. `LATER` on the card is an `<a>`. They
were styled by the same class and looked identical: one was a link wearing a
control's job and the other a control wearing a link's clothes, and from the
outside you could not tell which was which — or that either was pressable at
all. Not every pill is loud. But a choice is drawn as a choice.

### The lid

A mark and three icons. All three work with the script off; two of them are
disclosures, which is the same grammar as the chore's interval and the card's
reword box.

- **The icon** is 44px, a full pill, no fill and no border, tail cream,
  hovering to the cream wash. Open, it fills Notebook Violet with paper type —
  the product's own "this one".
- **Buddy is outside the menu**, because a conversation about what is in front
  of you should be one tap. It is the acorn at last, rather than the whole
  mascot at 46px where its face is mush. This is the case the acorn exists for.
- **Search is behind the second icon**, and opens on its own when there is a
  query — a search result never shows you a field you cannot see the words in.
- **The map is behind the third**, a hamburger that becomes the same cross that
  closes Buddy's sheet. One mark for *this is shut now*, not two.

**The floating acorn is gone, and both of its rules survived it.** It was fixed
bottom-right at 62px on every screen. It sat on top of a chore's timer row on a
phone and on top of the sheet's own send button on a wide screen, and a fixed
control over live ones turns a mis-tap into the wrong thing entirely. What it
must keep, wherever Buddy lives: it was `.askacorn` and not `.acorn` because the
card's title bar had that name first for the drawn badge, and taking it made
every note's badge a 62px circle in the corner of the screen for a release —
**the same mark in two roles needs two names.** And it never dims and never
carries a badge: a badge here would be a count, and a dimmed acorn would be the
product reading meaning into how a button gets used.

### The map

A panel of card stock, 6px padding, hanging under its icon. Three places — the
pile, the tasks, the chores — each a full-pill row at 44px in the Nav role.

- **Where you are is in the list and is not a link.** It is filled Notebook
  Violet with paper type and `aria-current="page"`. A menu that drops the place
  you are standing has items that move as you move, so *the second one* means a
  different screen on every screen.
- **Home is not in it.** The mark is the way home and has been since the screen
  existed.
- **Buddy is not in it.** It is one tap in the lid.
- **A screen that hangs off another answers with its parent.** The shelf is
  somewhere in the pile; the archive and the set-aside are somewhere in the
  tasks. The map says which room; the views row says which corner.

### The views row

Under a screen's title, and only under the screens that have corners: the pile
keeps a shelf, the tasks keep what is finished and what is stalled.

- 44px minimum height, 13.5px Nav, tail cream, underlined at 45% opacity —
  they go somewhere, so they are underlined.
- The one you are in drops the underline and steps up to `wght` 780 in paper.
- It sits under the title because that is what it is about: a link to the
  archive means nothing beside the chores.

**This is the half of the problem a consistent frame does not solve.** `/kept`
was reachable from the bottom of the pile and from an empty pile, and nowhere
else — the shelf of things deliberately kept for later sat behind having nothing
left to triage, which is the one state this product exists to say you need not
reach. `/moods` stays behind the check-in on purpose: Principle 5 traded a
construction-level guarantee for *shown back on request, and never on its own*,
and a link in a footer is closer to "on its own" than that bargain allows.

**These were 20.9px inline anchors — 47% of the minimum tap target** — and they
are the only way from a screen to its own sub-views.

### The Door

A card-stock link to one of the box's three halves, and most of the home
screen's body.

- **Stock and depth:** card fill, 3px outline, 14px radius, the result card's
  shadow. It presses flush like everything else you push — `translateY(5px)`
  against `0 5px 0 0`.
- **Name:** lowercase, `wght` 800, in the machine's axis, at the Note role's
  floor (21px, 17px on a phone).
- **Sub-line:** the meta role in headphone brown, saying what the half is.
- **Hover:** brightens to paper. A button hovers by deepening; a card you can
  walk into wants a "pick it up" affordance instead, and that difference is the
  point rather than an inconsistency.
- **Four, two by two, at every width.** Their equality is the screen's one
  statement, and a stacked column would read as first, second, third, fourth. A
  square says none of that. On a phone the art steps down to 50px and the name
  takes the Note floor rather than the phone's step-up — the words have to fit
  the cell, and the cell is what keeps them equal.

  **Four across was tried first and is why this is written down.** At 390px the
  frame is 364px: four cells measure about 84px, where *what you decided*
  already wraps to two lines at 115px. Breaking the row only on a phone fixed
  that and left the doors two different shapes depending on where you opened
  them — so the square is the shape everywhere. At 720px it gives 353px cells,
  which is roomy rather than merely adequate, and equality no longer depends on
  which device is reading it.

  **The cost is the fold, and it is accepted rather than unnoticed.** The block
  is about 270px tall and starts around 568px down, so on a 1280×800 laptop the
  second row is below the fold: you see two doors and scroll for the other two.
  Making them fit would need both rows inside 232px — about 109px each against
  192px now — which even the phone's 50px art does not reach. So it is a choice,
  not a tuning problem. Scrolling home is ordinary; four doors in two shapes was
  the worse trade.

  **A door's name has to fit one line.** All four names are ten characters or
  fewer in a shared grammar; at four across the cell gives about 117px of inner
  width, which is roughly eleven characters at the Note floor. A longer name
  wraps, and a wrapped name drops its own sub-line below its neighbours' — four
  equal boxes with unequal type, which is the equality rule failing quietly
  rather than loudly. The fourth door was called *what is coming* for an
  afternoon and did exactly that.

  **The agenda is named after the thing it holds, and the name was argued
  about.** *Agenda* is the calendar word, and a calendar is what `PRODUCT.md`
  refused. The owner chose it knowing that. What makes it a description rather
  than a slide back are the guard rails on the screen itself: only what is still
  ahead, never a count, and nothing anywhere about being late. If the name ever
  starts pulling the screen toward the thing that was refused, the rails are
  what failed.

A chore found by a search takes the Door's grammar at one line high, for the
same reason: it is a place to go, not a thing to do. What can be done to a chore
lives on the chores screen, and one owner of that means the two views cannot
disagree.

### Door Art

One drawing per door, above the name, in a fixed-height slot — 86px, 50px on a
phone. The fixed slot is the equality mechanism: every drawing renders the same
height whatever its aspect, so no door leads. Flat fills from the documented
palette inside the 3px outline (`vector-effect="non-scaling-stroke"` when the
drawing is SVG). Decorative to assistive tech — the name is directly beneath it.
No CSS shadow: the art is printed on the door's stock, not stuck to it, and its
own outline is its depth. A small mirrored tilt, ±1.5°, because nothing handled
sits square.

Its guard rails, which are a refused drawing written down as rules. Door art may
never depict a count, a progress state, a tick or a completion; never wears a
state colour *as a state* (amber as straw is a colour, a green tick is a claim);
never grey; never orange.

**This still applies to the drawing, and only to it, since 24 August 2026.** The
door wears a number on the pill beside its art now — see The thread — and that
is the pill's job rather than the picture's. A count drawn *into* the art would
be the same number said twice, in the one place it cannot be turned off.

**The exception the door art carries, and the one that was retired.**

The chores door wore a clipboard with two ticked boxes beside one empty, and a
grey clip — a progress reading drawn as a picture, in the `done` green, on the
product whose hardest rule is never a count in any form. It was refused on
those grounds, chosen anyway by the owner, and recorded here as a deliberate
exception rather than an oversight.

**On 20 August 2026 the owner replaced it** with a bucket, a mop, a sponge and
a spray bottle — violet and amber, no ticks, no checklist, nothing depicting a
completion and no grey. It obeys the rails. So that exception is retired rather
than defended, and the entry stays because the reasoning is the useful part: a
drawing that reads instantly as chores did not need a checklist to do it, and
the version that avoided one is also the version that stopped colliding with
the tasks door.

**The tasks door went back to the clipboard on 24 August 2026, and it is an
exception rather than a reversal of the reasoning.**

It is a clipboard with two ticked boxes beside one empty, a pencil, a sticky
note, and the acorn on the clip. Two of three is a progress reading drawn as a
picture, and the Don't list names checklists by name: the rule does not care
whether the total is a numeral. This is the same composition the owner removed
from this door — and from the chores door — on 20 August, for that reason.

Chosen anyway, knowingly, with the record put in front of him first. What it
buys: the errands drawing shared a calendar, a set of keys and an envelope with
the agenda door, and in a two-by-two grid those two sit one above the other in
the same column. Two doors that read alike is a legibility cost paid every time
the screen is opened; a checklist is a rule broken in a picture nobody counts.

Three things make it narrower than the drawing it replaces. The ticks are Acorn
Orange rather than `done` green, so no state colour is being used as a claim.
There is no grey. And the acorn on the clip is the product's second mark used
exactly as this document says it is available to be used — on a badge where the
full mascot would be too much.

**An exception, not a precedent.** The next drawing that depicts a count gets
refused on the same grounds this one was.

**What the tasks door carried before.** It was a clipboard with two
ticked rows and one empty — a progress reading drawn as a picture. The owner
replaced it the same day with errands: a calendar, a phone, an envelope, a set
of keys. **The count is gone**, which was the larger of the two faults and the
one the never-a-count rule actually cares about. What remains is a single green
tick on the calendar — a completion depicted, in the `done` colour, as ornament
— and a black key, which is the only grey left in the system.

**The four objects are four errands, not four affordances**, and the difference
is the whole reading of this drawing. The calendar is *an appointment to make* —
the task — rather than a date the task is due on. So are the others: someone to
ring, something to post, somewhere to drive. That is precisely what a door's art
is for, which is naming the kind of thing behind it, and it is why the drawing
does not imply the due dates a task deliberately does not have.

Worth writing down because the first reading of it here was the wrong one: a
calendar was taken for a deadline, which would have put a picture of lateness on
the one screen built to have none. It depicts a thing you decided to do. Read
door art as a subject, not as a control.

**Door art is the owner's, and the guard rails govern anything drawn by anyone
else.**

### The Slot

The way in, and the first thing on home. Deliberately not a card — a card is a
note, and this is the opening a note goes through.

- **Inset:** paper fill, 3px outline, 14px radius, and the shadow on the inside
  (`inset 0 4px 0 0 rgba(28,17,11,.16)` over `0 2px 0 0 rgba(28,17,11,.35)`).
  See The Inset Exception.
- **Focused:** the fill lifts to `#fefefe` and a 3px `#ff8a2b` ring joins the
  inset shadow rather than replacing it.
- **The field is his voice** — `CASL` 1 at 18px, the same axis the card sets a
  note in, because it is the same sentence before and after it is kept. It
  grows with its content and stops at 8.7em, so a very long thought scrolls
  inside the slot rather than pushing the doors off the screen.
- **The button says what happens**, in the product's own word: *tell it*. Not
  *add*, which is a word about a list, and not *save*, which is a word about a
  document. Orange, because this is the other thing in the product that makes
  something.
- **The camera is a 44px label wrapping a hidden input**, at the same height as
  the words and quieter than the button: photographing is the alternative to
  typing, not the headline. The input itself cannot be styled, so the label is
  the control.

**The confirmation is one word.** *kept*, in the meta role, on its own line
inside the box. It never names a total — it is the chat's ✅ in words.

**A failure is not the success in different words.** The held and refused
messages take the card's own material — cream stock, 3px outline, 3px offset —
rather than sitting as tinted type on the field. That is a legibility answer
before it is a visual one: orange on the lit purple measured 2.43:1, which made
the sentence reporting a failure the least readable thing on the screen. It is
also the truer shape: *kept* is a status you glance at, and these are messages
you have to read and act on, so they are objects on the field rather than more
of it. The refusal carries a 3px orange bar along its bottom; **held does not**,
because held is not failure — the words are somewhere, only the pile has not
seen them yet.

### The Photograph

A photograph with no words is a note, so a photograph is never an attachment
hanging off one.

- **On the card:** above the words, because when there is one it *is* the note
  and the words are the caption. Full width, `max-height: 46vh`, `object-fit:
  contain`, 3px outline, 10px radius, stock-shade behind it so a transparent or
  slow image is never a hole.
- **In a list:** the same object bounded harder — `max-width: 260px`,
  `max-height: 180px`, and `object-fit: cover`, because here you are scanning
  rather than deciding. Enough to recognise the thing, not enough to push the
  next row off the screen.
- **It follows the note everywhere.** The deck, the results, the shelf, the
  tasks, the archive. A photograph that vanished when a note was kept was a note
  that had lost its content.
- **Before it is kept:** an 84px square preview on its own line under the box,
  with a way to take it off. On its own line rather than in the 44px gap beside
  the button, because a photograph squeezed into that gap is a photograph you
  cannot check, and checking it is the entire reason it is there.

**The preview is drawn by the script and by nothing else.** With the script
absent there is no preview and the form still posts, which is the floor this was
built on.

### The check-in, and its answer

Five drawn faces, and they are not a scale and are never numbered: low and
frazzled are different states wanting different answers, which a one-to-five row
cannot say. The drawing is the button — no stock behind it and no outline of its
own, because each face already carries the system's 3px line and a card under it
would be a second border around a thing that has one. Sized by height, because
every drawing is the same size on the sheet they were cut from.

**The faces do not vary, and that is a decision.** Three things in this system
are chosen from the date — see The Day Rule — and the drawn faces were the
obvious fourth: they are the thing met most often on the surface habituation
would cost the most. They were left fixed on 23 August 2026 because the owner
likes them as they are. If the check-in ever starts going unseen, a second
drawing per face is the first thing to try, and the brief for it is in
`docs/art/2026-08-22-prompts-for-the-missing-drawings.md`.

Answered, the whole region becomes one chip: the face and the word inside a
single dark-wash pill, with the two ways to change your mind trailing behind it.
So home has exactly one interactive thing above the doors, and it is either the
question or the answer to it. There is no yesterday here and there never will be.

### The Offer

The one thing home hands you, composed only of parts that already exist:

- **Stock and depth:** card fill, 3px outline, 14px radius, the result card's
  shadow. A card you act on rather than a door you walk through, so it does not
  take the door's hover.
- **No page tab.** A tab says what a note ended up as; this has no outcome yet.
- **Head:** the meta role in headphone brown — *RIGHT NOW*.
- **What:** the Note role's floor at 21px, in the casual axis, because it is
  the owner's own words wherever there are any.
- **Because:** the Voice role, in headphone brown. One clause, lower case, no
  full stop.
- **Controls: one shape, and only colour separates them.** *did it* in the
  lifted done green, because one colour means "it happened" wherever it is
  said. *10 min* in paper, because orange means something was **made** and a
  timer makes nothing. *not now* filled with the card's own deeper stock,
  pushed to the far end: present, pressable, and plainly not the default.

**That third control is the one thing on this page that has been wrong twice.**
It was an unmarked link for months — a `<button>` drawn as a link, which is the
clearest single case of the vocabulary problem. Then it became an outlined pill
in a row of filled ones, which made the card offer four choices in four shapes:
filled green, filled paper, an outlined pill, and a chevron disclosure. Nothing
said which was the ordinary answer, so choosing meant comparing four *kinds of
thing* before you could compare four options. One shape now.

**I can't start** is quieter still, and is a disclosure: underlined text with
the chevron, opening in the same grammar the chore picker and the reword box
use. Its four answers are paper chips of equal weight — one of them louder would
be the product guessing which it is, and it does not know. The answer is one
line and at most one control; there is deliberately nowhere for a second step.

**A fixed point carries one button and no refusal.** It is the one thing on this
screen the world imposed rather than the product suggested, and a control
implying it can be moved would be a lie with consequences.

**The lid is the only place a number counts down**, and the offer never joins
it: the timer strip is a fact about a thing you chose to start.

### The Peek — removed, and what replaced it

An earlier home screen showed the newest note: readable, not actionable, one
card, no stack behind it, absent without residue when the pile was empty. It was
cut on sight. A home screen that shows what is waiting greets you with what is
waiting, however carefully it is dressed. That reasoning is still right and is
why this entry stays.

**On 20 August 2026 the offer took that space, and it breaks exactly one of the
seven prohibitions.** The Peek showed a slice of the backlog, chosen by arrival
order, that you could not act on. The offer is one thing *the product chose*,
with one way to do it and a one-press way to turn it down — so the prohibition
it breaks is "no action", and breaking it is precisely what stops the region
being a greeting from the pile.

The other six hold, and two more join them:

- never more than one, and never a list;
- never a count, a stack, or a *more*;
- no state colour and no urgency copy — it is never late, never red, never
  bold;
- **absent** rather than empty when there is nothing to offer;
- refusing costs one press, has no consequence and asks nothing back;
- it changes what is *offered*, never what is *true*;
- it is chosen deterministically and explains itself in one clause.

### Buddy's Sheet

The live-chat convention, rendered in this product's own materials.

**The exception it takes, recorded rather than taken quietly.** The rule below
says: *don't open a modal for the chore interval, or for anything else that
needs neither interruption nor protected focus.* The rule carries its own
condition, and this is the one thing in the system that meets it — a coach
conversation happens when everything else on screen is noise. The chore picker
was refused a modal because choosing an interval needs neither. That reasoning
is untouched and still governs everything else.

The chore picker is not a disclosure inside the card any more either: it is a
turn, which is neither a modal nor a thing that replaces the row. The question
arrives underneath what you asked it about, and what you asked it about is still
on the screen above it.

- **The button is the acorn, and it lives in the lid.** See The lid.
- **It is a real page** (`/buddy`) that the script upgrades into a `<dialog>`,
  so Escape closes it and focus stays inside without either file implementing
  either.
- **A screen when you go to it, an object when it comes to you.** Same markup,
  two treatments, and which one you get is the only thing that differs. Inside
  the dialog it is a card: cream stock, a 3px outline, its own little lid at
  17px, dark ink. Reached as a page it has no card at all — the conversation
  stands directly on the field the way `/enough` and every other screen's
  content does, and its name is that screen's title, in the wordmark's face at
  21px on paper, under The One Title Rule.

  For its whole life it was the object in both, which meant a 640px bordered
  card with a close cross floating in the top sixth of an empty field. A card
  is a note in this system, and a card in a room with nothing else in it reads
  as an overlay that failed to overlay. That page is where a notification tap
  lands, so as push becomes the way in it stops being a fallback and becomes a
  front door.

  **The two inks are tokens, set once per treatment.** `--sheet-ink` and
  `--sheet-quiet` mean dark-on-cream inside the dialog and paper-and-cream on
  the field, and the eight children that say "quiet" read them rather than
  naming a colour. The alternative was eight duplicated rules, which is eight
  chances for one of them to be left dark on a purple ground.
- **Bottom sheet on a phone, right panel on desktop.** The page behind stays
  visible and in place: the conversation is about what is on that screen.
- **`dvh`, not `vh`, and that distinction is a bug that shipped three times.**
  `vh` is the height the page would have if the browser's chrome were hidden;
  with an address bar on screen an 88vh sheet anchored to the bottom reaches
  higher than the window shows, and the first thing over the top edge is the
  lid, which is where the close button lives. It looked pinned in a headless
  window at exactly 390×844 because there is no browser chrome there at all.
- **Its own lid is sticky, and has its own background.** The sheet is what
  scrolls, and the lid is its first child, so reading an answer of any length
  carried the close button off the top of it. What was left was Escape, which
  wants a keyboard. It only happens once there is a conversation to scroll,
  which is why every test of it passed.
- **Its own little lid:** the mark at 36px, **the name Buddy** in Inter where
  the real lid says Squirrel, and a 44px cross. Same mark, different name.
- **On screen:** what the picker would hand you, painted on open and carrying
  no controls at all. An orange rule down its left rather than a second card:
  it is context for the conversation, not a second thing being offered.
- **Two voices, and only position tells them apart.** Yours on the tail cream,
  right-aligned, casual axis. Squirrel's on paper, left-aligned, with the
  sticker offset. No names, no avatars, and no timestamps: a time beside a
  sentence is a fact about how long you have been stuck.
- **The four chips are the Offer's own**, unchanged and equal in weight.
- **What it has cost this month**, in the lid, at the meta role's smallest size
  in headphone brown, with tabular figures so it does not shift as it climbs.
  This is the one number in the system that accrues and is still allowed on a
  screen — see `PRODUCT.md` for why. It is the quietest thing in the sheet on
  purpose: it is there to be findable, not to be watched.

**A shut dialog has to be told to go away.** `dialog.coachsheet { display: flex }`
beats the browser's own `dialog:not([open]) { display: none }`, so for three
releases the sheet closed correctly — the press landed, `close()` ran, `[open]`
came off, the backdrop went — and stayed exactly where it was. From the outside
that is a close button that does nothing.

### Cards / Containers

- **Corner:** 14px.
- **Background:** `#fdecd4` stock; the three cards behind take `#f6d5ab`, and so
  does an archived card — done, and quieter for it, because these sit underneath
  the ones that matter now. **Never crossed out**, which would be the product
  marking your own words as spent.
- **Title bar:** cap purple, paper type, 3px bottom outline, `11px 11px 0 0`
  radius so it sits inside the card's corner, with the acorn badge at its left
  and `LATER` at its right.
- **Shadow:** see Elevation.
- **Padding:** 30px/32px for the note, 22px/32px for the action tray.

**The note block is a column, and has to be** now that a note can carry a
photograph. As a row the photograph became a sibling of the words and landed
beside them — about seventy pixels wide on a phone, which is not a picture of a
letter, it is a swatch.

**Reworded, the note's own words leave the view.** The field holds them, and two
copies of the same sentence on one card is a question about which one is real.

### Inputs / Fields

Three, and they share one grammar: paper, 3px outline, an orange caret, a
placeholder in `#7e6857`, and focus that *adds* a ring rather than replacing
what is there.

**The placeholder is text and is measured as text.** It was `#8a7361` — 4.32:1
on paper, 3.85:1 on the card stock the search field is made of — and it was
written out by hand at four sites rather than named once, which is how the
search field came to be the only one of the three using a different colour
altogether. One token now, and it clears 5:1 on paper and 4.5:1 on stock.

- **The slot** and the new-task line are inset. See The Slot.
- **The new-chore name** is a full pill with a 3px offset, inside the dashed
  disclosure.
- **The search field** lives in the lid's panel and takes the casual axis at
  15px — what you type is your words, so it is set in your voice. It should
  step to at least 16px on a phone and does not; see the note under the
  Step-Up Rule.

### A fixed point, and what is on it

Two screens, added 24 August 2026, and they cost a rule: `PRODUCT.md` said
there is no list screen, *"because a browsable set of your appointments is a
calendar and a calendar is a thing you are behind on."* The owner overturned it
deliberately. What the rule protected is kept rather than argued away — the list
holds **only what is still ahead**, so there is nothing in it to be behind on.

**`/at` — what is coming.** Upcoming fixed points, soonest first, each one a
row in the Door's grammar at one line high, the way a chore in search results
already borrows it. Nothing past, nothing done, and never a count. Empty is the
empty-state drawing with a line in the Voice role and **no second heading** —
the screen already has its title, and the four true empty states carry their own
`h1` because they are the whole of their page, where this is not.

**`/at/{id}` — one fixed point.** The label is the screen's title. Then the one
sentence a fixed point says, which is the core's `LeaveWords` and is shared with
chat and with the notification so the three cannot drift about when to leave.

- **The take-line** is what to bring, on its own line rather than as a clause
  after a middot. The label is the Meta role in headphone brown; the words are
  his, so they take the casual axis at 15px. It is the thing you are standing in
  the hall without, and that earns an element rather than a tail.
- **`LEAVING` only inside the window.** Outside it the card carries no control
  at all, because the appointment is not yet something you can act on and a
  button that closes a thing three days early is a button pressed by accident.
- **The slot**, exactly as home has it, so anything typed becomes an ordinary
  note in the pile that happens to point at this appointment. There is no
  picker: a picker needs a browsable list of appointments to choose from, and
  here the appointment is the page you are already on.
- **The notes pointing at it** are ordinary result cards, each with *back in the
  pile* — which is the whole reversal, because the pointer was the whole of the
  change.

**Where a notification lands.** Pressing a leave-by warning goes to that fixed
point's own screen. It went to the front door for the product's whole life, on
the argument that *a link to something that has since been done is worse than a
page that says what is true now* — and that reasoning is kept rather than
dropped, because a fixed point inside its leave-by window is the one thing here
that cannot be stale: the notification and the window are the same fact.

Two carve-outs survive the change unaltered. A window already on the
destination is left alone, because navigating would reload it and throw away
whatever is half-typed in the slot. And a push carrying no destination still
opens the front door, which is what every notification did before.

*(This rule lived only in `sw.js`'s own comment until now. It is written here
because it is a design decision, and a decision recorded in one file's comment
is a decision the next person changes without knowing they are changing it.)*

A note pointing at an appointment leaves the pile and comes back when the
appointment is over. That is a read rule rather than a state: nothing accrues,
nothing runs on a schedule, and there is no fifth disposition for any screen to
learn.

### The Stamp

The signature component. When a note is actioned the card does not leave
immediately: a stamp lands on it — the state word in that state's ink, on paper,
in a 3px border of the state's fill, with a matching offset, rotated `-7deg`,
scaling from 1.7 with no rebound. The card then holds still for 1150ms with an
undo button on the card itself before flipping away over 470ms.

The hold is not decoration. Undo has to live on the thing it undoes, and if the
card leaves at once there is nowhere for it to be. Once the write has landed and
the page has been redrawn the card is gone, so the same offer moves to the top
of the field — which is also what the scriptless path has instead.

### Page Tabs

Search results carry a small coloured tab protruding from the card's right edge
— 26×34px, the state's fill, 3px outline, rounded on the outer corners only.
Straight off the notebook the mascot is holding. It replaces the coloured left
border that a card UI would reach for by default.

**A task has no page tab**, on the tasks screen or anywhere else: a tab says
what a note ended up *as*, and a task has no outcome until it has one. This is
also why the state word differs by kind — a row in the pile says `IN THE PILE`,
and the same row once it has been decided on says `DECIDED`. One word said the
truth about half the rows and a lie about the other half.

### Why a result matched

An orange underline at 3px with `text-decoration-skip-ink: none`, never a fill.
A filled chip on cream reads as a control, and every fill colour in this world
already means a state — a match is not a state.

### A Weekday Is Where You Are

A chore that comes back on a day reads that day where the person is, not where
the database is. Fixed 27 August 2026, the first Thursday after the rhythm
shipped.

**`extract(dow)` and `::date` both follow the session's timezone, which is UTC.**
So between midnight and 02:00 in Amsterdam the rule was asking about yesterday:
the bins were not due at 00:30 on a Thursday, and were due at 00:30 on a Friday.
The bins are the whole motivating example.

Issue #148 again, one feature further on. That fix threaded the location into
everything that *parses* a time and everything that *reads a row back*; this is
the third place — a comparison made inside SQL, where the zone is the session's
and nothing in Go can see it.

**The store hands the query its zone.** Every date in the weekday rule is read
`at time zone` the person's, falling back to the session's own when nowhere is
configured — which is exactly what the rule did before, so a deployment with no
location set is unchanged.

### The Box Opens a Place

The one box this product has can show you a place too. Added 26 August 2026,
the same evening the menu route learnt it, after *can you show me the chores?*
typed into the dock got *I can't see your chores from here.*

**Buddy opening a place shipped on the wrong surface first.** The tool went to
the route behind *ask Buddy* in the menu, which almost nobody presses. The dock
— the box you type into without thinking — reads what you typed through a
separate, deliberately cheap call that carried one tool and no facts at all. It
genuinely could not see the chores, and said so.

**One field on the call that already runs**, rather than the full toolset. The
reading path is a classifier — *thought or question, and what to say* — and it
stays one. Adding `open` alongside `say` and `keep` costs about 140 tokens and
one hundredth of a penny per question, and keeps the box a single call: what
you typed comes back in one round trip, as it always has.

Handing the dock the whole toolset was measured and rejected for now: five to
six times the cost is affordable, but it is two or three sequential calls on
the most-used input in the product, and it would mean typing *done with the
vet* completes a task. **That is a change to what the box is, not a fix**, and
it belongs in its own study.

**A request to be shown something is not a thought.** `show chores` has no
question mark and no asking opening, so the rule filed it — twice in one
minute, for somebody trying to look at their chores. The product answered a
request to look at something by writing it down. The rule now recognises a verb
of showing followed by the name of a place, and it takes **both**: `show mum the
photos` is still a thought, and so is `open the compiler`, which is why the
place has to be a whole word rather than a substring.

**One way of showing a place, reached three ways** — the menu, Buddy, and now
the box. All three draw the same turn from the same function.

### Buddy Opens a Place

Buddy can show you one of the six places, whole, as cards. Added 26 August
2026, after being asked *can you show me the tasks?* and answering that he
could not.

**He was right, and that was the bug.** Three rules met: the guard refuses any
reply containing a list, the brief is two sentences at most, and he had no tool
that opens anything. He could read your tasks and was forbidden to recite them,
so the only honest thing left to say was *no*. The menu, four inches above him,
did it in one press.

**When the places became messages, the menu learnt to open them and Buddy did
not.** This is that press, given to him.

- **He opens it; he does not offer it.** A chip would be a press on the most
  ordinary request in the product. Opening a place changes nothing, so the
  worst a wrong one costs is a scroll — and it needs no confirming, which is
  the whole reason it is not a proposal.
- **It arrives as his turn, not yours.** Pressing the menu is an utterance and
  goes into the record as one; Buddy opening a place because you asked is his
  answer. Putting *the tasks* in your mouth would be the record inventing a
  sentence you never typed.
- **Last in the turn, so the live edge is the cards.** What you asked to see is
  what you are now looking at.
- **He says something as well.** The place is drawn, not explained, and a
  screenful of cards with no sentence above it is a press that landed silently.
- **The same six the menu offers**, checked against the same vocabulary twice —
  once in the tool's enum and once as a lookup on the way in. A name that is
  not one of them is refused rather than drawn as an empty turn.
- **The screen only.** Chat has no cards; a place there would be the list the
  guard exists to refuse. The surface states what it can draw, because what a
  surface can draw is a fact about the surface.

### The First Run

What somebody who has never seen this sees. Added 26 August 2026, replacing an
empty conversation with a box in it saying *put it down here*.

**That box asks a stranger to trust the product with something before showing
them what it will do with it** — and what it asks for, the contents of
somebody's head, is the one thing they have most reason not to hand over first.

So Buddy plays the loop through once, dimmed, and then rules a line under it
and hands over. Three turns, because three is the whole loop: you say
something, it is kept and named, and later it comes back to you. A fourth would
be teaching the product rather than showing it.

**Seeing your own bubble beside Buddy's unbubbled words teaches the grammar of
the screen in one look**, which no sentence about it can do. That is the reason
it is dimmed rather than greyed: the colours are part of what is being taught.

- **It is never stored.** These are not turns, they are not written, and they
  are gone the moment there is one real turn. The first thing this product does
  must not be to write a false memory into a record whose whole value is that
  you made all of it.
- **It is inert by construction.** The verbs are spans, not forms, and there is
  no route they could reach if they were pressed. A worked example wired to
  real controls is a trap: the one thing worse than not knowing what DONE does
  is finding out on somebody else's note.
- **It carries its own classes, not the conversation's.** It looks the same —
  that is the teaching — but a picture of a card is not a card, and sharing the
  class would let a screen that must have no controls on it report one. The
  look is shared in the stylesheet, which is what a stylesheet is for.
- **It says what it is, in Buddy's voice**: *Here is what this is. None of it
  is yours.* A dimmed thing nobody has explained is a thing that looks broken.
- **It ends on a dashed rule and two words**: *now yours.* Everything above the
  line is nobody's and everything below it is theirs, and that is the one thing
  the visitor has to understand.

**It goes the moment there is anything of yours.** One turn in the record ends
it; so does anything Squirrel finds to say about your own world — an opening
line, or something to be handed. Things arrive through Campfire without a word
being said on this screen, and a person mid-sentence about their own things
does not need the loop explained.

**The example is ordinary and small on purpose.** `post the parcel back` is the
kind of thing this is for. An example built out of impressive things would be a
demonstration of the product rather than a picture of your Tuesday.

### The Readings

Six weeks by seven days, in five colours, with **the days you said nothing
drawn as dashed outlines**. Added 26 August 2026, replacing a list of the days
you had answered.

**The gaps are the honest part, and they are why this shape won.** You check in
on some days and not others, and the days you did not answer are most of what
is there. A list of answered days can only ever show answered days. A bar chart
hides the gaps more thoroughly still, by making them look like zero.

- **Six weeks, not the fortnight the command reads.** A fortnight drawn as a
  grid is two short rows and a lot of nothing; six weeks is the smallest window
  in which a pattern is visible at all, and a pattern is the only reason to
  keep these rows. It stops at six because this is still not a record of your
  year.
- **Weeks begin on Monday and the last row is the week you are in**, so today
  always has a cell and the bottom row is never full. You should be able to
  find where you are without being told.
- **A day that has not happened is drawn as nothing**, not as a gap. An empty
  Friday next week outlined like a missed day says you are already behind.
- **A day you answered twice shows what the day came to** — the last thing you
  said on it. One day is one square; the earlier answers are still in the table
  and this is not the page that reports them.
- **Nothing said at all draws no grid.** Six weeks of empty outlines is a
  picture of forty-two days you did not check in, which is the one judgement
  this page could still make by accident. The sentence stands in its place.

**No trend line, no average, no count, no number anywhere.** You read it; the
product does not. Buddy summarising it in a sentence was rejected: *mostly
calm* is a small judgement about a person, and it is still a judgement.

**The five mood colours are not the state colours.** Every fill in this world
means an outcome — done, kept, dropped — and lending green to *good* would make
one colour mean two things the first time a note and a reading shared a screen.
They are five hues of the same weight in no order, and none of them is the good
one: *low* and *frazzled* are different states wanting different answers, which
a scale from green to red cannot say.

| | good | calm | low | frazzled | wiped |
| --- | --- | --- | --- | --- | --- |
| | `#3fa08a` | `#6f9fd8` | `#8a5b8f` | `#d94f2b` | `#7a7f8a` |

A colour that means something has to say what it means, so the key sits under
the grid and every cell carries its day and its word for a screen reader.

### Results

A result is not a card. It has no border, no shadow, no verbs — a full-bleed
row with the name in the reading axis and, in the small caps, which of the
seven states it is in. Results are separated by a hairline and nothing else,
which is what lets six of them sit in one turn without the turn becoming a
wall of card stock.

**The reason it looks like this is that a result is not a decision.** You went
looking for a thing; you have not yet said anything about it. Every card in
this world is a thing you are being asked about, and dressing a search result
as one asks a question nobody posed.

**Tapping a result opens the ordinary card, in the next turn.** The quiet row
and the deciding card are two states of one note, and the tap is the step
between them. Nothing about search gets its own verbs: what opens is the same
card with the same DONE / KEEP / DROP the pile draws, built from the note's
real state. One at a time, which is the rule the whole conversation runs on —
the live edge holds one thing.

### The Conversation

**Buddy's words are not in a bubble. Yours are.**

Two kinds of thing wore one costume until 26 August 2026: a sentence and an
object were both cream rectangles with a 3px outline, so a conversation read as
one column of beige rather than as an exchange between two parties.

- **Buddy:** paper on the field, casual axis at `wght` 500, with a 1px outline
  text-shadow and the 40px face in the gutter. No stock, no border, no shadow.
- **You:** the cream bubble, unchanged, right-aligned.

What the bubble now means is not "somebody said this" but **"this is a thing
rather than a sentence"**. Every remaining rectangle on the screen is a note, an
appointment, a chore, or something you said. The same conversation goes from
seven outlined boxes to three.

Paper and not cream: on the purple field cream measures 4.8:1 and paper 8.9:1,
and this text has no stock behind it to sit on.

**What it costs:** the chat convention. Buddy's words no longer read as
"received", which is a real loss for somebody arriving for the first time. The
first-run worked example is what teaches the grammar in one look.

### The Six Bodies

A note, a task, a chore, an appointment, a photograph and something set aside
were one cream rectangle with different words in it until 26 August 2026. The
kind was knowable only by reading, which is the one thing a glance does not do.

Nothing new is invented. The notch, the page tab, the dashed edge and the hard
shadow are devices this system already owns; they are applied **per kind**
instead of uniformly.

| Kind | Body |
| --- | --- |
| note | plain stock, rotated `-.35deg` — a piece of paper somebody put down, not a row in a table |
| task | the notebook's page tab, in the kept amber the tab already uses |
| chore | its rhythm carried on the card, and `article.chore` as before |
| appointment | a **ticket**: notched on the left, time set large in Inter, what to bring below a perforation |
| photograph | a **polaroid** — paper border, the picture, the words underneath as a caption |
| set aside | recessed and dashed rather than raised |

**The appointment justifies the work on its own.** It is the only thing in this
product you can be *late* for, and it looked exactly like a note about a rattle.
It is also the one body where the meta line outranks the name: the time and the
leave-by are what you came for, and the label is the caption.

**The photograph needs no kind.** A card that carries one *is* one — which is
already true of the content, the picture being the note and the words beside it
the caption.

**Set aside bends "cream card stock, never white" by not being stock at all.**
That is deliberate and it is the only one. A parked note that looked like every
other object would be a parked note you keep trying to act on.

### The Menu

One 44px control in the lid, and the whole of this product's navigation.

The rail of four doors, the always-on `ask Buddy` / `look something up` chips
and the `stop whenever you like` line were all permanent fixtures in the
conversation, together about **45% of a phone screen before a word was said**.

- **Stock and depth:** card fill, 3px outline, 14px radius, the raised object's
  hard shadow. Positioned over the conversation rather than pushing it down — a
  menu that reflowed the screen would move the thing you were reading in order
  to show you a way somewhere else.
- **A `<details>`,** so it opens with the script off, like every other
  disclosure here.
- **Counts** on the places that earn one. Zero is no number and not a nought: a
  door reading "0" is a scoreboard, which is the rule the four doors carried and
  this inherited with them.
- **The way out is last, under a rule of its own,** in today's words. It is not
  a destination like the others; it is the end of the evening, and its place at
  the bottom says so without a louder word. Principle 3 is why it is not smaller
  or greyer — only elsewhere.

**This is a hamburger, and this product deliberately removed one on 25 August.**
Bringing it back is a decision rather than a drift: it held three things then and
holds eight now, and the argument against it — that a menu of three is emptier
than the space it costs — stops applying at eight.

### The Gate

The screen you sign in through, and the first thing anybody ever sees of this
product. Added 25 August 2026, when the application started doing OIDC itself
rather than sitting behind a forward-auth outpost that answered before any of
this was drawn.

Not called a door. **A door in this world is a section of the pile** — the
card-stock links above, with their own art and their own headings — and reusing
the word for the thing you sign in through would make every mention of either
ambiguous.

- **Composition:** `/enough`'s, exactly. The resting mascot at 300×207, a
  headline in the casual axis at `wght` 800 with the 2px outline, one sentence
  in cream, one control. Nothing new is invented for it: a screen with four
  states that invented its own treatment would be inventing four.
- **No lid.** The only page in the product without one. Every control up there
  goes somewhere a signed-out person cannot reach, so offering them offers a
  bounce straight back to here. `main` is the whole body and it centres, which
  it can because there is exactly one thing on the page.
- **The control:** the orange fill with dark ink, at 15px `wght` 800 with
  `.06em` tracking. It is this screen's make-something button and it is the
  only control, so it takes the colour that means *this makes something happen*
  rather than the outlined one that means *this is a choice among several*.

**One screen, four states, one composition.** A first arrival, a sign-out, a
refusal, and an Authentik that is not answering are the same moment arrived at
four ways. Only the sentence under the mark differs, and on a first arrival
there is no sentence at all — an arrival is not an error and must not read as
one.

| State | The sentence | The control |
| --- | --- | --- |
| cold | *(nothing)* | LET ME IN |
| signed out | you are signed out | LET ME IN |
| refused | that account cannot use Squirrel | LET ME IN |
| unreachable | I cannot reach the way in just now | TRY AGAIN |

**The refusal names no group.** Which group an account lacks is a fact about
the Authentik rather than about the person reading it, and printing it tells a
stranger exactly what to go and ask for.

## Do's and Don'ts

### Do:

- **Do** put the 3px `#1c110b` outline on every object. 2px is the one step
  down, for a small control inside a card; there is no third step and no
  hairline.
- **Do** pair the block offset with a real cast shadow on cards, and match the
  press translation to the offset exactly.
- **Do** set the owner's captured text in `CASL` 1 and every control in `CASL` 0
  — including the field he is typing it into.
- **Do** give every screen one `<h1>`, one class, in the same place.
- **Do** mark a disclosure with a chevron, a link with an underline, and a
  button with a fill — and let a disclosure in a row of buttons keep the row's
  shape while dropping its fill.
- **Do** use a state's ink for type and its fill for shapes — and its *lifted*
  fill when a control wears that state at rest, with the outline ink on top.
- **Do** say *that* there is more — the stack, a line of copy — and let the
  scroll carry the rest.
- **Do** let the lid shrink on a phone while everything else grows. It is
  chrome, and the card is what the screen is for.
- **Do** theme the browser's own surfaces: selection, caret, scrollbar track and
  thumb, focus ring, underline offset.
- **Do** randomise the stack's rotation on every card change. Habituation is a
  documented risk for this user, and a surface identical every time stops being
  seen inside a week.
- **Do** measure a colour against the type that will sit on it, and change the
  number until it passes. Two lifted fills and the field's own light were all
  set by eye first and all three were wrong.

### Don't:

- **Don't** emit a count, a total, a badge, a percentage or a page number, in
  any form, on any surface — including a title, which is where one would look
  most reasonable. This is the product's single hardest rule and the one most
  likely to be broken by accident. *One exception exists — the coach's monthly
  cost, in Buddy's own lid — recorded under Buddy's Sheet with its reasoning,
  the same way the Door Art exception is.*
- **Don't** use white or grey for any surface or any secondary type. Tint from
  the tail cream on purple, from the headphone brown on cream.
- **Don't** draw a count either. The rule above does not care whether the total
  is a numeral: a checklist, a progress bar, a row of ticks and a stack of cards
  are all counts in a costume. The one drawn exception in the system is recorded
  under Door Art, and it is an exception rather than a precedent.
- **Don't** put orange on a disposal action. Orange means something was made.
- **Don't** dress a chore at rest in orange either, or in the page-tab grammar
  that says what a note ended up as. A chore has a rhythm, not an outcome.
- **Don't** underline anything that is not a link to another screen.
- **Don't** give a control with no server consequence the loudest fill on its
  card. A disclosure that outshouts the buttons beside it is telling you the
  wrong thing is the point of the screen.
- **Don't** print how long it has been since a chore was done. Say it in words,
  softly, or say nothing.
- **Don't** put a gradient on anything that can be pressed.
- **Don't** open a modal for the chore interval, or for anything else that needs
  neither interruption nor protected focus. The picker replaces the action row
  in place, on the card. *The one exception in the system is recorded under
  Buddy's Sheet, and it is admitted because the rule's own condition admits it.*
- **Don't** take a letter key for navigation. Letters are actions; `k` is keep.
  Movement is space and the arrow keys.
- **Don't** let a control leave the surface it belongs to — undo lives on the
  card it undoes, not in a corner of the screen, and nothing floats over the
  content on a fixed position.
- **Don't** leave a screen reachable from one place, and never leave one
  reachable only from an empty state.
- **Don't** celebrate an empty pile. It is a normal ending, not an achievement,
  and a reward here is a counter wearing a different hat.
- **Don't** render an offer with nothing in it. The region is absent when there
  is nothing to hand over — an empty box, or a sentence encouraging you to find
  something, is the product deciding you ought to be busy.
- **Don't** give the offer more than one thing, at any width, in any state.
- **Don't** put a second permission prompt anywhere. One notification exists in
  this product and it is about having to leave.
- **Don't** write an author `display` value for a state that is meant to be
  gone. `[hidden]` is enforced globally with `!important` for exactly this
  reason, and the rule the three bugs behind it taught is the smaller one:
  **when something is meant to disappear, assert that it is invisible, not that
  it is unmarked.**
