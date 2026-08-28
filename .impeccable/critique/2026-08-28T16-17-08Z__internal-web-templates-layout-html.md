---
target: the header and the room navigation
total_score: 22
max_score: 40
na_heuristics: 
p0_count: 2
p1_count: 2
timestamp: 2026-08-28T16-17-08Z
slug: internal-web-templates-layout-html
---
⚠️ DEGRADED: single-context (session rule forbids sub-agents unless requested; treated as a decline)

## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | 2 | Current room = 4px down + 1px shadow; indistinguishable at a glance. Strip has no scroll affordance |
| 2 | Match System / Real World | 3 | Room names are the product's own vocabulary. But the brim is amputated at x=236 |
| 3 | User Control and Freedom | 1 | The way out is off-screen on every phone |
| 4 | Consistency and Standards | 1 | Navigation and content share one visual class; rail slab is the only surface without the 3px outline |
| 5 | Error Prevention | 3 | The button naming its consequence is good work |
| 6 | Recognition Rather Than Recall | 1 | Must recall that four more rooms exist and that the strip scrolls |
| 7 | Flexibility and Efficiency | 3 | Chore keys, roving focus, rooms in a new tab |
| 8 | Aesthetic and Minimalist | 2 | 236px for seven words, ~470px dead field, content on three axes |
| 9 | Error Recovery | 3 | Undo, clash notice, capture never eats words |
| 10 | Help and Documentation | 3 | First-run worked example is excellent |
| **Total** | | **22/40** | **Needs work** |

## Design Specificity Verdict

Exceptionally high and not the problem. Nothing else looks like this. The failure is structural, not stylistic: a beautiful visual world doing a job it was not designed for.

Detector: 3 warnings on the rendered chores page. 2x all-caps-body on `EVERY WEEK · LAST DONE LAST WEEK` — false positive, that is `.turnmeta`, a defined label role. 1x flat-type-hierarchy — real but mislocated: the collapse is the rail's labels occupying the same visual tier as card content, not font sizes.

No overlay injected. Evidence is screenshots (1440x900, 390x844) and measured browser geometry.

## Overall Impression

Both complaints share one root cause, and it is a line in DESIGN.md written the same day: "a room is an object in this world like every other object." That is wrong. A room is not an object in the box, it is the box's furniture. Dressing navigation in content's clothes is why the rail feels awful, and why adjusting the header alone cannot fix it.

## What's Working

- The room names. `the chores`, not `#chores`.
- The dock naming its consequence — better than a confirmation dialog.
- The card grammar is untouched and still excellent.

## Priority Issues

### [P0] The way out and search are off-screen on every phone
Measured in the strip layout: `look something up` x=928-1092, `leave it whenever` x=1106-1269, `the things you kept` x=734-914. Viewport 390px. No scrollbar (`scrollbar-width: none`). Principle 3 violated; DESIGN.md claims a rule that does not exist on a phone.
Fix: pin the way out and search; only the seven rooms scroll. Command: /impeccable adapt

### [P0] Navigation wears content's clothes
`the pile · 12` and `water the plants` are the same object: same fill, outline, radius, shadow. Seven cream cards read as a backlog on a product whose thesis is one thing at a time.
Fix: rooms get quieter material than card stock — the recessed `.held` treatment already means "present, not a thing to pick up." The Three Marks Rule already forbids this; it was applied to controls and not to surfaces. Command: /impeccable distill

### [P1] The header is not floating, and the rail amputates the brim
Lid is `margin-left: 236px`, butting the rail in a hard seam at x=236. The brim starts two-thirds across. The rail slab has no outline — the only surface in the product without one.
Fix: full-width lid with the brim intact; rail starts below it. Command: /impeccable layout

### [P1] The counts became the scoreboard, in the action colour
12/3/2/4 down the left edge, permanently visible, in orange — the colour reserved for "this makes something happen." A status value wearing the action colour. PRODUCT.md's reversal trigger is met.
Fix: take counts out of orange first; full reversal is still one call. Command: /impeccable quieter

### [P2] Three axes and a 470px hole
Dock left edge 380, card left edge 546, neither centred in the stage. ~470px of empty field between last card and dock.
Fix: one column axis for cards, heading and dock. Command: /impeccable layout

## Persona Red Flags

Ronald, phone, one thumb: wants to stop; `leave it whenever` is at x=1106 and nothing says the strip scrolls. Closes the tab — the one ending this product exists to make graceful.
Ronald, desktop: seven cream cards left, two cream cards centre; first scan cannot separate places from chores. Current-room signal invisible at scale.
Jordan (first-timer): no affordance says the strip scrolls; four of seven rooms do not exist to them.

## Minor Observations

- Phone strip clips `the chores` mid-pill: reads broken rather than "more this way."
- `.lookup`/`.leaving` are low-contrast cream on purple; worth an audit pass.
- `--railw: 236px` is sized for the longest label, not the content.
- Brim SVG keeps `preserveAspectRatio="none"`, stretching the curve at 1204px inset.

## Questions to Consider

- If the rail were ink on the field instead of stock, would it still need 236px?
- Is seven rooms the right number on a phone at all?
- What if `look something up` and `leave it whenever` never lived in the rail, because neither is a place?
