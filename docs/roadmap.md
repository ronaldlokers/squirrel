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
| **Devices** | Phone primary and better; desktop first-class. **Spec drafted 22 August**: `docs/superpowers/specs/2026-08-22-devices-design.md`. Not approvable yet — it turns on one undecided question, which is what gives so the lid can hold a fourth control. |

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
