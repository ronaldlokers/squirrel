---
version: 1
slug: "impeccable-comps-pile-first-viewport-html"
primary_target: ".impeccable/comps/pile-first-viewport.html"
related_targets: ["internal/web"]
---

# Surface: the pile

**Mode:** Operate. The visitor completes a task — reading back a thought and
deciding what it is. Scanability, the keyboard, and the real usage scene outrank
expression; the brand lives in the material and the details.

**Approved comp:** `.impeccable/comps/pile-first-viewport.html`, approved
2026-08-18. Its sidecar carries the approval flag. Screenshots of all five
states at both viewports are in `.impeccable/comps/shots/`.

**Not yet implemented.** The comp is the artifact; the screen itself is phase
5b, planned in `docs/superpowers/plans/2026-08-18-pile-screen.md` and specified
in `docs/superpowers/specs/2026-08-18-pile-design.md`. When it lands, this
brief's primary target becomes `internal/web/`.

## The composition

The lid is the mascot's cap and carries its brim. One card sits slightly above
centre with a handled stack beneath it, the date and the acorn badge in its
title bar, four actions at its foot with MAKE A CHORE the orange one. Search
lives in the lid and replaces the deck in place. Nothing counts anything.

## What must not be literalized

- **The stack is texture, not data.** Three cards behind the top one, rotated
  and offset, present whenever there is more. They are not three notes and must
  never scale with how many there are.
- **The randomised rotation is the point.** Habituation is a documented risk for
  this user. A stack that sits identically every load has failed even though it
  looks the same in a screenshot.
- **The 1150ms hold is a requirement, not a transition.** The card stays put
  after it is actioned so the undo has somewhere to be. Shortening it to feel
  snappy removes the undo's home.
- **The field's gradient and dot grid are the room, not an object.** They are
  the one gradient the world allows. Nothing pressable may take one.
- **Sample content is authored.** "buy milk", "kaas", the boiler notes and the
  meter reading are invented. The owner's real notes are in production and must
  never be pasted into a comp.

## Two things the comp resolves that the record did not

- **Letters are actions; movement is space and the arrows.** The spec's first
  draft asked for j/k to move *and* one key per action, which collides on `k`.
  The spec has since been amended to match the comp.
- **Search results are actionable.** Open notes carry all four actions, triaged
  ones carry BACK IN THE PILE. Beyond what the spec asks for, kept deliberately:
  two views one pile, and every transition reverses.

## Deferred on this surface

Skipping a note from the server-rendered screen (no cursor exists); the
interval picker as a chip row rather than a disclosure; search-as-you-type.
All three are recorded in the phase 5b plan.
