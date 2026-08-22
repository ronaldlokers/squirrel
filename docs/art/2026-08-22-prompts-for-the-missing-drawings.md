# What to ask for: the two drawings this product is missing

Prompts to paste into an image model, plus what to reject when it answers.

Both items are now settled and neither is outstanding — one shipped, one
declined. This is kept as the record of what was asked for and what came back.

1. **A resting pose for `/enough`** (#98). The stopping screen currently uses
   the same `logo.png` as the empty pile, the empty chores, the empty tasks and
   the empty archive. Rendered side by side they are the same composition, the
   same drawing, the same sizes — only the sentence differs. **Choosing to stop
   looks exactly like having nothing left**, which is the one equivalence this
   product exists to break.
2. **One alternate per mood face** (#108). The five faces exist in one moment
   each, on the surface that most needs to be seen honestly every day.

---

## The style contract

Paste this at the top of every prompt below. It is what the existing art
actually is, read off the files rather than remembered.

```
Sticker illustration. Thick uniform near-black outline (#1c110b) around every
shape, roughly 3px at 260px wide. Flat saturated fills. Soft rounded geometry,
no sharp corners. No text, no letters, no numbers anywhere in the image. No
drop shadow on the ground, no background scene — transparent background,
subject only. Centred, with a little air around it. Friendly and plain rather
than detailed; this is a mark, not an illustration.
```

**Two things the models get wrong every time**, so say them again at the end of
the prompt:

- **Transparent background.** If it will not do it, ask for a flat magenta
  (`#ff00ff`) ground and cut it out afterwards — never white, because the
  drawings contain white.
- **No text.** Image models add labels unprompted, and there is not one word
  anywhere in this product's art.

### The palette

| | hex | what it is |
| --- | --- | --- |
| outline | `#1c110b` | every outline in the system, and never pure black |
| orange | `#e66d0d` | the fur, and the one action colour |
| cream | `#fed6a7` | the tail's tip |
| purple | `#6c4da9` | the cap, and the room |
| brown | `#58413d` | the headphones and the acorn |
| paper | `#fffbf3` | the face and the raised things |

---

## 1. The resting pose, for `/enough` — **done, 23 August 2026**

*Shipped as `internal/web/static/resting.png`, 600×413, drawn at 300×207. The
prompt below is what produced it, kept for the companion pose and for whenever
it needs redrawing. What came back needed no cutting out — the model honoured
the transparent background — and one thing it did not honour: the fur is
modelled rather than flat, which was accepted and is recorded in DESIGN.md
rather than smoothed over.*

**Canvas:** 744 × 560 px, transparent PNG. It is drawn into a fixed 186 × 140
slot, so the aspect matters more than the size — anything at 4:3-ish will do,
anything squarer will letterbox.

**The moment it has to carry:** *that will do.* Not tired, not finished, not
pleased with itself. Stopping is a normal ending here, and a drawing that looks
relieved makes stopping into something you needed rescuing from, while one that
looks proud makes it an achievement. It wants to look like somebody who has put
a thing down and is fine.

```
[paste the style contract]

Draw a cartoon squirrel mascot at rest, seen from the front, head and
shoulders.

The character: bright orange fur; a big bushy orange tail with a pale cream tip
curling up behind one shoulder; two pointed orange ears; a flat purple
newsboy-style cap sitting low, with a small brown acorn badge on the front and
a short brown sprout with a round tip standing up from the crown; round brown
over-ear headphones. Its face is a white rounded speech-bubble shape — a
speech balloon used as a face — with two small oval black eyes and a small
simple smile.

The pose: leaning one elbow on the closed lid of a plain cardboard shoebox that
sits beside it, cap tipped slightly forward over the eyes, tail curled around
and settled rather than raised. Eyes soft and half-closed, mouth a small even
line, not a grin. Calm and unhurried — someone who has put something down, not
someone who has collapsed.

Transparent background. No text of any kind.
```

**Reject it if:** the eyes are closed all the way (that is asleep, and asleep is
a different screen), it is smiling broadly (that is congratulation, which this
screen must never do), the shoebox is open, there is a check mark or a tick
anywhere, or the tail is doing something energetic.

**Optional companion — the listening pose, for the true empty states.** Worth
having so the two screens stop being interchangeable from both directions, but
`/enough` is the one that matters.

```
[paste the style contract]

[paste the character paragraph above]

The pose: sitting upright and alert beside an open, completely empty cardboard
shoebox, looking slightly off to one side as if waiting for something to be
said. Ears up, one hand resting on the box's rim, tail up behind. Attentive and
patient. Not sad, not searching — the box being empty is fine.

Transparent background. No text of any kind.
```

---

## 2. One alternate per mood face — **not doing it, 23 August 2026**

Declined by the owner, who likes the five as they are. Recorded here rather
than deleted, because the reasoning that asked for it is still true and
somebody will ask again.

**What was asked for and why.** The five faces exist in one moment each, on the
surface that most needs to be seen honestly every day, and `PRODUCT.md`'s own
premise is that a surface which looks identical every time stops being seen
within about a week. #139 answered that everywhere else — the stamp's angle,
the field's light and the sentences met most often all come from the date now.
The faces are what is left fixed.

**The cost of the decision, stated so it is not a surprise later.** If the
check-in does start going unseen, this is the first thing to try, and the
absence of variation here is now a choice rather than an oversight. The free
version — a per-render couple of degrees of tilt, no art at all — was also
declined by the same answer; it is a smaller thing than a second drawing rather
than a cheaper version of it.

**If it is ever revisited**, the brief is harder than it looks and the first
attempt at it got this wrong: an alternate has to keep the mood exactly — `low`
must never read as `wiped`, since those are different states wanting different
answers, which is why the five are not a one-to-five scale — while being
visibly a different drawing met one at a time. Same colour, same tile, same
outline weight, same motif family; only the expression moves. A prompt written
by describing the faces produces five near-copies, which is what happened here.

## When the files arrive

Drop them in `internal/web/static/` under the names above and say so. The
wiring is small and is not written yet, deliberately: a template pointing at a
file that does not exist is a broken build, and a placeholder drawing is worse
than the repetition it replaces.

- `/enough` swaps its `src` and keeps `class="empty"`, so the layout does not
  move. Same decorative `alt`, because the drawing says which moment this is
  and a screen reader gets that from the sentence.
- The faces pick between the two per render, the way the card stack picks its
  angle and the sentences pick their wording — from the date, so both viewports
  agree and a reload is not a slot machine. See The Day Rule in `DESIGN.md`.
