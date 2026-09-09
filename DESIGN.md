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
  # A pill or a disc: the ops bar's chips and rail, the find field on a phone,
  # the writer's pill at the foot, and the dots down the rail. Never on
  # anything printed — a rounded slip of paper is a sticker.
  pill: "999px"
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

**The ending, decided 3 September 2026: the conversation retires.** It is gone —
see *Buddy's room — retired* at the end of this document, which is the current
description.

This paragraph said the opposite for two days. The decision of 1 September was
that the thread would stay at `/r/everything` as Buddy's room, reached by a link
in the ops bar, and that two worlds would live in this document. That was
reversed on 3 September and the reversal shipped in v0.75.0: there is one
surface inside the app, and Campfire beside it. The earlier reasoning is in git
history and is not repeated here, because a document that argues both sides in
two places is a document you cannot act on.

**Status.** This describes the running code. There is one world here now — the
board — and the section at the end records what the room was and what became of
each thing it did. The board is the front door at `/`, and `/r/{room}` is a
redirect.

Everything shipped through v0.55.1 is the previous world — the conversation,
where every object was a turn in a thread, drawn on cream card stock with a 3px
outline and a 14px corner. That world's document is in git history.

**What survived the redesign, because the owner pinned it:** the two faces,
Recursive and Inter, and the whole palette. Purple and orange are the product's
own colours and any surface that reads as neutral or white has failed to be
Squirrel. That is unchanged and non-negotiable.

**What did not survive:** the card, the chip, the bubble, the dock, the lid and
its brim, the sheet, and the 3px sticker outline that drew all of them. The
outline language stays; the sticker does not.

### One stylesheet, and the two small pages are in it

`pile.css` is gone. It was the room's stylesheet — 2,006 lines against the
board's 748 — and `/me` and the way in went on loading it for a day after the
room did not exist, so both pages shipped a design this document had already
declared dead: a 3px sticker outline, 14px corners, a 999px smoked-pill timer
and a pill to sign in with.

They are drawn from the board's own tokens now. The corners are `--r`, the
outline is `--line`, the stock is `--card` on the outline's 3px cast, and the
one register that survives the move unchanged is the underlined quiet action.
One shape was deliberately not carried over: *let me in* is a stamp — the same
rectangle, cast and caps every press on the board is drawn with. The other was
the running timer, which is moot since the body double left on 9 September
2026.

**A face stays round.** The board's own `.chip.face` is a circle and always has
been, and a picture of a person is the one place this product draws one. The
settings page agrees with the board rather than with the rectangle rule;
`TestBrowserYourFaceIsRoundEverywhere` holds both screens to it.

**The lid is the ops bar.** Same solid `--purple-dark`, same 2px rule beneath
it, no blur and no brim. It was translucent with a backdrop filter so the
conversation could be seen passing under it; there is no conversation to pass
under it.

**The band beside the bar.** `board.html` declared `theme-color` `#472e70`
while drawing a `#3b2560` bar, so an installed app met its own status band in a
seam. Both pages declare the bar they draw.

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
| `label-tight` | Recursive precise, 9px caps | The same word below 620px, where the five faces sit across a phone; and in the dial's own column, which is that narrow at every width. |
| `sign-tight` | Recursive precise, 10px caps | A door's sign below 620px, where three of them share a phone's width. Added 8 September 2026 with the doors; nothing else uses it. |

### Named Rules

**One hue, one meaning.** Orange is a thing to do, brown a fixed point the
world set, lilac the pile. A hue answers exactly one question and gives the
same answer on every screen.

Yellow retired on 9 September 2026, when a thing you do one time became a
rhythm rather than a kind. A chore and a once-thing are the same object cut by
how often it comes back, so a second hue for one of them was drawing a line the
product no longer holds.

A chore is **not** coloured by its rhythm. The rack it stands in and that
rack's own word already say which rhythm it has, and the spine was saying it a
second time — which is what made lilac mean both *daily* and *the notes* on a
screen that shows both. The rack signs keep a neutral cream at 36%, because a
sign that carries a hue is a sign making a claim the column already makes.

**The Printed Figure Rule.** A time, a date and a countdown are set in Inter
Black with tabular figures, because on a board a number is a printed thing and
not a sentence. This is the only content use of Inter in the product; everything
else that Inter used to do belongs to the wordmark. `figure-large` was the
running timer's, and nothing takes it since the body double left on 9 September
2026; the largest printed figure now is a fixed point's time in the diary.

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
across the middle to the bell and your face at the right. It is furniture:
nothing in it belongs to any one thing on the board.

**The find field is the middle of the bar** at every width, not only on the
phone. The middle used to be Buddy's chip, and when the room retired the bar's
right half kept its place while the middle went with the chip — half a bar of
bare purple. Search is the only navigation in this product besides the four
bays, and the navigation holds the middle rather than a corner; the phone bar
already said so, and the desktop is the same bar, not a second one.

**No working row.** The board's first line is the racks. The band that held the
pulled card and a running timer went on 9 September 2026 — the pick is on its
own row — and the timer went with it the same day, when the body double was
retired.

**The racks.** Three across and the dial at the right, `1fr 1fr 1fr
minmax(0, 264px)` — *daily*, *weekly*, *seldom*, cut by how often a chore comes
back. Equal, because none of the three is the important one; which rack a thing
is in is the fact the eye is meant to pick up, and a wider column would say one
of them mattered more. Each rack is a sign and then a channel of strips. The
channel is a recessed well that runs to the bottom of the board, so an unfilled
rack reads as *room in the rack* rather than as dead space.

**One writer for everything**, and the way in is a field. Floating over the
board at its foot — bottom right on the desk, edge to edge on the phone — paper
stock, a pill, with a `+` at its end. **Typing and pressing enter keeps a
thought**, which is one keystroke and is what principle 1 buys with this shape.
**Pressing `+` opens the writer** and carries what you typed into it, so saying
it is something else never costs the words twice.

Until 9 September 2026 the board had three writers — a chore writer and an
appointment writer under the racks, and a task inlet at the head of the once
rack. Three fields asking *what is it* in three places, so before you could
write anything down you had to decide which one to walk to. The modal asks the
words first and the kind second, which is the order the thought arrives in.

**One screen, and it commits at the end.** `Add something` in sentence case,
then the words, then four kinds as a row of pills — *a thought* · *to do once* ·
*comes back* · *a time*. Under them, in a nested panel of `card` stock, only
what that kind needs, and as presets rather than fields: **how often** is *every
day* · *every week* · *2 weeks* · *a month*, because the answer is nearly always
one of four and a number box with a unit beside it is two decisions where one
will do. A time asks for a day and a clock, and **and again** offers *just the
once* · *every week* · *2 weeks* · *4 weeks*.

Nothing is written until `KEEP IT`, which is the one orange stamp on the screen.
`NEVER MIND` and Escape both close it, and what you typed goes with them — the
words travel in the address, so shutting it clears them there too. A draft
Squirrel kept for you is a thought in a form field, which is the one thing this
product may never do.

**It is drawn without a script.** The bar is a link to `#add` and the writer is
`:target`, so it opens with the stylesheet alone. What the script adds is the
two courtesies: focus into the field, and Escape. Neither is what makes it work.

**And it is the one modal in this product.** *Don't add a modal* was a rule here
until this writer; it is now a rule with exactly one exception, and the reason
is that a writer must not be reachable from four places at once and must not
take a column of the board to sit in.

**The notes are a pill in the ops bar** — a lilac dot, the count, and the word
*notes* — beside the bell, and it leaves the board for `/notes`. A pill rather
than a glyph because it is the only thing in the bar that is a *place with
things in it*, and the count is what makes it worth looking at. The row of doors under the writers went with the once rack
on 9 September 2026: two doors, two signs and two questions were a second region
under the board saying what the board already had room to hold.

**The tray.** Fixed to the foot, `rgba(28,17,11,.34)` over the ground, ruled off
at the top. What left the board today, struck through, oldest first, with the
newest carrying `PUT IT BACK`. Nothing else lives here now — the check-in left
it for the dial.

### Breakpoint: 620px

One breakpoint, as before, and below it the phone is **its own screen rather
than a fold of the desk**. The racks go; the rail takes their place.

**The rail is today, in the order today happens.** A line down the left with a
dot at each thing, and a short spur from each dot to what it holds.

**At the head, the one thing that wants you now**, and it is its own object
rather than a strip: `paper` stock, a 14px holder, the reason above the name,
the name at 25px, and **its answers as two large stamps side by side** — the
only place on the phone where a thing can be answered. A fixed point inside its
leaving window takes the head if there is one; otherwise the first thing with an
hour does.

**Everything else hangs off the rail as a row**: a block on the left carrying
the hour, the words on the right, and no answers at all. **The tab carries the
time**, and it is the hour when there is one and the part of the day when that
is all there is — an appointment has a clock and a chore has only *mornings*. A
thing that wants you today with neither hangs last, with no block.

**It steps out as the day goes on.** Each row indents a little further than the
one above, to four steps and then no further. The day reads as a descent rather
than as a list.

**Off the rail, two rows and nothing else**, under *whenever you like*: *things
you decided*, which opens in place, and *in the notes*, which leads to the wall.
Each carries a count in a block the width of the rail's own hour blocks — orange
for what you decided, lilac for the notes.

The counts are the one place this product prints a number of things you have not
done yet, and they are here because the alternative is worse: a phone that lists
every chore not asking today is a phone you scroll past to find today. See
`PRODUCT.md` for the overrule and what it costs.

**A chore that is not asking today is not on the phone at all.** It is on the
desk, in its rack, where a thing you are not being asked about belongs. And
under *further ahead*, any fixed point that is not today, with its day rather
than its hour.

**The writer is the field at the foot.** Fixed, edge to edge, 26px clear of the
home indicator — the measurements the tab bar had, on the thing that replaced
it. Paper rather than smoke: it is stock you type ink onto, not chrome you look
through. The tab bar went on 9 September 2026: with the racks gone there were no
tabs to draw, and the phone's foot is better spent on the thing you came to do.

The dial folds away behind today's face in the ops bar. The tray keeps its
place. The notes stay a chip in the ops bar on the desk and a row off the rail
on the phone, because five chips leave the find field too narrow to type in.

**Both screens are drawn at both widths**, and which one you get is the
stylesheet. A strip is still a strip at both sizes, and every letter and press
that works on one works on the other — the script skips whichever set is not on
screen, which is what `offsetParent` is for in `board.js`.

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

**No bay is lit when you are not standing in one.** A search and an opened
strip light nothing, because a bar that says you are in the once rack while you
are reading a search result is a bar you stop believing. The wall lights its
own tab, because it is a place you are standing.

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

**44 is the floor, and it is walked rather than believed.** Every control a
thumb can reach clears 44×44 on a phone — the number the devices spec chose,
stricter than the 24×24 the standard asks for. It is now a test that walks the
real screens at 390×844 with touch emulation on and fails naming the element and
its measured size, because seven controls sat four pixels under it for a
fortnight and nobody had measured. The chips in the bar, the strip's chevron and
its way back, the blank strip's field and its `+`, and the search pill were all
40px.

Three were once knowingly below it and said so rather than being bent to fit.
One was never real: the compose field's exemption named a `textarea` that left
with the room it belonged to, and nothing on any screen matches that selector
any more. The other two are fixed, on the phone breakpoint only, and the test
that used to name and forgive all three carries no exemption of any kind now —
a control under the floor fails outright, naming itself.

**The chore's rhythm and the appointment's day and clock stay a sentence above
620px** — `every 7 days`, `05/09 14:30`, unboxed and underlined, exactly as
before. Below it the same fields become boxed: a 2px outline, the stock's own
corner radius, `--paper` behind the ink, 56px wide for the rhythm's count and
46px for each of the day, month, hour and minute, all 44px tall. Two sentences
either side of the one breakpoint that already redraws everything else, not
one field wearing two disguises.

**The lid takes the fix the ops bar's rail already had.** The flex spacer
beside the search icon that balances the missing wordmark stops growing on a
phone, so the find field takes the room the spacer used to hold instead of
splitting it, and the icon inside the field grows to 44px into the room the
field just gained — the same move that already gives the board's own bar a
search icon with nothing to take width from.

**And nothing scrolls sideways at 320px**, walked the same way. The assertion is
on `body` rather than on the document: `html, body` carry `overflow-x: clip`,
under which the document's own scroll width can never exceed its client width,
so the obvious test is one that cannot fail.

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

**The rail gives way.** Below 620px the board under the ops bar is one scrolling
deck, so what you are looking at scrolls and the bar at the foot does not. The
rail's head is the first thing in it and scrolls with everything else; nothing
holds the top. The ledge still sits at the foot of the channel rather than after the last
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

### The pick is on the row it is about

The picker still chooses one thing and still says why. Until 9 September 2026 it
said so on a card of its own at the top of the board — `paper` stock, an
`orange-lit` ring, its own copy of the words and its own four answers. The card
is gone. What replaced it is the row the pick is already about.

**The chosen row wears `PULLED` and the rule that chose it**, above its name, in
the quiet register. It keeps the answers every row of its kind has: `did it`
and `later`, or `done` and `drop`. Nothing is drawn twice, which is the whole
reason the card went — the picker only ever chooses a thing that already has a
row, and the board was drawing that thing in two places at once.

**Its two hard answers sit under the stamps, quiet**, beside *ask Buddy*:
`not this one` and `I'm stuck`. Quiet rather than stamped, because they are not
what the row is for; they are what you press when the row is wrong.

**`not today` and `not this one` are two different no's.** `not today` — which
is `later` on the row — concedes the pick and asks again tomorrow: right thing,
wrong moment. `not this one` does not concede it; the pick itself was wrong, and
it does not expire at midnight. Neither costs a model call, and neither ever
surfaces as a number, because a tally of wrong picks is exactly the report card
Principle 2 forbids.

**`not this one` is a chore or a thing you do once only.** A fixed point is the
world's business and cannot be the wrong pick, and those two are the only kinds
the picker now chooses among.

**Being stuck asks, and then says one sentence.** `I'm stuck` replaces the two
quiet presses with the product's own four — *too big*, *don't know how*,
*boring*, *not today* — and pressing one replaces them with the ladder's fixed
line. Which blocker you pressed lives in the address, so a reload shows the same
sentence instead of repeating a press. `not today` reached this way is the same
no as `later`, and leaves the same mark.

That sentence carries **no acorn**: the ladder's lines are fixed and are
Squirrel's own. **And it is the whole answer.** The ladder offered a `5 MIN` or
`10 MIN` beside two of its four sentences until the body double left on
9 September 2026; the sentences stayed, and *boring* says what its alarm used to
— a short go, ended by you rather than by a bell.

### A row with nowhere else to be

Almost everything the picker chooses has a row already. One case does not: a
pick further back than a rack draws — a thing you decided weeks ago, behind
`there is more further back`. It gets a loose row at the head of the board
rather than a card, because a choice you cannot see is not a choice.

The other two cases that used to live here — a running timer whose thing was not
on the board, and the breadcrumb — went with the body double on 9 September
2026.

### The opened strip corrects itself

A strip opened by name — from search, a notification, or a press — carries two
things the rack strips never do.

**Saying it another way.** *say it another way* is a link, quiet and lowercase,
and pressing it puts a field under the words holding what is there now. Only
the sentence changes: the time it arrived, the state it is in and its place in
the pile all stay, because those are facts about the note and only the wording
was wrong. It is not versioned — keeping the old text would make a note a
document with a history to read, which is a second place a thought can hide.
**No model may ever do this.** Rewriting your own words is a refused tool and
stays refused; this is the person's own press or it does not happen.

**The three states you cannot act on.** *waiting on* · *blocked on* · *someday*
sit under the ordinary answers, and only here — a rack strip must stay one line
and two stamps. They belong to a thing you decided to do; a note has no state
to be waiting in since the wall replaced the shelves.

### The opened strip breaks into steps

**Too big**, pressed on the pulled strip, no longer ends in a fixed sentence
alone. When the offer names a task, the sentence is broken into steps and the
board sends you to that task's own strip, opened — the sequence lives there
rather than under the pulled strip's own line.

**One step, never the list.** The opened strip draws the step's words, `done`
and `forget the steps` — the same shape *not now* already has, one press, no
consequence — and, on the last one, the words `the last one` beside it.
Nowhere is a position out of a total drawn: no `1 of 3`, no bar, no count.
That is not a drawing choice made here; the store this reads from has no
function that returns the sequence, so this strip could not draw one even if
asked to.

**A sequence belongs to the strip it is about, and to no other.** Opening a
different note or task while a breakdown is under way shows that strip as it
always was — no step, because the step is not about it. There is exactly one
sequence in flight for a person at a time, the same rule the timer keeps, and
it surfaces only under the one strip it was made for.

**When there is nothing to open onto, the fixed line is still the floor.** A
blocker that cannot be broken down, or a model that is slow, absent or wrong,
leaves the pulled strip's own sentence exactly as phase B drew it. Principle
10 costs nothing here: with no coach configured, `too big` has always meant
the fixed line, and still does.

### The opened strip corrects itself

A strip opened by name — from search, a notification, or a press — carries two
things the rack strips never do.

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
quiet morning. The wall and the search have the same line — *nothing in the
notes*, *nothing matched "…"* — because an empty bordered box says only that
something has gone wrong.

The line matters most on a phone, where one rack is the whole screen and there
is nothing beside it to compare against.

### The blank strip asks rather than guesses

A chore typed with no rhythm, and an appointment with no time in it, used to be
filed as notes — in another rack, found on the next refresh. The board asks
instead: the words come back into the field they were typed in, with the
question under them, and the interval or the day and time beside it are the
answer.

**One box per rack, and never a second.** A note box was pinned above every
phone tab but the notes for four days in September 2026, so a thought typed on
the chores tab could not become a weekly chore by accident. It is gone. Two
boxes on one screen, asking different questions, is the same decision it was
meant to spare you — made twice instead of once, and with a label explaining
which is which.

The problem it was aimed at was real and is now fixed in the field, which is
where it belonged. The count started at seven, so the interval was never empty
and the branch above could not run: a thought typed on the chores tab became a
weekly chore rather than a question. Seven is a placeholder now — it shows the
shape of the answer and sends nothing — so a chore typed with no rhythm is
asked about, like every other thing the board will not guess at.

The hint takes `--placeholder`, the colour every other blank strip already uses.
The rule for it was written `.strip.blank.asit .count::placeholder`, narrowed to
the agenda because the agenda's day and clock were the only counts with a hint;
it is `.strip.blank .count::placeholder` now. Left alone the browser draws its
own grey, which is 3.98:1 on this field and fails the contrast sweep.

**A rhythm is any number of days, weeks or months.** It is asked for on the
strip, as a number and a unit. Four preset chips stood under the field until
v0.68.0 and were removed there: with the interval already on the strip they said
the same thing twice, and a screen that can only offer four intervals is a
screen that
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

The phone's navigation. Four tabs — *now · daily · weekly · seldom* — in the
tab's own name and its count, no picture. The cell you are in takes
`orange-lit`; the other three sit at cream.

**The pictures went with the bays, 8 September 2026.** Each of the four bays had
a drawn icon, and they were the only raster art in the interface apart from the
mark. The four tabs that replaced them are not four things: they are one list of
chores cut four ways, and three of the cuts differ only in how often a thing
comes back. There is no drawing of *seldom*. Four illustrations invented for
four cuts of one list would have been four decorations, and a decoration that
claims to be a distinction is worse than a word. The files are deleted rather
than left unreferenced.

**The count sits beside the label.** `&middot;` and the number, in the tab's own
ink, beside its name — *weekly · 3* — and only when the rack holds something. No
rack ever wears a nought. An orange disc at a tab's corner is the platform's own
grammar for a notification demanding attention, which a rack's count is not.

### What is asking for you today

A rack is split in two by one question: **is this asking for you today?** It
comes back today, or today is the day you usually do it — ranks one to four,
the same cut the phone's *now* uses, so a thing can never be lifted on one
screen and resting on the other.

**What is asking sits on `paper`** and takes the deeper shadow. Paper is what
this world has always used to say *this one* — it is what the pulled strip is
made of — so nothing new was invented to say it. Its name steps up to 16.5px
and it keeps its reason.

**Below the seam, a row is its name and its rhythm.** The reason and the usual
time are still true and still one press away on the row itself; they are not
worth the room on something nobody is being asked about. A rack of seven was
carrying thirty-five things and now carries about twelve.

**Nothing is dimmed.** A resting chore is exactly as doable as it was
yesterday, and greying an available control to make its neighbour louder is
borrowing against the wrong account.

**The rack's sign counts what is asking, not what it holds** — *weekly 1*, not
*weekly 3*. A count of what wants you is the only kind this product allows.

**There is no separate usual-time line.** Every rank that asks for you already
carries a reason, and that reason names the usual time whenever the usual time
is why the row is where it is. A second line for it said the same thing twice
under one name, and it is gone.

### Once

The fourth rack, and the last column before the sidebar. A chore comes back and
a once-thing does not, which makes *never again* a rhythm like any other — and
the racks are already where a rhythm is drawn. It stands last because the
columns run by how long until a thing returns, and a once-thing never does.

**The one you decided last is what asks for you.** Nothing about today makes a
once-thing due, so the only claim the board can honestly make is which one is
freshest, and it says exactly that: *the last thing you decided*. Never the
oldest — how long a thing has been waiting is the count the racks have always
refused to draw.

**It is the only rack with a writer in it.** The three that come back share one
writer under the board, because the rack a chore lands in is what its interval
says. A once-thing has no interval to say it, so its writer sits at the head of
its own column, where the thing goes.

### The rack row

A chore's strip carries its name, its rhythm, and — only when it is asking for
you today — the one line saying why it is where it is.

A row whose turn is simply not today says **nothing at all**, and above all not
how long it has been waiting. That refusal is older than the racks and survived
them intact.

### The late mark

One chore in the product can say it is late: the kind the world put on a
weekday. `late — today`, in an outlined orange chip above the why-line, and the
row leads its rack.

**Only a chore with a weekday wears it.** A chore on an interval — every seven
days, every three — was put on that rhythm by you, and Squirrel marking you
late against a time you invented is the accruing judgement the product refuses.
A chore on a weekday was put there by the bin lorry.

**It says late and never how late.** No count, no days-since, no second colour
for worse. And it does not survive the day that caused it: tomorrow the bins
are an ordinary chore again, unmarked, because a mark that carries forward is a
mark that accrues.

**A chore nobody has ever done is not late.** It is new, and the sentence about
a thing you have never done is a sentence about you.

### What is coming

The sidebar's lower half, and the agenda's home since 9 September 2026. Every
fixed point still ahead, soonest first — not only today, because that is what
the door held and the rule the list is allowed under does not care how far
ahead it reaches: only what is still in front of you, nothing past, nothing
done, never a count of what you did not do.

**The time is a deadline and is set as one.** Inter Black, tabular, the
`figure` role — because an appointment is the one thing in this product with a
time the world imposed, and `PRODUCT.md` settled that on 20 August. The first
one today takes 24px; the rest take 19.

**The leave-by under it is arithmetic** about a distance, in the quiet
register, and it says so when the distance was a guess. It is never called a
deadline; the time above it is the deadline.

**One that comes round says how often**, in the same quiet register, above the
leave-by: *every week*, *every 2 weeks*. Weeks and never days, because what
comes round is a day of the week at a time of day. It never says which time this
is or how many there have been — a fixed point is not a streak.

**Inside the window where leaving matters, it is hoisted** out of the list and
above the dial, on paper with the orange outline. It *leaves the list* when it
does — a fixed point drawn twice in one column is the duplication the picker
was cured of. Nothing else reorders that column.

**A fixed point that has started stays for two hours and says `late`.** It
hoists exactly as one inside its leaving window does, and its leave-by comes
off — there is nothing left to leave for. The time the world set is the one
time in this product you can genuinely miss, so holding it is not Squirrel
inventing a deadline. After two hours it goes on its own, unpressed, which is
what keeps this from becoming something that piles up.

**The diary is read, not worked.** There are no answers on it. A fixed point
leaves on its own once its window is out, and closing one is what the pulled
strip is for while it is open. The agenda's four answers went with the door.

**The writer did not follow it.** Day, month, hour and minute cannot be read at
a sidebar's width — the comp proved it — so the appointment writer pairs with
the chore writer under the racks: the thing that comes back, and the thing the
world set, side by side.

### The dial

How you have been, in its own column beside the racks. A ring of seven arcs —
the week behind you, today closing the circle at the top left — around today's
face, then the five faces, then *the whole record*.

**A ring rather than the grid on the page about you.** Six weeks by seven days
is a stretch of past you look back over; seven days is a cycle you are inside.
Each arc is named by its weekday so no counting backwards is needed to find a
day, and each carries the day and the word as its title. A day you said nothing
is drawn in `rgba(28,17,11,.13)`, because the gaps are the honest part here
exactly as they are there.

**The five faces are always pressable.** This is the change of 8 September 2026:
they used to appear only when Squirrel wanted an answer, so saying how you were
was something you waited to be asked for. Being asked is still separate — the
*how do you feel?* line above them appears only when the last answer has stopped
describing now — but the answer is always available.

**Below 620px the dial is not drawn.** Today's face rides in the ops bar as a
chip, and pressing it puts the dial in the racks' place with a way back, the
same as a door. A ring, five faces and a link do not fit a phone's column
without taking a screen from the thing the phone is for.

### What a rack is not showing

On a wiped or frazzled reading the racks hold what comes back today and count
the rest: *2 more further into the week*. Drawn at the channel's foot, above a
2px rule, at 12px in `rgba(255,251,243,.62)` — quieter than any row above it,
because it is not a thing you can act on.

**The words are load-bearing.** Never *outstanding*, *left*, *still*, *waiting*,
*behind*. It is the size of the part of your life the board has decided not to
put in front of you today, not a number of things you have failed to do. A
browser test reads the sentence and fails on any of those words.

Not on a *low* reading. Low is how you feel; wiped and frazzled are how much you
have, and an emptied board handed to someone flat but functional reads as the
product agreeing they are finished.

### The channel

The rack itself: `rgba(28,17,11,.2)` inset behind a 2px `rgba(254,214,167,.2)`
edge, running to the foot of the board. Strips sit inside it at a 5px gap. The
channel is why an empty bay looks like room rather than like absence.

### The blank strip

The head of the wall, dashed, in cream at 80%, carrying the notes' own question
— *what is it*. It is the only blank strip left: the board's three became the
writer above, and this one stayed because it is the only one with a camera and
because capture on the wall must not need a modal to open first.

Typing turns it solid: `paper` stock, solid outline, the focus ring. The camera
lives on the same row as the plus, in the notes rack and nowhere else, and only
where there is a volume to keep photographs on — with nowhere to put them the
camera is never offered.

A photograph goes through the same path every other capture takes: the bytes
reach the volume and are fsynced there before the spool entry that points at
them exists. A photograph with no words is a capture, which is most of the point
of having a camera.

**Nothing has a second step.** A chore needs a rhythm and an appointment needs a
day, and the obvious build was to take the words and then ask — which would hold
a thought in a form field, the one thing this product may never do. So the
question sits beside the field rather than after it. The writer asks for its
interval on the same screen, a number and a unit, and answering it is the whole
act. It carried four preset stamps under the field
as well until v0.68.0, which removed them: the interval was already on the strip
and the stamps were a second row saying the same thing. The agenda's field
teaches its own grammar in the placeholder, `at 14:30 dentist`, which is the
sentence chat has always parsed, and its day and clock are both hand-built
fields — a native date or time input renders in the browser's locale and no
attribute changes that.

**And the floor under all four: words that are not what the kind asked for are
still a thought.** A chore typed with no rhythm and an appointment typed with no
time go to the notes, through the spool like any other capture, and the board
says where they went. A bay may refuse to make what you asked for. No bay may
drop what you typed.

**A photograph Squirrel will not take is refused the same way.** Choosing a PDF
or a kind this does not keep used to end at the failure page, which says nothing
has been lost while the words that came with it had been. It is a refusal, not a
breakdown: the words come back into the notes box and the line under the strip
says what happened and what to do — the wording the retired room used, kept.

A volume that will not write is a different thing and still fails visibly. The
picture was a picture; the memory is unreachable, and a screen saying "too big,
or a kind Squirrel does not take" about a working camera would send you looking
for a fault that is not there.

**Words typed with no network say so on the strip they were typed in.** The
worker takes them, and the board you come back to is the rack you were in, with
one line under the blank strip: *no network — I have it. It goes in when you are
back.* It is the same quiet line the two questions use, because it is the same
kind of thing — the screen telling you where your words are.

The worker knew only the room's route until 7 September 2026. Every board
capture went to `/board/new` or `/board/capture`, which nothing intercepted, so
with no network the browser's own error page replaced the board and the words
went with it. It holds the whole form now rather than one field: a note typed on
the chores tab has to go back to the chores when it lands, and a hold that kept
only the words filed it somewhere else.

**A photograph you have chosen is drawn under the box, and held.** Choosing one
hands the screen to another app, and an app handed away can be reclaimed and
comes back reloaded — with an empty input, looking exactly as it did before,
because it never looked any different. So it goes into the browser's own storage
the moment it is picked and comes back onto the input when the board is drawn
again, with the picture under the words and *take it off* beside it. The
photograph is a 84px square in the board's own 3px corner, not the room's 10px
one.

The blank strip is multipart only while it is carrying a photograph. A form that
always claimed multipart would give up the offline hold on every words-only
capture, which is the case the hold exists for. Without the script it stays
multipart and posts a photograph correctly, showing nothing and holding nothing
— the floor this was built on.

### The stamp

Rectangular, 2px outline, 3px radius, `0 3px 0` cast, caps, with the key letter
in a 2px-radius box at 70% opacity. Four fills:

| Fill | Meaning |
|---|---|
| `paper` | The neutral answer. |
| `orange` | This makes something happen: I'll do it. |
| `state-done-lifted` | Done, did it. |
| `card-deep` | This does nothing to the world: not today, not this one, later, stop. |

A stamp is always in the same place on the object it acts on, and always carries
its key. There are no pill buttons in this product any more.

**One fill per strip.** Orange is the accent — the bell's dot, the chores
holder — and a strip that lights it twice makes the loudest thing on the row
whichever action happens least. A note carries `did` on *done* and paper on
everything else, including *make a chore*: asking a question about the note is
not the thing the note is for, and only *I'll do it* on the pulled strip earns
the accent, because that strip has nothing else competing for it.

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

The markup said it and the stylesheet did not: `.strip .seen .off` carried
`text-transform: uppercase` with tracking and 750 weight until 7 September 2026,
so the one control the person has over what the board says about them was drawn
louder than the sentence it refuses.

### Asking about a strip, and the answer in the same margin

A strip carries one more press, *ask Buddy*, drawn as paper stock beside the
answers rather than as anything louder — it is not a disposition, and it must
not read like one. It is only there when a coach is configured: with no key the
press does not exist, and the board is what it was.

It shipped as a fifth stamp on 5 September 2026: same caps, same outline, same
cast as *done* and *drop*, riding the dispositions' own form — so it was struck
like one when pressed, and the row read as five things you could do to the
strip. It is its own form now, in the quiet register: underlined ink, no
outline, no cast, no key letter. The keyboard letters belong to the answers,
and this is not an answer.

**It says who answers.** *ask about this* named the strip; *ask Buddy* names
who replies. Principle 8 draws the line between what the rules produced and what
a model wrote, and a press that is about to spend a model call is the moment to
say which side of that line the sentence will come from.

**The letters belong to the strip you are focused in, whatever kind it is.**
Every strip that draws a key letter answers to it: the rack's rows, the opened
strip, the pulled strip, a search result. Until 7 September 2026 only the
rack's rows did — the dispatch asked for `.strip.answerable`, which the other
three are not — so D, K, X, R, W, B, Y on the opened strip, D, N, W, S, T on the
pulled strip and Z on a result that already left the pile were drawn and did
nothing. A key hint that lies is worse
than no hint: it teaches you a way of working that stops working when you move.

**An answer you asked for is something you are sent to.** Asking is a whole page
load, so nothing on the client can announce what happened before it. The answer
carries `tabindex="-1"` on the one strip you just asked about and focus is moved
there — a screen reader reads it out, and a keyboard is left standing where the
new thing is rather than at the top of the board. Only on the draw that follows
the press: every later draw leaves focus alone, because being sent back to
yesterday's answer on every page load is the same noise from the other side.

**Every field says what it is.** A field with no label, no `aria-label` and no
placeholder is an unnamed edit field to a screen reader, and the visible context
that would have explained it is a page load ago. `.wordfix` — the only way left
to correct a strip's wording — had none.

**The answer comes back in marginalia's own line.** Same rule above it, same
muted ink, same *not useful* at its end. That is the whole of the decision: the
board speaks in the margin, and it speaks there once. A second register — a
panel, a sheet, a bubble — would be a conversation growing back on the surface
that is replacing the conversation.

**One line in the rack; what came before it on the strip you opened.** The rack
draws exactly one noticed line per thing and always has — two lines under one
strip is a conversation, and a rack is not one. The lines before it are kept
now rather than overwritten, and they are read back on the opened strip under
*what Squirrel said about this before*. That is not the rack loosening its rule:
an opened strip is one thing, looked at on purpose, which is the same reason the
words can be corrected there and nowhere else. Until 7 September the row was
replaced, so "what did it tell me about the boiler last week" had no answer
anywhere.

The opened strip also draws the newest line, in marginalia's own register with
its *not useful* at the end. It did not before, so opening a strip lost the one
thing the rack had been showing about it.

A line you refused is not read back as history — refusing it said do not say
this again, and a record that reads it back to you every time you open the strip
is the opposite of that. It stays in the table, where the next pass is shown it.

**A press, where marginalia is a cadence.** These are the two ways a line
arrives and they differ in exactly one way: this one was asked for. Nothing
about the drawing distinguishes them, because nothing should — Principle 8 is
about whose voice a sentence is in, not about labelling it.

**Never a box.** Free text on the board goes into the capture fields and
nowhere else, and no capture field has ever reached a model. A field that asked
a question would be the general AI chat companion the product refused, moved
onto the board and given a different name.

**What an answer may not do is name a thing you could then act on.** A note is
a thought nobody has decided about yet. An answer that handed one back with a
control on it would be the product deciding for you, so only its words are
kept.

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

**Each thing it has worked out is a card, and it is dark ink on paper.** The
same stock and the same ink as every other raised thing in the product — a
stamp, a strip. It was cream on cream for eleven days: the rule asked for a
colour nothing defined, so the declaration was invalid and the text fell back to
what the purple page inherits, which is cream. The rows were there and the words
were rendering; only the colour was missing.

Nothing found it because the page only draws these cards when there is something
to show, and the dev screen answered that question with nothing — so the empty
state was what everyone looked at, including an accessibility audit of this
page. **The dev screen knows three things now.** A surface that cannot be seen
in development is a surface nobody reviews.

### Buddy, and the acorn — did not land

This section described a board mechanism that shipped between 3 and 4
September 2026 and was then pulled: the pulled strip carrying a model's
sentence in place of the picker's clause, an acorn beside it naming who wrote
the sentence, and a press under it — *that did not land* — to refuse one.
None of it survived. **It notices**, above, is the accurate record: the
clause is the picker's again, there is no mark on the pulled strip, and
nothing there to refuse.

**The refusal that is live is *not useful*, and it is on the board.** One
press at the end of a marginalia line or an answer Buddy wrote
(`/board/notuseful`). It marks the line rather than arguing with it, the words
are kept, and `WhatWasRefused` shows the next pass what not to write again.

*That went badly* — the press that did this in the conversation, at
`/buddy/badly` — went with the room on 6 September 2026. **The owner's
decision, taken afterward: reuse *not useful* rather than give the retired
signal a new control or drop it.** `nowFor` now reads `WhatWasRefused` into
every prompt in `BadlyLanded`'s old place; `BadlyLanded`, `LandedBadly` and
`LandedBadlyLatest` are gone along with the screen's `Store` entry for the
latter, since nothing called any of them once the room and its press left.
The two refusals are not quite one signal — *not useful* refuses a noticed
line about something on the board, *that went badly* refused an answer Buddy
gave directly — but both are Buddy's own words, rejected by the person who
read them, which is what the prompt slot has always needed.

### What Buddy has cost

One line on the page about you, in the same quiet register as everything else
there: what this month has cost and what it is allowed. It is the only accruing
number anywhere in the product, and the exception is argued in `PRODUCT.md`
rather than here — money, a fact about a machine, bounded by a ceiling set on
purpose.

Drawn only when there is a coach and only when the figure could actually be
read. A build with no key reporting `€0.00 of €10` would be reporting on a
thing that is not there, and an unreadable figure drawn as zero is a screen
that lies about a number whose whole job is to be true.

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

### The wall

`/notes`, its own page since 9 September 2026, and the only screen in this
product that is not the board. Reached from the notes chip in the ops bar, and
on a phone from the last tab in the bar at the foot.

**A sign, a writer, and pins.** The writer is the notes' blank strip, unchanged
and still the only one with a camera. Under it the pins fill a grid of
`minmax(248px, 1fr)` columns — as many as the screen has room for, one on a
phone — top-aligned, so a long note and a short one do not stretch each other.

**A pin is not a strip.** A strip is a row you answer; a pin is a thing you
look at. It carries its picture at the top, bled to the card's edges and ruled
off; then the words; then when it arrived; then two quiet presses. It keeps the
pile's lilac spine at 11px, because what a hue means does not change with the
shape it is on.

**The picture is on the pin.** A note with a photograph draws it, at the width
of the column, from `/photo/{id}/thumb` — and the picture is the link to the
whole one. The board's strips still never carry a thumbnail: a rack is a column
of rows you answer at a glance, and this is a wall you read.

**Two presses and no filing.** *drop*, and *make a chore*. Nothing else. The
shelves went with the triage: there is no *keep*, no *set aside*, no *done*, and
so no ledge, no seam and no second group under the live notes. What is on the
wall is every note you have, and the only ways off it are throwing it away or
deciding it is a thing you do.

### Named Rules

**The Open Strip Is The Focused Strip.** One rule for both worlds. On a desktop
hover and focus open a strip; on a touch screen a press does, and a press is
what focus means there. The keys follow it — an arrow moves to a strip and opens
it, a letter acts on the one that is open — so there is never a strip that is
focused and shut, or open and unfocused.

**The Blank Is Dashed.** A dashed edge means *there is nothing on this yet*. It
belongs to the blank strip and everything inside it — the camera, the four
rhythms — and to the trouble line, which is the one notice that is
not part of a rack. Nothing else. Anything you can press or follow is drawn
solid, because a dashed control reads as a placeholder, which is exactly what a
bay tab and a ledge tab looked like until 2 September.

**The Wall Never Counts What Is Not On It.** The sign counts the pins it is
drawing and nothing else. There is no shelf behind it to have a number, which is
what retired *A Shelf Never Counts*, *A Shelf Is Not A Bay*, *A Shelf Has No
Blank Strip* and *The Holder Says Where It Came From* on 9 September 2026. All
four were about the two shelves; the shelves are gone.

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

## Buddy's room — retired

The room is gone. It was the second conversation inside the app, and the
decision of 3 September was that there is only one: the board, with Campfire
beside it. Its shapes went with it — the pill, the 14px card, the gutter, the
dock, the transcript — and nothing on the board inherits them, because a strip
is printed and a conversation was not.

What it did lives on the board now. A strip can be asked about and the answer
hangs in the margin; the ladder is on the row the pick is about; search takes
the racks' place; the opened strip corrects its own words. What it held —
the turns — is untouched: the record is kept, and what went is the thing that
drew it.

**The two ways to talk to Squirrel are now the board and Campfire.** Campfire is
where a thought can be thrown from a phone with the room already open, and where
Squirrel can say something without the app in front of you. It is not where the
product lives, and no capability exists only there.

**What went with it, and is worth knowing.** The rail that cleared the lid on a
desktop; the dock's own reserve arithmetic; the slot; the faces line and the day
divider. The browser tests that measured those shapes went too — they asserted
on markup that no longer exists, and repointing them at the board would have
been a test that passes without testing.


## Do's and Don'ts

### Do:

- **Do** draw every content object as a strip, at every viewport.
- **Do** put a colour only where it means a bay, an action, or a state in the
  tray.
- **Do** print the rule that pulled a strip on the strip.
- **Do** keep the 1150ms hold before anything leaves the board.
- **Do** let an empty rack look like an empty rack.
- **Do** set times, dates and countdowns in Inter Black with tabular figures.
- **Do** let a note be a note: on the wall, with its picture, and with nowhere
  to file it.
- **Do** let a held strip carry what would move it where its mark would go.

### Don't:

- **Don't** reintroduce a card, a chip, a bubble or a bottom composer.
- **Don't** fill a strip with a state colour, or tint a row by category.
- **Don't** pull more than one strip.
- **Don't** give Buddy a face, a column, or more than one line on an object.
- **Don't** put a number on a bay sign that counts what you did not do.
- **Don't** use white anywhere, and don't let a surface read as neutral.
- **Don't** add a hover elevation or a floating panel; depth here only ever means
  distance from the rack. One modal exists, the writer, and it is the exception
  rather than the start of a habit.
- **Don't** give a note a state, a shelf, or a second place to be.
- **Don't** draw a held strip so it can be picked up.
