# What the sprint left behind

**Status:** proposal, 2026-08-22. Written against #110, and it argues with the
way that issue is framed.

## What was asked

The review's engineering-manager lens said: 263 commits in nine days, 6MB and
75+ markdown files of specs, plans and reports, on top of the binding
documentation — and recommended archiving or folding what no longer describes a
live decision.

The pace observation is right and worth keeping. The recommendation, measured,
turns out to be aimed at the wrong pile.

## What is actually there

| | Files | Lines | Has to stay true? |
| --- | --- | --- | --- |
| `docs/superpowers/plans/` | 9 | **18,882** | No |
| `docs/superpowers/specs/` | 9 | 2,739 | Yes |
| `docs/proposals/` | 6 | ~2,400 | No — they are arguments, dated |
| `DESIGN.md` | 1 | ~1,400 | **Yes** |
| `PRODUCT.md` | 1 | ~460 | **Yes** |
| `docs/roadmap.md`, `pile-screen.md`, `testing.md`, `running.md` | 4 | ~900 | Yes |

The plans are 78% of the volume, and they are the part that does not matter.

## Why the plans are not the problem

A plan here is step-by-step execution scaffolding — checkboxes, inlined test
bodies, instructions addressed to whoever is doing the work. `2026-08-20-tasks`
opens with *"REQUIRED SUB-SKILL: use subagent-driven-development to implement
this plan task-by-task"* and carries 29 checkboxes.

They describe **how work was done**, on a date, and every one of them is
finished. Nothing reads them for current truth, so nothing about them can drift
— they cannot become wrong, because they were never claims about the present.
They are already an archive; deleting them would only move them into git
history, which is where an archive already is.

**Documentation load is not volume. It is the number of sentences that have to
stay true**, and by that measure the plans cost nothing.

## Where drift actually happened

Twice, and both times in the same file: `DESIGN.md`. The 20 August critique
found it contradicting the shipped markup in seven places; a second pass on 22
August found the zoom rule overstating what survives — it claimed a deliberate
pinch still worked, and it does not, in the one context where the rule takes
effect.

So I checked the other must-stay-true documents rather than assuming.
`pile-screen.md`'s route table, diffed against `internal/web/pile.go`: **one
omission** — `/enough`, the stopping screen, was missing. Fixed in the same
commit as this proposal. Everything else in it is accurate, including the
`/coach` → `/buddy` redirect, which I initially mis-read as stale and which is
real.

That is the shape of the problem: not rot everywhere, but drift concentrated in
the one document that is largest, most detailed, and describes the thing that
changes most often.

## What to do

1. **Do not delete the plans.** They are inert and cost nothing to keep. If
   they are unwanted as clutter, that is a tidiness preference rather than a
   maintenance one, and worth naming as such.

2. **Mark them inert**, so nobody mistakes one for a current description.
   A one-line header on each — *"Executed. A record of how, not a description
   of what is."* — costs nine lines and removes the only real risk they carry,
   which is somebody reading one in a year and believing it.

3. **Put the effort where drift happens.** #104's remaining half — no change to
   `pile.css` or the templates merging without a `DESIGN.md` diff in the same
   commit — is worth more than deleting 19,000 inert lines. The appearance
   snapshot now catches the *code* half of that drift automatically; nothing
   catches the document half.

4. **On the pace itself**, the observation stands and needs no document: all
   four roadmap "next" items are shipped, and this is the natural point to slow
   down rather than start the next feature at the same cadence.

## What this proposal does not recommend

Consolidating the specs. They are cited by `DESIGN.md`, `PRODUCT.md` and the
roadmap, they hold reasoning that was written down precisely so it would not be
re-litigated, and 2,739 lines across nine documents is not a burden. The
proposal that produced this product's best decisions is on that shelf.
