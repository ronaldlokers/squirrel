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

**Status, and what this replaces.** This document describes the design of
record, not the running code, and it says so to the tests as well: while the
line `design-ahead-of-code` is in this file, the radius vocabulary is not
checked against the shipped stylesheet. Delete that line when the board ships
and the check closes again. Everything shipped through v0.55.1 is the previous
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
become a row of four sign-shaped tabs with one lit. The pulled strip stays above
the tabs because it belongs to no bay. The tray keeps its place at the foot and
wraps to two rows: what left the board, then the faces.

There is no second layout. The phone shows one rack of the same board, and a
strip is a strip at both sizes.

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

**Square, because it is printed.** Radii here are 3px and 2px, and the previous
world's 14px card corner is retired with the card. A flight strip is guillotined
from a sheet; the corner is the smallest radius that keeps a 2px outline from
looking chipped on a low-density screen.

**The grip.** A holder carries one mark: a 6×14 rounded slot at `rgba(28,17,11,.28)`,
centred vertically, which is where a thumb takes the strip out. On the pulled
strip it is 8×26. It is the only ornament in the system and it is functional —
without it the holder is a colour bar.

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
- The words column takes 8px 10px of padding and may run to two lines. Below
  the words, when the strip is cocked, its stamps.
- The mark column is the right edge: `mark` type, or `figure` when the thing has
  a time, which is the one case where the mark outranks the words in weight.
- A strip never carries an icon, a thumbnail, an avatar or a progress bar. A
  photograph on a note is shown by opening the strip, not by shrinking it into
  the row.

### The pulled strip

One per board, ever. `paper` rather than `card` — the only surface in the
product brighter than stock — with a 3px `orange-lit` ring, a 14px holder in the
colour of the bay it came from, and its stamps down the right edge in a column.

Above the words: the tag `PULLED` and, beside it in sentence case, the rule that
chose it. The rule is printed on the strip rather than hidden behind a why,
because an offer nobody can account for is a demand.

### The holder

11px of room colour, ruled off from the stock with the 2px outline, carrying the
grip. See The Holder Rule.

### The bay sign

Enamel: `purple-dark`, 2px outline, 3px radius, cream caps, and the rack's count
at the right in tabular figures. The count is **what is in the rack**, not what
is owed and never what is late. A rack with nothing in it shows no number rather
than a nought.

### The channel

The rack itself: `rgba(28,17,11,.2)` inset behind a 2px `rgba(254,214,167,.2)`
edge, running to the foot of the board. Strips sit inside it at a 5px gap. The
channel is why an empty bay looks like room rather than like absence.

### The blank strip

The head of every rack, dashed, in cream at 80%: `write on a new strip` on the
phone, and the bay's own question on the desk — *what comes back?* in the
chores, *what did you decide?* in the tasks. What you type lands in the bay you
are looking at, which is what retired the composer and its row of destination
chips together.

Typing turns it solid: `paper` stock, solid outline, the focus ring. The camera
lives on the same row as the plus.

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

### The pencil line

Buddy's whole surface. A 16px pencil glyph, `ink-soft`, sentence case, at most
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

The five faces, unchanged artwork, in 3px-radius tiles at the tray's right end.
Asked once an hour, drawn at the edge, never written into the record. A `wiped`
or `frazzled` reading thins the board — see the risk noted in the strip-board
comp; the thinning rule belongs to the picker, and until it is written this is
the design's largest unfinished edge.

### The ledge, and the two shelves

The lip at the foot of the notes rack, below every live strip: a hairline, then
two dashed tabs in sentence case — `what you set aside` and `the things you
kept`. They are not bay signs and are deliberately not set in caps, because they
are a way through rather than a place you are standing.

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

### The find field

In the ops bar, 260px, `rgba(28,17,11,.32)` behind a 2px cream edge. Search is
the only navigation in the product besides the four bays.

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
