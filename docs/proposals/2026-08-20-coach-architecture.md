# The coach: AI architecture for Squirrel

A proposal. Nothing is implemented. Section 15 lists what is still open.

**Decided 20 August 2026.** Twenty-one architectural choices were put to the
owner with both options argued. This
document reflects the decisions, not my preferences — where I argued the other
way and lost, the reasoning is kept in a *what this costs* note so the trade is
visible later rather than forgotten.

| | Decision |
| --- | --- |
| Who decides what-now | **The model**, picker as fallback |
| Context | **Retrieval tools** from day one |
| Writes | **Write tools** with a confirmation policy |
| Durations | **Model estimates**, learning corrects |
| Decomposition | **`breakDownTask()`** returns a sequence, app paces it |
| Interruptions | **The model** decides, on candidates the rules produce |
| Surface | **Both** — chat and screen from the start |
| Splitting | **Propose, then confirm** |
| Offer stability | **Cached, busted on real events** |
| Spend ceiling | **Degrade to the picker**, say so once |
| Coach turns | **Stored, excluded from the pile** |
| Coach surface | **A widget on every screen** — acorn button, sheet, `/coach` underneath |
| On open | **The current offer, already there** — cached, no call |
| Continuity | **Rolling window** — last three exchanges, ~30 minutes |
| "I can't start" | **Opens the coach**; the ladder becomes its fallback |
| Push rights | **Fixed points only** — a buzz always means "you have to be somewhere" |
| Rendering | **Appear at once**, over the first paint. No streaming. |
| Voice | **Reads the room** — lighter at capacity, plainer when low |
| The AI log | **Everything, kept indefinitely** |
| Provider | **OpenAI direct, gateway-shaped internally** |
| A/B | **Config switch only** |
| Repeated dismissal | **Nothing changes.** It is a button, not a nudge. |

### The pattern all three demotions share

Decisions 1, 3 and 14 each replace a deterministic answer with a model. None
deletes the deterministic answer:

| Was the answer | Is now | Still used |
| --- | --- | --- |
| `PickNow()` | the coach decides | fallback: offline, over budget, guard rejects |
| `UnstuckFor()` | the coach answers | fallback, **and the first paint** (below) |
| asking windows alone | `shouldInterrupt()` | the rules still bound it — the model can only be quieter |

**The deterministic version is never the thing that gets deleted. It becomes
the floor.** That is what makes eight AI-forward decisions safe to take in a
product whose whole value is that it works.

---

## 0. What the brief assumed, and what is actually here

The brief was written without access to the codebase and asks to be checked
against it. Six premises do not hold:

| The brief assumes | Squirrel actually |
| --- | --- |
| An existing AI implementation | **None.** Two dependencies: `pgx`, `testify`. Greenfield. |
| Routines exist | **No routines.** Chores are rhythms, tasks decided-once, notes thoughts, moments times the world imposed. |
| Priority, tags, categories | **Refused.** `internal/web/tasks_test.go` asserts *overdue, due, late, deadline, urgent, priority* never appear on the tasks screen. |
| A calendar to read | **Declined 20 August.** Fixed points are typed by hand so only deliberate things interrupt. |
| Durations exist to reason about | **Zero timers have ever run.** |
| Model cost at 100 / 1,000 users | **One user, permanently** (`PRODUCT.md`). |

Two things the brief asks for **already exist and shipped this morning**:

- `PickNow()` — six ordered rules answering "what now" in ~1 ms, offline, with a
  one-clause explanation. Now demoted to fallback by decision 1.
- `UnstuckFor()` — the four-answer ladder, including *"forget the rest of it,
  just do the smallest piece you can see"* plus a five-minute timer.

Neither is deleted. Both become the floor the coach stands on.

---

## 1. Existing architecture worth reusing

**Message transport — done.** `squirrel.Chat` is three function fields (`Send`,
`Update`, `Boost`) assembled by `internal/boot`. A reply is a `Message`: text
plus up to twelve `Action` buttons. Provider-agnostic already.

**Conversation persistence — already complete.** Every inbound message is stored
verbatim in `items`. That is a full log since 2026-08-15. `prompts` +
`prompt_lines` record everything Squirrel has said and what each numbered line
pointed at.

**Intent recognition — deterministic and load-bearing.** `Match()` classifies
every message, and an intent matches only if the *entire* trimmed message is
one, so *"done with the flux migration"* stays a thought. **The coach never sees
a message before `Match()` does.** A model deciding what a message *is* would
lose thoughts, which is the failure the product exists to prevent.

**Fuzzy matching — solved.** `findChore()` resolves *"bins"* → *"bins out"*.
The brief lists this as a nano job; it needs no model.

**State the tools will read:** `items`, `chores`, `events`, `checkins`,
`timers`, `moments`, `offers`.

---

## 2. Problems being solved

1. The offer's clause is generic — `choreBecause()` cannot read the note's words.
2. `UnstuckFor(BlockerBig)` is generic by construction.
3. A brain dump is stored as one note. Right for capture, wrong for triage.
4. **The genuinely missing capability:** the multi-item overwhelm turn. Nothing
   today accepts *"the house is a mess, I need groceries, I need to call the
   garage, I have work to do and I haven't showered"* and answers with one thing.
5. The picker cannot weigh *"you are low and this one is dreaded"* — it only
   knows rule order. This is what decision 1 buys.

---

## 3. Proposed architecture

```
                        every inbound message
                                 │
                    ┌────────────▼─────────────┐
                    │  spool → drain → items   │  unchanged. no AI
                    │  the thought is durable  │  before this line
                    └────────────┬─────────────┘
                                 │
                          Match(raw)             deterministic, capture-biased
                                 │
        ┌────────────────────────┼──────────────────────┐
     command                 fixed point             capture
     run it                  store it                   │
                                                  ┌─────▼──────┐
                                                  │ many things?│ Luna
                                                  │ → propose   │ confirmed
                                                  │   1–4 notes │ before write
                                                  └─────────────┘

                    "what now" — home open, or !now
                                 │
                    ┌────────────▼────────────┐
                    │  OFFER CACHE            │  hit → return, no call
                    │  busted by real events  │  (~70% of opens)
                    └────────────┬────────────┘
                                 │ miss
                    ┌────────────▼────────────┐
                    │  COACH  (Terra)         │
                    │  ┌───────────────────┐  │
                    │  │ read tools        │  │  now() open_work()
                    │  │ 1–3 round trips   │  │  next_moment() history()
                    │  └───────────────────┘  │
                    │  returns structured     │
                    │  Offer + reason         │
                    └────────────┬────────────┘
                                 │ unavailable / capped / malformed
                    ┌────────────▼────────────┐
                    │  PickNow()  — fallback  │  six rules, free, offline
                    └─────────────────────────┘

                              "I can't"
                                 │
                    ┌────────────▼────────────┐
                    │ the sheet opens, and    │  UnstuckFor() renders
                    │ PAINTS IMMEDIATELY      │  instantly, free, offline
                    │ with the fixed line     │
                    └────────────┬────────────┘
                                 │ ~1–2s later
                    ┌────────────▼────────────┐
                    │ coach replaces it with  │  Terra. If it never
                    │ a step naming THIS task │  arrives, nothing changes
                    └────────────┬────────────┘  and you never knew
                                 │
                    ┌────────────▼────────────┐
                    │ breakDownTask()  Terra  │  returns 3–5 steps
                    │ app stores, shows ONE   │  pacing stays deterministic
                    └─────────────────────────┘

                        scheduler tick (every minute)
                                 │
                    ┌────────────▼────────────┐
                    │ RULES: any candidates?  │  due + asking window +
                    │ ~1435 ticks/day: none   │  budget unspent + capacity
                    └────────────┬────────────┘
                                 │ ~5/day: yes
                    ┌────────────▼────────────┐
                    │ shouldInterrupt()  Terra│  go / no-go + wording
                    └─────────────────────────┘
```

**The pre-filter on the interruption path is arithmetic, not preference.**
Calling a model on every tick is 1,440 calls a day — about €130/month. Rules
narrow to candidates; the model makes the final call on those. ~5 calls a day.

### The offer cache

Decision 9. The coach's answer is held and re-used until something actually
changed:

| Invalidated by | Why |
| --- | --- |
| a mood check-in | the input that shapes everything |
| a timer starting or ending | rule 2 and rule 3 change |
| any completion or refusal | the answer may now be wrong |
| a moment entering its leave-by window | the world imposed something |
| 30 minutes elapsed | a floor, so nothing is stale forever |

**Shipped without a single invalidation hook.** The cache is keyed on the
picker's own answer, and `PickNow()` already reflects every row above: a
check-in changes capacity, a timer changes rules 2 and 3, a completion or a
refusal removes the row, a moment entering its leave-by window outranks
everything else. Comparing its answer catches all five at once, for the cost of
one `PickNow` per open — which every open already did. Hooks are the version of
this that gets forgotten when a seventh write path is added.

Two reasons, both already in the codebase: it cuts model calls on the idle
repeated action by roughly two thirds, and `pick.go` already argues an offer
that changes on every reload "reads as the product changing its mind."

---

## 4. Model routing matrix

*GPT-5 mini* → **GPT-5.6 Terra** ($2/$12 per Mtok). *GPT-5 nano* → **GPT-5.6
Luna** ($0.20/$1.20). Both current-family, 1.05M context. Luna beats last
generation's flagship on agentic benchmarks; its documented weakness is
long-context recall, which is why it never runs the tool loop.

| Operation | No AI | Luna | Terra | Reason |
| --- | :---: | :---: | :---: | --- |
| Creating a task from an explicit ask | ✅ | | | One insert. |
| **Deciding what to do now** | fallback | | ✅ | Decision 1. Rules become the floor, and the model only ever chooses *among what they found*. |
| Serving a cached offer | ✅ | | | ~70% of home opens. |
| Producing interrupt *candidates* | ✅ | | | Budget, windows, capacity. 1,435 of 1,440 ticks end here. |
| **Deciding to interrupt** | | | ✅ | Decision 6, on candidates only. |
| Leave-by arithmetic | ✅ | | | `starts − travel − ready`. |
| Which chore does "done" mean | ✅ | | | `findChore()`. |
| Intent classification | ✅ | | | `Match()`. A model here loses thoughts. |
| Extracting a time | ✅ | | | Regex. A guessed time is a missed appointment. |
| Low-capacity day | ✅ | | | `CapacityOf()`. |
| Evening reflection | ✅ | | | A generated summary is a sentence about the person. |
| **Duration estimate (first)** | | ✅ | | Decision 4. Cheap, replaced by measurement. |
| Duration estimate (after ~5 runs) | learned | | | Median of real timers overrides the guess. |
| **Split a brain dump** | | ✅ | | Structured, proposed, confirmed. |
| **Say the offer's clause** | | ✅ | | Rides on the decision call — no extra request. |
| Messy sentence, `Match` missed | | ✅ | | Rare by design. |
| **Make it smaller / breakDownTask** | | | ✅ | Decision 5. One call returns the sequence. |
| **The overwhelm turn** | | | ✅ | The new capability. |
| Notification wording | | ✅ | | Rides on `shouldInterrupt()`. |
| Weekly reflection | | | | **Not built.** Weekly anything is a scoreboard. |
| Pattern detection | | | | Learned in SQL (§9), never generated. |

**Where routing itself would add complexity:** no classifier picks the model.
The call site knows the job at compile time — splitting is always Luna, the
overwhelm turn always Terra.

---

## 5. Context architecture

### Always in the prompt (~180 tokens, cached prefix)

System preamble (§8) plus:

```go
type Now struct {
    Clock     string // "10:42"
    PartOfDay string // "morning"
    Capacity  string // "ok" | "low"  — derived, never the raw mood word
    FreeUntil *int   // minutes to the next fixed point, or nil
}
```

`Capacity` is derived deliberately: the model gets *"low"*, never *"wiped"* and
never a history. A signal, not a diagnosis.

### Retrieved on demand (decision 2)

The model asks; the tool answers with a compact struct. Results are capped so a
tool can never return an unbounded slice of the database — the cap is the
protection the brief wanted from retrieval, and it lives in the tool rather than
in the prompt.

### Long-term: structured columns, never a memory blob

| Learned | Where | Read by |
| --- | --- | --- |
| What a thing actually takes | `timers` → median per label | fit arithmetic; overrides the model's guess |
| What you turn down | `offers` | suppression |
| When you actually complete chores | `events.occurred_at` | asking windows (phase 5) |
| Which unstuck branch you pick, and whether a completion follows | `offers` + `items.state_at` | which intervention works |

Each is inspectable, correctable, deletable. None is a model's opinion about you.

---

## 6. Tool design

### Read tools

| Tool | Returns | Cap |
| --- | --- | --- |
| `now()` | clock, part of day, capacity, minutes free | — |
| `open_work(limit)` | open tasks + due chores inside their asking window | 10 |
| `next_moment()` | the next fixed point and its leave-by | 1 |
| `recent(limit)` | what was completed or refused today | 10 |
| `item(id)` | one item's text, kind, state, arrival | 1 |
| `history(label)` | median duration from real timers, or absent | 1 |

Six, not twelve. Every one is capped. There is no tool that returns the pile,
and none that returns mood history — `now()` gives the derived capacity and
nothing behind it.

**Five shipped, not six.** `history(label)` is the missing one, and it needs a
median over finished timers — which needs a history of finished timers, which
migration 0017 refuses on stated grounds: it becomes a record of what you
started and abandoned, which is a report card. There is probably a version that
is not one (completed runs only, label and length, never an abandonment, making
it a fact about the bins rather than about you), but reversing a written refusal
is a product decision and not a detail of the phase that happened to want it.
See open question 2.

**Two things moved out of the prompt and into the tools.** The caps, because a
cap the model is asked to respect is a cap it can ignore. And today's
refusals: `open_work` applies the picker's own suppression, so "not now" means
the same thing whichever of the two is choosing. Which also made
`recent(limit)` smaller than planned — it answers what was *done*, never what
was turned down, because a list of refusals adds nothing once they are already
absent from the work, except a record of what you keep saying no to.

### Write tools and the confirmation policy

Decision 3. The policy is grounded in rules the product already has, not
invented for the model:

**Runs directly** — already reversible in one press, and already exists as a
button the human could have pressed:

| Tool | Reverses via |
| --- | --- |
| `complete(item_id)` | `open` transition, one press |
| `complete_chore(chore_id)` | retraction — events are never deleted |
| `start_timer(label, minutes)` | stop, in the lid on every screen |
| `refuse(kind, ref_id)` | `UnrefuseToday()` |
| `snooze_chore(chore_id, until)` | say it again; the baseline never moved |
| `create_task(text)` | `untask`, then drop |

**Asks first** — creates a future interruption, or disposes of something:

| Tool | Why confirm |
| --- | --- |
| `create_moment(label, at)` | it will interrupt you later |
| `create_chore(name, every)` | it comes back on its own, forever |
| `retire_chore(chore_id)` | stops something recurring |
| `drop(item_id)` | disposal |
| `split(items[])` | decision 8 — capture must never be silently rewritten |


**Never available to the model:**

| Refused | Why |
| --- | --- |
| `reword(item_id, text)` | rewrites your own words, and it is not versioned. `!fix` is yours alone. |
| anything touching `checkins` | how you feel is said by you, never inferred |
| anything deleting a row | nothing in this product deletes; it retracts |

The confirmation surface is the button that already exists. A proposed
`create_moment` renders as the moment card with a *keep it* press — not a modal,
not a yes/no dialog.

**Splitting shipped at phase F, and it is not a write tool.** `Split` returns
strings; it has no way to write anything, which is decision 8 stated as a
property of the code rather than as an intention about it. The pieces travel in
the form that renders them and are written only by the press underneath — so
there is no stored proposal, nothing pending, and nothing to expire. That also
disposes of open question 1 for this case: an unanswered proposal lasts exactly
as long as the page it is on.

Two things it refuses that the design did not think to ask for:

- **A piece has to be made of the note.** Every word is checked against the
  original — dropping is allowed, because splitting drops the joins; inventing
  is not. "Use their own words" is exactly the instruction a model is most
  willing to improve on, and something a model wrote must never be proposed
  back as a thing you said.
- **The original is kept, not dropped.** It goes to the shelf, next to the
  things it turned into, through the same transition every other exit uses —
  so undo works on it normally. Dropping it would make a machine's reading of
  your words the only surviving version of them.

Not on capture, despite the phase's name. It runs when a press asks, on a card
that only draws the press when `Overwhelmed()` says the note looks like several
things — the same rule that recognises the overwhelm turn, because a brain dump
and an overwhelm turn are the same shape. Capture stays exactly as it was: no
model runs on the path a thought arrives through.

Screen only. Chat's confirmation surface would be a button carrying four
strings back through the tap mechanism, which is a worse fit than the card that
is already there; the PWA is the main interface and this is a triage step.

---

## 7. Deterministic vs AI

> **Deterministic** owns anything with a right answer *and* anything whose cost
> or latency makes a model absurd.
> **AI** owns anything with a better answer.

| Question | Owner |
| --- | --- |
| Is this chore due? | code |
| Is it a candidate for interrupting? | code — budget, window, capacity |
| **Is this the moment to actually interrupt?** | **AI** (decision 6) |
| When must I leave? | code |
| Does this fit in 20 minutes? | code, using the learned or estimated duration |
| **Which of these deserves attention?** | **AI** (decision 1), rules as fallback |
| Have I offered this today? | code |
| Is this message a command or a thought? | code — `Match()`, never the model |
| How long does this take? | **AI first, learning overrides** (decision 4) |
| What is the smallest first step? | AI |
| How do I say this without sounding like a demand? | AI |
| Is this one thought or four? | AI, confirmed |

**What decision 4 costs, recorded honestly:** the model does not know your
garage's hold time. Its first estimate is a guess wearing an authoritative
number, and the fit arithmetic will act on it. The mitigation is that a measured
median replaces it after ~5 real runs, and that an estimate is never *shown* —
only used to decide whether to suggest something.

---

## 8. Coach behaviour

Prompts are short because their jobs are narrow, and short prompts are what keep
the cached prefix cheap.

**Shared preamble (~120 tokens, cached):**

```
You are Squirrel. You help one person with ADHD by handing them one
manageable thing at a time.

Never produce a plan, a checklist, or numbered steps in what you say.
One thing. Two sentences at most. Plain words.
Never say "should", "just", or "simply".
If you cannot answer usefully, say nothing rather than something generic.
```

The last line matters most: a model that can decline produces silence, and the
deterministic answer takes over.

**The overwhelm turn adds:**

```
The person has listed several things at once. That listing is the
overwhelm — do not reflect it back.

Choose ONE. Prefer, in order: something at a fixed time; something under
five minutes; something bodily (eating, washing, sleeping) when capacity
is low.

Say what to do and one reason. Then say the rest is kept.
Do not list the rest.
```

The brief's worked example passes: five items, low capacity → bodily first →
*"Forget the rest for now. Get in the shower. I have the other four written
down."*

**Voice adapts to capacity (decision 17).** The `Capacity` field is already in
every prompt, so this costs nothing extra:

```
When capacity is "low", drop warmth and character. Shorter sentences,
plainer words, no turns of phrase. Say the thing and stop.
```

Worth knowing what this implies: **a plainer voice is itself legible.** When
Squirrel goes flat, that is a visible sign it noticed something. Under the old
Principle 5 that would have been forbidden as a statement about the person; you
opened that rule, so it is allowed — but it is a signal, not a neutral setting,
and it will be read as one.

**Shape guards, which are not content guards.** Principle 5 is open — the coach
may evaluate, compare, mention counts and streaks. That governs *what* it may
say. Form is separate and still enforced in `guard.go`: two sentences, no
numbered lists, no headings, no markdown. Anything failing is discarded and the
deterministic line used. **No retry** — a retry is a second chance to say
something worse.

`breakDownTask()` is the one exception to "no lists": it *returns* 3–5 steps,
which the application stores and reveals one at a time. **The list never
renders.** Pacing stays deterministic, which is the guarantee that keeps
decision 5 from becoming the twelve-step plan.

**Shipped at phase E, with the guarantee moved into the store.** There is no
function on `*Store` that returns the sequence — only `NextStep`, which returns
one — so no surface *could* render a list even if a later author wanted one.
The same device already keeps a caller from rendering a count of the pile or a
series of moods. `Last` says whether anything comes after; there is deliberately
no position and no total, because "step 2 of 5" is a count of what you have
left.

**Only "too big" asks for one.** The other three blockers already end somewhere
that is not a sequence: "don't know how" ends in a question whose answer is a
thought, and thoughts go in the pile; "boring" ends in a timer, because the
going is the point; "not today" is not an obstacle. `BreakingHelps()` says so in
the core, next to the ladder it belongs to.

**Synchronous, not first-paint-then-replace.** The diagram in §3 shows the fixed
line painting and a step arriving a second later. That needs polling, and §8a
already refused streaming for a related reason. What shipped instead: the call
happens on the press, and the fixed line is what renders whenever it does not
answer — so the failure is invisible rather than late.

---

## 8a. The coach surface

Decision 12: a widget on every screen, live-chat shaped.

### The exception it takes, recorded rather than taken quietly

`DESIGN.md` says: *"Don't open a modal for the chore interval, or for anything
else that needs neither interruption nor protected focus."*

The rule carries its own condition, and this meets it. A coach conversation
happens when everything else on screen is noise; protected focus is the whole
point of the surface. The chore picker was refused a modal because choosing an
interval needs neither — that reasoning is untouched and still governs
everything else.

### Shape

- **The button is the acorn.** `DESIGN.md` already establishes it as the
  product's second mark, *"available as a badge anywhere the full mascot is too
  much."* Cream stock, 3px outline, sticker offset, bottom-right. The live-chat
  convention rendered in the product's own materials rather than generic SaaS
  chrome.
- **Not in the lid.** The lid already carries mark, wordmark, timer strip, two
  cross-links and search, and the Lid Step-Down Rule exists because it was
  taking a fifth of a phone screen. A seventh element is not available.
- **Bottom sheet on a phone, right panel on desktop.** Thumb-reachable where
  the phone is primary.
- **Its own little lid:** cap purple header, acorn, close. Your words on cream
  in the casual axis; Squirrel's on paper in the Voice role. Same two voices as
  everywhere else.

### `/coach` is a real page

The sheet is `pile.js` upgrading a real route, not a JavaScript-only construct
— the same progressive enhancement the chore picker's `<details>` already uses.
Works with scripting off, deep-links, survives a reload. And because it is
chrome rather than a destination, **the home screen still has three doors.**

### Two behaviours it inherits

- **Opening costs nothing.** The sheet paints with the *cached* offer already
  there (decision 13). No model call on open — idle opens are free, which is the
  same trap the offer cache exists to close.

  *Shipped at C, broken at D, fixed after.* Phase D wired the decision into the
  shared `offerFor`, which the sheet also calls — so every acorn press became a
  chance to pay for a tool loop. The seam now carries whether the caller may
  spend: home may, the sheet may not, and the sheet shows a decision already
  paid for or the picker's own. Worth recording how it got through: the phase-C
  test asserting this checked only the conversational seam, so it stayed green
  while the picker's seam paid. A test that names a property has to check every
  way the property can be broken, not the one that existed when it was written.
- **What you typed never disappears.** Close mid-sentence and it is there on
  reopen. *A capture box that clears on failure is a capture box that eats
  thoughts* — the slot's rule, and it applies here for the same reason.

### Typing is not required

Decision 14 routes "I can't start" through the coach, which risks demanding
typing at the moment of least capacity. So the sheet keeps the four blockers as
one-press chips that fill the box — *too big · don't know how · boring · not
today*. One press, no typing, and the answer still comes back specific to this
task rather than generic.

That, plus the first paint above, means the worst case of decision 14 is that
you press one chip and read the same fixed sentence you would have got anyway.

### Rendering, and what closing it means

**No streaming (decision 16).** The fixed line is already on screen, so there is
no blank wait to fill — the better answer simply replaces it when it lands.
Nothing half-written is ever shown, which also means `guard.go` sees the whole
response before any of it does. Streaming would put text on screen that the
guard has not yet had a chance to reject.

**Closing it means nothing (decision 21).** The widget never initiates; you
opened it. Three opens with no engagement is not a signal, the acorn does not
dim, and nothing goes quiet. Reading meaning into how you use a button is the
product forming an opinion about you from behaviour rather than from something
you said — and the check-in already exists for saying it.

## 9. Memory and personalisation

**Within a session:** the last three exchanges, in memory, discarded after 30
minutes. Never replayed into a later call.

**Across sessions:** nothing conversational persists into prompts. Coach turns
are *stored* (decision 11) with a payload marker, using the same mechanism
`isNote()` already uses to keep commands out of the pile — the record stays
complete, the deck stays clean.

Everything durable is a column, queried in SQL:

- median timer duration per label → overrides the model's estimate
- which unstuck branch you choose, and whether a completion follows
- hour-of-day of real completions → shifts asking windows

**On estimate-versus-actual:** you chose to have Squirrel correct your
estimates. The learned median makes it possible. The rule keeping it from
becoming a scoreboard: the comparison drives fit arithmetic and is never
rendered *as* a comparison.

---

## 10. Proactive coaching

Decision 6 puts the model in this path. The architecture keeps it affordable and
bounded:

```
every minute:  rules → candidates?
               ├─ none (≈1,435 ticks/day) → nothing, no call
               └─ some (≈5/day)           → shouldInterrupt(candidates, now)
                                             → go/no-go + wording
```

**The rules the model cannot override:** the daily budget enforced by the unique
index on `(person, kind, date)`; quiet hours 22:00–06:00; the capacity gate; an
active refusal. The model may decline to interrupt when the rules would have
allowed it. It may never interrupt when the rules would not.

**Two of those four do not actually exist**, which phase H found rather than
assumed. The budget does — the unique index enforces it, and a chore already
raised today stops being due, so the day's second trigger ends before the model
is reached at all. Asking windows do. **Quiet hours and the capacity gate do
not**: a chore with no stated preference is open at every hour, and nothing on
the nudge path reads capacity. Neither was added here, because a rule about
when the product may speak is a product decision rather than a detail of the
phase that noticed it, and quiet hours in particular would make the test suite
depend on the hour it runs at. Both are in the roadmap's open list.

The asymmetry survives regardless, because it does not rest on those rules — it
rests on **where the call sits**. The model is only ever handed a candidate the
rules already produced, so there is no path by which it can say yes to anything
else. It fails open for the same reason, and that is the one place in this
architecture that inverts the usual rule: a coach that is down must not
silently turn off a feature that worked without one for months.

**Which channel it may use (decision 15).** Push is reserved for interruptions
*about a fixed point* — something the world imposed and that has a moment worth
catching. Everything else the coach raises goes to the Campfire room, which you
read when you read it.

That keeps a property worth keeping: **a buzz always means the same thing.** You
never have to wonder whether the phone is telling you to leave or offering an
opinion. It is the reason the leave-by push was worth building at all, and it
survives the coach getting interruption rights.

That asymmetry is deliberate — it can only ever make Squirrel quieter than the
rules permit, never louder.

---

## 11. Cost model

**Assumptions:** one user; ~15 home opens/day, ~70% served from cache after
event-based invalidation; ~5 interrupt candidates/day; ~2 breakdowns; ~1
overwhelm turn; ~2 splits. Retrieval adds 2–3 round trips carrying ~600 tokens
of tool schemas. Terra $2/$12, Luna $0.20/$1.20.

| | calls/day | €/month |
| --- | ---: | ---: |
| Offer decision (Terra, cache misses) | ~5 | 1.60 |
| Interrupt go/no-go (Terra) | ~5 | 0.60 |
| "I can't start" through the coach (Terra) | ~3 | 0.72 |
| breakDownTask (Terra) | ~2 | 0.48 |
| Overwhelm turn (Terra) | ~1 | 0.24 |
| Splits + estimates (Luna) | ~4 | 0.08 |
| Opening the sheet | any | **0.00** |
| **Total** | | **≈ €3.72** |

**About 37% of the €10 ceiling**, with the cache doing the heavy lifting.

| If… | €/month |
| --- | ---: |
| the cache is disabled | 6.05 |
| usage doubles | 6.00 |
| usage doubles *and* cache disabled | 12.10 — over |

**The spend counter (decision 10)** tracks tokens per calendar month. On the
ceiling it degrades to `PickNow()` and `UnstuckFor()` for the rest of the month
and says so once, in the room. Everything keeps working; it stops being clever.

**Hypothetical multi-user**, a product that does not exist: ≈€300/month at 100,
€3,000 at 1,000 — near-linear, no shared cache, no per-tenant fixed cost.

---

## 12. Migration plan

Additive — there is nothing to migrate from. Every phase ships its deterministic
version first and keeps it.

```
internal/coach/
  coach.go      Coach interface + NoCoach{} (zero value, the default)
  tools.go      six read tools, six write tools, the permission policy
  views.go      the structs a tool may return
  guard.go      shape validation before any output reaches a human
  budget.go     the monthly token counter and the degrade switch
  provider.go   one interface: base URL, model ID, key — all from config
  openai.go     the one implementation today (decision 19)
```

`NoCoach{}` as the zero value means the default build has no AI, every existing
test runs the deterministic path unchanged, and a provider outage is a silent
downgrade rather than an outage.

**OpenAI direct, gateway-shaped (decision 19).** No markup and no third party
reading every prompt, but the base URL, model ID and key are config values
behind one interface — so moving to OpenRouter or anyone else later is a config
change plus one file, not an integration.

**A/B is a config switch (decision 20).** No shadow-running, no per-call
alternation. Change the model ID, redeploy, live with it for a week. Coach
quality here is a feeling across days, not a metric across requests, and a coach
whose quality visibly varies is a strange thing to lean on when struggling.

**The log is kept (decision 18).** `coach_answers` holds the prompt, the
response, the model, the tokens and whether it was used, indefinitely. It is
what makes a bad answer inspectable afterwards and what makes the config switch
evaluable at all. It is also a permanent record of your worst moments and what a
machine said about them — kept deliberately, on the same reasoning that keeps
the check-in history.

---

## 13. Files affected

**Unchanged, deliberately:** the capture path (`capture.go`, `spool.go`,
`drain.go`, `sink.go`) and `intent.go`. The coach never precedes `Match()`.

**New:** `internal/coach/*`; migration 0020 (`coach_answers` — what was asked,
which model, what came back, whether it was used, tokens spent; the A/B and
spend surface); migration 0021 (`items.estimate_minutes`, nullable).

**Touched:** `pick.go` (becomes the fallback and the tool backend — the rules
are untouched), `stuck.go` (`breakDownTask` behind the ladder), `apply.go`
(`!coach`, split-on-capture), `schedule.go` (candidate pre-filter →
`shouldInterrupt`), `web/now.go` + a new coach view (decision 7: both surfaces),
`boot.go` (build the coach from config).

---

## 14. Implementation phases

| Phase | What | Value if the next never lands |
| --- | --- | --- |
| **A** | Skeleton: interface, `NoCoach{}`, guard, budget counter, config, `coach_answers`. No provider. | The boundary exists and is tested. |
| **B** | Luna behind it. `!coach <text>` in chat only. | You hear how it talks before it is near the product. |
| **C** | **The coach surface**: acorn button, `/coach` route, sheet, rolling window, first paint. ~~Deterministic content only — no model yet.~~ | The whole interface exists and is usable, running on `UnstuckFor()` and the cached offer. |
| **C2** | **The overwhelm turn** (Terra) behind that surface. | The genuinely new capability. |
| **D** | Read tools + **model-decides** with the offer cache and `PickNow()` fallback. | Decision 1 lands, cost-controlled. |
| **E** | `breakDownTask()`, and "I can't start" routed through the coach. | The ladder gets specific; it stays as the first paint. |
| **F** | Splitting a brain dump, proposed and confirmed. ~~On capture~~ — on a press, on the card. | Capture stops flattening four thoughts into one. |
| **G** | Write tools + confirmation policy. | The coach can act, reversibly. |
| **H** | `shouldInterrupt()` on rule-produced candidates. | Proactive, bounded. |

C is what justifies the project. H is the most dangerous and goes last on
purpose — it is the only one that speaks without being spoken to.

**C shipped with the box connected, which this table originally said it would
not.** The phase was written before B existed, on the assumption that the
surface would be judged with nothing behind it. By the time C was built B had
shipped a working provider, and a sheet whose text box could not be answered
would have been a worse thing to judge, not a purer one. What C2 still adds is
unchanged and is the part that matters: recognising the overwhelm turn and
escalating it to Terra. The four chips are still answered by `UnstuckFor()`
alone and stay that way until E.

### Settings (shipped at phase A)

| Variable | Default | What it does |
| --- | --- | --- |
| `OPENAI_API_KEY` | *(empty)* | **Empty means no coach**, and that is a supported shipping state, not a degraded one: the picker chooses, the ladder answers, and every screen works. Logged at info, not as a warning. |
| `COACH_BASE_URL` | `https://api.openai.com/v1` | Where to call. Present with one provider so that moving behind a gateway is a deployment change rather than a code change. |
| `COACH_MODEL_FAST` | `gpt-5.6-luna` | The routine tier. |
| `COACH_MODEL_DEEP` | `gpt-5.6-terra` | The escalation tier, for the turns where judgement matters. |
| `COACH_BUDGET_EUR` | `10` | The monthly ceiling, in whole euros. `0` disables the in-process ceiling; the provider's own hard spend limit is unaffected and remains the control that matters against a stolen key. |

A model the price table does not know is priced at zero, which means the
ceiling silently stops existing. Boot warns about exactly that at start rather
than leaving it to be discovered on an invoice.

---

## 15. Open questions

Everything architectural is decided. One small thing remains, and it can wait
until the phase that needs it:

1. ~~**How long does an unanswered `create_moment` proposal stay pending?**~~
   **Answered by construction at phase G.** A proposal is stored nowhere — it
   travels in the form that renders it, exactly as a split does — so it lasts
   as long as the page it is on and there is nothing pending anywhere to
   expire. No lifetime, no sweep, no lapse rule to get wrong.

2. **Should finished timers be kept, so durations can be measured?** *(raised
   at phase D, 20 August.)* `history(label)` was designed to answer "how long
   does this usually take" from real timers and replace the model's guess.
   Answering it needs a row per finished run, and migration 0017 refuses a
   timer history in writing: it becomes a record of what you started and
   abandoned, which is a report card.

   There is a narrower version that may not be: **completed runs only** — label
   and length, written when a timer reaches its end and never when one is
   stopped early. Nothing about abandonment is recorded, so there is nothing to
   read back as a failure rate, and the median is a fact about the bins rather
   than about you. It would also be the first thing in this product that
   *measures* rather than *remembers*, which is why it is being asked rather
   than assumed. Phase D shipped five tools instead of six; nothing is blocked
   on the answer.

*Resolved 20 August: the coach lives in a widget on every screen (§8a), which
answers the earlier route-versus-home question — it is chrome, so home keeps its
three doors.*

## 16. Records that must be amended before phase C

Not open questions — work items, listed so they are not forgotten:

- **`DESIGN.md`** gains the modal exception (§8a) and the coach sheet as a
  component. The existing no-modal rule stays exactly as written; the exception
  is recorded beneath it with its reasoning, the same way the door-art exception
  was.
- **`PRODUCT.md`** gains the coach as a surface, and the sentence that the
  deterministic answer is never deleted — only demoted to the floor.
- **`docs/pile-screen.md`** route table gains `/coach` and the coach's writes.
