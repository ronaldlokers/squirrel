---
target: overall design, and the header curve
total_score: 31
max_score: 40
na_heuristics: 
p0_count: 0
p1_count: 2
timestamp: 2026-08-28T21-32-20Z
slug: internal-web-templates-layout-html
---
⚠️ DEGRADED: single-context (session rule forbids sub-agents; treated as a decline)

## Design Health Score

| # | Heuristic | Score | Key issue |
|---|---|---|---|
| 1 | Visibility of System Status | 3 | The room is named in the lid and solid in the rail |
| 2 | Match System / Real World | 4 | The product's own vocabulary throughout |
| 3 | User Control and Freedom | 4 | Back works through rooms; the way out is always in view |
| 4 | Consistency and Standards | 2 | The dock is inset for a rail that moved; the brim stretches to a wonky line |
| 5 | Error Prevention | 3 | The button naming its consequence |
| 6 | Recognition Rather Than Recall | 4 | Rail always present, current room named twice |
| 7 | Flexibility and Efficiency | 3 | Chore keys, roving focus, rooms in a new tab |
| 8 | Aesthetic and Minimalist | 2 | Three axes, and a vast empty right field on desktop |
| 9 | Error Recovery | 4 | The unreadable-store state is exemplary |
| 10 | Help and Documentation | 2 | The worked example is broken on first run |
| **Total** | | **31/40** | Solid, with two visible defects |

## Priority Issues

### [P1] The dock is inset 234px for a rail that is no longer on the left
Measured at 1440px: column centre 609, cards centre 493, dock's field centre 715.
197px between the thing you type into and the things it produces. Cause:
`padding-left: calc(22px + var(--railw))` was added when the rail was on the
left, and the rail moved right without it. Removing it puts the slot's left edge
at 271px — exactly the turns' left edge, which is what the 676px slot width was
chosen for. Command: /impeccable layout

### [P1] The worked example is broken on first run
Carried from the product review. `.worked` reproduces the thread's markup with
none of `.thread`'s padding or gap, so it runs off the left edge and its cards
overlap the bubbles. Pre-existing since 26 August. Command: /impeccable layout

### [P2] The brim reads as a wonky line on desktop
A 30px arc with preserveAspectRatio="none" stretched across 1440px sags about
8px — not a curve, a line that looks slightly wrong. It works at 390px. One
shape cannot do both widths. Three options: straight everywhere; keep the curve
and drop the stretch so it holds its proportion; or curved on phone and straight
on desktop, which is two identities. Command: /impeccable polish

### [P3] STOP ASKING reads as disabled
Brown-grey against a green DID IT and a white HOW OFTEN. Meant as the quiet
third option; at that value it reads as unavailable.

## Corrected from earlier rounds
`look something up` and `leave it whenever` measure 6.56:1 — identical to the
room labels. They read as secondary because the rooms sit in dashed wells and
these do not, which is the intent. Flagged twice before as a contrast problem;
it is not one.

## Detector
3 findings, all noise: two all-caps-body on the card meta line (a defined label
role) and one flat-type-hierarchy across ten deliberate sizes.

## Systemic
Cream on this field measures ~4.8:1 at the lit centre of the radial — rail
labels, counts, Buddy's bubble. All pass 4.5 and none by much. No headroom: any
future lightening of the field pushes several things through the floor at once.
