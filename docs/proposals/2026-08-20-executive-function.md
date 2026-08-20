# Squirrel as an external executive-function system

A proposal, written after reading the whole codebase. Nothing has been
implemented. Section 15 lists what I want decided before anything is.

The brief asked four questions — *What should I do? How do I start? Am I still
on track? What should I do next?* — and asked me to be opinionated and to
challenge the brief where the built product already answers better. I have done
both. Roughly a third of the ideas in the brief are refused below, and the
reasons are given rather than implied.

---

## 1. Current app map

Squirrel is **not** a mobile app with screens and navigation. It is a single Go
binary with two front doors onto one Postgres database:

- a **Campfire chat bot** (webhook in, messages with buttons out), and
- a **web screen** at `squirrel.ronaldlokers.nl`, LAN-only behind Traefik
  ipAllowList and Authentik forward-auth, installable as a PWA.

There is one user. There is no AI, no LLM, no calendar, no push notification, no
scheduling beyond a once-a-minute tick, and no client-side framework. Two
dependencies: `pgx` and `testify`.

### Packages

| Package | Role |
| --- | --- |
| `cmd/squirrel` | main; signal handling |
| `internal/boot` | the only package importing both core and transports; wiring, retry-until-Postgres, shutdown |
| `internal/squirrel` | the core: intent matching, state, store, scheduler, message rendering |
| `internal/transport` | Campfire HTTP client and webhook |
| `internal/web` | the screen; a transport, not a core (import direction enforces it) |

### Data model (15 migrations)

- `people`, `identities` — one owner, transport identities reconciled at boot.
- `items` — every inbound message, stored verbatim. Columns that matter:
  `raw_text`, `payload`, `received_at`, `state` (`open|done|dropped|kept`),
  `state_at`, `kind` (`note|task`).
- `chores` — `name`, `interval_seconds`, `tolerance_seconds`, `active`,
  `snoozed_until`, `ask_days` (weekday bitmask), `ask_part`
  (`morning|afternoon|evening`).
- `events` — completions, with `source` (`ack|tap|chat|screen`) and
  `retracted_at`. The chore clock is `max(events.occurred_at)`, derived, never
  stored.
- `prompts` + `prompt_lines` — every numbered surface Squirrel has printed, its
  Campfire message id, and what each line pointed at (`chore_id` XOR `item_id`).
  This is how `done 2` resolves and how old buttons get disabled.
- `checkins` — mood history, append-only, **deliberately unreadable as a
  series** (`LatestCheckin` returns one row; no other read function exists).
- `timers` — one row per person, deleted when it fires. No history by design.

### Capture path (the spine)

Webhook → `Sink` (allow-list check) → **spool file, fsynced** → HTTP 200 with a
🐿️ → `Drain` (once a second, backoff on failure) → Postgres → `Applier.Apply`.
The 👀 reaction means "on disk", ✅ means "reached Postgres". Both survive; they
are two facts, not one status.

Consequence worth naming: **the screen's capture box has no spool behind it.**
`POST /capture` writes straight to Postgres. If the database is down the words
come back to the page and the screen says so loudly. That is documented as a
deliberate, owner-overruled cost.

### What the chat understands

Everything is a note unless it is unambiguously not. `Match` requires the
*entire* trimmed message to be a command form. `.` prefix forces capture.
Commands are `!`-prefixed. Vocabulary: `!notes !find !chores !chore !task
!untask !tasks !did !retire !snooze !undo !fix !timer !start !stop !mood
!help`, plus bare `done|keep|drop <n>`, bare `done`, `?`, `nvm`, and
`every <interval>: <name>` definitions (a count or a colon is required, so
prose does not become a chore).

### What the screen does

`/` home (three doors, capture slot, mood check-in) · `/pile` one card at a
time with done/keep/drop/task/make-a-chore/fix/skip · `/kept` the shelf ·
`/tasks` and `/tasks/done` · `/chores` with did-it, interval chips, stop-asking,
and 5/10/20-minute timer buttons · `/timer` · search across everything from the
lid, notes and chores together. Progressive enhancement throughout: every action
is a form or link that works with JavaScript off. Service worker caches assets
and holds a capture offline; it does **not** cache the pile.

### Scheduling and interruption

One goroutine, one tick a minute. Three things fire from it:

1. **The evening message** at 19:00 Europe/Amsterdam: what chores you completed
   today, what you captured since the last delivered evening message, and — only
   if nothing else claimed it — today's nudge riding along.
2. **The nudge**: exactly one chore a day, chosen by weighted random over how
   overdue (not most-overdue: that is the one nudging has already failed to
   shift), filtered by the chore's asking window, budgeted by a unique index on
   `(person, kind, date)`. Triggered by a message you send, by the presence
   webhook (Home Assistant, 2-minute delay, 10-minute debounce), or as the
   evening fallback.
3. **The timer** finishing: one line, `That's the kitchen. Stop wherever you
   are.` It asks nothing.

### Rules the code actually enforces (not just documents)

- **Never a count.** `OpenItems` returns `([]Item, bool)` — the caller is handed
  a boolean, so it *cannot* render a total. Enforced by signature.
- **Never a mood series.** Enforced by the absence of a store function.
- **Everything reverses.** Completions are retracted, not deleted. Undo is an
  ordinary transition.
- **Two views, one pile.** Both surfaces call the same store functions
  (`PromoteItem`, `SetItemState`, `RecordCompletion`); wording lives in
  `chorewords.go` below both.
- **Nothing loses a thought.** `Applier.Apply` and `Scheduler.Once` both recover
  panics, because a panic in a derived view once took the whole process down and
  the spool file was already gone.

---

## 2. What already works well — keep, do not touch

- **The spool.** Durability before intelligence. This is the correct spine and
  everything else should stay downstream of it.
- **Default-to-capture matching.** The bias is right and the escape hatches
  (`.`, `!`) are the minimum viable command language.
- **One chore a day, weighted random.** This is a genuinely good piece of
  behavioural design and better than what most ADHD apps do. Do not replace it
  with a list.
- **`kept` as a state.** The insight that reference notes are not tasks and will
  never be done is the thing that stops the pile rebuilding itself.
- **Undo everywhere, and the 1150 ms hold so undo has somewhere to live.**
- **Reversible snooze with an untouched baseline.** "Not today" that does not
  pretend the thing was done.
- **`SinceWords` / no numbers on elapsed time.** Correct, and correctly placed
  below both transports.
- **Nothing accrues.** No streaks, no counts, no percentages, anywhere. This is
  the product's actual moat and most of section 6 exists to defend it.
- **The design system.** It is unusually complete and it is doing real work
  (habituation defences, contrast measured not judged, keycaps dropped on touch).
  Reuse it; do not redesign.

---

## 3. ADHD / product problems in the current UX

Ranked by how much they cost.

### 3.1 Nothing ever answers "what should I do now"

This is the big one. Every surface is organised by **what kind of thing it is**,
not by **whether it deserves your attention**. Home is a chooser between three
containers. The pile is arrival order. Tasks is arrival order. Chores is
alphabetical. The only thing in the entire product that ever *chooses* is the
nudge — and it only ever chooses a chore, once a day, and only speaks in
Campfire.

So the person who opens Squirrel while overwhelmed gets: *which of these three
boxes would you like to open?* That is three decisions before any work, charged
to the resource the product's own accessibility section names as scarce.

### 3.2 Mood is collected and then does nothing

`checkins` is written from three places and read from exactly one: the home
screen, to echo one face back at you. It influences no nudge, no ordering, no
suppression. The migration comment says *"a nudge that knows you have been flat
for a while can be gentler, and that is the whole point of asking"* — that nudge
does not exist. Right now the check-in is a small tax with a decorative payoff,
which over time reads as pointless and stops being answered.

### 3.3 A decided task is a task that is never mentioned again

Chores get pushed. Tasks are pull-only. You promote a note to a task — the
deliberate act of deciding — and the system's response is to move it to a screen
you must remember to open. The *most* committed items get the *least* support.
This is backwards.

### 3.4 Home is denser than any other screen

Count the interactive regions on `/`: three doors, a textarea, a submit, five
mood faces. Nine targets, on the screen whose job is to reduce decisions. Every
other screen in the product is more focused than the front door.

### 3.5 Time does not exist in this product

No durations, no appointments, no "before you leave", no notion that 14:30
implies 14:05. The only time-shaped object is a countdown timer you start
manually. For a product built against ADHD, time blindness being entirely
unmodelled is the largest domain gap.

### 3.6 The evening message under-reports what you did

`Today` lists **completed chores only**. Tasks you finished, notes you triaged,
and focus time are all invisible. The one surface positioned to fix *"I did
nothing today"* currently answers with the narrowest possible slice of the day —
and on a day where you did four tasks and no chores, it says nothing at all.

### 3.7 The chat's numbering is stateful in a way that costs working memory

`done 2` resolves against the most recent delivered numbered prompt. It is
implemented carefully and the edge cases are handled — but the mental model
("which list am I numbering against right now?") is exactly the kind of state
ADHD working memory drops. It works because the screen exists as an escape. Do
not extend the numbered-position language any further.

### 3.8 There is no re-entry path after an interruption

The timer knows what you were doing. When it ends, the row is deleted and the
label is gone. Come back twenty minutes later and nothing in the system knows
what you were in the middle of. For a population defined partly by interruption
recovery cost, this is a cheap fix left on the floor.

### 3.9 After "done", a cliff

Completing anything produces an acknowledgement and then nothing. The moment
immediately after finishing is the cheapest moment to start the next thing —
momentum is already spent, the decision cost is at its lowest — and the product
walks away from it entirely.

---

## 4. Missing features (against the brief)

Against the brief's six pillars, honestly graded.

**🧠 BRAIN — external memory.** Genuinely strong. Fast capture from two
surfaces, verbatim, no metadata required, offline-held on the phone, searchable
across every state, correctable. Missing: nothing important. Voice is
unnecessary (the phone keyboard dictates into Campfire already). Multi-thought
splitting ("call the garage and also buy milk") is a nice-to-have.

**🎯 NOW — reduce decisions.** Largely absent. Now/Next/Later does not exist;
the closest thing is `open` vs `kept` and the nudge's one-a-day discipline.

**🚀 START — task paralysis.** Partially built: the timer is a real body double
and the 5/10/20 buttons on a chore are exactly the right shape. Missing: any
"I can't start" affordance, any decomposition, any two-minute start, any
reduction of an aversive thing into a smaller thing.

**⏱ TIME — externalise time.** Almost entirely absent, as above.

**🔄 LIFE — life maintenance.** The strongest pillar. Chores as rhythms rather
than deadlines, tolerance windows, asking windows, snooze, retire, no guilt debt
anywhere. Better than the brief's own suggestion — the brief proposes "3/4
completed, good enough", which is still a fraction; the product refuses to
render a fraction at all, which is stronger. Missing: appointments, and anything
that is fixed in the world rather than rhythmic.

**🐿️ COACH — executive-function assistant.** Does not exist in any form.

Plus, from the brief's later sections: reflection-without-shame is half-built
(§3.6); body doubling exists but does not follow along; mood-as-input is absent;
controlled novelty exists as *randomised card rotation, varied reactions and
varied nudge phrasing* and should go no further than that.

---

## 5. Blind spots — missing from the current app *and* from the brief

These are the ones I would fight for.

**5.1 Re-entry after interruption.** The brief mentions "recovery after getting
distracted" in a list and then never designs for it. The cost is not the
distraction; it is the reconstruction. A single persisted breadcrumb — *you were
on the kitchen, twelve minutes ago* — is nearly free and returns more than any
prioritisation feature.

**5.2 Fixed points are not deadlines, and the product's rule is over-broad.**
Squirrel bans deadlines because an invented due date accrues lateness. Correct.
But a dentist appointment at 14:30 exists whether or not the app knows about it,
and refusing to model it does not remove the lateness — it just means the app
cannot help you leave. The rule should be sharpened: *Squirrel never invents a
time you can be late for; it may hold one the world imposed.* Without this
carve-out the entire ⏱ TIME pillar is unbuildable.

**5.3 The end of a thing is the cheapest moment to start the next.** Neither the
current product nor the brief treats completion as a trigger. It is the single
highest-yield trigger available, and it costs one line of UI.

**5.4 Stopping is unsupported in the other direction.** Hyperfocus has no exit
ramp. The timer caps at 180 minutes and then says nothing. I am *not* proposing
a supervisor — but "an hour and a half on the kitchen, still going?" offered
only when you asked for it at the start is a different thing from surveillance.

**5.5 Transitions need loading, not reminding.** Leaving the house fails at
"where are my keys", not at "did I know I had to leave". Anything built for
14:30 must be able to hold *what to take*, or it is a louder clock.

**5.6 The notification channel is too weak for anything time-critical.**
Everything Squirrel says goes to a Campfire room you have to be looking at. For
"leave in ten minutes" that is not good enough. The PWA is already installed;
Web Push is the missing 150 lines.

**5.7 Estimation should be learned and never shown.** The brief asks for
"comparing estimated vs actual duration". Showing that comparison to an ADHD
user is a machine for generating self-criticism — it is a scoreboard for the
planning fallacy. Learn it, use it to answer "does this fit before 14:30", and
never render it.

**5.8 Energy is not a sixth face.** The brief suggests adding an energy axis
beside mood. That is a second decision at the moment the person is least able to
make one. The existing five already carry capacity: *frazzled* and *wiped* are
capacity words. Reinterpret, do not extend.

**5.9 Nothing knows what is already impossible today.** If it is 21:40, a chore
that only asks in the morning is not a candidate, and neither is anything that
takes an hour. Deterministic, cheap, and it removes items from consideration
without ever showing them — the single best kind of complexity-hiding.

---

## 6. Features we should NOT build

Stated flatly, with reasons, because several are in the brief.

- **XP, points, side quests, "+20 XP".** Direct violation of *nothing accrues
  that can be destroyed*. Points are a counter; a counter someone can lose is
  the exact loss-aversion mechanism PRODUCT.md is built against. The brief's own
  example — `SIDE QUEST · take out trash · +20 XP` — is a streak with different
  art. Refuse. Novelty stays in phrasing, art and randomised rotation.
- **"Beat the timer" challenges.** Same objection plus a failure condition.
- **Mood trends, weekly summaries, capacity graphs.** Already banned; the ban is
  right.
- **Projects, tags, folders, sub-tasks.** Already refused in PRODUCT.md and the
  tasks spec. Agreed: they are administration, and administration is what the
  product exists to avoid.
- **Voice capture.** Campfire on a phone already accepts keyboard dictation. The
  gain is a few seconds; the cost is an audio pipeline, storage, transcription
  and a new privacy surface.
- **Two-way calendar sync.** Reading a calendar might eventually earn its place
  (P3). Writing to one makes Squirrel a scheduling client and doubles the
  authority over your day.
- **A general AI chat companion.** "Talk to the squirrel about your feelings" is
  a different product, has no failure story, and puts an unbounded surface in
  front of a system whose value is that it is predictable.
- **AI-generated 12-step plans.** The brief names this as the failure mode; I
  agree, and the way to avoid it is to make the AI's *output shape* structurally
  incapable of being a plan (see §11: one step, ≤12 words).
- **Location context beyond the existing presence webhook.** Single user, one
  home, and the arrival ping already covers the only transition that matters.
- **Showing estimate-vs-actual accuracy.** See §5.7.
- **"Are you still on track?" mid-timer pings, by default.** A body double that
  interrupts is a supervisor. Opt-in per timer, or not at all.
- **A second capture surface.** Two exist and that was already one compromise.
  A third is not on offer.

---

## 7. Prioritised roadmap

The organising principle: **P0 is everything needed for Squirrel to answer
"what now" without an LLM and without new infrastructure.** Every P0 item is
deterministic, testable, and degrades to today's behaviour if it fails.

### P0 — core product identity

Without these, Squirrel is a very well-built external memory that still makes
you decide.

| # | Feature | Why P0 |
| --- | --- | --- |
| 0.1 | **The one thing** — home offers exactly one chosen item with an action on it | This *is* the product identity change. Everything else is support. |
| 0.2 | **Mood becomes an input** — a fresh low/wiped reading shrinks what is offered and suppresses the nudge | Closes §3.2. Cheapest large win in the codebase: the data is already there. |
| 0.3 | **Tasks become surfaceable** — the picker may name a task, not only a chore | Closes §3.3. Requires no new tables. |
| 0.4 | **"I can't start"** — a deterministic ladder attached to whatever is offered | Answers the second of the four questions. No AI. |
| 0.5 | **Completion offers one next thing, once** | Closes §3.9 for one line of UI. |

### P1 — high-value ADHD support

| # | Feature | Why P1 not P0 |
| --- | --- | --- |
| 1.1 | **Fixed points** (`at 14:30 dentist`) and the leave-by chain | Needs the rule carve-out in §5.2 agreed first, and a new table. |
| 1.2 | **Web Push** | 1.1 is not trustworthy over Campfire alone. |
| 1.3 | **Coarse duration buckets** on tasks (`a couple of minutes` / `a bit` / `a while`), volunteered only | Only useful once 1.1 gives it something to fit inside. |
| 1.4 | **Re-entry breadcrumb** — the last timer survives 30 minutes past its end | Cheap; sequenced here only because it is small. |
| 1.5 | **Evening message widened** to tasks done, notes triaged, focus time | Fixes §3.6. Pure rendering plus two queries. |
| 1.6 | **Capacity-aware nudge phrasing** | Follows 0.2 naturally. |

### P2 — intelligence / personalisation

| # | Feature |
| --- | --- |
| 2.1 | Learned durations from timer runs; used only to answer "does it fit", never displayed |
| 2.2 | Learned good moments — nudge timing from when completions actually happen |
| 2.3 | **AI: first tiny step** for "I can't start", behind the deterministic ladder |
| 2.4 | **AI: brain-dump splitter** — one rambling message becomes several notes, confirmable, never automatic |
| 2.5 | Kept-shelf contextual resurfacing (a kept note shown *only* when a related thing is raised) |

### P3 — experiments

| # | Feature |
| --- | --- |
| 3.1 | Body-double follow-along: two or three chores get named micro-steps |
| 3.2 | Read-only calendar (CalDAV) as a source of fixed points |
| 3.3 | Hyperfocus exit ramp, opt-in at timer start |
| 3.4 | Novelty in art and phrasing only — a different mascot pose on the empty pile, seasonal cards |

### Reasoning behind the ordering

P0 is chosen so that **no new dependency, no LLM and no new notification
channel** is required to make the product answer its own headline question. That
matters because everything in P1+ is a bet, and P0 is not: the picker is fifty
lines of Go over data that already exists, and if it turns out to be wrong it is
deleted without a migration.

Mood-as-input is P0 rather than P2 because the data has been accumulating since
migration 0012 and is currently pure cost.

Fixed points are P1 rather than P0 because they need a product-rule decision
from the owner, not because they are less valuable. If the answer in §15 is yes,
1.1 and 1.2 are the largest single improvement in the whole document.

AI is P2 in its entirety and *never* on the deterministic spine. See §11.

---

## 8. Proposed daily experience

**Morning, phone, still in bed.** You open Squirrel. It asks how you are —
because the last reading is from yesterday and yesterday is not now. You press
*wiped*. The page changes to one line: *`bins out — it is bin day`*, with
*not now* beside it. Nothing else on the screen is a task. The doors are still
there, below, and neither one is calling.

**Nine o'clock, desk.** You type into the slot: *`need to call the garage about
that weird noise sometime this week`*. It says `kept`. You are done; no
category, no date, no priority.

**Eleven.** Squirrel, in Campfire: *`While you're here — descale the kettle,
last done a while back, every month.`* Two buttons: the chore's name, and *not
today*. You press *not today*. Nothing is late; the clock did not move.

**One o'clock.** You open the screen. It says: *`ring the vet`* — a task you
decided on Tuesday and have avoided since. Underneath, one small link: *`I can't
start`*. You press it. It asks nothing about why by default; it offers the
smallest version first: *`just find the number.`* Then: *`10 minutes on it?`*
You press 10. The lid carries a countdown on every screen from then on.

**Ten past one.** *`That's ringing the vet. Stop wherever you are.`* Underneath,
one line: *`want the next small thing?`* You ignore it. Nothing happens.

**Half past two, you get up and wander off.** Nothing chases you.

**Quarter to three, back at the desk.** The screen's first line: *`you were on
ringing the vet, half an hour ago.`* One press picks it back up; one press says
it is done; ignoring it makes it disappear at the hour.

**Seven in the evening.** Campfire: what you handled today — the vet, the two
notes you cleared, twelve minutes of focus — then what you told it since
yesterday. No score, no percentage, no comparison with any other day.

**A day where you press *wiped* and mean it.** The one thing becomes the
smallest fixed thing or nothing at all. The nudge does not fire. The evening
message still says what you handled, because on those days it is usually more
than you think.

---

## 9. Home screen design

### The constraint I am working inside

The design system removed a home-screen "peek" on sight, with seven
prohibitions: no count, no stack behind it, no *more*, no action, no state
colour, no urgency copy, no residue when the pile is empty.

**I am proposing something that violates one of those seven, deliberately, and
keeps the other six.** The Peek showed *the newest untriaged note* — a slice of
backlog, unactionable, chosen by arrival order. What I want is the opposite:
**one thing the product chose, with exactly one action on it, and a one-press
refusal.** The prohibition it breaks is "no action", and it breaks it because an
unactionable offer is precisely what made the Peek a greeting from your backlog.

The other six hold, plus two new ones:

1. Never more than one. Not a list, not "and 3 others".
2. Never a count, a stack, or a *more*.
3. No state colour and no urgency copy. It is never late, never red, never bold.
4. No residue: when there is nothing to offer, the region is *absent*, not empty
   — home renders exactly as it does today.
5. Refusing costs one press and has no consequence and no follow-up question.
6. It never changes what is *true*, only what is *offered*.
7. **It is chosen deterministically and can always be explained in one clause**
   (*it is bin day*, *you decided this on Tuesday*, *you were doing it*).

### Layout

Desktop and phone, current design language, no new components except the offer
card (which is the door's stock and the chore card's controls).

```
┌────────────────────────────────────────────────────────────┐
│  🐿  Squirrel                              [ 🔍 search   ]  │   the lid, unchanged
└──────────────────────────────────────────╲________________╱─┘   (brim)

        ┌──────────────────────────────────────────────┐
        │  RIGHT NOW                                   │      meta role, cream stock
        │                                              │
        │  ring the vet                                │      note role, CASL 1
        │  you decided this on tuesday                 │      voice role, one clause
        │                                              │
        │  [ DID IT ]  [ 10 MIN ]   i can't start      │      lifted-green, paper, quiet link
        │                              not now  ✕      │
        └──────────────────────────────────────────────┘

  ┌────────────┐   ┌────────────┐   ┌────────────┐
  │   [art]    │   │   [art]    │   │   [art]    │             the three doors, unchanged
  │  the pile  │   │ the tasks  │   │ the chores │
  │ WHAT YOU   │   │ WHAT YOU   │   │ WHAT COMES │
  │   SAID     │   │  DECIDED   │   │    BACK    │
  └────────────┘   └────────────┘   └────────────┘

  ┌──────────────────────────────────────────────┐
  │  tell me a thing                             │             the slot, unchanged
  └──────────────────────────────────────────────┘  [ Keep it ]
```

**When the mood reading is stale**, the offer region is replaced by the
check-in — so home still has exactly one interactive thing above the doors:

```
        ┌──────────────────────────────────────────────┐
        │  how do you feel?                            │
        │   😄    🙂    😔    😵    😴                  │      the five drawn faces
        │  good  calm  low  frazzled wiped             │
        └──────────────────────────────────────────────┘
```

Answering re-renders home with the offer in that slot, shaped by the answer.
This is why mood moves *above* the doors: it is not a diary entry, it is the
question that configures the page.

**When there is nothing to offer** — nothing due, nothing decided, low capacity
with no fixed point — the region is absent and home is exactly today's screen.
That is the *stopping partway is a normal ending* rule made structural.

### What home still refuses to do

- It does not show the pile, or anything about the pile's size.
- It does not show what you did today. That is the evening message's job, and
  putting it here makes home a scoreboard.
- It does not show more than one thing, ever, under any capacity.

---

## 10. Key interaction designs

Each in the ten-part shape requested.

---

### 10.1 The one thing (P0 · 0.1, 0.2, 0.3)

**1. User problem.** "I know things need doing, I cannot pick one, so I pick
none." Every existing surface hands back a container instead of an answer.

**2. Trigger.** Opening `/`. Also the `!now` command and a bare `?` in chat
(which currently lists chores — this generalises it).

**3. UX flow.** Home renders → if the mood reading is stale, ask → otherwise
choose one item → render offer with two or three controls → user presses
*did it* / a timer / *I can't start* / *not now* → the page re-renders. *Not
now* suppresses that item for the rest of the day and offers nothing in its
place until the next reload. No "why not?" question, ever.

**4. UI changes.** One new template partial (`offer.html`), one new region in
`home.html`, existing button and card components. Mood block moves above the
doors. No new colours; *did it* reuses the chores screen's lifted green.

**5. Data required.** All existing: `chores` + `events` (via `DueChores`),
`items` where `kind='task' and state='open'`, `checkins` (latest, fresh),
`timers`. One new small table `offers(person_id, kind, ref_id, offered_at,
answer, answered_at)` so that *not now* can be honoured for a day and the same
thing is not offered twice in an hour. It is invisible and unreadable as a
series, exactly like `checkins`.

**6. AI involvement.** **None.** The picker is deterministic and must stay so —
it is the one thing the person must be able to trust and predict, and it must
work when everything else is down.

The rule, in order:

```
1.  a fixed point within its leave-by window        (P1; skipped until then)
2.  the timer's own subject, if one is running      ("you are on this")
3.  the re-entry breadcrumb, if within 60 min       (P1)
4.  a chore that is due AND inside its asking window
5.  the oldest open task that has not been offered today
6.  nothing
```

Capacity gate, applied first: if the freshest reading is `wiped` or `frazzled`
and less than 6 hours old, drop rules 4 and 5 entirely and keep only 1–3. A low
day offers what the world imposes and what you were already doing, and nothing
else.

**7. Notification involvement.** None. This is a pull surface. The nudge remains
the only push, and stays budgeted at one a day.

**8. Edge cases.** Nothing due and nothing decided → region absent. Everything
refused today → region absent. Database unreachable → home already renders
without it (it reads nothing today); the offer must fail *silently* and leave
home intact. Timer running for something that was since completed → fall through
to rule 4.

**9. Failure modes.** The one I fear: the offer becomes stale furniture you look
past. Defences — the explanation clause varies, the offer is absent whenever
there is genuinely nothing (so its presence stays meaningful), and *not now* is
honoured for a full day so the same item cannot follow you around.

Second: it offers something obviously wrong and burns trust. Defence — the
explanation clause is mandatory, so a wrong offer is legible rather than
arbitrary, and rule order is fixed rather than scored.

**10. Preventing it becoming overwhelming.** Hard cap of one. No badge, no
count, no list view, no "see all" — the doors are already that. And it never
appears twice on one page.

---

### 10.2 "I can't start" (P0 · 0.4)

**1. User problem.** Knowing what to do is not being able to do it. Currently
the product's only answer is a timer, which helps with boredom and not with
size, dread or not-knowing-how.

**2. Trigger.** A quiet link on the offer card and on a task card. Never a
button competing with *did it* — the shape matters: refusing to start should be
easier to reach than starting is, but visually smaller, or it becomes the
default.

**3. UX flow.** Press → the card's action row is *replaced in place* (never a
modal — the design system forbids one here) by four answers:

```
┌──────────────────────────────────────────────┐
│  clean the kitchen                           │
│  ─────────────────────────────────────────── │
│  what is in the way?                         │
│                                              │
│  [ TOO BIG ]  [ DON'T KNOW HOW ]             │
│  [ BORING ]   [ NOT TODAY ]                  │
└──────────────────────────────────────────────┘
```

Each answer produces one line and at most two controls:

| Answer | Deterministic response |
| --- | --- |
| too big | *forget the rest. just do the smallest visible piece.* + `[ 5 MIN ]` |
| don't know how | *what is the first thing you'd have to find out?* + a one-line capture that becomes a note |
| boring | `[ 10 MIN ]` + *I'll say when. Stop wherever you are.* |
| not today | same as *not now*: quiet for the day, no follow-up |

I have deliberately dropped the brief's *anxious* and *no energy* options. *No
energy* is what the mood check-in already says, and asking twice is a second
tax. *Anxious* invites a therapeutic response the product should not attempt —
and its useful action ("make it smaller") is already *too big*.

**4. UI changes.** One disclosure in the existing `<details>` grammar used by
the chore interval picker and the reword box. No new components.

**5. Data required.** None new for the deterministic version. If 2.3 (AI first
step) lands, it reads the item text only.

**6. AI involvement.** None at P0. At P2 the *too big* branch may replace its
fixed sentence with a generated first step, with hard constraints: one step,
imperative, ≤12 words, no list, no numbering, 3-second timeout, and the fixed
sentence on any failure. The user must not be able to tell whether it was
generated except by it being better.

**7. Notification involvement.** None.

**8. Edge cases.** Pressing it twice → the disclosure closes. Chore vs task →
identical, except *don't know how* is suppressed for a chore you have done
before (you know how; it is not the obstacle).

**9. Failure modes.** It becomes a fifth way to avoid the thing. Accepted: three
of the four branches end in a timer or a smaller action, and *not today* is an
honest exit that already exists elsewhere. Avoidance with a record is better
than avoidance without one.

**10. Preventing overwhelm.** Four options, one screen, no free text except the
optional one-line capture. Never asks a follow-up question after an answer.

---

### 10.3 Fixed points and leaving on time (P1 · 1.1, 1.2)

**1. User problem.** 14:30 is not a moment, it is a chain — wrap up, get ready,
gather, leave, travel, arrive — and the chain is invisible until it has already
failed.

**2. Trigger.** Capture that parses as a fixed point: `at 14:30 dentist`,
`tomorrow 09:00 school run`. Deliberately requires the `at`/time shape, and
falls back to a plain note when ambiguous — the same "when in doubt, capture"
bias as everything else.

**3. UX flow.**

```
you:        at 14:30 dentist, 20 minutes away
squirrel:   14:30 dentist. I'll say something at 13:55.
            [ what to take ]        ← optional, one press, one line
```

Then, at leave-time minus 10 (Web Push + Campfire):

```
squirrel:   dentist at 14:30. Leave about 14:05.
            keys · wallet             ← only if you said
            [ leaving ]  [ 5 more minutes ]
```

**4. UI changes.** A fourth region on home *only within the window* (rule 1 of
the picker). No fourth door. Fixed points do not get a browsing screen — a list
of your appointments is a calendar, and Squirrel is not one.

**5. Data required.** New table:

```sql
create table moments (
  id          bigserial primary key,
  person_id   bigint not null references people(id),
  item_id     bigint references items(id),   -- the note it came from
  label       text not null,
  starts_at   timestamptz not null,
  travel_secs int,                            -- null = unknown, defaults to 15 min
  ready_secs  int,                            -- null = default 10 min
  bring       text,                           -- one line, optional
  said_at     timestamptz,                    -- the leave-by was announced
  done_at     timestamptz
);
```

**6. AI involvement.** None. Parsing a time is a regex; guessing at one is a
missed appointment.

**7. Notification involvement.** This is the feature that requires Web Push.
Exactly two pushes per moment, maximum: the leave-by, and one at start time if
*leaving* was never pressed. Never more.

**8. Edge cases.** Past times → tomorrow. No travel time → 15 minutes assumed
and *said as an assumption* ("leave about 14:05 — I guessed fifteen minutes"),
because a silent assumption is how you end up late trusting the machine.
Multiple moments in a window → nearest only. Push denied → Campfire only, and
the screen says so once.

**9. Failure modes.** The worst outcome is a missed push that was trusted. Both
channels fire; Campfire is the fallback and is never skipped. Second: this
becomes a calendar. Defence — no list screen, no recurrence, no editing beyond
"never mind", and a moment that has passed is gone.

**10. Preventing overwhelm.** No moment is ever shown outside its window. A day
with four appointments shows one thing at a time, all day.

---

### 10.4 Re-entry (P1 · 1.4)

**1. User problem.** You were doing something. You got up. Reconstruction is
where the hour goes.

**2. Trigger.** Opening any screen within 60 minutes of a timer ending or being
stopped.

**3. UX flow.** One line in the picker slot: *`you were on ringing the vet, half
an hour ago.`* with `[ PICK IT UP ]` `[ DID IT ]` and a quiet `✕`. Ignoring it
lets it lapse at the hour.

**4. UI changes.** Reuses the offer card verbatim.

**5. Data required.** `timers` currently deletes on finish. Add `last_timer`
retention: either soft-delete with `ended_at`, or a `last_focus` row. Small.

**6. AI involvement.** None.

**7. Notification involvement.** **None, and this is important.** A push saying
"you got distracted" is the product breaking Principle 5. It waits for you to
come back.

**8. Edge cases.** Stopped deliberately → still shown; you may have stopped to
answer the door. Completed via *did it* → not shown.

**9. Failure modes.** Reads as nagging. Defence: pull-only, one hour, one press
to dismiss, and the words are neutral — *you were on X*, never *you did not
finish X*.

**10. Preventing overwhelm.** One, ever. It is the same slot as the offer, so it
cannot stack with it.

---

### 10.5 Reflection without shame (P1 · 1.5)

**1. User problem.** "I did nothing today" is usually false and always
expensive.

**2. Trigger.** 19:00, the existing evening message. No new schedule.

**3. UX flow.** Current shape, widened:

```
Today
 · bins out
 · rang the vet                    ← tasks, new
 · three notes cleared             ← triage, new, and deliberately vague
 · 22 minutes of focus             ← timers, new

Since yesterday
 · need to call the garage about that weird noise
 · ...
```

*Three notes cleared* is the one place a number appears, and I want to argue
for it rather than sneak it past: it is a count of things **that happened**, in
the past, that cannot go up while you are not looking and cannot be lost. The
banned counter is a count of *what remains*. If that argument is not accepted,
the line becomes *some notes cleared* and nothing else changes — but I think the
stronger reading of the rule allows it, and the sentence is much better with it.

**4. UI changes.** `EveningMessage` only.

**5. Data required.** Two queries over existing tables: items with
`kind='task' and state='done' and state_at >= midnight`; items with
`state <> 'open' and state_at >= midnight` for the triage count. Focus minutes
requires timer retention from 10.4.

**6. AI involvement.** None. Tempting and wrong: a generated summary of your day
is a sentence about the person, which Principle 5 forbids.

**7. Notification involvement.** Existing.

**8. Edge cases.** A day with nothing → the section is absent, as today. Never
"you did nothing", never "0".

**9. Failure modes.** It becomes a scoreboard read as a target. Defence: no
comparison to any other day, ever; no averages; no "best"; no total.

**10. Preventing overwhelm.** Capped at five bullets per section, with the
existing `…and more`.

---

## 11. AI coach architecture

### The position I am taking

**AI does not enter the spine.** Not capture, not the picker, not the nudge
budget, not scheduling, not "does this fit". Those must be predictable,
explainable, offline-capable and correct — and a model is none of the four.

AI earns a place in exactly three narrow spots, all P2, all optional, all with a
deterministic fallback that ships first and stays:

| Where | Input | Output | Fallback |
| --- | --- | --- | --- |
| *too big* branch of "I can't start" | one item's text | one imperative step, ≤12 words | the fixed sentence |
| brain-dump splitting | one captured message | 1–4 candidate notes, confirmed by the user | the message stays one note |
| nudge phrasing variation | chore name + coarse capacity | one sentence | the current phrasings |

That is the whole of it. If a fourth use appears, it needs the same three
properties: bounded output, no authority over state, and a working answer when
the model is absent.

### Why not a context-assembling coach

The brief asks for an AI that holds the whole picture. I think that is the wrong
bet **for this product**, for three reasons:

1. **It is a single-user homelab bot with two dependencies.** An LLM is a
   larger operational surface than everything else combined.
2. **Every conversational answer is unfalsifiable.** When the picker is wrong
   you can read the six rules and see why. When a model is wrong you get a
   plausible sentence and no recourse — and the person you are building for has
   an unusually low tolerance for a tool that is confidently wrong.
3. **The rules the product lives by are hard to hold in a prompt.** Never a
   count, never a sentence about the person, never a fraction, never a deadline.
   A model will violate those occasionally, and occasionally is enough to break
   the one thing that makes Squirrel different.

### The architecture, if and when it is built

If AI does go in, this is the shape — designed so that the *context* problem the
brief rightly worries about cannot occur, because the model is never handed a
database.

```
internal/coach/
  coach.go      Coach interface + NoCoach{} (the fallback; the default)
  ollama.go     one implementation, local, homelab
  views.go      the ONLY things a model may see
  guard.go      output validation: length, shape, forbidden words
```

**Views, not tools.** No function-calling loop, no retrieval, no database
access. The call site assembles a small struct in Go and passes it. Three views
exist, and they are the complete vocabulary:

```go
// What is being worked on right now. Never more than one item.
type Subject struct {
    Text     string     // the item's own words
    Kind     string     // "task" | "chore"
    EverDone bool       // chores only
}

// The coarse shape of now. No history, no series, no identifiers.
type Moment struct {
    PartOfDay string    // "morning" | "afternoon" | "evening"
    Capacity  string    // "ok" | "low"     -- derived, never the raw mood word
    FreeUntil *int      // minutes to the next fixed point, or nil
}

// A message the person just typed, for splitting only.
type Dump struct{ Text string }
```

Note what is *absent* and permanently so: no history, no mood series, no counts,
no other items, no dates, no names, no identifiers. A prompt cannot leak what it
was never given, and a model cannot produce a twelve-step plan about your whole
life when it has only ever seen one sentence.

**Guarding the output.** Every response passes `guard.go` before it reaches a
human: max length, single line, no numbering, no bullet characters, no
second-person judgement words (a small deny-list: "should", "just", "simply",
"failed", "behind"). Anything that fails the guard is discarded silently and the
fallback is used. There is no retry loop — a retry is a second chance to say
something worse, and the fallback is already acceptable.

**Where reasoning lives — the division:**

| Concern | Owner |
| --- | --- |
| what is due, what is next, what is offered | **deterministic** — always |
| what fits in the time available | **deterministic** — arithmetic |
| when to interrupt | **deterministic** — asking windows and the daily budget |
| whether capacity is low | **deterministic** — the freshest fresh check-in |
| how long a thing usually takes | **learned** — median of timers, never shown |
| when this person actually completes chores | **learned** — P2.2, shifts asking windows only |
| the first small step into an aversive task | **AI**, with a fixed fallback |
| turning one rambling message into several notes | **AI**, with confirmation |
| how a sentence is phrased this time | **AI**, cosmetic only |
| which chores exist, how often, when to ask | **user-controlled**, always |

**Privacy.** A local model (Ollama in the existing cluster) is strongly
preferred and is the only option I would ship without a per-feature opt-in.
`Coach` is an interface with `NoCoach{}` as the zero value, so the default build
has no AI at all and every test runs against the fallback path.

---

## 12. Technical architecture changes

Incremental throughout. No rewrites; the existing shape is good.

### Migrations

| # | Change | For |
| --- | --- | --- |
| 0016 | `offers(person_id, kind, ref_id, offered_at, answer, answered_at)` | 10.1 — honouring *not now*, avoiding repeats |
| 0017 | `timers`: `ended_at timestamptz`, stop hard-deleting on claim; retention 60 min | 10.4, focus minutes |
| 0018 | `moments` table (see 10.3) | P1.1 |
| 0019 | `items.minutes smallint null` — coarse bucket, volunteered only | P1.3 |
| 0020 | `push_subscriptions(person_id, endpoint, p256dh, auth, created_at)` | P1.2 |

`offers` must be written with the same discipline as `checkins`: append-only,
and the store exposes no function that returns a series.

### New core files

- `internal/squirrel/pick.go` — `PickNow(ctx, personID, now) (Offer, bool, error)`.
  The six rules, pure where possible, table-driven tests. This is the most
  important new file in the proposal and should be the most heavily tested.
- `internal/squirrel/capacity.go` — `Capacity(checkin, now) Capacity`. Three
  lines of logic, deliberately its own file so the single definition of "low
  day" cannot fork between surfaces.
- `internal/squirrel/moments.go` — parsing, store, leave-by arithmetic (P1).
- `internal/squirrel/push.go` — VAPID, subscription store, send (P1).
- `internal/coach/*` — P2 only.

### Changed files

| File | Change |
| --- | --- |
| `internal/web/home.go` | read the offer; move mood above the doors |
| `internal/web/templates/home.html` | offer region; reorder |
| `internal/web/templates/offer.html` | new partial |
| `internal/web/act.go` | `/now/act` — did-it, not-now, can't-start, timer-from-offer |
| `internal/web/pile.go` | routes |
| `internal/web/static/pile.css` | offer card (reuses door + chore-card rules) |
| `internal/squirrel/apply.go` | `!now`, `!at`, and the `?` generalisation |
| `internal/squirrel/render.go` | `NowMessage`, widened `EveningMessage` |
| `internal/squirrel/schedule.go` | capacity gate on `nudgeFor`; moment ticks |
| `internal/squirrel/nudge.go` | capacity-aware phrasing |
| `internal/squirrel/timer.go` | retention instead of delete-on-claim |
| `internal/web/static/sw.js` | push handler (P1) |
| `PRODUCT.md`, `DESIGN.md` | the rule carve-out and the offer component |

### Feature parity

PRODUCT.md's standing rule — anything you can do in chat you can do on the
screen and the reverse — applies to every item here. `!now` must exist for the
offer, `!at` for moments, and "I can't start" needs a chat form (`!stuck`).
Parity tests belong in the same round as the feature, not after.

### Analytics

There is no analytics and there should not be. The only behavioural data written
is what a feature needs to function: `offers` for suppression, timer durations
for fit arithmetic. Both are invisible and neither is readable as a series.
Anything more is a mood chart with a different name.

### Testing

- `PickNow`: table-driven over the six rules × capacity × time of day. Should be
  exhaustive; this is the function whose wrongness is most visible.
- Capacity gate: a `wiped` reading suppresses rules 4 and 5 and nothing else.
- Never-a-count tests, in the existing shape, on every new surface.
- Parity: each new chat verb and screen action land in one test.
- Moments: DST, midnight crossing, past times, missing travel time.
- Push: subscription round-trip; Campfire fallback when push fails.
- Coach (P2): guard rejects a numbered list, a long answer, and a judgement
  word; `NoCoach{}` is the default in every other test.

---

## 13. Implementation phases

Each phase is one PR, shippable and reversible on its own.

**Phase A — the picker, without touching the UI.** `capacity.go`, `pick.go`,
migration 0016, `!now` in chat. Ships behind a chat command only, so the
algorithm can be lived with for a few days before it takes over home. *This is
the phase most likely to teach us the rules are wrong, and it costs one file to
change them.*

**Phase B — home.** Offer region, mood moved above the doors, `/now/act`.
Deletes nothing; home without an offer is byte-identical to today.

**Phase C — "I can't start".** Deterministic ladder, screen and `!stuck`.

**Phase D — completion offers the next thing.** One line, everywhere a
completion lands.

**Phase E — re-entry and the widened evening message.** Timer retention,
focus minutes, tasks and triage in `Today`.

**Phase F — fixed points.** Parsing, table, leave-by, Campfire only.

**Phase G — Web Push.** Then F becomes trustworthy.

**Phase H — durations.** Buckets, learned medians, fit arithmetic used by
rule 1 of the picker.

**Phase I+ — coach.** Interface, `NoCoach{}`, guard, then Ollama behind it.

Phases A–E are P0 plus two P1 items and need no new infrastructure at all.

---

## 14. Files and components likely affected

**Read heavily, changed little:** `internal/squirrel/store.go`, `notes.go`,
`chores.go`, `prompts.go`, `intent.go`, `internal/boot/boot.go`.

**Changed:** `apply.go`, `render.go`, `schedule.go`, `nudge.go`, `timer.go`,
`checkin.go` (one derived reader), `internal/web/home.go`, `act.go`, `pile.go`,
`render.go`, `web.go` (the `Store` interface grows), all of
`internal/web/templates/`, `static/pile.css`, `static/pile.js`, `static/sw.js`.

**New:** `internal/squirrel/pick.go`, `capacity.go`, `moments.go`, `push.go`,
`internal/web/now.go`, `templates/offer.html`, `internal/coach/*`, migrations
0016–0020.

**Documents that must be amended, not appended to:** `PRODUCT.md` (the fixed
point carve-out; the offer's seven prohibitions), `DESIGN.md` (the offer card;
the Peek entry, which must record what changed and why rather than being
deleted), `docs/pile-screen.md` (route table).

---

## 15. Decisions, risks and what is still open

### Decided by the owner, 20 August 2026

1. **The fixed-point carve-out: yes.** Squirrel may hold a time the world
   imposed. PRODUCT.md's rule is rewritten as *Squirrel never invents a time you
   can be late for* — it may hold one the world did. Nothing is ever marked
   late, no lateness accrues, and Squirrel never generates a due date of its
   own. This unblocks P1.1, P1.2 and rule 1 of the picker.

2. **Home may carry one chosen, actionable offer: yes.** One thing, with its
   actions on it. This deliberately overrides the Peek's "no action"
   prohibition; the other six hold unchanged and are restated in §9, alongside
   the two new ones. DESIGN.md's Peek entry is amended rather than deleted — the
   reasoning is the useful part, and the record must show what changed and why.

3. **AI: three narrow spots, local model only.** A local model in the existing
   cluster. Views not tools, no database access, guarded output, and the
   deterministic version ships first and stays as the fallback. Never on
   capture, the picker, the nudge budget, scheduling or fit arithmetic.
   `NoCoach{}` remains the zero value, so the default build has no AI in it and
   every test runs the fallback path.

4. **Scope: phases A through G, straight through.** All of P0 and all of P1.
   Fixed points and Web Push are in.

### Still open — one product-rule question, answered by default

**`three notes cleared` in the evening message.** A count of what *happened*,
which cannot grow while nobody is looking and cannot be lost. My reading is that
the banned counter is a count of what *remains*, and that this sits outside it.

Phase E ships it as a count and writes that distinction into PRODUCT.md as a
considered boundary rather than a leak. If the reading is rejected, the line
becomes *some notes cleared* — one line in `EveningMessage`, and nothing else
moves.

### Prerequisite the owner must supply

Phase G needs a **VAPID key pair in the Proton Pass Dotfiles vault**, reaching
the pod the same way `PRESENCE_SECRET` does. Nothing in the repository may hold
it. Phases A–F do not need it, so it only blocks at the end of the run.

### Risks

- **The picker is wrong in a way that is boring rather than obvious.** Mitigated
  by Phase A shipping to chat first, and by the explanation clause making every
  choice legible. This risk is *higher* now that A–G is committed in one run:
  Phase A still has to be lived with for a few days before B lands, or the six
  rules get baked into home before anyone has felt them.
- **Home becomes the app and the doors atrophy.** Watch for it; the doors are
  how the pile stays reachable, and an offer that always exists would slowly
  hide them. The "absent when there is nothing" rule is the defence.
- **Feature parity debt.** Three new verbs across two surfaces is the largest
  parity surface the product has taken on at once. Each phase ships both halves
  or it does not ship.
- **Capacity gating could feel like being shut out.** A person who presses
  *wiped* and then wants to work must not hit a wall. The offer region on a low
  day still carries a quiet *show me anyway*, which offers rule 4/5 exactly once
  and does not persist.
- **Moments quietly becoming a calendar.** No list screen, no recurrence, no
  editing. If those get requested, that is the signal to stop and reconsider
  rather than to build them.
- **A–G is seven pull requests, and the value curve is front-loaded.** A–C
  change what the product *is*; D–E are polish on that change; F–G are a second
  product-shaped thing built on top of it. If the run stalls, it should stall
  after E rather than mid-F.

### Questions I could not answer from the code

- How many chores actually exist in production, and how often does the nudge
  currently find nothing due? That number decides whether rule 4 or rule 5 is
  the common case — and therefore what Phase A mostly does.
- Does the presence webhook fire reliably enough to be a picker input?
- Is Campfire's push actually noticed on the phone today? It no longer changes
  the decision to build G, but it changes how loud G has to be.
