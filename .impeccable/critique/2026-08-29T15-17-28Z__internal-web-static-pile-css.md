---
target: radii
total_score: 11
max_score: 20
na_heuristics: 1,3,5,7,9
p0_count: 0
p1_count: 2
timestamp: 2026-08-29T15-17-28Z
slug: internal-web-static-pile-css
---
Method: dual-agent (A: design review · B: detector + browser evidence, isolated and parallel).

## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | n/a | Status carried by fills, offsets, stamps; radius plays no part |
| 2 | Match System / Real World | 3 | Sticker/pressed/stuck-on metaphor encoded in corners; notch reuses 999px as a punched hole with no prose cover |
| 3 | User Control and Freedom | n/a | A corner cannot grant or withhold control |
| 4 | Consistency and Standards | 2 | Ten shipped values against four documented; four radii all meaning "pressable" |
| 5 | Error Prevention | n/a | Nothing prevented by a radius alone |
| 6 | Recognition Rather Than Recall | 2 | "If it's a pill I can press it" broken by .letmein, .face, .hit button, day cells |
| 7 | Flexibility and Efficiency | n/a | Not a radius property |
| 8 | Aesthetic and Minimalist Design | 3 | Surface reads as one soft world; 9/10/11 cluster is invisible clutter |
| 9 | Error Recovery | n/a | Nothing about corners recovers an error |
| 10 | Help and Documentation | 1 | Scored against DESIGN.md; prose contradicts its own frontmatter, documents a dead keycap |
| **Total** | | **11/20** | **55% — Acceptable** |

## Design Specificity Verdict

Authored core, generic accretion at the perimeter. press=pill / object=14 / stuck-on=10 is a real semantic system. 3em on Buddy's bubble is the most product-aware radius in the file. But ten distinct radii ship against a prose scale of four, concentrated in the newest components: gate, rail, search hits, day picker, worked example.

Deterministic scan found nothing, and that is the finding. layout.html:48 links the stylesheet root-absolute; the detector's resolver does path.resolve(fileDir, href), Node discards the first arg when the second is absolute, resolution lands on the nonexistent literal /static/pile.css, the read fails silently. pile.css is never parsed, so every cascade-dependent rule no-ops on this repo. Proved by reproducing with a relative href: the rule then fired correctly on 7px and 4px, matching lines 802 and 840. The design-system-radius findings seen during editing come from the edit hook, which scans the CSS directly; the CLI path is blind.

No overlays: both MCP browser tools require the absent chrome channel. Fallback was raw CDP against chromium — evidence only, not a user-visible overlay.

## What's Working

1. Pill/card is the load-bearing pair and survives every screen and both devices. Controlled test confirms it is the only radius distinction readable without side-by-side comparison.
2. The stuck-on family is coherent: every photograph is 10px (893, 1255).
3. Small-square elements show sound proportional instinct — 6px on a 15px tab, 4px on 13px, 7px on ~44px; each corner roughly a third of height. Screenshots confirm the 6px pagetab reads as visibly tighter than the card at phone size. The instinct is right, just unnamed.

## Priority Issues

[P1] .letmein (1520) is the only primary button that is not a pill. Same orange fill, same 3px outline, same 0 5px 0 0 sticker shadow as .enough .leavehere (1006, 999px), but var(--r). Undefended, on the first screen anyone sees. Fix: snap to 999px or write the defence at the site and in ## Shapes.

[P1] The detector cannot see this stylesheet. Root-absolute href defeats the CSS resolver; every cascade rule silently no-ops. This is why drift accumulated unseen, and why a clean CLI scan of this repo is false confidence.

[P2] The 9/10/11 cluster: .calgrid label.day 9px (1408), .roomsheet > summary 11px (1546), .hit button 11px (1794), .workedacts span 10px (1857). Controlled test: detectable only when adjacent, and they never appear adjacent. The 11px pair is a genuine third meaning and frontmatter already owns card-inner: 11px — pressable soft rectangles that are not objects yet. Name it, snap the day cells in, and decide what .workedacts is.

[P2] .frombuddy .said declared twice verbatim, pile.css:1132 and 1305, the second inside the 620px media block where it changes nothing. Drift trap.

[P3] The scale is prose, not code. Frontmatter declares seven tokens (chip 999, card 14, card-inner 11, stamp 10, tab 8, key 5, dot 3); ## Shapes names four; only --r exists in CSS. key 5px and dot 3px match nothing shipped; tab 8px disagrees with the pagetab's 6px; 7px, 9px, 4px, 3em appear in no token. On keycaps: pile.css:653 records the removal ("No keyboard, no keycaps"). DESIGN.md's key token and thread.js:150's live d/k/x/t bindings are what is out of step.

## Persona Red Flags

Jordan (first-timer): the two teaching surfaces are the two that lie — the gate's non-pill primary, and the worked example drawing the verbs at 10px when the real controls are pills.
Alex (desk): d/k/x/t fire from thread.js:193 with no keycap, hint or trace. The keyboard path is knowledge you must already have.
Casey (one-handed phone): the day picker's 9px cells are the least button-looking pressables, at the moment a mispress books the wrong day.

## Minor Observations

- ::-webkit-scrollbar-thumb 8px on a 14px thumb; 999px would put browser chrome on-language. Not visually verifiable in headless capture.
- .turncard.kat .notch reuses 999px as a punched circle; ## Shapes says 999 means "you press it".
- Polaroid nesting is off-concentric: 14px outer with 9px padding wants ~5px inside; shipped 10px.
- Unrelated detector false positive: its colour rule flags rgb(71,46,112) on body as outside the palette. That is --purple-bar, in :root and named in DESIGN.md three times.

## Questions to Consider

1. With four standing exceptions, has the real rule become "a pill when it stands alone, a soft rectangle when it stands in a set"? That is a better rule. Write it?
2. Why is the shape system the one system with no token, no snapshot dimension, and no test that would fail if .letmein drifted?
3. The keycaps died and the keys lived. Which was the decision?
