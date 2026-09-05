# Roadmap

What is built, what is decided, what is refused. Last reconciled **5 September
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

## Next, in order — 20 August, and shipped

Chosen once Buddy shipped, and argued in
`docs/proposals/2026-08-20-after-the-coach.md`. All four landed by 24 August.
What is next lives in **Decided, not yet built**, directly below: the board
retirement, decided 3 September, and Devices, settled 22 August.

| | | Why here |
| --- | --- | --- |
| ~~**1**~~ | ~~Three states for things you cannot act on~~ — **shipped** | An offer you cannot act on spends the one decision you were given. Excluded from every existing list by construction, so it was smaller than it sounded. |
| ~~**2**~~ | ~~The capture gap~~ — **shipped** | The front door acknowledged before the words were durable. Now it spools like the room, and one mechanism covers both. |
| ~~**3**~~ | ~~Mood readable, and resurfacing~~ — **shipped** | Both shown only on request, never as their own stream. |
| ~~**4**~~ | ~~Attachments, on a PVC~~ — **shipped** | The only one that added infrastructure. |

## Decided, not yet built

### The conversation retires into the board

Decided 3 September 2026, and it governs everything built after it rather than
being one more item on a list. **Spec written 5 September:**
`docs/superpowers/specs/2026-09-05-the-conversation-retires-design.md` — the
twelve capabilities still reachable only from the room, where each one lands,
six phases, and the shape question answered: Buddy on the board answers through
the strip you are looking at, never through a box, because a box on the board is
the refused general AI chat companion relocated rather than retired.

The conversation goes. What it does that the board does not yet — Buddy
answering, the readings, whatever is left of triage — moves into the board, and
the board becomes the whole of the app. **Chat, in the end, works only through
Campfire.**

**Campfire is always secondary to the app.** It is where a thought can be
thrown from a phone that has the room open, and where Squirrel can say
something without the app being in front of you. It is not where the product
lives, and no capability may exist only there.

What that means for anything built between now and then: a thing added to the
conversation is a thing that will have to move, so it is added to the board
unless there is a reason it cannot be. This release is the pattern — the
check-in was asked in the conversation, and it is asked on the board now.

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

### v0.71.0 — 5 September 2026

**The edges are bounded and quiet, and the board answers the press you made.**

The rest of the review, and the first release in which more findings were
*refused* than any single lens raised.

**The edges.** An open redirect fired straight after a successful login: the
guard rejected a leading `//` but not a backslash, and a browser reads `\` as
`/`, so `next=/\evil.com` left the host. The webhook had no rate limit, which
is what made brute-forcing the unsigned integers behind `!action` realistic.
The OIDC flow asked for no nonce. A failing migration retried once a second
forever, where the drain loop twelve files away had doubled its backoff for
months. The migration pass took no advisory lock, so a rolling deploy could
fire the one log line the runbook says means *roll the image back* — a false
page rather than an outage, which is worse in its way, because the runbook is
read at two in the morning.

**The quiet.** A chore's own name was being written to the log on the ordinary
*decided not to interrupt* path, not on a failure. `coach_answers` had a stated
horizon in the schema — **dropped on 5 September**, because nothing ever
enforced it and a promise the product did not keep was worse than the honest
"this is kept" — and `!unsay` drops the last exchange. Photographs keep their
metadata, location included, and `PRODUCT.md` says so rather than leaving it to
be discovered.

**The board.** With nothing focused, a letter acted on the first card — so a
second press finished a note nobody had read, and a blind user could not reach
the second note by the letter the screen was showing them. The 1150ms hold
removed the stamps from the accessibility tree and said nothing for a second.
The find field's focus ring had *no border to colour*, so there was no focus
indicator at all, at any width, and had never been one. The phone's tab badges
stopped borrowing the bell's unread grammar. A thought typed on the chores tab
stopped becoming a weekly chore.

**Three findings were refused on contact with the running product**, and that is
the part worth keeping.

The health endpoint the review said was missing has existed all along, and
deliberately does not check the database: a readiness probe failing on a
Postgres outage would take the pod out of its Service, and Campfire does not
retry a webhook. Adding the check would have traded a survivable outage for
permanent loss.

The twelve buttons standing in the notes bay were an artefact of the review's
own screenshots. `hover: hover` collapses them on a desktop and the accordion
collapses them on a coarse pointer; a headless browser is neither, so it alone
draws them all. The finding shipped with a mockup of something that does not
happen.

And the four rhythm stamps were not missing — v0.68.0 deleted them the day
before, because they and the interval field said the same thing twice. They were
rebuilt from `DESIGN.md`, which still described them, and caught before merge.
The record was what was wrong, and it is corrected.

**What the drill found.** It was run for the first time, into a scratch
namespace, and passed: 66 rows, the join returning its one photograph, the file
intact with both its markers. It also found that the nightly pair was taken in
the wrong order — the volume three hours *before* the dump, when a photograph is
written to disk before its row — so a restore could hold a row pointing at a
file no backup had. Fixed in `homelab`, and the runbook's step 1, which
prescribed that order while giving the right reason against it, is corrected.

### v0.70.0 — 5 September 2026

**A capture fails fast and says so, and the picker chooses everywhere.**

A cross-functional review read the whole product at v0.69.0 — twelve lenses,
then six verifiers whose job was to refute what the first twelve found. Three
findings did not survive that second pass and are not in this release; what is
here is what held.

**Two ways capture could leave you with nothing to read**, both opened by
v0.68.0's move of the write into the request.

The first was silence. With the database wholly unreachable the refusal was
built correctly and then handed to `keepSaid`, which tried to record it on the
same broken store, dropped it, and returned nothing — so the fragment path
answered 200 with an empty body, and the script reads an empty answer as *there
was nothing to keep*. A total failure of the store was indistinguishable from a
keystroke that did nothing. A turn that cannot be written down is now kept in
the answer and dropped only from the record; it carries no id, which the
template already reads as "not a row", so it is shown once and gone on the next
draw. That reaches every one of the thirty-odd handlers that call `keepSaid`,
not only capture.

The second was hanging. Nothing bounded the write: no deadline on the request's
context, no `WriteTimeout` on the server, a bare DSN handed to `pgxpool`. A
database that is *down* was always handled — the dial fails and the box says
so. A database that is merely *degraded* had nothing to stop it, and the person
got a spinner instead of the sentence v0.68.0 promised. Three bounds now, one
per leg, and a bound written into the address still wins.

**The picker chooses in the conversation too.** v0.69.0's entry said it already
did, "on the board and in the conversation both". That was true of the board and
false of chat, where `judged` still handed the picker's answer to a model that
could return a different row and a different reason. Made true rather than
corrected — the model is off the offer path entirely, and about 1,400 lines went
with it: the `Offers` cache, `ForgetOffer`, `Decide`, the `choose` tool.

That deletion closed a second finding on its own. Chat cached its decision and
the board only invalidated that cache from the pulled strip, never from a bay,
so finishing a chore where it lives could leave chat offering it for another
half hour. With nothing cached there is nothing to disagree about.

**Three tests were deleted for passing.** With `Decide` gone from the interface
nothing can consult the fake coach, so every *a model was asked* assertion had
become vacuously true — including the one whose whole point was that no model is
invited when the rules find nothing. That guarantee is the type system's now:
there is no method left to call, which is stronger than the assertion was. The
review had already found one of these hollow by mutation, a counter wired to
nothing that passed whether its guard existed or not.

**Four smaller things, each one line.** The letter shortcuts did not count a
`<select>` as typing, so pressing *d* for "days" in the chore rhythm jumped to
the first note's DONE. The agenda's own capture field was crushed to nothing by
the date and clock beside it, so `at 14:30 dentist` — the placeholder that
teaches the grammar — had never been visible at a desk. The face chip was
hidden above 620px and is the only route to `/me`, so the page holding the
readings could be reached on a phone and not on a desktop. And the record was
corrected in six places where it still described the screen the board replaced.

**What the review found that this release does not fix**, so it is not lost: the
restore drill has still never been run, and the nightly backups take the
photographs volume three hours *before* the database dump. A photograph is
written to disk before its row, so that order is the one that produces a row
pointing at a file no backup holds — the exact direction `docs/running.md` warns
about and then prescribes. Both live in `homelab`.

### v0.69.0 — 4 September 2026

**Pressing a button on the board costs nothing.**

The report: *"When I click 'not today' it takes a few seconds before something
happens on screen."* The press was never the cost — it writes a refusal and
redirects. The redraw was.

Since v0.65.0 the pulled strip's clause was written by a model for whatever the
picker had just chosen. Answering an offer invalidates that cached decision *by
design* — that invalidation was itself a fix, for a card that redrew itself
unchanged — so the very next draw was a guaranteed miss, and deciding is a tool
loop of up to three round trips run inside the render. Press, wait for a model,
then see the next card.

**The picker chooses and the picker says why**, on the board and in the
conversation both. `judged`, the `Decide` option, the acorn mark and *that did
not land* went with it: the mark said where a sentence came from, and with every
sentence coming from the rules there is nothing to attribute and nothing to
refuse there.

**What stays.** `ForgetOffer` — the core still caches a decision for Campfire's
own *what should I do now*, and answering on the board has to invalidate it or
the chat goes on offering something already dealt with. And marginalia, which is
where the product speaks unbidden: in the margin, once a day, where nothing
waits for it.

**This was written down before it happened.** v0.65.0 recorded the risk while
taking it: *"A surface that has to cost nothing to open may not spend a call...
If that turns out to be the wrong trade, the fix is a cadence — notice once a
day rather than once per pick, and it is a small change."* It was the wrong
trade, the owner found it by pressing a button, and the fix was the small change
it was predicted to be. The value of writing the risk down was that nobody had
to work out what had gone wrong.

**Two intermediate answers were rejected on the way**, and both are worth
naming. Filling the decision in the background would have removed the wait while
still spending a call per newly picked thing — three presses, three calls.
Asking only about a thing offered more than once would have kept a model on the
board for the case where the picker's clause has already failed. Neither
survived the question the owner actually asked: why is a model involved when I
press a button?

### v0.68.0 — 4 September 2026

**A press does the thing. And the board is worked rather than navigated.**

The owner's report was that clicking did nothing and a refresh showed the new
state. The cause was not the spool's durability but its asynchrony: a capture
from the screen went to a spool, a background drain wrote the row, and the board
you were redirected to was drawn before the row existed. One of the two capture
paths did not even ask for the settling pass that hid this — and it is the one
the notes bay uses whenever photographs are configured, which is the most
ordinary act on the board.

**The screen writes the row itself now.** Inside the request, before it answers.
The spool and the drain stay where they earn their keep: Campfire has nobody in
front of it and nowhere to report a failed write, so the extra hop is worth it
there. The screen has both.

**What was traded, stated rather than hidden.** With Postgres unreachable the
box says it could not keep the words instead of accepting them and settling them
later. The words stay in the box, and the turn that says so is unchanged. It was
argued against first, reaffirmed, and then done: the owner's call, not a quiet
one.

**It removed a class of bug rather than an irritation.** A question typed in the
dock was dropped from the pile by matching its *words*, because the row it
wanted did not exist yet when the answer came back. Two notes saying the same
thing could drop the wrong one, and a slow drain dropped nothing. It takes the
id of the row it just wrote.

**Six design changes, each picked from three variants at the screen.** The top
bar became two objects instead of seven siblings — identity leading, everything
you operate in one cluster, the clock outside it as information. The chores
strip asks for its interval on the strip, because four preset stamps and a
second row said the same thing twice. The agenda strip is shaped like what it
makes, and its clock is 24 hours *by construction*: a native time input renders
in the browser's locale and no attribute changes that, so the hour and the
minute are two fields. The notes are one list — what needs deciding first, then
what was set aside and what was kept under their own seams — and the ledge with
its two shelf trips is gone. Bay colour rides the sign's holder rather than a
tint-by-category wash.

**Two anti-patterns found mechanically and fixed, both mine.** The marginalia
line shipped in v0.66.0 with a 3px orange left border, which is the single most
recognisable tell of generated UI and is banned outright by the craft floor. And
the touch path animated `max-height` with a 320px guess in it. A hairline rule
and `grid-template-rows` respectively.

**The design loop itself was the week's real blocker**, and three things fixed
it: `?only=<bay>` draws one rack in development, so a picked element is one
element; `/dev/redraw` reloads the page when a template changes, which is what
hot reload looks like on a Go server; and the dev fixture keeps what it is told
instead of answering false to every write, which is why pressing things there
had felt dead. A test now refuses to let a live-overlay script tag reach a
committed template.

**A TypeScript rewrite was costed rather than argued about** —
`docs/proposals/2026-09-03-typescript-rewrite.md`. 27,178 lines of source,
41,215 lines of proved tests, four direct dependencies, and a blocker the
rewrite would not have fixed: one component rendered per bay is four instances
in React too.

### v0.67.0 — 3 September 2026

**The reason line may point at the rest of the board.**

Register C could only ever speak about the one thing it was pulled from — the
mock said so as its cost, and it was true of what shipped in v0.65.0. It is now
shown what else is on the board: the things written down and not decided about.
So the clause can say the detail this one needs is written down already, or this
is the same errand as that. That is the one thing a clause can say which a
person cannot get by reading the list, and it is the test marginalia is held to.

**What it may point at, it may not hand you.** The notes go in as text rather
than through a tool, and `coach.Written` carries no id at all. A tool result is
something the model was handed, and anything it was handed is something
`choose()` will accept; text is not, and a note it names is a number it invented
— which was already refused. This is the exception `OpenWork`'s comment
predicted and refused in writing, taken on the one path that cannot reach the
offer. A note is a thought nobody has decided about, and offering one would be
the product deciding for you.

**It costs nothing new.** The board rides in on the first message rather than as
a fourth read tool, so there are no new round trips and `maxRounds` is untouched.
The alternative — a `notes` tool — would have needed a fourth round and would
have put note ids in front of `choose()`, which is the thing that must not
happen.

**Two existing tests were changed on purpose**, which is worth saying plainly
because a changed assertion is how a regression usually gets through:
`TestDecideReadsThenChooses` and `TestDecideCapsWhatAToolWillReturn` now expect
the board to be read on the way in. Both still fail if the read stops happening.

Eleven mutations, each asserted to have changed the file and to compile before
its test was believed. The three that matter: dropping the id check in
`chosen()`, so a note becomes choosable; swapping `OpenItems` for `Tasks`, so
the pile stops being the pile; and letting a failed board read take the offer
down with it.

### v0.66.0 — 3 September 2026

**Marginalia: the board says one thing about itself, in the margin.**

Register A, the one the mocks chose. A line now hangs under an ordinary strip —
small, muted, set in behind an orange rule — in the margin of the rack rather
than in a conversation. There is no name on it, no face, and no press that asked
for it.

**It is written on a cadence, never on a press.** Once a day the whole board is
read and at most two lines come back. This is the trade v0.65.0 said would be
made if the per-pick call turned out wrong, made here from the start: nothing
the person does spends a call, nothing waits while a line is written, and a line
is either already there or it is not. It is the difference between a note in the
margin and a chat.

**A line earns its place by connecting two things.** The detail one note needs
is written in another; several of these are one errand; this cannot start until
that is done. A line that restates the strip it hangs under is worse than an
empty margin, because it still has to be read. The preamble refuses the rest
absolutely — never a count, never anything about the person, never an
instruction or a question — and it says twice that nothing is better than
something.

**Refusing is the only control, and it is real.** *not useful* keeps the words
and marks them refused rather than deleting the row, and the next pass is shown
them as something not to write again. A refusal that only cleared the screen
would leave the same line to be written tomorrow.

**Three places refuse, not one.** The preamble, the tool's shape, and what the
caller will accept back: a note about an id that was not on the board is dropped
rather than hung on whatever row happens to carry that number, and a note takes
the kind of the thing it names rather than one the model chose. A board with
fewer than two things on it is not read at all — the whole value is what one row
says about another. A pass that could not run writes nothing, so the clock does
not move and the next tick tries again.

**A rack whose marginalia cannot be read draws anyway.** That is the rack this
product had for its whole life, and nothing is said about the failure because
nothing was promised.

**The browser found a defect the string tests could not.** The line was a `<p>`
with the refusal form inside it, and the parser closes a paragraph when a form
opens — so the button rendered outside the line it belonged to. Every assertion
about the rendered string still passed, because the string contained everything
it was asked about. It is a `div` now, and reverting that kills the test.

Twenty-five mutations, each asserted to have changed the file and to compile
before its test was believed. Three of them were build errors dressed as passes
and one was a genuine false pass: the mutation removed a `return nil` that was
redundant with the nil map behind it, so the code did the same thing. The
mutation had to be the defect itself — the failed read taking the rack with it —
before the test proved anything.

### v0.65.0 — 3 September 2026

**It notices rather than answering.**

The pulled strip's line appears because something was worth saying about the
thing you were handed, not because you pressed for it. The press is gone, and so
is the route behind it: a line you have to think of asking for is a line this
product has failed to give you, and thinking-of is the thing it exists to
remove.

**The mark stays and stops being a person.** It says where the sentence came
from — a model wrote it, or the picker's rules did — and not who said it,
because nobody said it. There is no name on it and no verb of speech. The
squirrel is the product's mark rather than somebody's face.

**The refusal was already real**, which was the condition for any of this. *That
did not land* is written into the record the next prompt reads, so a line you
refused shapes the ones that follow. Unbidden text without that is something
that happens to you rather than something you are in a relationship with.

**What it costs, and it is a rule being traded rather than kept.** A surface
that has to cost nothing to open may not spend a call — written for a screen you
might open twenty times an hour. Opening the board now spends one on the first
render of a newly picked thing. The cache keys on what was picked, so it is one
call per thing offered rather than one per render, and the budget is what says
no. If that turns out to be the wrong trade, the fix is a cadence — notice once
a day rather than once per pick — and it is a small change.

**Where this is going**, decided from three mocks: the reason line is register
C, and marginalia on ordinary strips is register A, which is next and needs
somewhere to store a line against an item plus a policy for what is worth saying
at all. A third register — an observation as its own strip in the rack — was
refused: a board with a feed on it is a board you scroll rather than work.

Buddy's name survives in the chat chip and his room, because the conversation is
still there. It goes when the room does.

**Two mutations of mine did not bite until they were fixed**, and both are
shapes this project has hit before. One silently did not apply, because gofmt
had moved the line it matched on. One did not compile, because it left a
variable unused. Neither showed as anything but a pass, which is the whole
danger: a mutation that does not run is indistinguishable from a mutation the
test caught. Each one now asserts that it changed the file before the test is
believed.

### v0.64.0 — 3 September 2026

**The check-in is on the board.**

The five faces at the tray's right end, which DESIGN.md has specified since the
board was drawn and which was never built. On a phone they take their own row
above the day's departures: the tray scrolls sideways, and a question sharing
that row is a question you can push off the screen. Asked once an hour, the same
rule the conversation used, from the same five.

**Nothing is said back.** The conversation answers a check-in with a turn
because it is a record of what was said; the board is a record of what there is,
and a reading is neither a strip nor something to answer.

**This is the first of the moves the direction above calls for** — the
conversation retiring into the board — and it is the pattern for the rest: the
question was asked in one place and is asked in the other, without the answer
becoming something the board has to draw.

**A test of mine passed against broken code, and the fix is worth keeping.**
Checking that a word which is not one of the five keeps nothing, by asserting
the recorded mood was empty, could not tell "nothing was written" from "the zero
mood was written" — the store keeps whatever it is handed. It asserts the write
never happened at all now, and only then did the mutation bite.

Two tests changed because the tray grew a row rather than because a promise
moved. Both measure against the thing they are about now — the rack against the
racks area it fills, the pill against the foot — instead of past whatever
happens to sit between, and their fixture answers the check-in so the faces are
not drawn.

Also in this release: the fake store reads back what it was told. It kept
appends in one list and read from another, so a write was never visible to a
read and any test that reloaded could not see the press before it. One test
passed only because it never reloaded; it presses the real control now.

### v0.63.0 — 3 September 2026

**The rooms' furniture goes, and who you are is a page.**

**The room sheet was still being drawn.** It was the phone's control for seven
rooms, and it outlived the rooms by a day: on a phone it painted its label over
the bar's chips. There is one room and the bar is the navigation, so it goes.

**The rail goes with it.** What it held last was the way back, the way to look
something up, and who you are — two chips and a page now. The body's grid is one
column again rather than one column plus a rail that is not there, which is what
was throwing the new page's layout out.

**Who you are is its own page.** The picture, the name, notifications, what
Squirrel knows, how you felt before, and the way out. It was a disclosure inside
the rail, on the argument that settings is state rather than a conversation and
this product had no third thing for it to be. It has one now — and a panel that
lives inside a conversation is a panel you reach by first going somewhere you
did not want to be.

**The face in the bar fills its chip** rather than sitting on an orange square:
`object-fit: cover` under `overflow: hidden`, so the orange is the fallback
ground behind a monogram and not a ring around a photograph. The test that
checks a face is round checks three places now — your own turns, the settings
page, and the chip — where it checked two.

**Six tests followed the panel to the page** rather than being deleted: the push
setting's three, the two about who you are and the way out, and the one about
nothing painting over the way out. That last was written for the open room sheet
and holds the same thing without it — a control that is on the screen and cannot
be pressed is the failure, whatever is over it.

One had to change shape rather than target. The readings-alignment test asked
for the grid from the settings panel, which is a full navigation now, and the
fake store records appended turns separately from what it reads back — so a
reload cannot see them and the old path only worked because it never reloaded.
It drives the same request through the fragment path instead, and says so. The
fake is worth fixing on its own account: a store that forgets what it was told
can hide exactly this.

### v0.62.0 — 3 September 2026

**One bar in both places, and the face opens the settings.**

The conversation's lid holds the board's chips now — your face, the field, the
way to the other place, the bell. It keeps its own frame, fixed and translucent
and ruled off, because the transcript is seen to pass under it and the board has
nothing to do that for.

**The middle chip is always the other place.** A chat bubble on the board, the
board in his room. That is the answer to how you get back out of a conversation
with no rail and no mark, and it is one slot rather than a second bar.

**The face opens what can be changed** rather than the room it was usually
already in: notifications, what he knows, how you felt before, and the way out.
The rail keeps that panel and loses the two controls the bar took over.

**Why this was backed out of v0.61.0 and worked here.** One component could not
sit in two stylesheets until three things were made the same. `--line` is 2px on
the board and 3px in the conversation, so a shared control that read it was
drawn at two weights. Only one of the two sets `box-sizing` globally. And
pile.css carries a 44px control minimum that a chip inherits, which is exactly
why his room's chips came out four pixels taller than the board's and every
offset test failed. The chip carries its own weight, its own box model and its
own height now, and `--lid-h` follows from that.

**One behaviour changed rather than moved.** Searching from his room used to
answer inside it, as a turn carrying hits. The field in the bar is the board's,
so a word typed there leaves the room and answers on the board. That is one
search rather than two — and it is a real change, not a tidy-up: the
conversation can no longer show a result without leaving it. The test that held
the old shape is retired where it stood, with the reason in it.

Also: `TestARackHandsYouWhatItHoldsAndSaysThereIsMore` forbade the substring
`60` anywhere on the page and began failing because an asset's version hash
happened to contain it. It checks the number as it would be drawn.

### v0.61.0 — 3 September 2026

**The board asks rather than guessing, and a capture appears at once.**

Four things anyone could hit in a minute of using the board, and one correction
about testing that outlives them.

**A note captured on the board was not on the board you were sent back to.**
Capture spools first — that is what makes it durable — and the drain moves it
into the pile on a one-second interval, so the redirect beat the drain. The
capture was safe the whole time and looked lost, which is the one impression
this product may never give. There is a pass over the spool in front of the
person now, before the redirect. The write is what must not fail; the pass may,
and the background drain still finds it. Two drains over one spool is safe by
construction: a redelivery is absorbed by `ON CONFLICT DO NOTHING`, the same
guarantee that lets Campfire retry a webhook.

**A chore typed with no rhythm became a note.** In another rack, found on the
next refresh. The same went for an appointment with no time in it. Both are
questions now: the words come back into the field they were typed in, with the
question under them, and the chips or the pickers are the answer. Nothing is
filed anywhere until it is answered. Two tests pinned the old behaviour and now
assert the opposite, with the reason in place.

**A rhythm is any number of days, weeks or months.** The four chips are a
shortcut, not the vocabulary: the fourth chore you have comes back every three
days, and a screen that can only offer four intervals is one that makes you
round.

**The agenda has its pickers back** — a day and a time beside the field, which
is what the conversation had and the rack lost. A sentence with a time in it
still works and is still quicker.

**Your face is on the chip** where the mark was, read from the same place the
conversation reads it so the two cannot drift.

**The ground is painted on `html` rather than on the body.** A background on the
root propagates to the canvas, which is what the phone paints under the status
bar and inside the safe areas — so the strip above the bar was flat purple while
everything below it was dotted. The board also gained a `theme-color` and the
mark as its favicon, neither of which it had. The bar reserves the top inset and
nothing on top of it.

**The correction.** Four defects in the bay bar were written up in v0.58.1 and
v0.59.1 as invisible to CI, on the grounds that `env(safe-area-inset-*)` reads
zero in a headless browser. **That is wrong.**
`Emulation.setSafeAreaInsetsOverride` sets it, and this project was already
using it — `TestBrowserTheDockDoesNotStackItsOwnPaddingOnTheInset` has done so
since the dock was built. It was found by accident, while retargeting the lid's
tests. There are real tests now for the pill's height above the home indicator
and for the top inset, both mutation-proved. Those two entries overstated the
limit, and the lesson is the ordinary one: before recording that something
cannot be tested, look for the tool in the project's own suite.

**Not done, and deliberately not smuggled in.** Buddy's room does not carry the
board's bar yet. It was built and backed out: the conversation's layout is
driven by `--lid-h`, computed from the logo's height, and the transcript, the
rail and the sheet all offset from it — three browser tests fail on the offsets
alone. The answer to *how do you get back* is built and waiting as `boardchip`:
the middle chip is always the other place, a chat bubble on the board and the
board in his room.

### v0.60.0 — 2 September 2026

**The top bar is chips, and the bell has a record behind it.**

On a phone: a monogram where the mark was, the find field across the middle,
Buddy's chat chip, then the bell. No ground and no rule, so the board begins at
the top of the screen — the same move the bay bar made at the foot, and for the
same reason: a band of furniture costs a strip. The mark, the wordmark and the
clock all go. On a desktop they stay, and the two chips replace the *talk to
Buddy* link so there is one Buddy control at both sizes.

**Buddy's acorn becomes a chat bubble** and sits in front of the bell. Every
chip is a glyph with an `aria-label`, and a test counts the chips against the
names, because a control drawn as a glyph and named nowhere is a control nobody
can follow.

**The find field stops hiding behind its own glyph.** It was drawn to nothing
for one afternoon: a search that costs a press before you can read what you
typed is a search you cannot correct.

**A push had been fire-and-forget since it shipped.** The payload went to the
push service and nothing on this side remembered that it had, so the app could
not answer the question a bell implies — *what did you tell me?* — and a phone
that was off, or a notification swiped away unread, lost it for good.

`said` is written **at the fan-out rather than per subscription**: two browsers
on one account are two deliveries of one thing said, and a list that showed it
twice would be a list about plumbing. Nothing is written when no push service
took it, because a row there would be the app claiming it said something it did
not say.

**The bell wears a dot, not a number.** A count there would be things you have
not read, which is the one shape this product refuses. The bays' numbers count
things that exist, which is a different claim — and the difference is worth
keeping in writing, because the two look identical on a screen.

**This is the first schema change in seven releases**, and it is additive: one
new table and its index, nothing altered. Rolling the image back leaves the
table behind unread rather than leaving the app unable to boot, so the rollback
is still the image tag alone.

### v0.59.1 — 2 September 2026

**The pill leaves the same ground the reference does.**

Giving back 24 of the home indicator's 34pt put the floating bar against the
indicator rather than clear of it. Eight is the right amount to give back, and
the number came from measuring rather than from taste.

**How it was measured**, because the method is the useful part. Both
screenshots — the reference app's bar and this one's — were scanned column by
column for the row where the bar's own colour ends and the ground begins. The
reference leaves 49 image pixels under its bar; this left 12. At the
screenshots' scale that is 21 CSS px against 5, so it was four times tighter
than the thing it was copying, not slightly.

**Where the missing five went.** The pill's `5px` hard shadow sits below its
border and is part of the object, so half the offset was spent before the gap
began. `--foot` reads 26 on a phone with an indicator and the visible gap reads
21.

Confirmed by rendering with the inset stubbed at 34px and scanning the render
the same way: the outline ends 42 pixels above the foot at twice scale, which is
the 21 the reference has.

This is the fourth thing in this bar that no test in the project could see, and
the second where the answer was to measure pixels rather than to write an
assertion. `env(safe-area-inset-bottom)` is zero in a headless browser, so the
gap CI renders is never the gap a phone shows.

### v0.59.0 — 2 September 2026

**The bays float at the foot, under smoked acetate.**

A pill clear of all three edges — 12px at the sides, about 10px off the true
bottom — rather than a band welded along the bottom of the screen. The board
scrolls beneath it, the cell you are in is tinted rather than filled, and the
four bays are named without their article: Notes, Chores, Tasks, Agenda. The
rack's own sign still reads *the notes*.

**Translucency in this world is not glass.** `rgba(59,37,96,.86)` over a 13px
backdrop blur is the board's own purple at strength, so a strip passing beneath
comes through as a diffused shape rather than a frosted pane — and the pill
keeps its 2px outline and its shadow, which glass in the iOS sense has neither
of. It would otherwise be the one surface in this product that is not a thing
you could pick up. It is the acetate a strip board is covered with. Solid where
the browser has no backdrop filter, and solid again under
`prefers-reduced-transparency`.

**The gap under the buttons is gone by construction rather than by patch.** It
was the well behind the lit cell: the well stopped above the home indicator's
reserve, so the reserve read as a hole beneath the buttons. A floating bar has
no edge to leave a gap against. The reserve also shrank — `--foot` is
`max(8px, env(safe-area-inset-bottom) - 24px)`, because a floating bar does not
owe the indicator the whole 34pt the way a fixed band does: the indicator is
drawn over it, and the cells' own padding keeps the labels clear.

**Two things had to change for the translucency to mean anything.** The reserve
moved inside the scroller rather than sitting on the page, because nothing ever
passed beneath the pill and the blur showed only ground and dots. And the labels
went to full-strength cream: measured on the rendered pixels with a cream strip
diffused directly under a label, 72% opacity gave 4.35:1, which is under the
floor for text that size. At full strength it is 7.0:1.

**The third blind spot in one bar, recorded together.** A headless browser
reports `env(safe-area-inset-bottom)` as zero, so no rendered test can see how
much of the inset is reserved; the contrast walk renders a page where nothing
sits under the bar, so it cannot see a label losing contrast against what shows
through; and the collapse in v0.57.0 hid the same way. Everything checkable here
is checked, and the rest was verified by rendering with the inset stubbed at
34px and by sampling rendered pixels — which is a method, not a test, and is
written down as such.

### v0.58.1 — 2 September 2026

**One safe area at the foot, not two.**

The rack kept padding for the home indicator after the bar took the bottom of
the screen, so both claimed it: a band of ground between the rack's last edge
and the bar, on every phone that has one.

**Why nothing caught it, which is the part worth keeping.**
`env(safe-area-inset-bottom)` is zero in a headless browser, and the gap is
exactly that inset — so every screenshot and every browser test of the new bar
rendered it flush against the rack. It exists only on a device with a home
indicator, and it took a photograph of a real phone to see it. The same blind
spot will hide the same class of defect again: a headless run cannot see any
rule whose value comes from the device's own chrome.

So the guard is a rule about the stylesheet rather than about a rendered page,
because a rendered page in CI will keep measuring zero: inside the 620px block,
`.baytabs` is the only selector allowed to name the inset. It is proved both
ways — giving the inset back to the rack fails, and taking it off the bar fails
— and it protects the next move of the furniture as much as this one, since
whatever ends up last on the phone has to be the thing that pads, and only that
thing.

### v0.58.0 — 2 September 2026

**The bays are a bar at the foot, and the chevrons line up.**

Four cells across the bottom of the phone — a drawn icon over the bay's own
name, with the count as an orange badge on the icon. The bar is the last thing
in the document, under the tray, so the way between places sits where the thumb
is and the tray goes back to being content rather than furniture.

**Where the bays have been.** One row at the top that scrolled sideways with no
scrollbar, which put the fourth place off the right edge; then two rows of tabs,
which cost 98px of an 844px screen and put navigation at the far end of a reach;
now a bar that cannot fail either way — four cells, one grid, no scroll at any
width, always in the same place.

**No bay is lit when you are not standing in one.** A shelf, a search and an
opened strip light nothing, because a bar that says you are in the notes while
you are reading a search result is a bar you stop believing.

**The icons are illustrated and full-colour**, which is a deliberate exception
to a world drawn in flat shapes and 2px outlines, and it is recorded in
DESIGN.md as one. They belong to the mark's register rather than the board's —
the squirrel in the ops bar is drawn the same way — and four line glyphs at this
size would have been four grey rectangles at a glance. 64KB for all four,
exported at 128px so they hold at three times.

**The badge is the strongest form a count has taken in this product.** It is a
choice rather than drift: the rule against counts was retired for doors, and the
bar is the door. No bay ever wears a nought.

**The chevrons line up.** The opener that press-to-open added sat between a
strip's words and its mark, and a mark is as wide as its words — *every week*
against *every 28 days* — so the chevron stepped in and out down the rack. It is
the last thing in the row now, in a column of its own.

**A real defect, found through a flaky assertion that was telling the truth.**
Every page load drew the stamps open and animated them shut: the collapse
carried its own transition, so the moment the script applied the collapsing
class, 160ms of closing ran on every strip. The transition is gated behind a
second class the script adds a frame later.

Two tests changed because the furniture moved rather than because a promise did,
and both are worth knowing. The rack-fills-the-screen test asserted a magic
760px that the bar's own height broke; it reads the top of whatever sits below
the rack now, so it cannot rot the same way again. The sticky-tabs test went
with the sticky tabs, and the half worth keeping — that the pulled strip gives
way — is its own test.

The asset guard also changed: the four icons are named by a range over the bays,
so no literal file name appears in the template source it scanned. It reads the
rendered board now, which is stronger evidence than template text.

### v0.57.0 — 2 September 2026

**A strip opens when you press it.**

A note strip was 162px on a phone, so three of them filled the screen and one
and a half were left once something was pulled. Closed it is 44px — its words, a
chevron and its mark — and five notes are in view. Pressing anywhere on a strip
shows its stamps and shuts whatever was open; Escape closes; the chevron carries
`aria-expanded`.

**The gate is `(hover: none) and (pointer: coarse)`, not the width.** A tablet
has no hover either, and the desktop's open-on-hover is no use to it.

**The pulled strip gives way.** Below 620px everything under the ops bar is one
scrolling deck — the pulled strip, then the tab row, then the rack — with the
tab row sticky inside it. What you are looking at scrolls and the way between
bays does not. It held the top of the board before whether or not you were
looking at it.

**The rule this settles** is written into DESIGN.md as *The Open Strip Is The
Focused Strip*: hover and focus open a strip on a desktop, a press does on a
touch screen, and a press is what focus means there. An arrow moves to a strip
and opens it, a letter acts on the one that is open, and there is never a strip
that is focused and shut.

**On the script.** This is the release where the standing reading of "no script"
was corrected. What the suite actually pins is that no *place* needs JavaScript
to be reachable — `TestTheRoomsNeedNoScript` — and that the offline capture path
survives. Interaction was never covered by it, and `board.js` has owned the
1150ms hold, the strike and the whole keyboard layer since the board was
mounted. Designing around the wider reading was producing worse mechanisms; the
base layer here is not a fallback but what already shipped, and with `board.js`
gone every strip is open exactly as in v0.56.1.

**The bug inside the fix, recorded because of how it hid.** The collapse was
first written as the `grid-template-rows: 0fr → 1fr` pattern, which clips: a
strip's words are a grid item stretched by the row their own content sets, so
the fraction resolves against an imposed height and the second row of stamps is
cut. It sliced `MAKE A CHORE` in half. It is `max-height` now — and the reason
nothing had caught it is that **a headless browser reports `hover: none`**, so
every desktop screenshot and browser test in this project has been taking the
flex path. The grid path had never been rendered by anything that looks at it.

Also in this release: the 15.8MB compiled `devscreen` binary that had been
committed at the repository root since the dev screen arrived in v0.46.0 is deleted, and its
path ignored. Nothing ran it — every caller uses `go run` — but `go build` wrote
straight over it, so looking at the dev screen once left 15.8MB of unrelated
diff staged.

### v0.56.1 — 2 September 2026

**The board on a phone.**

The screen the new world was never checked on. A photograph of a real phone
showed six things wrong, four of them one bug each: an ops bar that never shrank
below 620px, so five things fought for 390 pixels and the link to Buddy wrapped
onto three lines while the find field was clipped mid-placeholder; four bay tabs
scrolling sideways with no scrollbar, which put the agenda off the right edge; a
clock reading 11:29 two inches under the phone's own 11:44; an empty rack that
said nothing at all; and half a viewport of purple under the ledge.

**The tab row is the one worth recording.** A sideways-scrolling row of places
with no scrollbar is how the fourth place became unreachable on 28 August, as
the rooms. It came back a week later as the bays and nothing caught it, because
nothing tested that every bay is on the screen. Something does now.

**The ops bar loses its words** — two drawn glyphs, a magnifier and Buddy's own
acorn, both keeping their words in the markup where a screen reader still reads
them. The clock goes: it is rendered when the page loads, so it is wrong by
however long the page has been open, and the phone's own is right there.

**The glyph is the field's label**, so pressing it focuses the field and
focusing it is what opens it, across a second full-width row. No script is
involved in any part of that. A field carrying a query stays open, because a
search you cannot read is a search you cannot correct.

**A rack says when it is empty**, in its own words — *nothing in the notes*,
*nothing comes back today*, *nothing in the tasks*, *nothing left today* — and
never at the same time as the trouble line, because a rack that could not be
read must not also report a quiet morning. The shelf and the search say the same
kind of thing rather than drawing an empty bordered box.

**Dashed means one thing now**: there is nothing on this yet. It belongs to the
blank strip and what sits inside it, and to the two notices that are not part of
a rack. The bay tabs and the ledge tabs were wearing it and read as
placeholders; they are solid. A test holds the vocabulary closed.

**The body stops being a grid with a fixed row count.** It had four named rows
and between three and six children depending on what was happening, so with
nothing pulled the racks landed in a `min-content` row and the app ended halfway
down the screen. It is a flex column and the racks are the part that grows,
which closes the same latent bug on the desktop.

Also: the dev screen did not build — its store had gone four methods stale
behind the `Store` interface, so `make dev` failed on the branch that shipped
them.

Not addressed, and recorded so it is a decision rather than an oversight: on
touch every strip shows its stamps at all times, which is why three notes fill a
phone screen. That is the strip board's thesis rather than a phone bug.

### v0.56.0 — 2 September 2026

**The app is a strip board, not a conversation.**

The complaint was that everything felt scattered and there was no way to focus
on two things at once, which is what a conversation is: one thing at a time, in
the order it was said, and the state of anything you are not looking at is
somewhere further up. Three worlds were mocked against it — a dashboard of
panels, a desk, and a strip board. The board won on the one thing the other two
could not do: a strip is a thing you can pick up, answer and put down, and four
racks of them are readable at a glance without any of them being a number.

**Colours and fonts are the only things kept.** Everything else was replaced:
`board.html`, `board.css`, `board.js` and `board.go` are a second application
sharing the store, and it took the front door rather than sitting beside it.
DESIGN.md was rewritten from 2,987 lines to 650 around the new world's own rules — The Holder
Rule, The Four Holders, The State Is A Mark Rule, Green Is Only Done, Never A
White Surface, and the rest.

**A strip carries its own answer.** The two-step it replaced — press a row, then
answer the card that appears — is gone: every stamp is on the strip itself, so
nothing is ever held in a form field waiting for a second decision. What leaves
a rack lands in today's tray at the foot of the board, with 1150ms of hold to
undo it.

**The four rooms became four bays**, and then stopped being places at all. Their
URLs answer 301 to the rack that holds what they held. The record keeps its
rooms — every turn still carries the room it was said in, new turns still file
by subject, and Buddy's room reads the whole record — so nothing that was ever
said is out of reach. The four survive as sets he can be asked to draw.

**The conversation is Buddy's room.** It is where you talk to him and nowhere
else; answering happens on the board, so triage left the conversation with the
doors.

**A rack says that there is more and never how much** — *there is more further
back*, under the last strip it can hold. The same rule the whole product has
followed since the beginning: a capped list may say *that* there is more.

**Two capabilities were retired on purpose** and are recorded as decisions
rather than omissions: the notice the coach could make about a set, and the
live edge, which a rack is by construction.

**The phone shows one bay at a time**, chosen by four tabs above the racks; the
desktop shows all four side by side. One application, collapsing, rather than
two.

**The keys were drawn and never read.** Every stamp carried a letter from the
day the board was mounted and nothing listened for one — found only when the
rooms' own keyboard tests came looking for a home during the retirement. Letters
act on the strip you are focused in, arrows move between strips, and a letter
nothing answers to does nothing.

No schema change: the rollback is the image tag alone.

### v0.55.1 — 31 August 2026

**How you felt before is a turn, not a page.**

The last screen in this product that was not a conversation. Six weeks by seven
days, reached by a link from a chip that lasted one turn — so asking for it took
you out of the room you were standing in, and it was reachable from nowhere else
at all. It is a press now, from that chip and from the settings panel, and it
draws where you are.

**Kept rather than drawn**, unlike a room's list, and the difference is the
point: a list is the state and goes stale, where a reading you asked for in
August is what August looked like. The same rule the two shelves follow.

**The grid is unchanged** — the gaps drawn as empty outlines, nothing counted or
trended. It moved into `drawn` and spans the turn's full width the way the faces
and the pickers do.

**A vocabulary rule caught the settings label.** "How you have been" was the
obvious name and *you have been* is on the same list as *overdue* and *streak*:
a phrase that characterises the person rather than reporting what they said.
`TestTheCheckinSaysNothingAboutYou` scans the whole page and refused it. It is
*how you felt before* in both places now, which is the name the chip already
had.

**`/moods` answers 301** — it may be on a home screen — and `backFrom` loses its
one remaining destination, because there is nowhere else a timer can be stopped
from.

### v0.54.0 — 31 August 2026

**How you feel, asked every hour, in the room where the asking happens.**

Six hours before, on the owner's instruction. What makes an hour bearable is
the second half: the question is drawn at the edge and never written. It was a
turn on the argument that a question the record cannot show you answering is
half-recorded, and that argument was made when it came four times a day. At
hourly it inverts — sixteen "how do you feel?" would be most of a record whose
job is to hold what you said. Your answer is still kept; the reading has always
lived in its own table.

**Two clocks where there was one.** *Should I ask again* is an hour;
*does this reading still describe you* is six. `Checkin.Fresh` answered both,
so changing the number would have quietly shortened how long a wiped afternoon
keeps Squirrel gentle. `JustAsked` is the new one and `Fresh` is unchanged.

**Two guards had to learn the question left the record.** "Do not hand somebody
a job while you are asking how they are" was enforced by looking at the end of
the conversation, and the question is not in the conversation any more. Both
now ask whether it is being asked *now* as well.

### v0.53.0 — 31 August 2026

**Everything about you is in one place, and it is where it says who you are.**

Three account-level concerns were transient chips that scrolled away: a floating
button reading *tell me when to leave* that hid itself the moment it was
answered either way, *what do you know about me* beside one model-written reply,
and the mood history behind a chip that lasted a turn. None had a state you
could see or come back to. Once notifications were on there was nothing that
said so; once they were refused there was nothing at all.

**The identity block is a disclosure now.** Face and name as the summary;
notifications, what Squirrel knows, and the way out inside. A panel rather than
a page — settings is state rather than a conversation, the one thing on this
screen that is neither.

**A setting says which way it is set**, and the state comes from two places
because it lives in two. `Store.Notifying` answers whether anything would be
sent to; only the browser knows whether the permission was refused, and a
refusal cannot be re-asked by a site, so what is offered there is where the
switch is. A state that cannot be read says so rather than guessing.

**And there is a way off**, which there has never been. `StopNotifying` retires
every browser with the same `gone_at` a dead endpoint gets — a subscription you
turned off and one that stopped answering are the same thing to the sender, and
a second column would be a second thing to check before every send. The script
drops the browser's own subscription in the same press: either half alone
leaves a notification from nowhere, or a row sent to for ever.

**The label names the setting; the sentence says what it does.** *tell me when
to leave* is a lovely sentence and a poor name for a thing with two positions —
the same lesson "close the door" taught two releases ago.

**The mood check-in stays in `everything`,** on the owner's call. It is not a
setting; it is a question the product asks, and it belongs where the asking is.

### v0.52.0 — 31 August 2026

**A room's list stops being a turn, and the chores stop going stale.**

Reported as "can't get to the list of chores", and it was worse than that. A
room wrote its list into the conversation and then refused to write it again
while the last turn still had something on it to act on — which a list always
does. So every visit after the first showed a photograph of the first one: a
chore you had done still asking, a count that no longer counted anything.

**The category error is the finding.** A list is not something somebody said.
The record is the conversation; the list is the state; state kept as history is
stale by definition. `#edge` is a second container below `#thread` holding what
is true now, drawn on every arrival and written nowhere. `endsOpen` no longer
guards it, because there is nothing to duplicate into.

**A press answers in two places** — the conversation gains what you said, and
the edge is asked for again and replaced. The answer carries `X-Edge: 1` and the
script refuses anything without it: a handler that did not know the header
answered with the whole screen, and it landed inside the element that asked. The
front door was doing exactly that for an hour.

**Three things only a browser could have caught.** The edge had none of the
transcript's layout, so `--gutter` was undeclared and Buddy's face rendered at
118px. The keyboard shortcuts assumed the live edge was the last turn in the
conversation, which is where it stopped being. And replacing the edge threw away
whatever had focus — on the chores the focus *is* the selection, so the next
letter would have acted on nothing.

**A test of mine was vacuous and a mutation found it.** It asserted the room
redraws by looking for a chore name anywhere on the page, and the name was also
in the scrollback. It reads the edge alone now, and only then did restoring the
old guard fail it.

### v0.51.0 — 31 August 2026

**The conversation says when it was.** The record goes back weeks and nothing on
it said when anything happened, so "what did I tell it about the boiler" could
be found and not placed.

**One time per run, under the face.** Consecutive turns by one speaker are one
utterance — already why a face is drawn once per run — and one time is the whole
truth about an utterance. A clock on every bubble is a minute-by-minute record
of your afternoon, which is what the run-resumption sentence refuses a clock
time to avoid; the rule that sentence was written under is intact.

**The day is a seam.** *today*, *yesterday*, or the date, centred across the
conversation and drawn only where the day turns over. Those two by name because
they are the days you are most often looking for.

**Three attempts at where it goes, and only a browser could tell them apart.**
In the words' column the time pushed every run down by the height of an avatar.
In the gutter on a line of its own it read as a label on whatever came next.
Face and time are one element now. All three were correct markup, which is why
the test measures boxes rather than reading the DOM.

**Read in your zone.** `Options.Location` reaches the turn through the
middleware that already carries who you are, rather than being threaded through
fifty call sites — fifty chances to hand over the container's zone, which is
issue #148.

**A test caught its own time-of-day dependency.** `TestTheConversationSaysWhenItWas`
put a fixture two hours before `now`, which is the same day at noon and the day
before at one in the morning. It was written at one in the morning and failed
immediately; the clock is pinned now.

### v0.50.0 — 31 August 2026

**Seven rooms became five, and two of the seven were never rooms.**

The rail had been carrying two different kinds of thing under one shape. Five
were places you keep something. Two — *the things you kept* and *what you set
aside* — were **states a note is in**, promoted to doors on 28 August because
there was nowhere else to put them. A state with a door on the rail reads as a
fourth pile to stay on top of, which is the weight this product exists to
remove.

**everything · the notes · the chores · the agenda · the tasks.** The pile and
both shelves are one room; the shelves are two chips on every branch of its
turn, drawing where you already are. Buddy's room is named for what it holds
rather than for who is in it — *general* was the other candidate and is Slack's
word, the register `#chores` was refused for when these rooms were built.

**A shelf is still a place and is no longer a room**, which is the distinction
the whole change is about. `placeName` covers seven things where `doorName`
covers five, so Buddy's `open` still draws a shelf by name and `/open` still
pages one. Nothing about either shelf's contents changed.

**The record moved with the rooms.** Migration 0035 collapses three room keys
into one and renames a fourth, so it is a data migration and not a schema one —
and it is proved by rewinding its own row in `schema_migrations` and running it
over turns that are already there, which is the only ordering it will ever meet.
Reverting either `update` fails the test.

**The four URLs that stopped being rooms answer 301.** A room was a URL you
could put on a home screen.

**Stopping went, on the owner's call.** `/enough` was one sentence behind a link
at the foot of the rail, and the link was the whole of it — a door to a
sentence, under a rule as if it were the important thing there. What the screen
was for still happens: the run is forgotten where a place is entered rather than
where a link was pressed. Its composition was already the way in's, so the
drawing stays and only the screen goes.

**The button walk follows the shelves.** `TestEveryButtonAnswersWithAFragment`
walked seven GET screens; a shelf that stopped being a URL would have quietly
left its cards uncovered, so the walk presses both chips and walks what comes
back.

### v0.49.0 — 31 August 2026

**A press answers in the room it was made in.** Four reports from the phone in
one evening, and three of them were one defect wearing different clothes.

`pile.js` bound the dock's whole enhancement to `.slot[action="/capture"]`.
That is the dock in Buddy's room, the pile and the two shelves — and not the
dock in the agenda, the chores or the tasks, which post to their own routes. In
those three nothing intercepted the press: the browser submitted the form
itself, the server answered its 303 to `/`, and you arrived in Buddy having
filed the thing in the room you had been standing in. The selector was written
when there was one dock and never revisited when there were seven.

**Binding it everywhere then found the second half.** Those three docks have no
camera, and the handler read `input.files` off nothing. A throw inside a submit
handler that has already called `preventDefault` is the worst shape a failure
can take: the press does nothing, and says nothing about doing nothing.

**Sixty-one redirects named `/`.** `answerWith` and every fallback in the
package sent you to Buddy's room, which was the only room there was until
28 August. They ask the request which room it is in now. This is the scriptless
floor for the same bug, and it is why the fix is in two places rather than one.

**Turning the calendar's month said the question again.** Paging to November
left five "Which day?" in a record that is never rewritten — paging is not
something you said. The turn comes back under the id it already has carrying
`X-Replaces`, and the script swaps it where it stands; nothing is kept, so a
reload shows the question once, in the month it was first asked in. Without the
script it is kept and redrawn the old way, which is the floor holding.

**An appointment can be at a quarter past.** The three times were the
vocabulary rather than a shortcut, so this product could hold an appointment at
09:00, 14:30 or 18:00 and no other. There is a field that takes any time now,
with the three beside it filling it in, and the guard went from a membership
test to a clock check. `TestATimeThatWasNeverOfferedDoesNothing` had pinned the
old rule; it now pins what still holds — something that is not a time at all
still does nothing.

**The way out is named rather than narrated.** "close the door" was the
product's voice doing the wrong job on the one control you look for by name at
the moment something has gone wrong. It is **log out**, under an authored door
with the way through it, on the same 2.4px round-capped stroke as the chevron
and the camera.

**Your face is round in the rail.** It drew a bare `<img class="youface">`, and
the rule that rounds a picture is `.youface img` — which can never reach it. One
markup in both places now.

**Ten mutations, ten assertion failures.** Two of them are the ones worth
keeping: reverting the dock's selector fails three rooms with "the press did
nothing at all", and reverting the camera guard fails four. Neither would have
been caught by reading the diff.

### The dev screen — 28 August 2026, unreleased

**Everything this product looks like is compiled into the binary**, and that had
a cost nobody had named: `//go:embed templates/*.html` and `//go:embed static`
mean editing `pile.css` does nothing to a running process, and `pages` is parsed
once at package init. Three things had nowhere to run because of it — impeccable's
live mode, the design detector's overlay, and any test of the service worker by
hand, which needs a real origin and a real network to cut.

`make dev` serves the screen on a port with invented contents: no database, no
model, nothing that survives the process. Templates and static files come from
the working tree, templates re-parse per request, and nothing is cached, so an
edit is a refresh.

**The build tag is the safety argument rather than a convention.**
`EnableDevelopment` and `DevServe` live behind `//go:build dev`, so a binary
built without it does not contain the code that could set `devDir` — verified by
`strings` on both binaries, which finds neither symbol in the production one. The
checks that read `devDir` compile in either way and are simply never true.

**It caught a bug in its own making, which is the part worth recording.**
`stampOf` walks the tree it is given and used to be handed the whole embedded FS
to walk a `static` prefix. Handing it the static directory itself — which is what
serving from disk needs — made the walk find nothing and return the SHA-256 of
empty input: a constant stamp, in every asset URL, under `max-age=31536000`.
That is exactly the v0.7.0 failure the comment on `assetVersion` describes, and
it is silent. `TestTheStampIsOfTheFilesAndNotOfNothing` now fails on it.

**What it unlocked immediately: the offline path, proved.** Flagged five times
since 28 August and never verified, because it needs a real origin, a real
service worker and a real failure — and the worker is the one part of this
screen Go never runs. `node scripts/offline-path.mjs` starts a dev screen, kills
it, types a chore into the dead server, brings it back and reports what
returned. On 29 August: the worker held
`{text, room:"chores", action:"/chores/name", field:"name"}`, the chore came
back a chore with its how-often picker, **0 in the pile**, **0 still held**.

**The first attempt passed while proving nothing**, which is the part worth
keeping. CDP's `Network.emulateNetworkConditions` with `offline: true` does not
reach the service worker's own fetch: the POST returned 200, the worker held
nothing, and the chore arrived correctly by the ordinary online path. Stopping
the server process is the only cut that reaches through.

### Rooms — 28 August 2026, unreleased

**One conversation became seven, each with its own Buddy.** Spec:
`docs/superpowers/specs/2026-08-28-rooms-design.md`.

**A room is two things, and the second is the point.** It keeps its own
conversation, and it narrows what Buddy can do in it. A room that kept its own
history while answering with the whole product would be a filter on a
transcript.

- **`turns` gained a `room` column**, defaulting to `buddy` — which is what
  backfills the record. Everything said before rooms existed was one
  conversation, and the room a conversation lives in is Buddy's, not the
  pile's: the record holds his openings and his answers as well as your notes.
  `EverythingSaid` is the one unscoped read, for the learning tick: rooms
  partition the screen, not the person.
- **Doors became rooms.** A door was a POST that appended "the pile" to the
  record on every press, which made a record of walking around. A room is a GET
  at `/r/{room}`, and it draws its own state on arrival only when the
  conversation ends with nothing to act on. `/open` survives as a redirect for
  an installed home screen's cached forms, and keeps "the rest", which is paging
  inside a room and genuinely something you said.
- **The menu became the rail.** Furniture beside the conversation on a desktop,
  a sideways strip above it on a phone. Counts on the four rooms that earn one,
  kept against the scoreboard evidence on the owner's decision of 27 August.
- **The dock names its consequence** — *Make a chore*, *Put it in the pile* —
  because the room decides the filing with no confirmation step and a
  placeholder is invisible by the third day.
- **Buddy is narrowed per room.** The chores cannot complete a task, make one,
  or be handed one when they ask what is open. The agenda cannot write at all
  and may only propose a moment. The shelves only talk. Enforced twice: the
  tools he is offered, and a refusal at dispatch for the ones he was not, since
  a model can name a function that was never in its list.

**What the tests found that the plan had not:**

- `sw.js` held `{text}` and replayed everything to `/capture`, so a chore typed
  with no network came back a pile note.
- `Home` meant both "the front door" and "this is a conversation", so every
  room rendered without `thread.js` — the fragment posting, the live edge and
  the chore keys all live there. Nothing looked broken, because the forms fall
  back to full navigations. Only a browser test found it.
- The test mux resolved overlapping routes by pattern length, so `/r/{room}`
  beat `/r/buddy` by one character and every test asking for Buddy's room
  reached the generic handler while the server reached his own.
- `noticeAbout` was asked with the display name — "the tasks" is not a room
  key — so the door's own line went unnarrowed under a name that looked right.
- `TestNoChipInTheConversationIsALink` banned links outright; the rule
  underneath was that a chip must not point somewhere dead. It now resolves
  every href against the route table *and* checks the query is one a handler
  reads — checking the path alone is what let `/?open=chores` ship, since that
  is `/`.

**The appearance snapshot now records `transform` and `box-shadow`.** Without
them `.rail .room.in` recorded byte-identically to `.rail .room`, so the one
mark saying which room you are in was pinned by nothing. It is also the first
thing that has ever watched the sticker offset, which is the depth rule of the
whole system.

**The cost of per-room Buddy, measured 28 August 2026.** Serialising the tools
exactly as they go on the wire, plus the room's own preamble line. Bytes are
exact; tokens are bytes/4, an approximation used only to put the number in the
ceiling's unit.

| room | tools | bytes | ~tokens | vs Buddy |
| --- | --- | --- | --- | --- |
| Buddy | 15 | 4756 | 1189 | — |
| the chores | 9 | 3390 | 847 | −28.7% |
| the tasks | 11 | 3371 | 842 | −29.1% |
| the pile | 8 | 3096 | 774 | −34.9% |
| the agenda | 5 | 2234 | 558 | −53.0% |
| the things you kept | 4 | 1463 | 365 | −69.2% |
| what you set aside | 4 | 1461 | 365 | −69.3% |

**It goes down, and not by enough to matter.** At the routine tier's 20 cents
per million input tokens the largest saving — 824 tokens in a shelf — is about
€0.00016 a request. Against ≈€3.72/month, nothing here moves the ceiling in
either direction, and the feature was never justified by cost.

**The window was the direction that could have gone the other way, and did
not.** Keying it by `(person, room)` means more windows, but each request
carries one of them and a room's own is a subset of what the single window
held. Per request it can only shrink.

Both directions are pinned by tests rather than left as arguments —
`TestARoomNeverCostsMoreThanBuddysOwn` and
`TestARoomsWindowIsNoBiggerThanTheOneItReplaced`.

**What the measurement cannot see** is the behaviour: losing the thread when
you change rooms could make you repeat yourself, which is *more turns* rather
than bigger ones. Nothing here would catch that, and only use will.

### v0.43.0 — 26 August 2026

**The box can show you a place, and a request to see one is not a note.** Two
defects, found by using the product an hour after v0.42.0 shipped.

**The fix shipped on the wrong surface.** v0.42.0 gave Buddy an `open` tool on
`/buddy/say` — the route behind *ask Buddy* in the menu. The dock is a different
path entirely: what you type goes through `Reads`, a deliberately cheap one-shot
call carrying a single `answer` tool with `say` and `keep`, no facts and no
places. It could not see the chores and said so, which was true. The dock is, in
its own preamble's words, "the one box this product has", so the capability had
landed on the route almost nobody presses.

`open` joins `say` and `keep` on that same call rather than arriving as a second
tool, because they are one judgement: *this was a request to see the chores* and
*here is what to say about it* are the same sentence being written.

*The wider version was measured and deferred rather than skipped.* Handing the
dock the full toolset costs five to six times as much per question — affordable
against a €10 ceiling — but it is two or three sequential calls on the most-used
input in the product, and it would mean typing "done with the vet" completes a
task. That is a change to what the box is rather than a fix for this, and it
gets its own study.

**A request to be shown something was being filed as a thought.** `show chores`
has no question mark and no asking opening, so the rule kept it — twice in one
minute, for somebody trying to look at their chores. The product answered a
request to look at something by writing it down. The rule now recognises a verb
of showing followed by the name of a place and takes both: `show mum the photos`
is still a thought, and so is `open the compiler`, which is why the place has to
be a whole word rather than a substring. That case is in the table because the
unbounded version survived the first mutation round.

**One of the new tests was vacuous, and a mutation caught it.** The thought case
used words that never reach the reading path at all, so it passed with the
behaviour reverted. It now uses words the gate lets through and asserts the path
was reached — which is the case that actually happens: the rule sent it to be
answered and the model read the whole sentence and disagreed. Seventh time this
shape has been caught in this project, and the first time it was in a test
written the same hour as the fix it covered.

Seven mutations. No migrations.

**A weekday chore read its day in UTC.** `extract(dow)` and `::date` follow the
session timezone, not the person's, so a chore set to come back on a Thursday
was Wednesday's chore between midnight and 02:00 Amsterdam — the hours this
product is most likely to be open. The scan now reads the day where the person
is, and the test pins 00:30 Thursday in Amsterdam, which is Wednesday in UTC.

The zone that was wrong here is the *database session's*, which is `Etc/UTC`,
and it is a different clock from either of the two the pod carries: `TZ` and
`DIGEST_TZ` are both Europe/Amsterdam, and neither reaches a query. So the SQL
has to be told — `Store.zone()` is the name it hands to `at time zone`, falling
back to `current_setting('TimeZone')` when nowhere is configured, because
`time.Local` stringifies to "Local" and Postgres does not recognise that.

Same family as #148 and the two after it, and a different clock again: those
were the process's, this one is the connection's.

**The deck's stylesheet and script outlived its markup, and kept each other
alive.** The one-card triage screen came out in v0.41.0 when the conversation
replaced it. Only the HTML went. Every reachability check said the CSS was in
use because `pile.js` declared `const card`, and said the JS was in use because
the stylesheet had `.card`. Neither renders.

`pile.css` went from 3,740 lines to 1,596 — 300 rules whose classes appear in no
template and no script, thirteen colour variables nothing reads, two
`@keyframes` nothing plays. `pile.js` went from 971 to 557: the stamp, the
delayed-write machine whose debt was only ever incurred inside the stamp, live
search over a field no page renders, and the letter keys that acted on a card.
The undo chain in Go went with them — `undoFrom` is called from nowhere and
`view.Undo` is never set, so `undoView`, `saidWords` and `backTo` were feeding a
template block that could never render. Also ten unexported functions, two
exported ones superseded and left behind, an `aria-live` region nothing has
written to since live search went, and 62 KB of embedded PNGs no page names.

*The appearance snapshot regenerated byte-identical at every step*, which is the
whole proof: nothing removed was drawn.

**Nine tests were guarding the wrong side of a change.** The two that said Mount
refuses without a spool and without an owner had byte-identical bodies, so the
second passed on the first's check and `Gate` and `Login` had no test at all.
The isolation sweep — the test that says one person cannot read another's pile —
listed `/photo/` twice where it meant `/photo/` and `/photo/{id}/thumb`, so the
thumbnail route was never aimed at somebody else's row. The link walker had a
branch keyed on a map entry that does not exist, could not match a `{id}`
wildcard, and read only `href` and `action` — never `src` — so a card's
photograph could point at a dead route unnoticed. The route table named 55
routes and asserted 56. Nine assertions checked for strings that exist nowhere
in the repository, and three read a 301's forty-two-byte body.

**The race detector had never run.** `TestConversationsSurviveConcurrentUse` has
no assertion of its own — the detector is the assertion — and its comment said
CI ran it that way. CI did not. Both `make` targets pass `-race` now, at a cost
of about ninety seconds, and removing a mutex produces a data race report.

**Five things are now checked that were not.** That `truncateAll` reaches every
table, which is a property of the schema rather than of the seven tables it
names: a table added without a foreign key to them would leak rows into the next
test, and the symptom is a test that passes alone and fails in a suite. That
every embedded asset is one some page asks for. That a snooze has a floor as
well as a ceiling — only the ceiling had a test, and zero hours hands back a
deadline already past. That the four cards Buddy asks questions with are cut
from one description of the card stock, measured in a browser. And that a
comment naming a test names a test that exists, which had already drifted once,
in the same commit as the rename that caused it.

Eleven mutations. No migrations. Net −5,658 lines.

### v0.42.0 — 26 August 2026

**Buddy can open a place.** Asked *can you show me the tasks?*, he answered that
he did not have the task list available. He was right, and that was the bug.

Three rules met to make the question unanswerable. `Guard` refuses any reply
containing a list — bullets, numbers, headings. The brief is two sentences at
most. And he had no tool that opened anything. So he could read the tasks,
`open_work` having returned them, was forbidden to recite them, and had no way
to show them: the only honest thing left to say was no. The menu, in the same
lid, did it in one press.

**When the places became messages in v0.41.0, the menu learnt to open them and
Buddy did not.** This is that press, given to him. One tool, `open`, and the
place arrives as cards in the next turn — the same turn `placeTurn` has always
produced for a door.

*He opens it rather than offering a chip.* Opening a place changes nothing, so
it asks no permission and the worst a wrong one costs is a scroll; a chip would
put a press on the most ordinary request in the product. *It arrives as his
turn, not yours* — pressing the menu is an utterance and belongs in the record
as one, where Buddy opening a place because you asked is his answer, and putting
"the tasks" in your mouth would be the record inventing a sentence you never
typed. `placeSaid` is split out of `placeTurn` for exactly that. *The screen
only*: `Turn.CanOpen` is set by the surface, because what a surface can draw is
a fact about the surface, and a place in chat would be the list the guard exists
to refuse.

**One mutation survived the first pass, and it was the seam.** `Open` is a field
in an inline literal in `coachWeb`; dropping it left every test green while
Buddy said "here they are" above nothing at all — a reply claiming to have done
something it had not, which this codebase calls the worst failure available. It
is the fourth field to earn a wiring check for the reason `Push` cost three
releases to learn, and `TestTheScreenIsGivenThePlaceToOpen` is that check. Eight
mutations in total.

`nowFor` is nil-safe now, which its own comment already asked for: every read in
it fails soft, so a missing store should be the softest failure of all rather
than a panic.

No migrations.

### v0.41.0 — 26 August 2026

**The second draft of the conversation.** Everything moved into the thread over
the previous week, and then the thread was never redrawn for what it had become.
Three rounds of mockups, argued and recorded in
`docs/superpowers/specs/2026-08-26-the-second-draft-design.md`, and all eight
sections of it are here.

**Buddy loses his bubble.** His words are set plainly and yours stay in a bubble,
which is the whole grammar of the screen — and it was invisible while both wore
the same shape. Two voices drawn the same way is a transcript; one bubbled and
one not is a conversation.

**The frame comes off.** The rail of four doors, the chip row under every turn
and the stop link below that took about a fifth of a phone before a word was
said. All three went together. Every destination is in the menu, permanently,
with its counts — navigation and copy are two different jobs, and Buddy
mentioning a place in a sentence only covers the second. Doing only that makes
the product unnavigable ten turns in.

**Six kinds of thing, six bodies.** A note, a task, a chore, a fixed point, a
thing set aside and a thing kept are six different kinds of thing that were one
card with different words in it.

**Four verbs end a card, and the questions are behind a press.** Triage is two
tiers now: the four verbs that finish a thing, and `something else?` for
everything that does not. A card that offers eight things is a card that asks
you to choose how to choose.

**What's coming, first.** The most common reason to open Squirrel was three
interactions deep behind a door press while the conversation opened with
history. Buddy opens with it, as a turn rather than a standing strip — a fixture
that rewrites itself is a record that rewrites itself.

**A search result opens into a card.** A hit is a quiet full-bleed row with no
verbs, because a result is a thing you went looking for rather than a thing you
are deciding about, and dressing one as a card asks a question nobody posed.
Tapping it draws the ordinary card with the ordinary verbs, built from the
note's real state. The quiet row and the deciding card are two states of one
note and the tap is the step between them.

**The readings are a calendar.** Six weeks by seven days, replacing the list of
the days you had answered. *The gaps are the honest part*: the days you said
nothing are most of what is there, and a list can only ever show the days you
answered. Days that have not happened are drawn as nothing rather than as gaps —
an empty Friday next week outlined like a missed day says you are already
behind. A day answered twice shows what the day came to. Nothing said at all
draws no grid, because forty-two empty outlines is a picture of forty-two days
you did not check in, which is the one judgement this page could still make by
accident. Five mood colours of their own, deliberately not the state colours: a
fill in this product means an outcome, and lending green to `good` would make
one colour mean two things the first time a note and a reading shared a screen.

**A first run plays the loop through once.** An empty conversation with a box in
it saying *put it down here* asks a stranger to trust the product with the
contents of their head before showing them what it will do with it. Buddy plays
the loop dimmed instead — a capture, an answer, a chore offered — then rules a
line and hands over. Three turns, because three is the whole loop. It is never
stored: the first thing this product does must not be to write a false memory
into a record whose whole value is that you made all of it. It is inert by
construction — the verbs are spans and there is no route they could reach — and
it sits outside the thread, so the live edge can never be a turn nobody said.

**Two defects the suites caught rather than a reading did.** The browser tests'
live-edge selector matched the worked example's last turn, which is what moved
the example out of `#thread`. And the mood cells carried offscreen labels
sitting on a coloured ground at 2.67:1; each cell is `role="img"` with an
`aria-label` now and has no text node in it at all.

Eleven mutations this round, and one that *failed* to kill — a redundant rail
check on the first run — so the check was removed rather than left as an
unpinned condition. No migrations.

### v0.40.0 — 26 August 2026

**Four things it did not do.** Chosen from a design study — three rounds of
mockups, recorded in `docs/superpowers/specs/2026-08-26-the-second-draft-design.md`
— and built cheapest first. None of the four needs a model.

**Keeping your place.** You start triaging, the phone rings, and forty minutes
later the conversation opens as if nothing had been happening. Losing your place
is the failure this product is built around and it kept no memory of a run in
progress. Buddy says where you got to before anything else and offers both
honest answers, because after forty minutes away either can be the truthful one.

*Expiry is the design, not a tuning knob.* Three hours, and if it is ever
changed it should get shorter: coming back to yesterday's half-finished pile is
being nagged about an afternoon you have already had. The clock measures silence
rather than duration, so a long afternoon of triage never goes stale. No count
is stored — a stored one could lie the moment a capture lands from Campfire —
and the sentence never names a clock time, because "40 minutes ago" is a fact
about the gap where "you stopped at 14:12" is a record of your afternoon.

**Chores that come back on a day.** A chore was `every N seconds` from 0002 and
almost nothing real is. The difference is not vocabulary: an interval measured
from the last completion *slides*, so doing the bins a day late once makes every
reminder after it a day late too. `interval_seconds` is still written for a
weekday chore, so the tolerance gate, the asking window and everything that
renders "how often" keep reading the column they always read. The week parity
counts from the chore's creation rather than an ISO week number, which wraps at
the turn of the year and would flip every alternating chore in the house on
1 January.

**Something you set aside can speak up.** The three states shipped in August as
a one-way door: you park something waiting on the surgery precisely so that you
do not have to hold it, and that only works if something else is holding it.
The oldest one that has gone quiet comes back once, with how long it has been.

*someday never speaks up*, and that is the design. It is the state that means
"not now, and do not ask me". It is a mention rather than a task — no question
mark, no "should you" — and `still waiting` is the cheapest of the three
answers, because if saying "still" were harder than closing it this would be a
screen that pushes you to finish things.

**The hyperfocus exit ramp**, last because it is the only one that can annoy
you. The problem is not that you cannot stop; it is that the decision to stop
never arrives. Opt-in at the moment the timer starts, on the screen and nowhere
else — the chat, the coach and a nudge all start timers nobody ticked a box for.
It says once, and is marked said in the same breath as being drawn: if the mark
cannot be written the line is not drawn at all, because not said beats said
twice. `leave me alone` means today rather than this timer.

**Three real bugs, each found by a test written to bite.** A clearing branch in
the chore rhythms read the chore returned by `UpsertChore`, whose `RETURNING`
does not carry the new columns, so it was dead in production. `StartTimer` did
not clear the exit ramp's opt-in, so a timer started by the chat would have
inherited it and interrupted somebody who never asked. And a table proving a day
is refused against days and months used counts the *count* check refuses anyway,
so it passed with the unit check deleted — the fourth time this month a test
looked like proof and was not.

Three migrations: `0030_runs`, `0031_chore_weekday`, `0032_exit_ramp`.

### v0.39.1 — 26 August 2026

**An unreachable authentik costs the way in, not the product.** v0.39.0 was
tagged, deployed, and took both clusters down. Production squirrel crash-looped
for about half an hour, and because Campfire does not retry a delivery it could
not make, anything said in the room during that window is gone.

Two bugs. The first is ordinary: the config trimmed the trailing slash off
`WEB_OIDC_ISSUER`, copied from the `WEB_URL` line above it. go-oidc compares
the issuer it discovers against the one it was given byte for byte, and
authentik publishes this one *with* a slash. The config test asserted the trim
was correct, so it shipped with a test pinning it — a reminder that a test
written from the same misunderstanding as the code defends the bug.

The second is a design mistake worth keeping the reasoning for. `NewAuthentik`
did OIDC discovery at boot and a failure was a boot that failed, on the
argument that "a Squirrel with no way in is not a working Squirrel". That
argument is wrong, and the shape of the outage is the proof: what went down
alongside the screen was capture, the drain and the Campfire webhook, none of
which have anything to do with signing in.

**A failure costs a feature and never the product.** The spool exists so that
an unreachable Postgres does not lose a note. An identity provider the *screen*
needs must not be a harder dependency than the database the *whole product*
needs — and it was.

The split is now by what a failure means. Configuration that is missing or
dangerous is refused synchronously, without a network, because it cannot come
right on its own: an empty `WEB_REQUIRED_GROUP` still refuses to mount.
authentik being unreachable is not refused at all — the gate says so, in the
sentence the screen already had for it, and tries again every thirty seconds
rather than on every press.

The immediate cause was a missing NetworkPolicy: squirrel's pod had no egress
to authentik's hostname, because forward auth never needed one. That is a
homelab fix. This is the half that means getting it wrong again costs the login
screen rather than the room.

One unrelated fix rides along. `TestSomethingTodayIsWhatBuddyOpensWith` built
an appointment three hours from now and asserted the card reads "today", which
is false after nine in the evening — on any branch, for a reason having nothing
to do with anybody's change.

No migration.

### v0.39.0 — 25 August 2026

**Proper OIDC: the application does its own authentication.** Squirrel's whole
authentication was one line — Traefik called an Authentik forward-auth outpost,
Authentik decided, and `guard` compared one header to one configured string.
That was the right size while there was one person and one pile. The outpost
could only ever say "somebody Authentik likes" and never *which* somebody, so a
second person meant a redeploy.

`guard` keeps its name and its position and loses its body. It reads a session
cookie, resolves it through a minute of memory, and puts the person and the
OIDC `sub` on the request. Sessions live in Postgres, hashed, so a database dump
is a list of hashes rather than a set of live sessions.

**`Options.Owner` is deleted.** It was a process-global `atomic.Int64` that
forty-nine call sites read through `opts.person()`, and it could not survive two
people: a second person's request would have read the first one's owner and
drawn their pile. Nothing about that was visible at a call site, which is why
the refactor landed as its own commit with every suite passing unchanged before
anything about authentication moved.

**The gate**, not the door — a door here is a section of the pile and has been
since v0.10.0. One screen, `/enough`'s composition, four states differing only
in the sentence under the mark, and a first arrival says nothing at all because
an arrival is not an error. The refusal never names the group somebody lacks:
that is a fact about the Authentik rather than about them.

**Two ceilings.** The coach's monthly budget was one number for the process,
applied to whoever asked, so every demo account would have been another monthly
allowance. `COACH_BUDGET_GUEST_EUR` is what anybody who is not the owner may
spend.

**The isolation sweep** is the part worth keeping even if the rest were thrown
away. Every store function already took a `personID` and every one was scoped by
it — and nothing tested that, because for the product's whole life there was one
person and nothing to leak into. Two people now get one of everything, and
thirty reads are walked as one of them. Five real scopes were removed one at a
time to prove it bites; it caught all five and named the leaked row each time.

Three dependencies arrive with it — `go-oidc`, `x/oauth2`, `go-jose` — in a
repository that had been pgx and testify since it started. The alternative was
hand-rolling JWKS fetching, key rotation and RS256 verification, where being
subtly wrong is a login that accepts a forged token.

**A migration**, `0029_sessions`. And a deployment order that matters: the
forward-auth middleware must not come off before the OIDC client exists and
Squirrel is serving the gate, or the deploy locks you out of your own pile.

Adding somebody to a group in Authentik is now the whole of admitting a second
person or a demo account. There is no redeploy, which was the point.

### v0.38.0 — 25 August 2026

**Buddy is the mascot, unframed.** v0.37.0 gave him a drawn acorn on a purple
disc. Both halves of that turned out to be wrong, and both were settled by
rendering rather than by arguing.

The acorn was the right glyph and the wrong idea: DESIGN.md's rule that it
exists for exactly this size was written when the only alternative was the
whole character, tail and all, which is mush at 40px. A **head-only** portrait
is a case the rule did not consider — the artwork's centre of mass is the head,
so a square crop centres it for free. The version with a tail needed every crop
to fight its own composition.

And the disc was a second outline on top of the first. The artwork already
carries the 3px ring this system draws on everything, so the two stacked read
as a black mass beside bubbles made of light cream. Bare, the head sits in the
field the way the door art sits on its cards — which is the world's own
language: a sticker has no plate under it.

Forty pixels rather than thirty-four. A circle crops to the face; a silhouette
has to fit the ears and the headphones inside the same width, so at the same
size the face comes out about a fifth smaller.

One test assertion was wrong and passed with a disc deliberately put back: the
computed border width is `3px`, not `px solid`, and a background *colour* leaves
`background-image` reading `none`. Each property is read on its own now.

### v0.37.0 — 25 August 2026

**Buddy has a face.** The product is built around a mascot who appeared nowhere
in the conversation he was having. It is the acorn, filled cream on purple,
exactly as the mascot wears it on its cap — and only Buddy has one, because
there is one person using this and he knows which words are his.

A turn became a gutter and a column, which also gave the place's name somewhere
to belong: it had been floating in the field beside the bubble it titled.

Three things the pass found rather than added. Buddy's bubble was `--paper`,
which is near enough to white on a surface whose second stated characteristic
is that white is not Squirrel. The two permanent chips were centred at the same
weight as chips that had just arrived, so they read as a page footer. And every
gap in the conversation was the same size, which makes a reply no more part of
the exchange than the next thing entirely.

One thing it refused to change: the stopping line is an outlined pill and stays
one. Pills are pressable and underline means *going somewhere else to look at
something* — a documented decision, and polishing it would have been undoing it
on taste.

**`.face` was already taken** — it is the check-in's mood button, and it
carries a 44px tap target, so Buddy's disc came out 34 wide and 44 tall. The
third class collision on this surface after `.tcard` and `.say`, and the second
found by measuring the rendered box rather than by reading the stylesheet.

### v0.36.0 — 25 August 2026

**The box asks the house first.** v0.35.0 sent every capture abroad to be read,
which is the opposite of the architecture this product states — *"rules narrow,
and the model answers the few that survive"*, the argument the interruption
pre-filter and the splitter are both built on. Ronald caught it the same day.

Three tiers, cheapest first.

**The rule** needs no model, no network and no cluster.
`LooksLikeAQuestion` says yes only when the sentence is doing nothing else: a
question mark at the end, or one of the openings that is a thing you say to
somebody rather than about something. It is biased towards keeping, and
deliberately — a question read as a thought is a note in the pile, and a
thought read as a question is a thought dropped out of it.

**The house** is a small model on the cluster, asked about everything typed. It
costs electricity in a cupboard rather than money abroad, so it may run on
every capture where the hosted one may not. It answers one word. Anything that
is not one of the two words is no answer rather than a guess, and the rule
underneath is better than a coin toss.

**Abroad** only answers what survives both — a question, which is rare, and the
one place quality is the whole point.

And when the rule reads it wrong, the acknowledgement carries **one press** that
hands the words to Buddy properly.

On a normal day of putting thoughts down, that is no hosted calls at all rather
than one per note.

Two mutations survived a restore during this work and one of them — the line
wiring the house into the screen — left the suite green with every capture
going abroad again. The wiring is a value a test can inspect now, which is what
`Push` cost three releases to learn.

### v0.35.0 — 25 August 2026

**Typing into the box is talking to Buddy.** It was a capture slot that said
"Kept.", which is what a filing cabinet says — and that is why the thread still
did not feel like one.

He reads what you typed, answers it, and says whether it was a thought worth
keeping or a question he has just dealt with. A question does not stay in the
pile.

Ronald asked for this knowing the risk, which was named at the time: a model
between you and the capture promise can be wrong. **The order is where that
risk is managed, and it is the whole design.** The words are spooled and
already a note before Buddy is asked anything. Nothing in the new path can stop
that; what it can do is drop a note afterwards, which is a state the product
already has, which the pile reverses, and which leaves the words in the
database either way.

So every failure lands in the same place. No coach, a spent budget, an
unreachable model, a reply that fails its shape, a wrong judgement — each costs
a note sitting in the pile you did not want there. None costs a thought.

A photograph is kept and never read: it is not words, there is nothing to
answer, and it is the one capture that is hardest to make again.

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
it — the *process* clock, which is a different thing from the person's. So
anything captured after ten in the evening wore the previous day's date on the
corner of its card.

*(Corrected 27 August 2026: this said "the pods run in UTC on purpose since
#148", which is backwards. #148 is what gave the pods a timezone — `TZ` and
`DIGEST_TZ` are both Europe/Amsterdam. The clock that is UTC is the database
session's, which is what v0.43.0's weekday defect turned out to be about.)*

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
