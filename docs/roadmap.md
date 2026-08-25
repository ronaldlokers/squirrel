# Roadmap

What is built, what is decided, what is refused. Last reconciled **20 August
2026**.

This is an index of *state*, not of reasoning. Every decision here was argued
somewhere else, and the argument is the useful part:

| Document | What it holds |
| --- | --- |
| `docs/proposals/2026-08-20-executive-function.md` | The audit that started this: what the product could and could not do, graded against six ADHD pillars, and the P0–P3 roadmap it produced. Marked shipped for phases A–G. |
| `docs/proposals/2026-08-20-coach-architecture.md` | The coach: 21 architectural decisions, the model routing matrix, the cost model, and what a brief written without access to the codebase got wrong about it. |
| `PRODUCT.md` | The binding record of product rules. A rule here is enforced somewhere in the test suite. |
| `DESIGN.md` | The design system, including the exceptions it has granted and why. |

**When this file and a proposal disagree, this file is newer.** When this file
and `PRODUCT.md` disagree, `PRODUCT.md` wins — it is the record, this is the
plan.

---

## Shipped

### v0.13.0 — 20 August 2026

Phases A–G of the executive-function proposal, live in both clusters. Four
migrations, applied on first boot, zero errors.

Squirrel stored well and chose nothing. Every surface was organised by *what
kind of thing a row is* and none of them by *whether it deserves attention now*.
That is what this release changed.

| | |
| --- | --- |
| **The picker** | Six deterministic rules, fixed order, ~1 ms, offline, one clause of explanation. `internal/squirrel/pick.go` |
| **Capacity** | The check-in became an input. A fresh *wiped* or *frazzled* drops the two rules that are Squirrel's own initiative and keeps the three that are the world's business and yours. *Low* is deliberately not a capacity word. |
| **The offer on home** | One thing above the doors, sharing its region with the check-in. Absent, never empty. |
| **"I can't start"** | Four answers, each one sentence and at most one control. `stuck.go` |
| **The hand-off** | One more thing on the message that says you finished something. Never after triaging a note, never mid-timer, never on a low day. |
| **Re-entry** | The timer's row survives its own ending by an hour. Never mentions finishing. |
| **A fuller evening** | Chores *and* tasks *and* cleared notes. |
| **Fixed points** | `at 14:30 dentist`, the leave-by chain, `!bring`, `!leaving`. |
| **Web Push** | RFC 8291 + 8292 against the standard library. No new dependency. |
| **Two floors under the nudge** | Nothing arrives unasked between 22:00 and 06:00, and nothing is raised on a low day. |

**Rules that moved to allow it**, all recorded in `PRODUCT.md`:

- *Squirrel never invents a time you can be late for. It may hold one the world
  did.* — the carve-out that made fixed points buildable.
- Home may carry one **actionable** offer. This overrides one of the seven
  prohibitions written when the old preview was cut; the other six hold.
- One count of what **happened** is allowed (*three notes cleared*). The banned
  counter counts what remains.
- A sixth principle: **Squirrel chooses, and can say why.**

---

## Buddy — all eight phases shipped

Chosen over the personalisation half on the grounds that durations and learned
timing are data-starved until the product has been used for a month. Built A
through H in one run on 20 August 2026; every phase is a merged pull request
with its own argument written out.

**What it actually is, in one paragraph.** An acorn on every screen opens a
conversation with **Buddy** about what is on that screen. It can hand you one thing and say
why, break something into steps you are shown one at a time, take five things
at once and answer with one, separate a brain dump into the things it was, do
six things on your behalf, and ask before doing four more. Once a day, when a
chore is already due and already inside its window, it may decide not to
mention it.

**What did not change.** Capture. No model runs on the path a thought arrives
through, and none ever will. Nothing rewrites your words. Nothing counts.
Nothing deletes. A buzz still means one thing.

**21 decisions are settled** — see the coach proposal for each argued in full.
The load-bearing ones:

- The model decides what-now; **`PickNow()` becomes the fallback**, not a
  deletion.
- Retrieval tools from day one, with a written trigger to revisit
  (context past 10,000 tokens).
- Write tools with a confirmation policy grounded in existing product rules:
  runs directly if already reversible in one press *and* already a button; asks
  first if it creates a future interruption or disposes; **never** `reword` —
  that rewrites your own words.
- A widget on every screen: the **acorn**, bottom-right, opening a sheet over a
  real `/coach` route. Home keeps its three doors, because the button is chrome.
- Opening it costs nothing — it paints with the cached offer.
- Push stays reserved for fixed points, so **a buzz always means the same
  thing**.
- **≈€3.72/month**, about 37% of the €10 ceiling.

**The principle underneath all of it:** every deterministic answer handed to a
model — the picker, the ladder, the asking windows — is kept as the floor.
Nothing is deleted. That is what makes eight AI-forward decisions safe in a
product whose value is that it works.

### Phases

| | | |
| --- | --- | --- |
| **A** | Skeleton: interface, `NoCoach{}`, guard, budget counter, config, `coach_answers`. No provider. **Needs no key.** | shipped |
| **B** | Luna behind it, `!coach` in chat only. | shipped |
| **C** | The coach surface: acorn, sheet, `/coach`, first paint, rolling window. The box is connected; the four chips stay deterministic. | shipped |
| **C2** | The overwhelm turn — recognising it, and escalating it to Terra. *The phase that justifies the project.* | shipped |
| **D** | Read tools, model-decides among what the picker found, offer cache, picker fallback. | shipped |
| **E** | `breakDownTask()`; "too big" routed through the coach, one step at a time. | shipped |
| **F** | Splitting a brain dump, proposed and confirmed. On the screen; chat is unchanged. | shipped |
| **G** | Write tools and the confirmation policy. Six run, four ask, three are refused. | shipped |
| **H** | `shouldInterrupt()` on rule-produced candidates. *Last on purpose — the only one that speaks without being spoken to.* | shipped |

**Live configuration:** the API key is stored (SOPS + vault, project-scoped,
spend limit set) and already wired into the base Deployment as an optional
`secretKeyRef`, so **the coach turns itself on the moment a release carrying it
is deployed** — `gpt-5.6-luna` for routine work, `gpt-5.6-terra` for escalation,
under a €10/month in-process ceiling with the provider's own hard limit behind
it. With the key removed it is `NoCoach{}` and the product is exactly what it
was in v0.13.0.

**Seven deviations from the proposal, each recorded where it happened:** C
shipped with its box connected rather than deterministic-only; D shipped its
cache with no invalidation hooks at all, and five read tools rather than six
until the sixth was decided; E
is synchronous rather than paint-then-replace; F runs on a press rather than on
capture, and on the screen only; G answered the open question about pending
proposals by making them unstorable; H found that two of the four rules it was
told bound the model do not exist.

---

## Next, in order

Chosen on 20 August once Buddy shipped, and argued in
`docs/proposals/2026-08-20-after-the-coach.md`.

| | | Why here |
| --- | --- | --- |
| ~~**1**~~ | ~~Three states for things you cannot act on~~ — **shipped** | An offer you cannot act on spends the one decision you were given. Excluded from every existing list by construction, so it was smaller than it sounded. |
| ~~**2**~~ | ~~The capture gap~~ — **shipped** | The front door acknowledged before the words were durable. Now it spools like the room, and one mechanism covers both. |
| ~~**3**~~ | ~~Mood readable, and resurfacing~~ — **shipped** | Both shown only on request, never as their own stream. |
| ~~**4**~~ | ~~Attachments, on a PVC~~ — **shipped** | The only one that added infrastructure. |

## Decided, not yet built

### Product

| | Decision |
| --- | --- |
| ~~**Three new states**~~ | **Shipped 20 August.** *waiting on someone* · *blocked on a thing* · *someday*, named separately rather than one "parked" state. |
| ~~**Mood series**~~ | **Shipped 20 August.** One page and one command, both asked for by name. Nothing else reads them. |
| ~~**Resurfacing**~~ | **Shipped 20 August.** One kept note, roughly one evening in three, riding along with the evening message. |
| ~~**Attachments**~~ | **Shipped 20 August.** From the PWA — camera or gallery — one per note, on a volume beside the pod. Shown back before it is kept, and held on the device from the moment it is picked. |
| **Devices** | Phone primary and better; desktop first-class. **Spec settled 22 August**: `docs/superpowers/specs/2026-08-22-devices-design.md`. Five acceptance criteria, and the lid question is answered: three icons, the wordmark stays, a fourth is a redesign rather than an addition. |

### Structural

**PWA primary, Campfire secondary.** This inverts three things and none of them
is free:

1. ~~**Capture durability is backwards.**~~ **Fixed 20 August.** The slot goes
   through the same fsynced spool the room does. The drain does not apply a
   capture with no conversation, which is what keeps the slot a slot rather
   than turning it into a command line.
2. **Push becomes the primary channel**, not an improvement on Campfire.
   Unblocked 22 August: the off-LAN question is settled — the phone is on an
   always-on VPN, now recorded in `PRODUCT.md` — and the buzz's vocabulary was
   renegotiated deliberately. The nudge and the evening message may both push,
   with distinct tags and silent delivery for the nudge, so an audible buzz
   still means *leave now*. The notification's destination was the first brick
   and landed in #112.
3. ~~**Feature parity relaxes** to best-effort in one direction.~~ **Written
   into the record 22 August.** It had been decided here and contradicted in
   `PRODUCT.md`, which said parity was absolute — and that file wins, so the
   relaxation was not in force and the screen-only split was a live breach of
   it. `PRODUCT.md` now carries the one-directional rule and enumerates the
   five things chat keeps forever.

**Principle 5 is open.** The coach may evaluate, compare, and mention counts and
streaks. Shape guards — two sentences, no lists, no headings — are separate and
still enforced.

### Learned, once there is data to learn from

Both are dated to roughly late September, because they need a month of ordinary
use before they have anything to say:

- **Durations** from real timer runs, correcting your own estimates. Used for
  fit arithmetic, never rendered as a comparison.
- **Asking windows** shifting toward the hours you actually complete things.

### Experiments, kept on the list

Hyperfocus exit ramp (opt-in at timer start) · body-double follow-along
micro-steps for two or three chores · ~~novelty in **art and phrasing
only**~~ **— phrasing shipped 22 August.**

The four sentences met most often — the empty slot, the offer's label, the way
out of the deck, and the stopping screen's own line — have several wordings
each, chosen from the date. Deterministic rather than random, so both viewports
agree all day and a reload is not a slot machine; and produced by rules rather
than by a model, so it is Squirrel's voice under Principle 8 and the
deterministic floor never needs a key to speak.

**Every control label is untouched and stays that way.** Muscle memory is what
Principle 6's "the same every time" protects: a sentence you read is worth
varying, a button you press without reading is not.

The art half is still open — the alternate mood faces and the stopping
screen's own pose need drawings.

---

### v0.22.0 — 24 August 2026

**A fixed point you can put things on.** A note can point at an appointment;
`/at` is what is still coming and `/at/{id}` is one of them, showing when to
leave, what to take, and the notes pointing at it. The leave-by notification
lands there instead of the front door.

It cost two recorded decisions, both amended rather than deleted:

- `PRODUCT.md` refused a browsable list of appointments — *a calendar is a thing
  you are behind on*. Overturned by the owner, with the guard rails that replace
  it: the list holds only what is still ahead, so there is nothing in it to be
  behind on.
- A notification went to the front door, on the argument that a link to
  something already done is worse than a page saying what is true now. A fixed
  point inside its leave-by window is the one thing that cannot be stale.

Notes point at the appointment rather than the appointment growing fields, so
every thought stays in one pile and `!find` still reaches all of it. The pointer
is the disposition — no eighth state, nothing to migrate, and the reversal is a
null. Home carries a fourth door, two by two on a phone.

### v0.23.0 — 24 August 2026

**Buddy is the app.** Home is gone; `/` is one conversation. The four doors are
a rail pinned under the lid carrying what is waiting behind each, the check-in
and the offer are turns, the dock is the slot, and every press appends to the
record instead of reloading the page. Phase 1 of
`docs/superpowers/specs/2026-08-24-the-thread-design.md`; the number × unit
interval picker, the day/time picker, and the other three doors as messages are
phases 2–4.

It cost three recorded decisions, all amended rather than deleted:

- **Principle 2 — *nothing accrues that can be destroyed*.** Retired by the
  owner. The doors carry numbers. What is left is narrower and still refused
  elsewhere: nothing here is a score, nothing counts what you did not do, and no
  number survives being dealt with. It reverses cleanly — the numbers are
  computed at render time and stored nowhere.
- **Progressive enhancement.** Retired the same day. The thread requires the
  script. The single rendering path is kept: handlers return HTML from the same
  templates whether the browser asked for a page or for the turns a press made.
- **One reading, never a series.** Narrowed. A conversation keeps what was said
  in it, and what you answered is something you said. Only the newest turn draws
  the faces, so scrollback holds one line each; `/moods` is still the only
  surface that groups them.

Three defects it turned up, none of them visible in a diff: the offer was being
appended on every page load, `not now` rendered cream-on-cream at 1.18:1 the
moment its class changed, and `main`'s centred flex row shrank the conversation
to 240px of a 390px phone.

### v0.24.0 — 24 August 2026

**The chores and the tasks join the conversation.** Phase 2 of the thread. A
door is something you say: the rail posts, what is behind it arrives as cards in
Buddy's reply, and every action on them answers as a turn. `/chores`, `/tasks`
and `/tasks/done` are gone.

**How often is a number and a unit.** Two rows in one form — `every` 1/2/3/4/6/8
and `of these` days/weeks/months — replacing the four fixed chips. It needed no
core work: `ParseEveryAsking` already accepted any count against any unit, and
the picker composes the same sentence a person would type and hands it to the
same parser.

What it cost, stated rather than discovered:

- **A door cannot be opened in a new tab, and the back button does not step
  through doors.** Opening a place writes to the record, so it has to be a POST.
- ~~**The digit keys that answered the interval question are gone.**~~
  **Restored in v0.29.0.** They belonged to a disclosure inside the chore card;
  the question is a turn now, and the carve-out came with it.
- ~~**The new-chore form went with its screen.**~~ **Restored in v0.32.0**, as
  the guided version this entry called a bigger thing: two turns, with the name
  travelling in the picker's own hidden fields so nothing is written until the
  interval is answered.
- **The archive has nowhere to be drawn** until the pile joins the conversation
  in phase 3. What you did is still stored and `!find` still reaches it.

One hardening that came free: `taskActHandler` wrote against a bare id, because
`SetItemState` takes no person. Reading the row for the turn's words is a lookup
that is already person-scoped, so a row that is not yours is now not yours to
act on.

Three defects found on the way: `html/template` cannot put an action in a tag
name and the version that tried rendered nothing at all; the chore keys cached
their card list at page load, which is empty for cards that arrive in a
conversation; and the door, having become a button, sat at the browser's default
13.33px until the appearance snapshot caught it.

### v0.25.0 — 24 August 2026

**The agenda joins the conversation, and gains the day picker.** Phase 3a. The
agenda door draws what is still ahead; opening one draws it with what to take
and the notes pointing at it; and an appointment can be made for any day rather
than only today or tomorrow.

That last one needed the core. `ParseMoment` built from today's date, so a date
was not sayable at all. Two answers were refused — a grammar that takes dates,
which widens the bar that stops a stray thought becoming an interruption, and a
time built in the web package, which is a second place to be wrong — and the
third was to anchor the same parser to a chosen day. `ParseMoment` is now
literally `MomentOn(now, s, now)`.

`/at` is gone. `/at/{id}` stays until phase 4, because a notification sent
yesterday is still on a lock screen.

The lid's menu is down to one entry: the pile is the last screen that is a
screen, and the way back to the conversation is the mark.

**Phase 3b, half done: triage in the conversation.** The pile door hands you
one note with DONE, KEEP, DROP and A TASK; acting says what happened and hands
you the next; *later* skips one without deciding; and the way to change your
mind travels with the answer, because the card is about to be scrollback.

The deck still stands. Deleting it is 137 references to `/pile` across 39 test
files plus 54 to its sub-screens, and a triage loop nobody has used yet is not
a reason to take away the one that works.

The three questions a note can be asked came with it: making a chore out of it
reuses the interval picker, rewording it opens a box of its own — not the dock,
which keeps everything you type, where these words replace something — and "i
can't act on this" offers the same three whys.

The split proposal came too. Nothing is written when it is drawn: the pieces
are words in a turn until the press, and a proposal in scrollback has lost its
button by the live edge rule — which is what the deck got from keeping it only
as long as the page it was on.

**Everything the deck can do, the conversation can do.** What is left is the
deletion itself: 137 references to `/pile` across 39 test files plus 54 to its
sub-screens.

### v0.34.2 — 25 August 2026

Three things a phone showed that a laptop could not, all reported from one
screenshot.

**The dock covered the end of the page.** The clearance was on `.thread`, and
`.thread` is not the last thing on the page — the two chips and the way out sit
below it, outside its padding, so there was no way to scroll them clear. It is
on the whole column now, and the reserve is the dock's measured height rather
than a guess, so a slot grown to four lines pushes the conversation up instead
of sitting on top of it.

**"How do you feel?" was asked on every visit.** A question you have not
answered is still on the screen; asking again does not make it easier to
answer, it makes a column of the same question. The reading going stale is what
makes it worth asking; having asked and been ignored is what makes it not worth
asking again.

**And the opening line landed on top of it.** The check-in is a question with
its answers drawn on it, exactly like the picker, and it was left out of
`endsAsking` when that was written — so the two alternated down the screen.

The browser test for the first of these **passed with the defect deliberately
put back**: its fixture had no turns, so the page did not scroll and nothing
could overlap. Rebuilt against a conversation long enough to scroll, where it
fails by eighteen pixels.

### v0.34.1 — 25 August 2026

**A note carries your date, not the container's.** The other half of v0.33.1,
found by looking for the same shape somewhere else rather than by waiting for
it to be reported.

`ReceivedAt` came back from the driver in UTC and `toView` called `.Local()` on
it — the *process* clock, and the pods run in UTC on purpose since #148. So
anything captured after ten in the evening wore the previous day's date on the
corner of its card.

The conversion moved to `Store.here`, which now serves both tables, and it is
applied where a row comes out of the database rather than where it is printed —
because "each print site" is what let this happen twice. `pick.go` had already
patched one of them by hand, which is what that looks like from the inside:
correct, local, and no help to the next reader.

Both integration tests run under `TZ=UTC`, which is what the clusters do.

### v0.34.0 — 25 August 2026

**Buddy comes to know you.** He read the last few things said and nothing
else — helpful about the sentence in front of him and never about the person
typing it, because there was no memory outside a rolling window that forgets by
design.

The turns table made this possible on 24 August: it is a complete record of
everything that has happened on the screen. Once a week Squirrel reads it back
and writes down at most six short observations about how you actually work —
what you finish, what you put off, when in the day you do things — and those
are shown to the model on every later turn.

Once a week rather than once a turn, and that is a truth argument before it is
a cost one: an observation worth keeping is one that survived a week of
evidence, and a model asked "what have you noticed" after every message will
notice something after every message.

What it may conclude is bounded in three places, because a preamble is a
request rather than a rule. Never a count — "always" and "never" are a count
with the number taken off. Never a judgement about you as a person, only about
how you work. Six at most: a model asked for twenty things it has noticed will
produce twenty, and the last fourteen will be invented.

**You can read it and throw it away.** This is the only thing in the product
that holds an opinion about somebody rather than something they said, and a
product that quietly builds a picture of you that you cannot read is not this
product. It is one press off Buddy's own reply, in the model's own words rather
than a summary of them, and one more press to forget all of it.

The model is told never to say any of it back. Nothing can enforce that, which
is exactly why it is written down.

**The door cache has a bottom**, and a key that names the person in digits
rather than as a rune — see the review of 25 August, items 6 and 7.

### v0.33.3 — 25 August 2026

**The opening line stopped swallowing the offer.** Shipped in v0.33.0 and found
the same night by asking the question directly rather than by reading the code.

`endsOpen` counts anything Buddy has put on the table, and the opening line
carries one chip to the place it is about — so on every day it spoke, which is
any day with something on the agenda, a chore due or notes in the pile,
Squirrel opened with a line and then said nothing about what to actually do.
The offer is the product's whole argument, and it was off.

An opening says what is true and asks nothing. It is not something on the
table. Nothing failed, because the offer's own fixtures have no agenda.

### v0.33.2 — 25 August 2026

**"The rest" shows the rest.** All three of them were dead, and had been since
the doors became messages.

The chores' and the tasks' pointed at `/?open=chores` and `/?open=tasks` —
a query the thread has never read, so pressing them reloaded the conversation
and did nothing. Search's pointed at `/pile?q=…`, a route deleted with the
deck, which answers 404. Nobody had found out because twelve cards is more than
most people have; v0.33.0 dropped that to five, which is what would have made
somebody press one.

They are forms now, carrying where to carry on from, and the agenda gained one
it never had. Search has no second page, so the offer is narrowed to *say it
more exactly* rather than inventing pagination to make a chip work.

### v0.33.1 — 25 August 2026

**A fixed point comes back in your clock, not the driver's.** Issue #148 one
layer further out, found an hour after v0.33.0 shipped by probing what
`Upcoming` actually returns.

A `timestamptz` is an instant and carries no zone, so pgx hands it back in UTC.
Everything then *prints* it — "at 14:30", "leave about 14:05", the new opening
line — and printing the right instant with the wrong digits on it is a missed
appointment. On a process running in UTC, which is what the #148 fix
deliberately left the pods as, every one of those read two hours early in
summer.

The #148 fix threaded the location into everything that *parses* a time. The
reading side was never audited. The conversion is at the one place a moment
comes out of the database, because "each call site" is exactly what let it
happen.

### v0.33.0 — 25 August 2026

**Buddy talks.** Four things, all reported as the same complaint: it does not
feel conversational yet.

**He says the first thing now.** Every turn in the product was a reply — you
arrived and the conversation sat there until you pressed something, which is
not what a conversation is. The bar is meaningfulness rather than a budget:
something on today or tomorrow, a chore that has come back, notes waiting to be
decided about. Nothing worth saying means nothing said, which is the common
case on a quiet afternoon. It is one thing, not a summary of all four — the
rail already says all four — and it does not say the same sentence twice.

**The words vary.** The acknowledgements met several times a sitting — *Good.*,
*Kept.*, *This one.* — go through the same day-seeded pools the slot's line has
used since novelty shipped. Deterministic, so both viewports agree and a reload
is not a slot machine, and the wording that shipped still leads every pool.

**A door hands you five, not twelve.** Twelve was chosen so that "there is
more" would be rare, which is the wrong thing to optimise: a reply that arrives
as twelve cards is a screen of list with a sentence on top, and reading it is
the work the conversation was supposed to replace.

**And Buddy can say what he noticed.** One model call per set, bounded to
fifteen words, cached so opening a door twice costs one call. Never advice —
that is checked here rather than asked for in the prompt, because a prompt is a
request and this is a rule. The product's own sentence comes first: Principle 8
draws the line at authorship, and the count is Squirrel's own fact.

### v0.32.0 — 25 August 2026

**A new one, at every door.** Making a thing from nothing went with the
screens: each had a form at its foot, and when the screen became a message the
form had nowhere to be. v0.24.0 recorded that for the chores and said making
one from nothing is a sentence in the dock — which is true of a note and is not
true of a chore or an appointment, because neither is a thought you had. You
decide to have them.

Each door's turn carries one chip, on every branch including the one that says
there is nothing here — an empty list is the moment you are most likely to want
to add to it. Never over a question.

A chore is two answers, and nothing is written until the second, so walking
away halfway leaves no half-made chore behind. `/chores/new` now takes the
picker's number and unit as well as the four strings the old form posted.

### v0.31.0 — 25 August 2026

**The lid comes off.** The acorn, the search field and the hamburger all went
the same day, and with them roughly three hundred lines of script that existed
to make a sheet behave like a sheet. What is left in the lid is the mark and
the running timer.

**Buddy is turns.** The sheet brought a conversation with it because there was
not one to join; there is one now. `ask Buddy` is a chip at the foot of the
thread, the exchange is turns, and everything that does something came with it:
the four blockers, the steps a thing breaks into, the four proposals he must
ask permission for, and saying that a reply landed badly. Closing did not come:
a sheet is a thing that can be open and a conversation is not — you stop
talking.

Two guarantees the sheet held by hand are now the thread's own. A proposal
still cannot be applied by anything but a press, because it is stored nowhere
and travels in the form that renders it — and in scrollback it has lost its
button by the live edge rule. And the step you were on is still there when you
come back, drawn under the same rule as the offer so it cannot append a copy of
itself on every load.

**Search is a chip.** It was a disclosure in the lid because it had to be
reachable from seven screens.

Two defects this turned up. The chips were on the live edge first, which is the
newest Buddy turn — so a brand new conversation, which has none, had no way to
Buddy at all. They are at the foot of the thread instead. And **the focus ring
on the dock measured 2.33:1**, against the 3:1 WCAG 1.4.11 asks: the violet
override keyed off `.sheet`, and the dock is the same cream stock without it.
It had been wrong the whole time, on the one control that is on screen at every
moment, and only surfaced because the test that measured the sheet had to be
pointed somewhere else.

### v0.30.0 — 25 August 2026

**The shelf and the set-aside come into the conversation**, and the shelf stops
being unreachable. `/kept` and `/held` were the last two screens in a product
whose every place is a message. What made it a defect rather than an
untidiness: the only way to the shelf hung off the card drawn in the pile's
turn, so the moment there was nothing left to decide about, it was reachable
from nowhere at all — the exact bug the comment beside that chip had been
warning about since the deck came out.

Neither is a door. The rail is four and its equality is the whole statement it
makes, and a door for the things you explicitly set aside would put them back
in front of you, which is the one thing setting them aside was for. They are
chips on the pile's turn — on every branch of it now, including the one that
says there is nothing to decide about.

The shelf offers one thing and it is the way back. A kept note was never going
to be done, so DONE there would answer a question nobody asked.

**The conversation gained a way to stop.** The stopping line lived on those two
pages and nowhere else, so deleting them would have left the product with no
exit at all. Found by a test that had been asserting it on one of them.

**Two accessibility tests were deleted rather than retargeted.** Both asserted
on the deck's markup — the key badges and the stack behind the card — and
neither renders anywhere since it came out, so both had been comparing zero to
zero. A test that cannot fail reads like cover.

### v0.29.0 — 25 August 2026

**The interval question answers to keys again**, which is the last thing the
thread took away and did not give back. A digit is a count, a letter is a unit
by its own first letter, and Enter answers — so `d` means *days* while the
question is up, rather than DONE on the note behind it.

Only that question: a digit inside the day picker would be a day of the month,
and guessing which of the two was meant would book the wrong one.

### v0.28.0 — 25 August 2026

**Time is where you are, not where the process is.** Issue #148, fixed durably.

`TZ` on the clusters corrected the symptom in August and was never the fix:
nothing in this repository asked for it, no test failed without it, and the
staging manifest replaces its env list wholesale. The location is threaded now,
the way the scheduler's quiet hours and evening message always were —
`ParseMomentIn` and `StartOfDayIn` take it, the Applier and the Store are given
it at boot, and the web's Options carry it to the picker and the coach.

Three tests, and the third is the one that matters:

- a fixed point parsed on a UTC process lands at the right instant;
- "today", for a refusal, is the person's day and not the container's;
- **and the whole chat path books it correctly.** Reverting the wiring — one
  setter — fails that one while both parser tests stay green, which is exactly
  how the fault survived the first time.

`ParseMoment` and `StartOfDay` remain for callers that only ask whether
something *has the shape* of a fixed point, which is what `Match` does on stored
rows where the resolved date is never read.

### v0.27.0 — 25 August 2026

**The notification lands in the conversation, and a refusal has a way back.**
Phase 4, and the last of the thread.

`/at/{id}` stops being a page and becomes the tap itself: it writes the
appointment as a turn and redirects to `/`, so a warning opens the thing you are
about to leave for at the live edge. The URL keeps working forever, because one
sent last week is still on a lock screen. `sw.js` needed no change at all.

**Issue #147 is closed.** A browser told no will not ask again and this site
cannot make it, so a button would be a control that cannot work. What is shown
instead — once, and only to somebody who has actually refused — is the sentence
that says where to turn them back on.

### v0.26.0 — 25 August 2026

**The deck came out.** `/pile` and its templates are gone: the card, the split
proposal, the results screen, the bottom, the empty state and the interval
chips. Triage is the conversation and there is no second way to do it.

Search moved with it — the lid's field posts and the answer is a turn — and the
shelf hangs off the pile's turn, because the deck's foot was the only way to it.

The lid's menu is empty now. Every place is a message; the way to one is the
rail, and the way to the rail is the mark.

What crossed and what did not, stated rather than discovered:

- **The letters crossed.** `d`, `k`, `x`, `t` press the note Buddy is holding
  out. The machine around them did not: the stamp, the hold that gave an undo
  somewhere to be, the tray. A conversation has no card that needs to hold
  still, because the answer is a new turn.
- **An empty place is a sentence, not a drawing.** The empty states were their
  own treatment; a place that is a message has no screen to be absent.
  `/enough` is the last screen in the product that is an absence.
- **Keeping your place went away because there is no place to lose.** The deck
  carried a cursor through every redirect so that acting on the third note down
  did not return you to the first. The conversation never moves you.

Fifty-five unit tests and fifty-seven browser tests were reconciled. Most were
repointed; the ones that went are the deck's own machinery, each carrying the
reason where the test used to be.

**The rest of phase 3b, not started: the deck comes out.** It is the deck — one-card triage with
split, undo, paging, search and seven states — and it is the hardest surface in
the product. Splitting it out landed the day picker rather than hiding it
inside a larger change.

One test found doing nothing: `TestWhatIsComingCountsNothingAndScoldsNobody`
asked for a route that had just been deleted, so every assertion passed over an
empty body.

---

## Open

*(The nudge's two missing floors were decided and built on 20 August — quiet
hours 22:00–06:00 on the unasked path, and no nudge on a low day. See the
shipped list.)*

1. ~~**Attachment storage.**~~ **Decided 20 August: a PVC on the pod.** No
   object storage exists in the cluster and adding MinIO or Garage would mean a
   service to patch, back up and keep alive for one feature. The cost, stated:
   attachments join the pod's lifecycle and the restore drill has to grow to
   cover the volume.
2. ~~**`create_moment` proposal lapse.**~~ **Answered by construction at phase
   G:** a proposal is stored nowhere, so it lasts as long as the page it is on
   and there is nothing pending to expire.
3. ~~**Records to amend before coach phase C.**~~ **Done** — `DESIGN.md`,
   `PRODUCT.md` and the route table were all amended before C shipped.
4. **The vault note** for `squirrel openai key` still carries the *known
   exposed* paragraph from the revoked key. App-only edit; `pass-cli` reaches
   named fields but not the note body.
5. **Whether `/v1/responses` is worth the portability.** Buddy decides without
   extended reasoning today, because tools and reasoning cannot both be asked
   for on `/v1/chat/completions`. Two calls of evidence is not enough to spend
   `COACH_BASE_URL`'s portability on. Revisit if the overwhelm turn reads
   shallow.

---

## Refused, and staying refused

Listed so they are not re-litigated. Each was considered and declined with a
reason; the reasons live in the two proposals.

**Because they accrue something that can be lost:** XP · points · streaks ·
beat-the-timer challenges · mood charts and trends · showing
estimate-versus-actual accuracy.

**Because they are administration, which is what the product exists to avoid:**
projects · tags · folders · sub-tasks · priority levels · recurring tasks (that
is a chore).

**Because they duplicate or dilute a working surface:** a third capture surface
· voice capture (the phone keyboard already dictates into Campfire) · a general
AI chat companion · morning planning · weekly reflection.

**Because they import a shape the product does not have:** calendar import ·
two-way calendar sync · a browsable list of appointments · deadlines on tasks ·
"someday" as a note state rather than a task state.

**Because the local hardware cannot serve them well:** a local model for the
coach — 4-core arm64, no GPU, 10–20 s replies against under a second hosted, to
save about €3 a year.
