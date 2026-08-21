# One frame — design brief

*21 August 2026. Produced by `/impeccable shape` after "the app feels super
messy… on the stack, there is buttons, but also links. It's a mess" and "I
don't want to guess on fixes anymore."*

**Status: awaiting confirmation.** No code follows this until it is approved,
and a clickable comp follows the approval before anything touches the app.

---

## 1. Job and audience

One person, Ronald, who has ADHD, on thirteen screens of a product that is his
external memory. Two situations and no others — **phone in gaps**, one thumb,
interrupted; and **desktop deliberately**, keyboard available, triage as a
chosen task. A tablet must not break, and is not designed for.

Mode: **Operate**. He is completing a task, not being persuaded of anything.
Scanability, consistency and native expectation outrank expression; the brand
lives in precise details, which is where it already lives well.

## 2. Outcome and proof

The screens stop reading as thirteen things designed separately. Two tests of
that, both observable:

- He can tell what a control will do **before** pressing it.
- Every screen opens the same way, so arriving somewhere new costs nothing.

**Product truth that constrains every choice here:** nothing accrues that can
be destroyed. No counts, no streaks, no completion, no "N outstanding" — and
that now explicitly includes titles, which is the one place a number would
look most reasonable.

## 3. Selected direction

A **refinement**, not a redesign. The world stays exactly as it is: the
shoebox, the two type axes, the sticker shadows, the four state colours, the
mascot, the copy. The material was measured and is already consistent — 22 of
the radii are `999px`, 17 of the shadows are one value, 29 tap targets are
44px. Nothing about the look is the problem.

The thesis is two sentences.

**One frame.** Every screen is title → content → the way onward, at one width
with one vertical rhythm. Today there are five heading treatments, ten of
thirteen templates with no `<h1>` at all, and two different empty-state
grammars. A screen that opens the same way as the last one is most of what
"coherent" means to someone looking rather than measuring.

**Three kinds of control, one mark each.**

| pressing it | means | mark |
| --- | --- | --- |
| button | something changes, on the server | filled, bordered, the hard shadow it already has |
| link | you go to another screen | underline, and underline means nothing else |
| disclosure | something opens here; nothing changes | a chevron that turns |

Today all three are underlined text in places — `LATER →` goes somewhere,
`fix the words` opens in place, `i can't act on this` opens in place, and they
are indistinguishable. Meanwhile disclosures are not consistent with each
other either: a filled pill on chores, a `+` on the new-chore form, plain
underlined text on the card. Four appearances for one idea.

## 4. The part that is not cosmetic

**Half the screens are reachable from exactly one place, and one of them is
reachable only by emptying the pile.**

| screen | reached from |
| --- | --- |
| pile · tasks · chores | the lid, always — two of the three, plus home's doors |
| **kept** | **the bottom of the pile, or an empty pile** |
| held | the tasks screen only |
| archive | the tasks screen only |
| moods | the check-in on home only |
| enough | the pile's footer only |

"It's weird to get into the kept notes" is not a feeling. The shelf of things
he deliberately kept for later is behind having nothing left to triage —
which is the one state the product exists to tell him he need not reach. The
same shape, less severely, hides held, archive and moods.

This brief treats reachability as part of the layout, because a frame that is
identical everywhere and still cannot be navigated to has solved the smaller
half of the problem.

## 5. Scope and boundaries

**In:** all thirteen templates; the frame; the control vocabulary; how each
screen is reached.

**Untouched, and the comp must show them unchanged:** the deck's composition —
card, stack, stamp, the 1150ms hold, the undo that lives on the card; the
palette and both type axes; the mascot and door art; every word of copy except
where a heading's case changes; the capture slot's behaviour, which was fixed
last night and works.

**Anti-goals.** No counts anywhere, including titles. No new colours. No
component library, no build step, no framework — this binary has none and does
not want one. No second visual world; if a choice here would look better in a
different world, it is the wrong choice.

**Open, and a builder must not invent an answer:** whether the card keeps its
question gate; whether a wide desktop gains a second column rather than a
720px strip on a 1280px page; whether chores survives now the picker exists.
All three are real questions and none of them is this.

## 6. States and ranges

Every screen at: empty, one item, a typical few, and more-than-fits. Plus the
states that only exist in failure — the four capture messages, the offline
hold, a photograph refused, the 503 that currently drops the design system
entirely.

Realistic ranges: the pile holds tens, not thousands. Chores are single
figures. A note is a sentence; occasionally a paragraph; sometimes only a
photograph and no words at all.

## 7. Interaction, layout, constraints

One column, `min(720px, 100%)`, both situations — the phone is the column at
full width. Frame slots: title, content, onward. Feedback happens where the
action was, which the capture slot now does and nothing else does yet.

Binding constraints, all of them already true and none negotiable:

- **Progressive enhancement is absolute.** Every action is a form or a link
  that works with JavaScript switched off. The script may only make it nicer.
- 44px minimum on anything pressable; AA contrast on every text colour.
- `[hidden]` means hidden — enforced globally as of this morning, after the
  same bug hid three separate defects in one week.
- Keyboard letters keep acting on the deck; `space` keeps skipping.
- One heading element, one class, on every screen that is a screen.

## 8. What happens next, in order

1. **A clickable comp** — one self-contained page showing the frame across
   every screen and every state, at phone and desktop, opened and judged
   before anything else happens.
2. On approval: the frame, then the vocabulary, then reachability — each its
   own release, to staging, looked at on a real phone before it goes further.

The order and the staging step are written down because the alternative was
tried on 20 August: four screens' worth of design went to production together,
none of it had been used by anyone, and the app came back unusable.
