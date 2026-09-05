---
name: Squirrel
description: A controller's strip board — every thought a printed strip in a coloured holder, and the one that matters pulled proud of its rack.
colors:
  purple: "#6c4da9"
  purple-deep: "#58388a"
  purple-dark: "#3b2560"
  # The chassis. Every fixed furniture surface — the ops bar, a bay sign — is
  # this, and nothing else in the product is.
  purple-bar: "#472e70"
  violet: "#5e23b1"
  violet-ink: "#56209f"
  orange: "#e66d0d"
  orange-lit: "#ff8a2b"
  outline: "#1c110b"
  tail-cream: "#fed6a7"
  # Strip stock. Every strip is printed on this, whatever bay it belongs to.
  card: "#fdecd4"
  # The same stock, exposed: a strip that has been struck, and the tray.
  card-deep: "#f6d5ab"
  # The pulled strip only. One surface in the product is brighter than stock,
  # and it is the one thing Squirrel is asking about.
  paper: "#fffbf3"
  paper-deep: "#f7e9d4"
  headphone-brown: "#58413d"
  # Pencil. Buddy's marginal notes and a strip that has been struck through.
  ink-soft: "#6a4f3e"
  placeholder-ink: "#7e6857"
  field-lift: "rgba(150, 110, 220, .35)"
  shadow-cast: "rgba(28, 17, 11, .55)"
  shadow-cast-soft: "rgba(0, 0, 0, .7)"
  # The four holders. Room identity, and never state — see The Holder Rule.
  holder-notes: "#6c4da9"
  holder-chores: "#e66d0d"
  holder-tasks: "#ffb300"
  holder-agenda: "#8a6a55"
  state-done: "#529414"
  state-done-ink: "#37640d"
  state-kept: "#ffb300"
  state-kept-ink: "#8a5c00"
  state-dropped: "#8a6a55"
  state-dropped-ink: "#6a4f3e"
  state-chore-ink: "#b0530a"
  # Lifted fills: a state colour raised until the outline reads on it. Stamps
  # only. See The Lifted Fill Rule.
  state-done-lifted: "#71a73e"
  state-done-lifted-hover: "#89b65f"
  state-dropped-lifted: "#9f8574"
  state-dropped-lifted-hover: "#af9a8b"
  mood-good: "#3fa08a"
  mood-calm: "#6f9fd8"
  mood-low: "#8a5b8f"
  mood-frazzled: "#d94f2b"
  mood-wiped: "#7a7f8a"
typography:
  # The wordmark, and only the wordmark.
  display:
    fontFamily: "Inter, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "19px"
    fontWeight: 900
    letterSpacing: "-0.028em"
  # A printed figure: the clock, a departure time, a countdown. Inter black
  # with tabular figures, and the one place this face appears in content.
  # See The Printed Figure Rule.
  figure:
    fontFamily: "Inter, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "19px"
    fontWeight: 900
    letterSpacing: "-0.02em"
    fontVariantNumeric: "tabular-nums"
  figure-large:
    fontFamily: "Inter, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "26px"
    fontWeight: 900
    letterSpacing: "-0.02em"
    fontVariantNumeric: "tabular-nums"
  # What the pulled strip says. The largest thing anybody reads here.
  pulled:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "27px"
    lineHeight: "1.16"
    letterSpacing: "-0.01em"
    fontVariation: "'MONO' 0, 'CASL' 1, 'wght' 620"
  # A strip's own words. His sentence, at the size a dense rack can hold.
  strip:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "14.5px"
    lineHeight: "1.3"
    fontVariation: "'MONO' 0, 'CASL' 1, 'wght' 500"
  # The rule that pulled a strip, and Buddy's pencil in the margin. Same size,
  # same ink, because both are annotations on somebody else's words.
  pencil:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "13.5px"
    lineHeight: "1.4"
    fontVariation: "'MONO' 0, 'CASL' 1, 'wght' 470"
  # An enamel bay sign, and the tray's own label.
  sign:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "12px"
    letterSpacing: "0.1em"
    textTransform: "uppercase"
    fontVariation: "'MONO' 0, 'CASL' 0, 'wght' 800"
  # A stamp's face.
  stamp:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "12.5px"
    letterSpacing: "0.09em"
    textTransform: "uppercase"
    fontVariation: "'MONO' 0, 'CASL' 0, 'wght' 800"
  # The mark at a strip's right edge: when it arrived, when it is due.
  mark:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "11px"
    letterSpacing: "0.08em"
    textTransform: "uppercase"
    fontVariation: "'MONO' 0, 'CASL' 0, 'wght' 750"
    fontVariantNumeric: "tabular-nums"
  # The key letter inside a stamp.
  keycap:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "11px"
    fontVariation: "'MONO' 0, 'CASL' 0, 'wght' 800"
  # What you type: onto a blank strip, into the find field.
  written:
    fontFamily: "Recursive, 'Helvetica Neue', Helvetica, Arial, sans-serif"
    fontSize: "14.5px"
    lineHeight: "1.35"
    fontVariation: "'MONO' 0, 'CASL' 1, 'wght' 500"
  # Below 620px, the roles that step up. See The Step-Up Rule.
  strip-phone:
    fontSize: "15.5px"
  pulled-phone:
    fontSize: "21px"
  pencil-phone:
    fontSize: "13.5px"
rounded:
  # Everything printed or stamped. A strip is a slip of paper, not a sticker.
  strip: "3px"
  stamp: "3px"
  # The key box inside a stamp, and the smallest marks.
  mark: "2px"
  # The holder's grip, and a face tile in the tray.
  grip: "3px"
spacing:
  # The outline weight. Two, not the old three: a strip is printed, not drawn.
  line: "2px"
  # The holder's width on a strip, and on the pulled strip.
  holder: "11px"
  holder-pulled: "14px"
  # Between strips in a rack. Tight — a rack is full or it is not.
  rack-gap: "5px"
  # The rack channel's own inset.
  channel-inset: "7px"
  # Between racks.
  bay-gap: "14px"
  # The board's outer margin. There is no other page padding.
  board: "18px"
components:
  strip:
    backgroundColor: "{colors.card}"
    textColor: "{colors.outline}"
    typography: "{typography.strip}"
    rounded: "{rounded.strip}"
    border: "2px solid {colors.outline}"
    shadow: "0 2px 0 0 {colors.shadow-cast}"
  strip-cocked:
    transform: "translateY(-3px)"
    shadow: "0 5px 0 0 {colors.shadow-cast}, 0 14px 20px -14px {colors.shadow-cast-soft}"
  strip-struck:
    backgroundColor: "{colors.card-deep}"
    textColor: "{colors.ink-soft}"
    textDecoration: "line-through"
  strip-blank:
    backgroundColor: "transparent"
    border: "2px dashed rgba(254, 214, 167, .45)"
    textColor: "rgba(254, 214, 167, .8)"
  pulled:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.outline}"
    typography: "{typography.pulled}"
    rounded: "{rounded.strip}"
    border: "2px solid {colors.outline}"
    shadow: "0 6px 0 0 {colors.shadow-cast}, 0 20px 30px -18px {colors.shadow-cast-soft}"
    outline: "3px solid {colors.orange-lit}"
  stamp:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.outline}"
    typography: "{typography.stamp}"
    rounded: "{rounded.stamp}"
    border: "2px solid {colors.outline}"
    shadow: "0 3px 0 0 {colors.shadow-cast}"
    padding: "8px 11px"
  stamp-go:
    backgroundColor: "{colors.orange}"
    textColor: "{colors.outline}"
  stamp-did:
    backgroundColor: "{colors.state-done-lifted}"
    textColor: "{colors.outline}"
  stamp-no:
    backgroundColor: "{colors.card-deep}"
    textColor: "{colors.outline}"
  baysign:
    backgroundColor: "{colors.purple-dark}"
    textColor: "{colors.tail-cream}"
    typography: "{typography.sign}"
    rounded: "{rounded.strip}"
    border: "2px solid {colors.outline}"
    padding: "6px 11px"
  channel:
    backgroundColor: "rgba(28, 17, 11, .2)"
    border: "2px solid rgba(254, 214, 167, .2)"
    rounded: "{rounded.strip}"
    padding: "{spacing.channel-inset}"
  ops:
    backgroundColor: "{colors.purple-dark}"
    borderBottom: "2px solid {colors.outline}"
    padding: "9px 18px"
  tray:
    backgroundColor: "rgba(28, 17, 11, .34)"
    borderTop: "2px solid {colors.outline}"
    padding: "10px 18px 13px"
---

# Design System: Squirrel

## Overview

**North star: the strip board.**

An air-traffic controller works a rack of printed paper strips, one strip per
flight, each in a coloured holder. A strip that needs something gets pulled half
out of the rack so it stands proud of the others. When the flight leaves the
airspace the strip comes out and is gone. The board carries no totals, no
history, and no record of what the controller did not get to.

That is this product's oldest rule wearing somebody else's uniform, so the whole
app is built as one of those boards. Every note, chore, task and appointment is
a strip. The four rooms are four bays. Squirrel choosing one thing is not a
highlight or a badge, it is a strip pulled out of its rack — a physical act you
can read from across the room. Answering something strikes it and drops it into
the day's tray, and the tray is empty again tomorrow.

**The ending, decided 1 September 2026: the conversation is not retired. It
becomes Buddy's room.** The plan of record was that it would go once the board
could do everything; the board can, and what that made obvious is that the last
thing it cannot do is the one thing a conversation is actually for. So the
thread stays, at `/r/everything`, reached by the last link in the ops bar —
*talk to Buddy* — and it stops being where the pile is worked.

That decision is what closes this document's own gap. Two worlds live here now,
and both are described: the board, which is everything above, and **Buddy's
room**, which keeps the shapes the conversation was drawn in.

**Status.** This document is ahead of the running code
and is landing behind it. **As of 1 September 2026 the board is the front door**:
`/` draws it, and the conversation kept its own address at `/r/everything`, with
every press made inside it coming back there. The board is also still at
`/board`, and drawing the four bays, the pulled strip, the running timer and the
ledge from the store; a strip can be answered on it, with the strike, the
1150ms hold, the travel and the tray; and every rack can be written into,
the notes through the spool, a chore with one of four rhythm stamps, an
appointment as the sentence chat already parses.

The board also searches, keeps a photograph and opens the strip that carries
one, asks Buddy on a press and marks what he wrote, makes a chore out of a note,
and closes an appointment. What is left in the conversation, and left there on
purpose, is Buddy's longer conversation: that is what the link in the ops bar is
for, and a conversation belongs where conversations are. Nothing here is described from intention: what the route renders
is what this document says, and the parts it does not render yet are named in
the surface brief.

The radius vocabulary is checked against both stylesheets again: the board's
values are in the frontmatter above, and the conversation's are the table under
*Buddy's room*. The exemption this document carried while the board was being
built is gone. Everything shipped through v0.55.1 is the previous
world — the conversation, where every object was a turn in a thread, drawn on
cream card stock with a 3px outline and a 14px corner. That world's document is
in git history (`git show HEAD~1:DESIGN.md`) and it stays the accurate
description of production until this one is built.

**What survived the redesign, because the owner pinned it:** the two faces,
Recursive and Inter, and the whole palette. Purple and orange are the product's
own colours and any surface that reads as neutral or white has failed to be
Squirrel. That is unchanged and non-negotiable.

**What did not survive:** the card, the chip, the bubble, the dock, the lid and
its brim, the sheet, and the 3px sticker outline that drew all of them. The
outline language stays; the sticker does not.

**Key characteristics**

- One unit. A strip is the only content object in the product, at every size.
- One grid. Every band on the board — working row, racks, tray — aligns to the
  same row rhythm, including the bands that hold sentences.
- State is a mark, never a fill. A strip is struck and moved; it is never
  repainted green.
- Colour on a left edge is a holder, never an accent.
- Exactly one strip is pulled, ever.
- Nothing is drawn as speech. Buddy writes in the margin, in pencil.

## How this document stays true

The rules below are named so they can be cited in review and broken on purpose
rather than by accident. When a rule is broken, the entry says who broke it and
what was traded — this document has never been a style guide, it is the record
of decisions.

Values here are the design of record. Where the built code disagrees, the code
is wrong or the rule changed; either way one of the two gets edited the same
day. A value that appears in code and in no rule here is drift.

## Colors

The palette is the mark's own and is not up for discussion. What the redesign
changed is which surface each colour is allowed on.

### Primary

| Token | Value | Where |
|---|---|---|
| `purple` | `#6c4da9` | The notes holder. The mark's cap. |
| `purple-deep` | `#58388a` | The board's ground, under the light. |
| `purple-dark` | `#3b2560` | Furniture: the ops bar, a bay sign. |
| `purple-bar` | `#472e70` | The chassis behind the ground. |

### Secondary

| Token | Value | Where |
|---|---|---|
| `orange` | `#e66d0d` | The chores holder, and every stamp that makes something happen. |
| `orange-lit` | `#ff8a2b` | Focus, and the ring on the pulled strip. |
| `tail-cream` | `#fed6a7` | Type on the chassis. Never a fill on stock. |

### Tertiary

| Token | Value | Where |
|---|---|---|
| `holder-tasks` | `#ffb300` | The tasks holder. |
| `holder-agenda` | `#8a6a55` | The agenda holder. |
| `violet` | `#5e23b1` | Focus, on stock only. See The Focus Ring Rule. |
| `state-done` | `#529414` | The tray's done mark. Green appears nowhere else. |

### Neutral

| Token | Value | Where |
|---|---|---|
| `card` | `#fdecd4` | Strip stock. |
| `card-deep` | `#f6d5ab` | A struck strip; a stamp that does nothing to the world. |
| `paper` | `#fffbf3` | The pulled strip, and stamps. |
| `outline` | `#1c110b` | Every border and every dark ink. |
| `ink-soft` | `#6a4f3e` | Pencil: annotations, and struck words. |
| `placeholder-ink` | `#7e6857` | What a blank strip says before you type. |

### Named Rules

**The Holder Rule.** The colour strip down a strip's left edge is a *holder* —
the plastic sleeve the slip sits in — and it must read as one: eleven pixels
wide, separated from the stock by the same 2px outline as everything else, and
carrying a grip mark. It says which bay the strip belongs to and nothing else.
It is never a severity, never a state, never a category accent. The moment it
reads as a coloured border on a card, the mechanism has been lost and the strip
is back to being a list item with decoration.

**The Four Holders.** `notes` purple, `chores` orange, `tasks` amber,
`agenda` brown. Amber and brown are also two of the four state colours, and that
is safe **only** because of The State Is A Mark rule: since no state ever fills a
strip, a colour can mean a room on the edge and a state in the tray without
either being ambiguous. If a state ever becomes a fill, this collision is the
first thing that breaks, and the answer is to change the holders rather than the
rule.

**The State Is A Mark Rule.** A state is drawn by striking the words through in
pencil and by the strip's position — in the rack, or in the tray. Never by
filling the strip with a state colour, never by a coloured dot on the row, never
by a badge. A board where twelve strips are tinted twelve ways is a chart of how
you are doing, which is the reproach this product exists not to make.

**Green Is Only Done.** `state-done` appears on the did-it stamp and on a mark
in the tray. It is never a holder, never furniture, never an accent. Green in
this product means one thing.

**The Orange Ink Rule.** Ink on orange is the outline, never white or paper.
White on `#e66d0d` measures 3.1:1; the mark's own near-black measures 5.78:1.
Inherited from the previous world unchanged, because it is a contrast fact
rather than a style.

**The Lifted Fill Rule.** A state colour used as a stamp fill is raised toward
white until the outline reads on it — `state-done` becomes
`state-done-lifted`. The unlifted value is for marks and ink, not for fills
under type.

**The Focus Ring Rule.** `orange-lit` on the field and on anything sitting on
the purple ground; `violet` on stock, where orange-lit measures 2.03:1 against
the 3:1 that WCAG 1.4.11 asks of an indicator. Every surface that takes `card`
or `paper` is in the violet list, and the way this goes wrong is by omission.

**Never A White Surface.** Card stock is the tail's cream lightened. There is no
white in the interface. A surface that reads as neutral has failed to be
Squirrel — a brand commitment from PRODUCT.md, not a preference.

**The Field Exception.** The board's ground keeps the previous world's lit
radial at `.35` alpha and its 22px dot grid, unchanged, including the reasoning:
`.5` put cream at 4.19:1 on the lit centre, and `.35` puts it at 4.8:1. The
light's horizontal position is the day's, between 8% and 26%. This is the one
gradient the product allows and nothing pressable may take one.

## Typography

Two faces, both already in the product, both self-hosted.

**Recursive** carries every word a person wrote or would say. `CASL 1` on his
own words and on Buddy's; `CASL 0` on the machine's furniture — signs, stamps,
marks. `MONO` is pinned off everywhere, deliberately: this is a warm tool and
not a terminal, and the ASCII direction was declined over exactly this.

**Inter Black** is the wordmark, and — new in this world — the printed figure.

### Hierarchy

| Role | Size / face | Job |
|---|---|---|
| `display` | Inter 900, 19px | The wordmark in the ops bar. Nowhere else. |
| `figure-large` | Inter 900, 26px | A countdown on the running panel. |
| `figure` | Inter 900, 19px | The clock; an appointment's time on its strip. |
| `pulled` | Recursive casual, 27px | What the pulled strip says. |
| `strip` | Recursive casual, 14.5px | A strip's words. |
| `pencil` | Recursive casual, 13.5px | The rule that pulled a strip; Buddy's margin. |
| `sign` | Recursive precise, 12px caps | A bay sign, the tray's label. |
| `stamp` | Recursive precise, 12.5px caps | A stamp's face. |
| `mark` | Recursive precise, 11px caps | A strip's right-edge mark. |
| `written` | Recursive casual, 14.5px | What you type. |
| `glyph` | Recursive precise, 17px | The `+` that keeps a strip, and the notification count. A single character sized to a control, never a word. |
| `label` | Recursive precise, 9.5px caps | A word under a picture: the five faces in the tray. |
| `label-tight` | Recursive precise, 9px caps | The same word below 620px, where the five faces sit across a phone. |

### Named Rules

**The Printed Figure Rule.** A time, a date and a countdown are set in Inter
Black with tabular figures, because on a board a number is a printed thing and
not a sentence. This is the only content use of Inter in the product; everything
else that Inter used to do belongs to the wordmark. A duration you can watch
change — the running timer — takes `figure-large`, and it is the second largest
type on the board after the pulled strip.

**Two Axes, One Face.** His words and Buddy's are casual; the furniture is
precise. A sign, a stamp or a mark set in the casual axis reads as somebody
speaking, which the furniture must never do.

**The Step-Up Rule.** Below 620px the reading roles step up — `strip` to
15.5px, `pulled` to 21px — while the furniture stays put. A phone is held
further from a tired face than a monitor is, and the strip's words are the part
that has to survive that.

**No All-Caps Sentences.** Caps are for signs, stamps and marks: a name, never
a sentence. The rule that pulled a strip is set in sentence case with a
small caps tag beside it — `PULLED` — because forty characters of uppercase is a
sentence somebody has to decode rather than read.

## Layout

The app is the viewport, exactly, and the regions inside it own their scroll.
Four bands, top to bottom, and their order never changes:

**The ops bar.** Fixed height, `purple-dark`, ruled off with the 2px outline.
The mark and wordmark at 42×32, then the clock and the day, then the find field
and settings at the right. It is furniture: nothing in it belongs to any one
thing on the board.

**The working row.** A grid of the pulled strip and, at its right, a 292px panel
for a running timer when one is running. When nothing is pulled and nothing is
running, this band closes to nothing rather than showing an empty frame.

**The racks.** Four bays across, `1.18fr 1fr 1fr 1fr` — the notes bay is wider
because its strips carry the longest sentences and four answers. Each bay is a
sign, then a blank strip, then a channel of strips. The channel is a recessed
well that runs to the bottom of the board, so an unfilled bay reads as *room in
the rack* rather than as dead space.

**The tray.** Fixed to the foot, `rgba(28,17,11,.34)` over the ground, ruled off
at the top. What left the board today, struck through, oldest first, with the
newest carrying `PUT IT BACK`. The check-in's five faces sit at its right end.

### Breakpoint: 620px

One breakpoint, as before. Below it the four racks become one, and the bay signs
become four sign-shaped tabs with one lit. The pulled strip stays above the tabs
because it belongs to no bay. The tray keeps its place at the foot and wraps to
two rows: what left the board, then the faces.

There is no second layout. The phone shows one rack of the same board, and a
strip is a strip at both sizes.

**The ops bar becomes chips on no ground.** The mark, the wordmark and the clock
all go: a monogram where the mark was, then the find field across the middle,
then Buddy's chat chip and the bell. The bar keeps no background and no rule, so
the board begins at the top of the screen — the same move the bay bar made at
the foot, and for the same reason: a band of furniture costs a strip.

Every chip is a glyph with an `aria-label`, because a control drawn as a glyph
and named nowhere is a control nobody can follow. **The first chip is your
face** when there is a picture to show and the monogram when there is not, read
from the same place the conversation reads it so the two cannot disagree.

**The ground is painted on `html`, not on the body.** A background on the root
propagates to the canvas, which is what the phone paints under the status bar
and inside the safe areas — so with the ground on `body::before` the strip above
the bar was flat while everything below it was dotted. The board carries a
`theme-color` of `purple-bar` and the mark as its favicon for the same reason:
the parts of the window this design does not draw still belong to it.

**The bar reserves the top inset and nothing on top of it.** The status bar's
own band is the margin; a second one under it is space this screen cannot spare.

The clock goes because the
phone shows one two inches above it and ours is server-rendered, so it is wrong
by however long the page has been open.

**The find field is the middle of the bar and always open.** It was drawn to
nothing behind a glyph for one afternoon; the reference app puts a control in
that position and so does this. A search that costs a press before you can read
what you typed is a search you cannot correct.

**The four bays float at the foot.** A pill clear of all three edges — 12px at
the sides and 21px of visible ground beneath it — under the 2px outline and the
board's own shadow, with four cells inside it: a drawn icon over the bay's name,
capitalised and without its article, so *the notes* on the rack's own sign is
*Notes* here.

**The pill is smoked, not glass.** `rgba(59,37,96,.86)` over a 13px backdrop
blur: the board's own purple at strength, so a strip passing beneath comes
through as a diffused shape rather than a frosted pane. It is the acetate a
strip board is covered with, which is why it keeps its outline and its shadow —
glass in the iOS sense has neither, and would be the one surface in this product
that is not a thing you could pick up. Solid where the browser has no backdrop
filter, and solid again under `prefers-reduced-transparency`.

**The board scrolls beneath it**, and the reserve lives inside the scroller
rather than on the page, so a strip travels under the pill while you move and
the last one still clears it at rest. Labels are full-strength cream for this
reason: measured against a cream strip diffused directly under one, they hold
7.0:1.

**The cell you are in is tinted, not filled.** Orange ink and a full-strength
icon against 72% cream and icons at 55%. It had a well behind it while the bar
was a band along the bottom, and the well was the trouble: it stopped above the
home indicator's reserve, which made the reserve read as a hole under the
buttons rather than as the ground the bar floats over. A floating bar has no
edge to leave a gap against.

Before this the bays were two rows of tabs at the top; before that, one row that
scrolled sideways with no scrollbar, which is how the fourth place became
unreachable twice in this product — once on 28 August as the rooms, once on
2 September as the bays. The bar cannot fail that way: four cells, one grid, no
scroll at any width.

**The inset is named once, and part of it is given back.** `--foot` is
`max(10px, env(safe-area-inset-bottom) - 8px)`: a floating bar does not owe the
home indicator the whole 34pt the way a fixed band does, because the indicator
is drawn over it and the cells' own padding keeps the labels clear. Reserving
all of it left the bar hovering visibly high; giving back 24 of the 34 put it
against the indicator.

**The number came from measuring, not from taste.** Both screenshots — the
reference app's bar and this one's — were scanned pixel by pixel for the row
where the bar's own colour ends and the ground begins. The reference leaves 21
CSS px of visible ground; this leaves the same. The `5px` hard shadow counts as
part of the object, which is why `--foot` reads 26 on a phone and the gap reads
21. Everything that
has to sit above the home indicator reads that one value: the bar's offset and
the page's reserve. It is a rule with a test because a headless browser reports
the inset as zero, so nothing rendered in CI can see it being claimed twice —
which is exactly how the rack came to pad for it after the bar took the foot,
putting a band of ground between them on every phone that has one.

**No bay is lit when you are not standing in one.** A shelf, a search and an
opened strip all light nothing, because a bar that says you are in the notes
while you are reading a search result is a bar you stop believing.

**The lit rack takes the screen it is on.** It stretches to the foot so its
channel has a bottom edge and the ledge sits on it. The purple below a short
rack was not room, it was the app stopping in the middle of the screen.

**A stamp is 44px tall and the keycaps go.** Every stamp's letter is drawn only
where `hover` and a fine pointer say there is a keyboard to press it with. The
keys themselves still work the moment one appears.

**A strip opens when you press it.** On a touch screen a strip is its words, its
mark and a chevron at the end of the row — 44px — and pressing anywhere on it shows its stamps and
shuts whatever was open. Five notes are in view where three were. The gate is
`(hover: none) and (pointer: coarse)`, not the width: a tablet has no hover
either, and the desktop's open-on-hover is no use to it.

This is a script, and the base layer under it is the one that already shipped:
with `board.js` gone every strip is open, exactly as in v0.56.1, so nothing is
unreachable and nothing needs a fallback drawn for it. The script adds the
chevron, carries `aria-expanded` on it, and closes the open strip on Escape.

**His room carries the same bar.** The conversation's lid holds the chips now:
your face, the field, the way to the other place and the bell. It keeps its own
frame — fixed, translucent, ruled off — because the transcript is seen to pass
under it, which the board has nothing to do.

**The middle chip is always the other place.** A chat bubble on the board, the
board in his room. One slot, one shape, and the way out of wherever you are
standing is never somewhere else.

**The chip carries its own weight and size**, rather than reading `--line` or
inheriting a control minimum: the two stylesheets set that token differently, so
a shared component that read it would be drawn at two weights. It is the same
40px circle with the same 2px edge in both places.

**Your face opens a page of its own** — *who you are*: the picture, the name,
notifications, what he knows, how you felt before, and the way out. It was a
disclosure inside the rail, on the argument that settings is state rather than a
conversation and the product had no third thing for it to be. It has one now,
and a panel that lives inside a conversation is a panel you reach by first going
somewhere you did not want to be.

**The rail and the room sheet are gone.** The sheet was the phone's control for
seven rooms; there is one room, and the bar is the navigation. What the rail
held last — the way back, the way to look something up, and who you are — is two
chips and a page. The body's grid is one column again.

**The pulled strip gives way.** Below 620px the board under the ops bar is one
scrolling deck — the pulled strip, then the rack — so what you are looking at
scrolls and the bar at the foot does not. The ledge still sits at the foot of the channel rather than after the last
strip, because the channel still stretches to fill a short rack; the prediction
that it would have no foot to sit on was wrong.

## Elevation & Depth

Three depths, and nothing floats.

| Name | Value | What it is |
|---|---|---|
| `strip` | `0 2px 0 0 rgba(28,17,11,.55)` | A slip of paper lying in a rack. |
| `cocked` | `0 5px 0 0 rgba(28,17,11,.55), 0 14px 20px -14px rgba(0,0,0,.7)` | A strip pulled part-way out. |
| `pulled` | `0 6px 0 0 rgba(28,17,11,.55), 0 20px 30px -18px rgba(0,0,0,.8)` | The one strip standing proud of the board. |

### Named Rules

**Depth Is Distance From The Rack.** The only thing a shadow may say here is how
far out of its holder a strip has been pulled. There is no hover elevation for
its own sake, no floating panel, no modal on a scrim. A thing that is not in a
rack is either pulled or in the tray, and both are positions rather than layers.

**The Hard Shadow Is A Cast, Not A Costume.** The offset-with-no-blur shadow is
inherited from the previous world and stays because the light in this one is
still a single hard source from above. It is paired with a real soft cast on the
two lifted depths; a hard shadow alone at those sizes is a sticker.

## Shapes

**Square, because it is printed.** Radii here are 3px, 2px and 999px, and the
previous world's 14px card corner is retired with the card. A flight strip is guillotined
from a sheet; the corner is the smallest radius that keeps a 2px outline from
looking chipped on a low-density screen.

**The grip.** A holder carries one mark: a 6×14 rounded slot at `rgba(28,17,11,.28)`,
centred vertically, which is where a thumb takes the strip out. On the pulled
strip it is 8×26. It is the only ornament in the system and it is functional —
without it the holder is a colour bar.

**The pill, and only where the frame is.** 999px belongs to the chrome and never
to the board's own furniture: the bay bar on a phone, the tool cluster in the
top bar, and the chips inside them. Everything that holds content — strips,
signs, stamps, racks — keeps the 3px corner. A pill on a strip would be a
capsule, and this world is guillotined paper.

**The 2px line.** Every border in the product, including the ones that used to
be 3px. Thinner than the sticker world by exactly one pixel, and that pixel is
most of why this reads as printed rather than drawn.

## Motion

**Everything moves on the rack's axes and nothing fades.**

| Name | Value | Where |
|---|---|---|
| `cock` | `120ms cubic-bezier(.2,.8,.2,1)` | A strip lifting 3px out of its rack. |
| `strike` | `160ms linear` | The line drawing itself through the words. |
| `hold` | `1150ms` | Before an answered strip travels. Not a transition — see below. |
| `travel` | `260ms cubic-bezier(.2,.8,.2,1)` | The strip crossing into the tray. |
| `push` | `180ms cubic-bezier(.2,.8,.2,1)` | The rack making room for a new strip at its head. |
| `reduced` | `0ms` | Reduced motion: the strike and the new position, no travel. |

### Named Rules

**Cocking Is The Only Selected State.** A strip that has your attention lifts
3px out of the rack; the strip being worked stays there. This replaces every
highlight, tint, badge and left-border the product might otherwise grow. The
pulled strip is the same gesture at full size, which is why they share a shadow
family.

**The 1150ms Hold Is A Requirement.** An answered strip strikes, then stays
exactly where it is for 1150ms before travelling to the tray, because the undo
has to have somewhere to live while you decide. Shortening this to feel snappy
removes the undo's home. Carried unchanged from the previous world, where it was
the card's hold.

**Nothing Fades.** A fade is what a screen does; a rack does not do it. The only
opacity transition in the system is the reduced-motion fallback, and that one
exists because motion is what it is replacing.

## Components

### The strip

The single content object. A three-column grid: holder, words, mark.

- 2px outline, 3px radius, `card` stock, `0 2px 0` cast.
- **Its stamps are behind the cock.** Where there is a pointer, a strip's
  answers are collapsed to nothing and the row opens along the rack's axis when
  the strip is cocked or focused; where there is no hover they are always on the
  strip, because a thumb has nothing to hover with and an answer you cannot
  reach is not an answer. The buttons stay in the markup either way, so the
  keyboard reaches them and a page with no stylesheet still works.
- The words column takes 8px 10px of padding and may run to two lines. Below
  the words, when the strip is cocked, its stamps.
- The mark column is the right edge: `mark` type, or `figure` when the thing has
  a time, which is the one case where the mark outranks the words in weight.
- A strip never carries an icon, a thumbnail, an avatar or a progress bar. A
  photograph on a note is shown by **opening the strip**: the words become the
  way in, a note with no words says *a photograph*, and what opens is that one
  strip drawn whole above the racks — the picture at up to 38vh, contained
  rather than cropped, with the answers it would carry in its rack. A note whose
  photograph is of words is a note that cropping loses.

The words are a paragraph of their own inside the well, with the stamps under
them, which is what lets the stamps collapse on a touch screen without taking
the words with them. On touch a chevron sits at the end of the row, past the
mark: it is the strip's accessible control, and it is drawn by the script,
because without the script there is nothing to open. Past the mark and not
before it, because a mark is as wide as its words — *every week* against *every
28 days* — and a chevron placed inside that column steps in and out down the
rack.

### The pulled strip

One per board, ever. `paper` rather than `card` — the only surface in the
product brighter than stock — with a 3px `orange-lit` ring, a 14px holder in the
colour of the bay it came from, and its stamps down the right edge in a column.

Above the words: the tag `PULLED` and, beside it in sentence case, the rule that
chose it. Below them, its three answers — `I'll do it` in the orange every
make-something stamp takes, `not today` and `I'm stuck` in the fill that does
nothing to the world.

**Being stuck asks, and then says one sentence.** Pressing `I'm stuck` replaces
the three answers with the product's own four — *too big*, *don't know how*,
*boring*, *not today* — and pressing one of those replaces them with the
ladder's fixed line. Which blocker you pressed lives in the address rather than
in the server, so a reload shows the same sentence instead of repeating a press.
`not today` reached this way is the same no as turning the offer down, and
leaves the same mark.

That sentence carries **no acorn**: the ladder's lines are fixed and are
Squirrel's own, and the acorn is what marks a sentence a model wrote. Principle
8 decides which mark a line gets, not who is speaking on that strip. The rule is printed on the strip rather than hidden behind a why,
because an offer nobody can account for is a demand.

### The holder

11px of room colour, ruled off from the stock with the 2px outline, carrying the
grip. See The Holder Rule.

### The bay sign

Enamel: `purple-dark`, 2px outline, 3px radius, cream caps, and the rack's count
at the right in tabular figures. The count is **what is in the rack**, not what
is owed and never what is late. A rack with nothing in it shows no number rather
than a nought.

### What a rack says when it is empty

One line in the middle of the channel, in that rack's own words: *nothing in the
notes*, *nothing comes back today*, *nothing in the tasks*, *nothing left
today*. Never a shared sentence, and never a drawing.

It is not the same thing as `.trouble`, and the two are exclusive by
construction: a rack that could not be read says so, and does not also report a
quiet morning. The shelf and the search have the same line — *nothing on this
shelf*, *nothing matched "…"* — because an empty bordered box says only that
something has gone wrong.

The line matters most on a phone, where one rack is the whole screen and there
is nothing beside it to compare against.

### The blank strip asks rather than guesses

A chore typed with no rhythm, and an appointment with no time in it, used to be
filed as notes — in another rack, found on the next refresh. The board asks
instead: the words come back into the field they were typed in, with the
question under them, and the rhythm chips or the day and time beside it are the
answer.

**A rhythm is any number of days, weeks or months.** The four chips are a
shortcut and not the vocabulary: the fourth chore you have comes back every
three days, and a screen that can only offer four intervals is a screen that
makes you round.

**A capture is settled before the board is drawn again.** The spool is what
makes it safe; a pass over the spool in front of the person is what makes it
visible. Without it the board you are sent back to is drawn before the drain has
run, and a note you have just written is not on it — which reads as the capture
having been lost, which is the one thing this product may never do.

### The bell, and what Squirrel told you

A push has been fire-and-forget since it shipped: the payload went to the push
service and nothing on this side remembered that it had. So the app could not
answer the question a bell implies — *what did you tell me?* — and a phone that
was off, or a notification swiped away unread, lost it for good.

One row per push, written where the fan-out happens rather than once per
subscription: two browsers on one account are two deliveries of one thing said,
and a list that showed it twice would be a list about plumbing. Nothing is
written when no push service took it, because a row there would be the app
saying it said something it did not say.

The bell wears **a dot, not a number**. A count there would be things you have
not read, which is the one shape this product refuses — the bays' numbers count
things that exist, which is a different claim. The dot says only that there is a
record to look at.

The list itself is strips in the agenda's holder: the title as the words, what
it said under it, and the time as the mark. It is a record and not a pile, so
nothing on it can be answered.

### The bay bar

The phone's navigation, and the only place a bay is named there. A drawn icon at
28px over the bay's own name — *the notes*, not *notes* — in 10.5px precise
type. The cell you are in takes `paper` ink over a `rgba(254,214,167,.13)` well;
the other three sit at 78% cream with their icons at 55%.

**The icons are illustrated and full-colour**, which is a deliberate exception
to a world drawn in flat shapes and 2px outlines. They belong to the mark's
register rather than the board's: the squirrel in the ops bar is drawn the same
way, and four line glyphs at 28px would have been four grey rectangles at a
glance. They are the only raster art in the interface apart from the mark.

**The count is an orange badge on the icon**, `orange` under a 2px outline with
Inter's tabular figures, at the icon's top-right — and only when the bay holds
something. No bay ever wears a nought. This is the strongest form a count has
taken in this product, and it is a deliberate choice rather than drift: the old
rule against counts was retired for doors, and the bar is the door.

### The channel

The rack itself: `rgba(28,17,11,.2)` inset behind a 2px `rgba(254,214,167,.2)`
edge, running to the foot of the board. Strips sit inside it at a 5px gap. The
channel is why an empty bay looks like room rather than like absence.

### The blank strip

The head of a rack, dashed, in cream at 80%, carrying that bay's own question —
*what is it* in the notes, *what did you decide?* in the tasks. What you type
lands in the bay you are looking at, which is what retired the composer and its
row of destination chips together.

Typing turns it solid: `paper` stock, solid outline, the focus ring. The camera
lives on the same row as the plus, in the notes rack and nowhere else, and only
where there is a volume to keep photographs on — with nowhere to put them the
camera is never offered.

A photograph goes through the same path every other capture takes: the bytes
reach the volume and are fsynced there before the spool entry that points at
them exists. A photograph with no words is a capture, which is most of the point
of having a camera.

**Every bay has one, and none of them has a second step.** A chore needs a
rhythm and an appointment needs a day, and the obvious build was to take the
words and then ask — which would hold a thought in a form field, the one thing
this product may never do. So the question moved into the strip instead of after
it. The chores rack carries four rhythm stamps under its field — *a day*, *a
week*, *2 weeks*, *a month* — and pressing one is the whole act. The agenda's
field teaches its own grammar in the placeholder, `at 14:30 dentist`, which is
the sentence chat has always parsed.

**And the floor under all four: words that are not what the rack asked for are
still a thought.** A chore typed with no rhythm and an appointment typed with no
time go to the notes, through the spool like any other capture, and the board
says where they went. A bay may refuse to make what you asked for. No bay may
drop what you typed.

### The stamp

Rectangular, 2px outline, 3px radius, `0 3px 0` cast, caps, with the key letter
in a 2px-radius box at 70% opacity. Four fills:

| Fill | Meaning |
|---|---|
| `paper` | The neutral answer. |
| `orange` | This makes something happen: I'll do it, make a chore. |
| `state-done-lifted` | Done, did it. |
| `card-deep` | This does nothing to the world: not today, later, stop. |

A stamp is always in the same place on the object it acts on, and always carries
its key. There are no pill buttons in this product any more.

**What each bay answers.** A note takes *done*, *keep*, *drop*, and *make a
chore*; a chore takes *did it* and *later*; a task takes *done* and *drop*; an
appointment takes *it is over*, because it is not answered the way work is — you
left, or it stopped mattering, and nothing records which of the two it was.

**Make a chore asks on the strip.** The note already exists, so there is nothing
being held while the question is asked: pressing it swaps that one strip's
answers for the four rhythms, and pressing a rhythm promotes the note. One strip
asks at a time — a rack where every row asks a question is a rack you have to
answer to read.

### It notices

What the product noticed is written in the margin of the board, once a day.
Nothing else on the board says anything a model wrote, and nothing waits for
one.

**The pulled strip's clause is the picker's.** The rules choose the thing and
the rules say why. A model wrote that clause between 3 and 4 September 2026 —
one call per newly picked thing, made inside the render — and it was removed for
the reason it was flagged as risky when it shipped: *not today* invalidates that
decision by design, so the press that means "not this one" waited seconds for
the next card, and a board that costs a model call to press is a board you stop
pressing.

**No mark on the pulled strip, and nothing to refuse there.** The mark said
where a sentence came from; with every sentence coming from the rules there is
nothing to attribute. *That did not land* went with it — marginalia carries its
own refusal, and that is the only place the product speaks unbidden.

### Marginalia

A line may also hang under an ordinary strip, in the margin of the rack rather
than on the pulled strip. It is set small, in the muted ink, indented under the
strip's own words — a pencil note against a line on a list, not a second voice
in the room.

**It is written on a cadence, never on a press.** Once a day the board is read
as a whole and at most two lines come back. Nothing the person does asks for
one, and nothing waits while one is written: a line is already there or it is
not. This is the difference between a note in the margin and a chat.

**A line earns its place by connecting two things.** The detail one thing needs
is written in another; several of these are the same errand; this cannot start
until that is done. A line that restates the strip it hangs under is worse than
an empty margin, because it still has to be read.

**Never a count, never about the person.** No numbers, no *again*, no *still* —
those are the sentences that turn a board into a report card. The line is about
the things, and it never instructs and never asks.

**Refusing is the only control, and it is real.** *not useful* sits at the end
of the line, quiet and lowercase. The words are kept, not deleted, and the next
pass is shown them as something not to write again.

### Capture writes the row

**The screen writes what you typed, before it answers.** A capture from the
board or the dock is one row, written in the request, so the board you are sent
back to has the strip on it. It went through a spool and a background drain
until 4 September 2026, and the delay was the whole of what made the product
feel broken: a press did nothing, a reload showed it had worked.

**Campfire still spools.** That path has nobody in front of it and nowhere to
say a write failed, so durability there is worth an extra hop. The screen has
both — it can refuse out loud, and a refusal is better than a promise it cannot
keep.

**What that trades.** With Postgres unreachable the screen now says so instead
of accepting the words and settling them later. That is the honest report: a
capture box that clears on failure is a capture box that eats thoughts, and the
words stay in the box.

### The page about you

Who you are, what Squirrel has worked out about you, and how you have been —
one screen, drawn on arrival.

**Nothing on it is asked for.** Both readings were a press that answered in the
conversation, which put the two things this product holds *about you* behind a
door inside a room you had already opened. You go to this page when you wonder
about yourself; wondering is the asking.

**An opinion about you is readable and refusable.** What it has worked out is
shown in its own words, one line per thing so that none of it reads as a
paragraph of assessment, with no dates — when it worked something out is not a
fact you can act on. *Forget all of it* is one press with no confirmation, and
the empty state that follows says what forgetting cost.

**A read that fails is a sentence where the thing would be.** Not an error
page, and never a blank: the rest of the page does not depend on either read.

### Buddy, and the acorn — did not land

This section described a board mechanism that shipped between 3 and 4
September 2026 and was then pulled: the pulled strip carrying a model's
sentence in place of the picker's clause, an acorn beside it naming who wrote
the sentence, and a press under it — *that did not land* — to refuse one.
None of it survived. **It notices**, above, is the accurate record: the
clause is the picker's again, there is no mark on the pulled strip, and
nothing there to refuse.

**A different press with the same name is still live, and it is not this
one.** Buddy's own replies, in the conversation rather than on the board,
carry their own refusal — *that went badly* — one press, on any reply a model
wrote (`/buddy/badly`, `coachBadlyHandler` in `internal/web/coach.go`, tested
in `internal/coach/badlylanded_test.go`). It marks the answer rather than
arguing with it, and the marked ones are what the next prompt is shown as
examples of what does not work here. It has nothing to do with the acorn: it
was never attached to the pulled strip, and it was never removed.

### The pencil line

A pencil glyph, `ink-soft`, sentence case, at most
one underlined action in the line. It sits in the strip's margin under the
words, above the stamps. He has no face here, no bubble, no timestamp and no
column of his own; nothing renders the distinction between his sentences and
Squirrel's, which is Principle 8 and unchanged.

A strip carries at most one pencil line. A second one is Buddy talking to
himself.

### The tray

The day's departures. Strips at 22% stock with their words struck, their holder
at half opacity, the state as a `mark`. The newest carries `PUT IT BACK` as a
stamp like any other. It empties overnight, and that is what stops it becoming a
history you can be behind on.

### The check-in

The five faces, unchanged artwork, in 3px-radius tiles at the tray's right end —
and on a phone, on their own row above the day's departures, because the tray
scrolls sideways and a question sharing that row is a question you can push off
the screen. Asked once an hour, drawn at the edge, and nothing is said back:
the conversation answers a check-in with a turn because it is a record of what
was said, and the board is a record of what there is. A reading is neither a
strip nor something to answer. A `wiped`
or `frazzled` reading thins the board — see the risk noted in the strip-board
comp; the thinning rule belongs to the picker, and until it is written this is
the design's largest unfinished edge.

### The ledge, and the two shelves

The lip at the foot of the notes rack, below every live strip: a hairline, then
two tabs in sentence case — `what you set aside` and `the things you kept`. They
are not bay signs and are deliberately not set in caps, because they are a way
through rather than a place you are standing. Solid-edged since 2 September,
under The Blank Is Dashed.

Pressing one turns the notes rack into that shelf and nothing else on the board
moves. The sign becomes the shelf's name in the dashed variant, and where every
rack opens with something to write on, a shelf opens with `back to the notes`.

**A held strip** is recessed rather than raised: `rgba(28,17,11,.16)` behind a
2px dashed cream edge, cream ink, no shadow, and it does not cock. Present, and
not a thing you can pick up. Its holder stays the colour of the bay it came
from, and where a live strip carries a mark, a held strip carries **what would
move it** in his own words — *when he replies*, *when the part arrives*,
*someday*. The three ways to say you cannot act stay three, because they end
differently.

**A kept strip** is flat printed stock — `card-deep`, no shadow, no mark, no
cocking. A fact rather than a job, which is what `kept` has always meant: a
serial number or a link is not a task and will never be done.

Both carry exactly one stamp, `back in the pile`, because every transition in
this product reverses.

### Named Rules

**The Open Strip Is The Focused Strip.** One rule for both worlds. On a desktop
hover and focus open a strip; on a touch screen a press does, and a press is
what focus means there. The keys follow it — an arrow moves to a strip and opens
it, a letter acts on the one that is open — so there is never a strip that is
focused and shut, or open and unfocused.

**The Blank Is Dashed.** A dashed edge means *there is nothing on this yet*. It
belongs to the blank strip and everything inside it — the camera, the four
rhythms — and to the two notices that are not part of a rack: the trouble line
and a shelf's sign. Nothing else. Anything you can press or follow is drawn
solid, because a dashed control reads as a placeholder, which is exactly what a
bay tab and a ledge tab looked like until 2 September.

**A Shelf Never Counts.** The sign carries the name and no number, and this is
the one place in the product where that is a rule rather than a preference. A
number beside stalled work is a reproach, and the point of setting something
aside is to stop being asked about it. PRODUCT.md's retirement of the count rule
does not reach here: it permitted counts on doors, and a shelf is not a door.

**A Shelf Is Not A Bay.** Four bays, and there is no fifth sign on the board. A
shelf borrows the notes rack for as long as you are reading it. A door for
stalled work would put it in front of you every morning, which is precisely what
setting it aside was for.

**A Shelf Has No Blank Strip.** Nothing is kept or set aside by being typed —
both states are reached by deciding about a strip that already exists. A shelf
you could write into would be lying about where its contents come from.

**The Holder Says Where It Came From.** A held task keeps its amber holder and a
held note its purple one, so one shelf holds both without needing a second
label. This is the holder rule doing real work rather than decorating.

### The find field, and what it finds

In the ops bar, behind a 2px cream edge. Search is the only navigation in this
product besides the four bays, and it is a GET: what you looked for is in the
address, so it survives a reload and can be kept.

**What matched takes the racks' place.** One rack, centred, headed
`what matched "…"`, opening with the way back. Not a fifth bay, not an overlay:
the board has one place where things are, and looking for something puts what
you found there.

**Every state, on one screen**, which is what the pile has always promised. What
a result carries is decided by where it is: something still in the pile keeps
its four answers, and something that already left carries `back in the pile` and
nothing else — offering the exits to a note that has taken one is asking a
question that has been answered. Its mark is the state it went to.

## Buddy's room

One room, and the only place the conversation's own shapes survive. It is not a
lesser board and it is not a second pile: it holds what was said, which is a
different kind of thing from what is waiting.

**Its shapes, documented rather than tolerated.** The board is square because a
strip is printed; a conversation is not, and its radii stay what they were:

| Value | What it is |
|---|---|
| `999px` | A chip, and the scrollbar's thumb. A pill is the shape of a thing you say back. |
| `14px` (`--r`) | A card in the conversation, and the boxes that hold one. |
| `10px` | The stamp, and a photograph inside a card. |
| `3em` | Buddy's speech, which is Campfire's own message radius and deliberately far from any card corner: at a glance the two can never be the same object. |
| `7px`, `6px`, `4px` | The small marks — a page tab, a keycap, a notch. Deliberately not tokens, because each is a drawing rather than a size. |

**What the room keeps:** the bubble, the face in the gutter, the day divider,
the dock, and the rail. **What it gives up:** being where a note is triaged.
Those presses have a board now, and two places to answer the same thing is the
problem this product exists to solve rather than one it may create.

**One piece of it landed on 2 September 2026:** Buddy's room reads the whole
record rather than one room's share of it. Nothing was rewritten — every turn
keeps the room it was said in — and what changed is that the reading stopped
asking. That is the piece the retirement needs and the piece that is true
whatever happens to the rooms: a record with four rooms' worth of holes in it
would be the thing that made retiring them lossy.

**Done on 2 September 2026.** The four object rooms are retired. Their URLs
answer 301 to the bay that holds what they held, and the decisions the sweep
was waiting on are these:

- **The record keeps its rooms.** Nothing was rewritten. Every turn still
  carries the room it was said in, and Buddy's room reads the whole record
  rather than one room's share, so nothing said is out of reach.
- **The four are sets, not places.** He can still be asked to show you the
  chores, and he draws them; what they stopped being is somewhere you can
  stand. New turns are still filed by subject, so the record keeps saying what
  a turn was about even though no room draws it.
- **A rack says that there is more, never how much.** *there is more further
  back*, under the last strip it can hold.
- **There is no rail.** Buddy's room has one link where it was, back to the
  board, and the board's four bay signs are the navigation.
- **The notice a door could carry about its set is gone**, and this is the
  record of it: the coach could say one thing about a set of rows — "these are
  all about the car" — and no surface asks for that now. It was a door's
  feature, and the doors went.
- **The live edge is gone with them.** It was a room's current state under the
  conversation; a rack is that by construction, and re-reads on every load.
- **Triage left the conversation.** His room draws no card to answer, because
  answering happens on the board. That is what "it stops being where the pile
  is worked" means in the code rather than in a sentence.

**And the keys are wired.** The board drew a letter on every stamp from the day
it was mounted and read none of them, which the room's own keyboard tests
caught the moment those rooms went. Letters act on the strip you are focused in,
arrows move between strips, and a letter nothing answers to does nothing at
all.

## Do's and Don'ts

### Do:

- **Do** draw every content object as a strip, at every viewport.
- **Do** put a colour only where it means a bay, an action, or a state in the
  tray.
- **Do** print the rule that pulled a strip on the strip.
- **Do** keep the 1150ms hold before anything leaves the board.
- **Do** let an empty rack look like an empty rack.
- **Do** set times, dates and countdowns in Inter Black with tabular figures.
- **Do** keep the shelves on the ledge, below the strips you have not decided
  about.
- **Do** let a held strip carry what would move it where its mark would go.

### Don't:

- **Don't** reintroduce a card, a chip, a bubble or a bottom composer.
- **Don't** fill a strip with a state colour, or tint a row by category.
- **Don't** pull more than one strip.
- **Don't** give Buddy a face, a column, or more than one line on an object.
- **Don't** put a number on a bay sign that counts what you did not do.
- **Don't** use white anywhere, and don't let a surface read as neutral.
- **Don't** add a hover elevation, a modal, or a floating panel; depth here only
  ever means distance from the rack.
- **Don't** give a shelf a bay sign of its own, a count, or anything to type
  into.
- **Don't** draw a held strip so it can be picked up, or a kept strip so it
  looks like work.
