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

- Capture happens in **both** places. It was Campfire only for the screen's whole
  life, and the reasoning is worth keeping rather than deleting: two capture
  surfaces means two places to look for a thought, which is the problem the
  product exists to solve. The owner overruled it on 20 August 2026, choosing a
  slot on this screen over a relay through Campfire. What makes that survivable
  is that both surfaces write the same row to the same table through the same
  fsynced spool — one pile with two doors into it. What it costs is that the
  Campfire room stops being the complete record.
- Reached at `squirrel.ronaldlokers.nl`, LAN-only via a Traefik ipAllowList,
  behind Authentik forward-auth. Runs in a homelab Kubernetes cluster.
- **The phone is on an always-on VPN into the homelab**, which is what makes
  LAN-only compatible with the second usage scene. Recorded 22 August 2026
  because it had never been written down and the two halves of this document
  contradicted each other without it: the screen is reachable only from the
  LAN, and one of the two confirmed scenes is a phone in a queue. It is a
  precondition of "phone primary" rather than an incidental fact about the
  network, and if it ever stops being true the answer is an architecture
  decision, not a paragraph. Without it: capture still survives, held in the
  service worker until the phone is home; the pile is unreadable; and a push
  notification tapped in that queue opens a page that refuses to load.
- The same notes are also reachable from chat: `!notes`, `!find <text>`,
  `done <n>`, `keep <n>`, `drop <n>`, `!chore <n> every <interval>`. The screen
  and the chat commands are two views of one pile and must agree.
- An evening message at 19:00 lists what was captured since yesterday and what
  was completed today. That window asks when a note arrived, not what became of
  it, so a triaged note still appears there.

## Capabilities and Constraints

**Confirmed functionality for this surface:**

- One screen, no navigation. Notes newest-first.
- **One question per note, and the answers behind it.** The card asks *what is
  this?*; pressing it reveals four answers — **done · keep · drop · a task** —
  and one thing you can make from it, **make it a chore**, which takes an
  interval. What stalls is not which of the five, it is whether you are
  deciding about this thing at all right now, so that is what the card asks.
  Skipping and the letter keys both work from a shut card; correcting the words
  sits outside the question, because it is a repair rather than an answer.
- **Stopping is a place.** "stop whenever you like" is a link to a screen that
  says this was a normal way to finish. Chosen, never triggered: a screen that
  appeared after four cards would be a screen with an opinion about how many
  cards are enough, and that number would be a count wearing a kind face. It
  reads nothing and reports nothing about what you did.
- Search across every state, on the same screen.
- Undo lives on the screen, and a row stays in place for a moment after it is
  actioned so the undo has somewhere to be.
- Keyboard-first: move between notes, one key per action.
- **Buddy**, behind the acorn on every screen: a conversation about what is
  on that screen. Chrome rather than a fourth door — home still has three. It
  is a real page (`/coach`), so it works with scripting off and survives a
  reload; the sheet is an upgrade over that, not a requirement. Opening it
  costs nothing and calls no model. Four one-press chips mean typing is never
  required, which is what makes it usable at the moment of least capacity —
  the moment it exists for. Closing it means the conversation is over and
  means nothing else: nothing is counted and the acorn never dims.

**Constraints that are not negotiable:**

- **Never a count.** No badge, no total, no "N to review", no page count. A
  capped list may say *that* there is more, never *how much* more. This is the
  single rule most likely to be broken by accident.

  **One exception, added 20 August 2026: what the coach has cost this month.**
  It sits in the coach sheet's own lid and on no other screen. The rule bans
  the number that accrues against your work and implies a target of zero; this
  is money, it is a fact about a machine rather than about you, and it is
  bounded by a ceiling that was set on purpose rather than open-ended. What it
  buys is that the ceiling stops being invisible until the month it is reached
  — before this, the only way to know what the coach cost was SQL. It is an
  exception rather than a precedent, and it is the reason no running cost
  appears on home.
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

**A photograph instead of typing.** The case is a letter, a serial plate, a
parking sign: the moment you want this is usually the moment you are stood in
front of the thing holding a phone. But **camera-first is not camera-only** —
the letter you photographed this morning is the same case one hour later, and
the browser's own chooser offers both. One photograph per note — a note that can
carry five is an album, and an album is a thing you organise rather than glance
at. **A photograph with no words is a note**, which is most of the point of
having a camera.

**A photograph you have chosen is already a capture.** It is shown back before
it is kept, and it is held on the device the moment it is picked — choosing one
hands the screen to another app, and an app handed away can be reclaimed before
it comes back. Capture is sacred one layer further out than it used to be.

They live on a volume beside the pod rather than in object storage: this
cluster has none, and adding some would mean a service to patch, back up and
keep alive for one feature. What that costs is stated rather than discovered —
photographs join the pod's lifecycle, and **the restore drill has to cover the
volume**. Nowhere to put them is a supported state: with no volume the camera
is never offered.

**The shelf may speak, once in a while.** A kept note can ride along with the
evening message — never its own message, never more than one, and never when
there was nothing else to say. A shelf that taps you on the shoulder is a
second inbox, which is the thing this product exists not to have. Roughly one
evening in three, chosen at random rather than from a queue: a queue would give
the shelf a front, and a front is a place to be behind.

**Three ways to say you cannot act on it:** *waiting on someone*, *blocked on a
thing*, *someday*. Three rather than one because they end differently — a
waiting-on ends when somebody replies, a blocked-on when something arrives, and
a someday only when you say so. One "parked" state would make *chase the vet*
and *learn to solder* the same kind of thing, which is how a someday list
becomes a guilt list.

Each carries what would move it, in your words. Nothing counts them and nothing
ever will: a number beside stalled work is a reproach, and the point of setting
something aside is to stop being asked about it. They are read on their own
page, reached from the bottom of the tasks — never a fourth door on home, which
would put them back in front of you.

**Two kinds:** a **note** is a thought you had; a **task** is a thing you
decided to do, once. The same row at different moments — promoting a note takes
it out of the pile, because the pile holds what you have not decided about and
that is what makes triage mean anything. A task that is done is archived, not
deleted, and every step of that reverses.

**Four states:** `open` (the pile), `done`, `dropped`, `kept`. `kept` is
load-bearing — a serial number or a link is not a task and will never be done, so
without it every reference note sits in triage forever.

**Feature parity relaxes in one direction, and only one.** Amended 22 August
2026, because the rule as written and the roadmap had been contradicting each
other since 20 August: the roadmap recorded that parity relaxes to best-effort
in one direction, this file said it was absolute, and by this project's own
precedence this file wins — so the relaxation was not actually in force, and
the screen-only brain-dump split had been a live breach of the record the whole
time.

What the original rule was protecting is kept, and it is Principle 4: two views
of one pile that *disagree about what a note is* are a bug in the product.
Nothing in this amendment touches that. What is retired is the stronger reading
— that the two views must be able to *do* the same things — which was never
what Principle 4 said and had begun forcing chat twins nobody asked for.

So:

- **The screen may do things chat cannot.** A brain-dump split, a photograph
  chosen from the camera, a picker chip: these are gestures, and a gesture is
  not a sentence you can type into a room.
- **Chat may not do things the screen cannot.** That direction stays closed.
  The screen is the surface being made primary, and a command with no home on
  it would be a feature you have to remember a room for.
- **Whatever either does, both must read the same.** State is not a feature.
  A note set aside from the card is set aside in chat's eyes the moment it
  lands, and the reverse.

**The floor chat keeps forever**, whatever else moves to the screen — because
this is the surface that works when the other one cannot be reached at all:

  1. Capture. Typing a thought into the room keeps it, and always will.
  2. Answering a nudge — `did`, `later`, and the leave-by chain.
  3. The four dispositions: `done <n>`, `keep <n>`, `drop <n>`, `!fix <n>`.
  4. Setting one aside: `!waiting`, `!blocked`, `!someday`.
  5. Reading the pile back: `!notes`, `!find <text>`.

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
5. ~~**Say nothing about the person.**~~ **Opened by the owner on 20 August
   2026.** The coach may evaluate, compare, and mention counts and streaks in
   what it says. What the rule protected against is still real — the accruing
   number on a surface — and rule 2 is what protects against it; this one was
   about *speech*, and it was making the coach useless at the only thing a
   coach is for. What it costs, and it is not nothing: the coach is now allowed
   to say something that lands badly on a bad day. `coach_answers` keeps every
   exchange for exactly that reason — so "it was tactless" can be told apart
   from "I remember it as tactless".

   **And as of 22 August the record is read back.** One press on the sheet says
   the last thing Buddy said did not land, and the last few of those are shown
   to the model as examples of what does not work here. Examples rather than an
   instruction, because an instruction nobody can check is a wish. Never a
   count: how often something lands badly is a fact about the person, and rule
   2 forbids one on every surface — including the prompt, which is a surface
   the person never reads.
6. **Squirrel chooses, and can say why.** One thing at a time, by rules that
   are fixed, readable and the same every time. An offer nobody can account for
   is a demand.
7. **The readings are yours to look at, and nobody else's to use.** The
   check-in was unreadable by construction for the product's whole life: the
   store returned one reading and no function could return more, which is
   stronger than a rule someone has to remember. **That guarantee was given up
   on 20 August 2026** and replaced with a narrower one — there is one page and
   one command, both of which you have to go to by name. Home shows today's
   answer and a link, never a series. Nothing else reads them: not the evening
   message, not the picker, not Buddy, which is handed "ok" or "low" derived
   from a single reading and cannot ask for more. And nothing anywhere totals,
   averages or compares them. What it means is yours; the product only hands
   back what you said.
8. **Anything a model wrote is Buddy's; anything the rules produced is
   Squirrel's.** Buddy is the name of the thing you talk to, and the line is
   about authorship rather than about features. The picker's own clause, the
   ladder's fixed sentences and a nudge that fires because a chore is due are
   Squirrel. The clause a model chose, the steps it broke a thing into, the
   wording it gave a nudge and every word in the sheet are Buddy. Nothing
   renders that distinction — you are not shown a label saying which — but it
   is what decides whose voice a sentence is written in, and it is why the
   deterministic floor never sounds like a degraded Buddy. It is Squirrel,
   speaking as it always did.
9. **Squirrel does not speak first at a bad moment.** Nothing arrives unasked
   between 22:00 and 06:00, and nothing is raised on a low day. Both are floors
   under the one path that speaks without being spoken to; neither can be
   lifted, including by the coach, which is asked afterwards or not at all.
   The chore's own clock keeps running while it waits, so nothing is lost —
   only the asking stops. A message you *chose* the hour of, like the evening
   one, is not covered by this: it is not arriving unasked.
10. **The deterministic answer is never deleted; it becomes the floor.** Every
   place a model speaks has a fixed answer underneath it that shipped first and
   keeps working — the picker chooses, the ladder answers, the asking windows
   decide when to interrupt. No key, no network, or a month's budget spent must
   leave a product that works exactly as it did before the model existed. This
   is what makes the coach safe to add and safe to switch off.

## Accessibility & Inclusion

Built for an ADHD brain, which sets requirements beyond the usual baseline:

- **Decision load is the scarce resource.** Self-regulation draws on a depletable
  pool and every choice spends it. Fewer decisions per note beats more power.
- **Habituation is the enemy.** A surface that looks identical every time stops
  being seen within about a week.
- Keyboard operation is a first-class path, not a fallback.
- The screen is used on a phone in poor conditions as often as at a desk.
