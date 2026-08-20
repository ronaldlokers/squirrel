# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

One person: Ronald, the owner and only user. He has ADHD, and the product exists
because of how that specifically fails him — not as a general productivity tool
that happens to suit him.

Two situations, both confirmed, and the design has to serve them equally rather
than optimising for one:

- **Desktop, deliberately.** Sat at the machine, keyboard available, triage as a
  chosen task.
- **Phone, in gaps.** On the sofa, in a queue, one thumb.

A successful session is **a few notes cleared, then stopping.** Not emptying the
pile. Stopping partway is the normal ending, not an abandoned job.

## Product Purpose

Squirrel is an external memory that lives inside a Campfire chat room. You type a
thought and it is kept; it tells you about a chore when it thinks the moment is
right.

This surface — the pile — is the first part of Squirrel you look at rather than
talk to. Until v0.5.0 a note was write-only: shown once in the evening message
and never again. The pile is where notes are read back, searched, and disposed
of.

Success is that opening it does not feel like opening an inbox you are behind on.
The measure is whether it gets opened again next week.

## Positioning

Squirrel is built around a single rule that a neighbouring product could not
copy without abandoning its own metrics: **nothing accrues that can be
destroyed.** No streaks, no counts, no completion percentage, no badge of how
much is outstanding.

That rule comes from the ADHD literature this project is designed against —
loss aversion makes losing hurt about twice as much as the equivalent gain
pleases, an all-or-nothing counter makes one miss read as total failure, and the
abstinence violation effect turns that into abandonment. Every competitor in this
category sells the counter as the feature.

## Operating Context

- Capture happens **only** in Campfire. This screen never accepts a new note, and
  that is permanent rather than a current limitation: two capture surfaces means
  two places to look for a thought, which is the problem the product exists to
  solve.
- Reached at `squirrel.ronaldlokers.nl`, LAN-only via a Traefik ipAllowList,
  behind Authentik forward-auth. Runs in a homelab Kubernetes cluster.
- The same notes are also reachable from chat: `!notes`, `!find <text>`,
  `done <n>`, `keep <n>`, `drop <n>`, `!chore <n> every <interval>`. The screen
  and the chat commands are two views of one pile and must agree.
- An evening message at 19:00 lists what was captured since yesterday and what
  was completed today. That window asks when a note arrived, not what became of
  it, so a triaged note still appears there.

## Capabilities and Constraints

**Confirmed functionality for this surface:**

- One screen, no navigation. Notes newest-first.
- Four actions per note: **done · drop · keep · make it a chore.** The chore
  action takes an interval.
- Search across every state, on the same screen.
- Undo lives on the screen, and a row stays in place for a moment after it is
  actioned so the undo has somewhere to be.
- Keyboard-first: move between notes, one key per action.

**Constraints that are not negotiable:**

- **Never a count.** No badge, no total, no "N to review", no page count. A
  capped list may say *that* there is more, never *how much* more. This is the
  single rule most likely to be broken by accident.
- ~~**Never a capture box.**~~ **Overruled by the owner on 20 August 2026.**
  The screen now captures, and it writes straight to the pile rather than
  relaying through Campfire. The rule existed because two capture surfaces
  means two places to look for a thought; what makes this survivable is that
  the two surfaces write to the same table and the same list reads it back, so
  there is still one pile. What it costs, precisely, and it is not nothing:
  the Campfire room stops being the complete record of everything you told
  Squirrel, and a note captured on the screen has no spool behind it, so an
  unreachable database is a note that was never taken. The screen therefore
  fails loudly and keeps the words on the page rather than clearing the field.
  The alternative — posting into the room and letting the webhook bring it
  back — was offered and declined.
- Every state transition is reversible, and repeating one is a no-op rather than
  an error.
- Postgres is on this screen's request path and that is correct — the spool
  invariant protects capture, not retrieval. If the database is down the screen
  fails visibly and nothing is lost.

**Terminology:** the untriaged notes are called **the pile**. Considered and
rejected: "inbox", which imports inbox-zero and its counter. The owner chose
"the pile" over the objection that it may read as self-deprecating.

**Two kinds:** a **note** is a thought you had; a **task** is a thing you
decided to do, once. The same row at different moments — promoting a note takes
it out of the pile, because the pile holds what you have not decided about and
that is what makes triage mean anything. A task that is done is archived, not
deleted, and every step of that reverses.

**Four states:** `open` (the pile), `done`, `dropped`, `kept`. `kept` is
load-bearing — a serial number or a link is not a task and will never be done, so
without it every reference note sits in triage forever.

**Feature parity is a standing rule.** Anything you can do in chat you can do
on the screen, and the reverse. It is Principle 4 generalised: two views of one
pile that can do different things are two views that will eventually disagree
about what the pile is.

**Decided on 20 August 2026:** a note's text can be corrected — `!fix <n>` in
chat, "fix the words" on the card. Only the words change; the arrival time, the
state and the place in the pile stay, because those are facts about the note
and only the sentence was wrong. Not versioned: keeping the old text would make
a note a document with a history to read, which is a second place a thought can
hide, and the thing being corrected is usually a typo.

**Explicitly undecided:** tags, folders and projects; full-text ranking;
resurfacing old notes unprompted (rejected, not deferred —
it would become a second stream competing with the nudge for the same attention).

**Decided on 20 August 2026: Squirrel chooses.** Every surface was organised by
what kind of thing a row is and none of them by whether it deserves attention
now, so opening the product while overwhelmed asked which of three boxes to
open first. There is now one offer — one thing, chosen by six deterministic
rules in a fixed order, carrying one clause that explains itself. It is on home
above the doors, and it is `!now` in chat.

The rules, in order, and the order is the design rather than an implementation
detail: a fixed point inside its leave-by window; the timer you are already
running; what you were on before you got up; a chore that is due and inside its
asking window; the oldest thing you decided to do; nothing. When the picker is
wrong you can read six rules and see why, which is not true of anything that
scores, learns or generates.

**The check-in is an input now, not a record.** A fresh *wiped* or *frazzled*
reading drops the two rules that are Squirrel's own initiative and keeps the
three that are the world's business and yours. *Low* is deliberately not one of
them: low is how you feel and those two are how much you have, and an empty day
handed to someone flat but functional reads as the product agreeing they are
finished. A quiet *show me anyway* lifts it once and is remembered nowhere.

**Refusing is one press, has no consequence, and is never followed by a
question.** It lasts the day, it reverses, and it touches nothing about the
thing itself — a chore turned down in the picker is exactly as due as it was,
because the picker's memory and the nudge's budget are two different things.

**Decided on 20 August 2026: Squirrel may hold a time the world imposed.** The
deadline rule is sharpened rather than broken, and the new wording is the one
to hold everything against:

> Squirrel never invents a time you can be late for. It may hold one the world
> did.

A dentist appointment at 14:30 exists whether or not this app knows about it,
and refusing to hold it does not remove the lateness — it only means Squirrel
cannot help you leave. What is still refused: nothing is ever marked late,
nothing accrues, there is no recurrence (that is a chore), and there is no list
screen, because a browsable set of your appointments is a calendar and a
calendar is a thing you are behind on. A moment is shown only inside the window
where leaving matters, and afterwards it is simply over.

**Decided on 20 August 2026: one count is allowed, and only this one.** The
evening message may say *three notes cleared*. The banned counter counts what
*remains* — it grows while nobody is looking, sits beside an implied target of
zero, and can be lost. This counts what happened, on one day, in the past. If
that reading is ever rejected the line becomes *some notes cleared* and nothing
else moves.

**Decided on 20 August 2026: one thing may interrupt you, and only one.** The
leave-by warning goes to the room and to the browser. Nothing else is ever
pushed — a nudge is a suggestion, and a suggestion that waits is doing its job.
The permission is asked for by a quiet line you have to go and press, never on
load, and once answered either way the control disappears.

**Decided on 20 August 2026: knowing what to do is not being able to do it.**
Every offer carries a quiet way to say you cannot start it, and four answers
behind it — too big, don't know how, boring, not today. Each produces one
sentence and at most one control; there is nowhere in the answer to put a
second step, which is what makes the twelve-step productivity reply impossible
rather than merely discouraged. *No energy* is absent because the check-in
already asks it, and *anxious* is absent because it invites a therapeutic
response this product should not attempt and its useful action is already the
first option.

## Brand Commitments

The name is Squirrel. In chat it speaks in short, plain sentences and never
scolds. Two emoji are load-bearing information rather than decoration: 👀 means
the thought reached disk, ✅ means it reached the database. A varied reaction —
🎉 ✨ 🙌 💫 🌟 — is drawn at random when a chore is completed, deliberately
unpredictable and deliberately non-cumulative.

**The mark is the visual authority, and it was redrawn on 18 August 2026.** Two
files, and they are not interchangeable:

- `assets/logo.png` — transparent, wide. Head and tail, a purple cap badged
  with a white acorn, a loose acorn above it, brown headphones, and a face that
  is a speech bubble because this is a chat bot. **This is the one that goes on
  a coloured surface.**
- `assets/avatar.png` — square, and it carries the notebook, but its white
  background is baked in. Chat avatar only; it cannot sit on the purple.

The drawing language is unchanged: a `#1C110B` outline around flat saturated
fills, soft rounded geometry, no gradients. The outline is darker than the
`#2A1D19` this record used to claim.

| | |
|---|---|
| purple | `#6C4DA9`, deep `#58388A`, field `#3B2560` |
| orange | `#E66D0D` |
| outline | `#1C110B` |
| cream (the tail) | `#FED6A7` |
| brown (the headphones) | `#58413D` |
| white | `#FEFEFE` |

Purple and orange are not a preference to be honoured, they are the product's
own colours. Any surface that reads as neutral or white has failed to be
Squirrel — including card stock, which is the tail's cream lightened, never
white.

Three things the redraw settles that were previously guesses:

- **The acorn is the product's second mark.** It is on the cap in the drawing,
  so it is available as a badge anywhere the full mascot is too much.
- **The notebook's page tabs are green, violet, amber and cream — four tabs,
  and there are four states.** `done` is green `#529414`, `kept` is amber
  `#FFB300`, `dropped` is the headphone brown `#8A6A55`, and a chore takes the
  mark's own orange because it is the one action that makes something rather
  than disposing of it. Each has a darker sibling for type, because the tab
  colours are picked to be seen and type has to be read.
- **The cap has a brim**, which is why a header on this product can be a lid.

**Requested feeling for this surface: warm and personal.** It holds the owner's
own words, so it should read closer to his handwriting than to a system's
database view. That sits in productive tension with the keyboard-first mechanics,
which pull toward a precision tool — the design has to be both, not average them.

## Evidence on Hand

- Spec: `docs/superpowers/specs/2026-08-18-pile-design.md` — the binding source
  for this surface's behaviour.
- Working chat implementation of the same pile: `internal/squirrel/notes.go`,
  `render.go`, `apply.go`.
- Real content is the owner's own notes, already in production. No sample data
  exists in the repository and none may be fabricated as though it were his.
- No screenshots, no imagery, no photography, no testimonials, no press. Nothing
  to cite and nothing to imply.

## Product Principles

1. **Capture is sacred; everything else is a view over it.** No feature may make
   a thought harder to record or easier to lose.
2. **Nothing accrues that can be destroyed.** No counter, no streak, no
   percentage — not on any surface, in any form.
3. **Stopping partway is a normal ending.** The design must never make leaving
   look like failure.
4. **Two views, one pile.** Chat and screen must agree about what a note is and
   what state it is in; a disagreement is a bug in the product, not a difference
   in surface.
5. **Say nothing about the person.** Report what was said and what happened.
   Never grade, rank, compare, or imply a norm.
6. **Squirrel chooses, and can say why.** One thing at a time, by rules that
   are fixed, readable and the same every time. An offer nobody can account for
   is a demand.

## Accessibility & Inclusion

Built for an ADHD brain, which sets requirements beyond the usual baseline:

- **Decision load is the scarce resource.** Self-regulation draws on a depletable
  pool and every choice spends it. Fewer decisions per note beats more power.
- **Habituation is the enemy.** A surface that looks identical every time stops
  being seen within about a week.
- Keyboard operation is a first-class path, not a fallback.
- The screen is used on a phone in poor conditions as often as at a desk.
