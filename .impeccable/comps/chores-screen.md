# Comp notes: the chores screen (`/chores`)

Comp: `chores-screen.html` (open it from its place in the repo — fonts and the
mark are linked relatively to `internal/web/static/`). Two screens in one file:
the list with one picker open, then, past a labelled annotation band, the same
screen empty.

## What changed from the draft, and why

- **A chore is no longer a search result.** The draft dressed chores in
  `rcard` + the orange `state-chore` page tab. The tab vocabulary means "what
  state this note ended in", and a column of identical orange tabs puts a
  dozen oranges on one screen — the Orange Reserve Rule allows one, at the
  moment something is made. DESIGN.md already says the orange CHORE mark is
  the moment of promotion, shown once. So: a chore at rest is plain cream
  stock, no tab, and **this screen carries no orange at all** except the
  browser surfaces (caret, selection, scrollbar) that always do.
- **Elapsed time is soft and colourless.** The draft set `LAST DONE {{.Last}}`
  in the state-ink slot, which colours it like a status. It is now part of one
  headphone-brown meta line — `EVERY 2 WEEKS · LAST DONE THIS WEEK` — because
  reporting what happened is not a state. Buckets, never numbers: *today /
  this week / a while ago*. A chore never done shows only its rhythm; what has
  not happened is not reported.
- **The name leads, in his voice.** He named the chore, so it is set casual
  (`CASL 1`, wght 520) at the Note role's floor — 21px, stepping to the Note's
  23px on the phone — not the 18px Result role, because a chore is decided on,
  not scanned past.
- **Actions grew to the full control scale** (44px, 4px offset/press; 48px in
  the phone grid) — they are the screen's purpose, not a result card's
  conveniences. DID IT presses to done green, STOP ASKING presses to dropped
  brown and sits apart at the row's far end (desktop) or on its own full-width
  row (phone), mirroring the deck's "the different one is separated" rhythm.
- **The picker replaces the row in place, scriptlessly.** Still a `<details>`
  disclosure (plain HTML, no modal), but `form:has(.often[open])` steps the
  other two actions aside exactly as the deck's JavaScript does, the open
  summary becomes the system's dashed *never mind*, and the chips get their
  `COMES BACK` lead so each chip reads as a sentence. Without `:has()` the
  actions simply stay put and the chips open beneath — still the working page.
- **The current interval wears cap purple.** The picker should say where
  things stand without a count or a grade; purple is the product's own colour,
  not a state and not a reward. Pressing it again is the usual no-op.
- **The pile link moved into the lid** (`.lidlink`, which pile.css already
  has), per the shared-lid rule; the foot line is now the product's own quiet
  sentence, *squirrel asks when it's time*.
- **The empty state stopped implying an expectation.** "nothing comes back on
  its own **yet**" lost its *yet*; the second line states a fact ("when a note
  becomes a chore, this is where it lives") and suggests nothing.
- Kept from the draft: `WHAT COMES BACK` as the screen's head (it says what
  these are without counting), the four interval values, and plain form
  controls throughout. List order is creation order and never re-ranks.

## New tokens and rules (nothing invented quietly)

- **No new colours, radii, sizes or shadows.** The chore card borrows the
  Result card's shadow (`0 5px 0 0` + cast); every size is a documented step.
- **New use of an existing size:** the chore name is the Note role *pinned at
  its floor* (21px / mobile 23px) instead of the viewport clamp — a list can't
  take the deck's 27px.
- **New rule candidate:** *purple may mark where a setting currently sits.*
  It reports how things are; it is not a fifth state.
- **Extension of the press rule:** DID IT presses to done, STOP ASKING to
  dropped — the existing "a button presses to its own state colour".

## DESIGN.md amendments — accepted 2026-08-19

All five landed, along with a sixth the recolouring below required: **The
Lifted Fill Rule**, which is how a control wears the colour of what it does
without failing contrast. The list is kept as written for the reasoning.

## What was proposed

1. Add a line beside the CHORE-stamp paragraph: **"a chore at rest is not
   orange"** — orange is the moment of promotion only. (This is the rule the
   draft broke.)
2. Record the **soft-elapsed vocabulary** as a copy rule: *last done today /
   this week / a while ago*; never a day count; the clause is absent when
   nothing has happened. It is a product rule currently living nowhere.
3. Put the Step-Up Rule's mobile sizes (23 / 31 / 18 / 18.5 / 12 / 16px) into
   the frontmatter typography ramp — the design detector reads only the ramp
   and flags the rule's own documented values as violations (it did so here;
   the 23px and 31px findings in this file are those false positives).
4. The Two Voices Rule's sentence "nothing is set in the casual axis unless
   the owner typed it" contradicts Headline/Voice being casual product speech
   two sections earlier. Suggest: "…unless the owner typed it *or the product
   is speaking in full sentences* (Headline, Voice)."
5. Document `.lidlink` (the quiet cross-link in the lid) — it ships in
   pile.css but is absent from DESIGN.md's components.
