---
target: radii
total_score: 19
max_score: 28
na_heuristics: 1,3,9
p0_count: 0
p1_count: 2
timestamp: 2026-08-29T22-31-49Z
slug: internal-web-static-pile-css
---
Method: dual-agent (A: design review · B: detector + browser evidence, isolated and parallel).

## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | n/a | State carried by fill, shadow, dash; corner plays no part |
| 2 | Match System / Real World | 3 | Strong metaphors; the pill keycap abandons the rounded-square kbd convention |
| 3 | User Control and Freedom | n/a | A corner offers nothing to undo |
| 4 | Consistency and Standards | 3 | Test-enforced; docked because 10px now means stamp, photograph and fake button |
| 5 | Error Prevention | 2 | The day circle clips its own hit area; a dead day wears cursor: pointer |
| 6 | Recognition Rather Than Recall | 3 | Pill-means-pressable is near-total; the worked example teaches a shape no button has |
| 7 | Flexibility and Efficiency | 3 | Keycaps right; the opacity split is per-site, not a rule |
| 8 | Aesthetic and Minimalist Design | 2 | 31 rings on the phone picker; three nested curved strokes on the answer button |
| 9 | Error Recovery | n/a | Corners recover nothing |
| 10 | Help and Documentation | 3 | Token sheet, closure test, recorded retirements; the in-UI teaching surface shape-lies |
| **Total** | | **19/28** | **68% — Good** (prior run: 11/20 = 55%) |

## Where the assessments disagreed

A said the day cells became circles that clip their touch targets. B said they are stadiums, measuring ~98x76px on phone. B's number is arithmetically impossible: seven 98px columns need 686px in a 393px viewport; B reached the picker by injection and escaped the thread's column constraint. The CSS comment states the real geometry — "44px tall, and at 390px about 43 wide" — so on a phone the cells are square and 999px makes circles. Verified independently: a point 3px inside a 43x44 cell's corner hits DIV.calgrid, not the cell. ~21% of each cell is dead.

## Design Specificity Verdict

A designed vocabulary now, not a tidied one: four meanings plus one honestly unvalued category, held shut from both ends by a test. But the pill rule was made exceptionless by force-fitting a calendar cell and a wrapping search row. What the old sprawl was quietly doing and is now missing is an intermediate shape for "a pressable cell inside a larger structure" — the retired 9px day and 11px hit were both that, and both of this round's regressions sit where that meaning used to live.

Deterministic scan: the resolver defect still reproduces, [] on templates. Scanning a scratch directory gives zero radius findings for the wrong reason — findDesignRoot() finds no DESIGN.md there, so the design-system checks never activate. Proved properly by removing the three ignores: exactly three findings at pile.css:820 (7px), :858 (4px), :1709 (6px). Also: isAllowedRadiusRaw() passes any var( without resolving and any literal >=99px, so the detector has never validated the eight var(--r) declarations against 14px. TestTheRadiusVocabularyIsClosed is the only thing doing that.

## Priority Issues

[P1] The day circle clips its own touch target. .calgrid label.day (:1399) puts 999px on the hit element. Verified dead corners, ~21% of each cell, on the control reached one-handed to book a date. Fix: keep the label rectangular for hits, paint the circle on an inner element or ::before.

[P1] The worked example teaches a retired shape. .workedacts span (:1848) is 10px while every real button is a pill, and its comment says the shape is meant to be learnt. Also makes 10px mean a third thing, which the closure test cannot catch — it closes the value set, not the meaning map.

[P2] A dead day wears a pointer cursor. .calgrid span.gone (:1411) overrides background, border, colour and opacity but not cursor.

[P2] The pill rule over-reaches onto wrapping content. .hit button (:1785) is a full-width left-aligned two-to-four-line stadium, a silhouette drawn nowhere else.

[P3] The keycap opacity split is a patch wearing a rule's clothes. .72 fails 4.5:1 on any non-paper fill, not just orange.

## Persona Red Flags

Casey: the picker's dead corners, aiming at a bottom-row 31. Jordan: the screen built to teach button shapes shows the wrong one. Sam: hit-testing follows the radius, so this is an accessibility defect, not a fat-thumb one; .gone misreports a non-control. Alex: .dots i flexes to ~85px with a fixed 7px corner.

## Minor Observations

- The scrollbar thumb as a pill reaches browser chrome, and the thumb is genuinely draggable.
- .hit (search result) and .chip.hit (keyboard flash) are a grep-trap.
- The rigor is all in one channel: radius got a token sheet and two tests while the border system drifted to 2.5px dashed on a navigating control the same week.

## Questions to Consider

1. Is the real rule "a pill is a single-line label you press", and would stating it that way have kept the day grid and the search hit out honestly rather than by force?
2. Nothing locks border weights. Does every guarded channel just relocate drift?
3. Is 10px becoming the junk drawer — the token any shape borrows to hide from the test?
