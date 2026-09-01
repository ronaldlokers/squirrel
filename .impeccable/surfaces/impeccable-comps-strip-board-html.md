---
version: 1
slug: "impeccable-comps-strip-board-html"
primary_target: ".impeccable/comps/strip-board.html"
related_targets: ["internal/web"]
---

# Surface: the board

**Mode:** Operate. The visitor works a rack — reads what was pulled for them,
answers a strip in front of them, writes a new one, and stops. Scanability, the
keyboard, and the real usage scene outrank expression; the world lives in the
material, not in ornament.

**Direction locked 1 September 2026.** The roll assigned the departure board;
the owner took the pick card, the strip board, from the same hand. The other
cards on that table — the iridescent edge, the modular system, and the four
declined challengers whose disciplines were donated into the assigned card — are
spent and carry no approval.

**Reference comp:** `.impeccable/comps/strip-board.html`, which carries its own
written build contract: world, first viewport, visitor path, signature
interaction, motion grammar, cross-surface reach, honest risk. Built code-led
(no image generation on this machine), so that contract is what the finish
review audits rather than a rendered comp.

**Landing, phase by phase.** As of 1 September 2026 the board is mounted at
`/board` and draws the four bays, the pulled strip, the running timer and the
ledge from the store. A strip can be answered on it: a stamp
strikes it, the strike holds 1150ms, the strip travels, and it lands in today's
tray carrying `put it back`. Without scripting the same stamp posts a form and
the board comes back with the strip in the tray, so the answer path works with
the script off and the animation is the enhancement.

The notes and tasks racks can be written into: the notes bay writes to the spool
the way every other capture in this product does, the tasks bay inserts a
decision. The chores and agenda racks have no blank strip, because both need a
second answer first.

Not built: the second step for a chore's rhythm and an appointment's day, the
pulled strip's own three stamps, making a chore from a note, answering anything
in the agenda, a photograph, and the phone's bay tabs. `/` is still the
conversation.

`DESIGN.md` still runs ahead of the stylesheet, and both say so: the document carries the line `design-ahead-of-code`, and
`TestTheRadiusVocabularyIsClosed` stands down while it is there. Everything
shipped through v0.55.1 is the conversation. When the board lands, delete that
line and the check closes again.

## The composition

Four bands, top to bottom, and their order never changes: the ops bar (mark,
clock, day, find, settings); the working row (the one pulled strip, and a
running timer's panel beside it when one is running); the racks (four bays,
`1.18fr 1fr 1fr 1fr`, each a sign, then a blank strip, then a channel); the
day's tray along the foot, with the check-in's five faces at its right end.

At the foot of the notes rack, inside its channel and below every live strip,
the ledge: two dashed tabs in sentence case, `what you set aside` and `the
things you kept`. Pressing one turns the notes rack into that shelf — the sign
becomes the shelf's name, the blank strip becomes `back to the notes`, and
nothing else on the board moves.

Below 620px the four racks become one and the signs become tabs. There is no
second layout.

## The shelves, resolved 1 September 2026

They are the ledge under the notes rack, not a fifth bay. What that buys, in the
order the rules were argued:

- **A shelf is not a door.** A fifth sign would put stalled work in front of you
  every morning, which is what setting something aside exists to stop. The ledge
  sits below every strip you have not decided about, so it is reached by going
  past your own pile rather than by being shown.
- **A shelf never counts.** The sign carries a name and no number. PRODUCT.md
  retired the count rule for doors; a shelf is not a door, and a number beside
  stalled work is a reproach.
- **A shelf has no blank strip.** Nothing is kept or set aside by being typed,
  so a shelf opens with the way back instead.
- **Two different objects.** Set aside is recessed and dashed, carrying what
  would move it where its mark would go — *when he replies*, *when the part
  arrives*, *someday*, still three because they end differently. Kept is flat
  printed stock with no mark: a fact rather than a job.
- **The holder does the labelling.** A held task keeps its amber holder and a
  held note its purple one, so one shelf holds both.

Both drawn in the comp as the notes rack in its two shelf states.

## What must not be literalized

- **Exactly one strip is pulled.** The pulled strip is not a "featured card" that
  could become two on a busy day. Two pulled strips is the picker having failed
  to choose, drawn as though it succeeded.
- **The holder is a sleeve, not an accent.** Eleven pixels, ruled off with the
  same outline as everything else, carrying the grip. The moment it renders as a
  coloured left border it is a list item with decoration and the mechanism is
  gone.
- **State is a mark.** Struck words and a change of position. A strip is never
  filled with a state colour, and the four holder colours are only safe because
  of this.
- **The 1150ms hold is a requirement, not a transition.** An answered strip stays
  put while the undo has somewhere to live, then travels. Shortening it to feel
  snappy removes the undo's home. Carried from the previous world unchanged.
- **An empty rack is room, not absence.** The channel runs to the foot of the
  board whether or not it is full. Collapsing it to its content turns a quiet day
  into a screen that looks broken.
- **Habituation is a documented risk for this user.** In the previous world the
  answer was the deck's randomised rotation. Here it is the field's light, whose
  horizontal position is the day's, between 8% and 26%. A board that sits
  identically every morning has failed even though it screenshots the same.
- **Sample content is authored.** The boiler code, the meter reading, kaas, the
  bins, the dentist. The owner's real notes are in production and must never be
  pasted into a comp.

## What this surface still owes

Named rather than discovered later. None of these are drawn yet:

- **The thinning rule.** The honest risk on this direction is density. The
  mitigation written into the comp is that a `wiped` or `frazzled` reading thins
  the board to the pulled strip and the tray. That is a picker rule and it is
  specified nowhere.
- **Capture costs a press on the phone.** Pick the bay, then write, where the
  dock was one gesture. Capture is Principle 1, so this is the trade to argue
  with first if the sofa scene suffers.
- **A photograph on a strip.** Shown by opening the strip, never by shrinking a
  photograph into a 42px row. Not drawn.
- **The phone's bay tabs.** The document says four racks become one with the
  signs as tabs. What is built stacks the four racks in one scrolling column
  instead, which works and is not what is written.
- **The second step.** A chore's rhythm and an appointment's day. Until it is
  drawn, those two racks have no blank strip at all — an empty field that drops
  what you typed while it asks a question is the one thing capture may never be.
- **A photograph on the board.** The conversation's capture takes one; the
  board's does not yet.
- **The pulled strip's stamps.** The offer draws and cannot be answered from the
  board; its three answers already exist behind `/now/act`.
- **Making a chore from a note**, which needs a rhythm and so a second step.
- **Search results, first run, the gate, and the evening message.** All still
  wear the previous world.

## Deferred

The interval picker, the day picker's calendar, and Buddy's longer answers all
need a home in this world and have not been given one. They were sheets and
turns; a rack has no sheet.
