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

**Rules that moved to allow it**, all recorded in `PRODUCT.md`:

- *Squirrel never invents a time you can be late for. It may hold one the world
  did.* — the carve-out that made fixed points buildable.
- Home may carry one **actionable** offer. This overrides one of the seven
  prohibitions written when the old preview was cut; the other six hold.
- One count of what **happened** is allowed (*three notes cleared*). The banned
  counter counts what remains.
- A sixth principle: **Squirrel chooses, and can say why.**

---

## Next: the coach

Chosen as the next body of work over the personalisation half, on the grounds
that durations and learned timing are data-starved until the product has been
used for a month.

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
| **D** | Read tools (five of six), model-decides among what the picker found, offer cache, picker fallback. | shipped |
| **E** | `breakDownTask()`; "too big" routed through the coach, one step at a time. | shipped |
| **F** | Splitting a brain dump, proposed and confirmed. On the screen; chat is unchanged. | shipped |
| **G** | Write tools and the confirmation policy. Six run, four ask, three are refused. | shipped |
| **H** | `shouldInterrupt()` on rule-produced candidates. *Last on purpose — the only one that speaks without being spoken to.* | |

**Ready:** the API key is stored (SOPS + vault, project-scoped, spend limit set),
and the model IDs are verified against the live API: `gpt-5.6-luna` for routine
work, `gpt-5.6-terra` for escalation.

---

## Decided, not yet built

### Product

| | Decision |
| --- | --- |
| **Three new states** | *waiting on someone* · *blocked on a thing* · *someday*, named separately rather than one "parked" state. Each exists because the picker keeps offering things you cannot act on. |
| **Mood series** | Readable, and shown back on request. Reverses the rule that made it unreadable by construction. |
| **Resurfacing** | A kept note may come back — **only** riding along with something already being raised. Never its own stream. |
| **Attachments** | Through the PWA, camera-first on mobile. *Storage undecided — see open questions.* |
| **Devices** | Phone primary and better; desktop first-class. |

### Structural

**PWA primary, Campfire secondary.** This inverts three things and none of them
is free:

1. **Capture durability is backwards.** Campfire capture is protected by an
   fsynced spool before anything acknowledges it. The PWA's `/capture` writes
   straight to Postgres with no spool — accepted when it was secondary, wrong
   now it is the front door. The service worker already holds captures when the
   network is gone; the gap is a live network and a dead database.
2. **Push becomes the primary channel**, not an improvement on Campfire.
3. **Feature parity relaxes** to best-effort in one direction. Chat keeps
   capture and answering a nudge; it does not need every screen feature.

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
micro-steps for two or three chores · novelty in **art and phrasing only**.

---

## Open

1. **Attachment storage.** No object storage exists in the cluster. Either a PVC
   on the pod — simplest, but the restore drill has to cover it — or add MinIO
   or Garage. Needed before attachments.
2. **`create_moment` proposal lapse.** How long an unanswered proposed fixed
   point stays pending. Defaulting to an hour, matching the breadcrumb, unless
   decided otherwise. Needed at coach phase G.
3. **Records to amend before coach phase C.** `DESIGN.md` gains the modal
   exception and the coach sheet; `PRODUCT.md` gains the coach surface and the
   floor principle; `docs/pile-screen.md` gains `/coach`.
4. **Two homelab commits unpushed** — the coach key and its rotation.
5. **The vault note** for `squirrel openai key` still carries the *known
   exposed* paragraph from the revoked key. App-only edit; `pass-cli` reaches
   named fields but not the note body.

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
