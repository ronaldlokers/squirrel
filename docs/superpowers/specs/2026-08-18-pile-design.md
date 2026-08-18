# Phase 5 — the pile

Builds on the phase 1 to 4 specs. Those still bind. This one gives notes a way
out.

## Why

Squirrel is a write-only memory.

`items` has ten columns and not one of them is a state. A note is raw text and
a timestamp, permanently. The evening message shows captures since yesterday,
so every note is visible exactly once — the day you wrote it — and then never
again. There is no way to find one, no way to finish one, and no way to say a
thought stopped mattering.

That is a strange shape for an external memory. The kickoff named three
bottlenecks: friction at capture, reminders that stop landing, and no record of
whether something already happened. Phases 1 to 4 answered all three for
*chores*. Notes got the capture half and nothing else.

The failure is not that the pile is large. It is that the pile has an entrance
and no exit, so the only thing you can do with a thought you had three weeks ago
is remember that you had it.

## What changes

- A note gains a **lifecycle**: `open`, `done`, `dropped`, `kept`.
- A **command prefix**, `!`, and with it search, the pile, and help — a
  vocabulary that today exists nowhere.
- A note can **become a chore**.
- A **web UI**, for triage and search only.

## What does not change

Everything phases 1 to 4 guarantee. Capture is still the default and a command
still the exception; raw text is still spooled and fsynced before it is
acknowledged; **Postgres is still never on the capture path**; overdue is still
a gradient; the nudge is still one chore, once a day.

And the rule that outlasts all of them: no streaks, no counts of times missed,
no scolding — **nothing that accrues and can be destroyed.**

Two consequences of that rule are decided here rather than left to taste.

**The pile is never counted.** No badge, no total in a header, no "341 notes",
no "you have N to review". A number that grows while you are not looking, next
to an implied target of zero, is the streak mechanism wearing a different hat:
it accumulates, it can be lost, and the day it is finally cleared it refills by
Tuesday. The screen shows notes. It does not score them.

This binds the capped lists too. `!notes` shows the newest N and says there are
more — "there is more" is a fact about the list, "there are 341 more" is a
score. The screen scrolls rather than paginating, for the same reason: a page
count is a total in disguise.

**It is called "the pile", not the inbox.** Considered and rejected: `inbox`
imports inbox-zero, which is the accumulating-counter mechanism above, complete
with a target you will never hold. The counter is the real risk and it is banned
outright either way, but the word carries the convention in with it. Recorded
because the argument against "the pile" — that it may read as self-deprecating
every time it is opened — is real, and was overruled deliberately rather than
missed.

**Capture stays in one place.** The web UI never accepts a new note. Two
capture surfaces means two places to look for a thought, which is the problem
this project exists to solve. The screen is read-and-triage, permanently.

## The lifecycle

| State | Means |
|---|---|
| `open` | Untriaged. The default, and what every existing row becomes. |
| `done` | It was a task and it happened. |
| `dropped` | It stopped mattering. |
| `kept` | Not a task. Reference. Leaves triage, stays searchable. |

**`kept` is the load-bearing one.** A serial number, a link, a name someone
mentioned — these are not tasks and will never be `done`. With only `done` and
`dropped` on offer, every reference note sits in triage forever and the pile
reappears inside the thing built to clear it. `kept` is the exit for notes that
were never going anywhere.

**Every transition is reversible**, including back to `open`. Undo is a
transition, not a special case. Phase 3 spent two review rounds getting undo
right for chore completions, and triage makes more state changes in a minute
than the chat did in a week.

**Two columns on `items`, not an events table.** Chores use events because a
chore recurs and its history *is* its clock — `max(occurred_at)` is how "how
overdue" is computed at all. A note does not recur. It has one current state,
and its history is something nobody reads. An events table here would be
ceremony.

**The evening message is unaffected.** It shows captures in a recent window,
which is a question about `received_at`, not about state. A note triaged an
hour after writing it still appears that evening. This is deliberate: the
evening message reports what you told Squirrel, not what is outstanding.

## The command surface

`?` is the only command today. `!help`, `!notes` and `!chores` were discussed
during phase 4 and never built.

**`!` is a prefix, not a keyword, and that is load-bearing.** Every bare word is
a capture by design. `find my keys` is a thought, not a search; `notes to self`
is a thought, not a listing. A keyword-triggered command would eat them. Phase 2
hit this exact trap when `every day i think about leaving` had to stay a capture
while `every day meds` became a chore, and the answer then was the same as the
answer now: make the command shape unambiguous rather than guessing at intent.

| Command | Does |
|---|---|
| `!find <text>` | Search raw text across every state, newest first, capped |
| `!notes` | The pile — `open` only, newest first, capped |
| `!chores` | What `?` does today |
| `!help` | The vocabulary |

`?` keeps working as an alias for `!chores`. It is muscle memory, and phase 4
leaned on it as the on-demand escape from the one-a-day nudge budget.

**Results are numbered and reuse the phase 3 prompt machinery.** A numbered
result is a prompt of a new kind, so `done 2` resolves against it exactly as it
resolves against a nudge — the "numbered prompt owns the numbered surface" rule
already exists and already has the bugs beaten out of it.

**But that machinery is chore-shaped and has to be generalised first.**
`prompt_lines.chore_id` is `not null references chores (id)`, `RecordPrompt`
takes `chores []Chore`, and every resolver returns a `Chore`. A line becomes
"either a chore or an item": `chore_id` and `item_id` both nullable, with a
check constraint that exactly one is set. This is the largest single piece of
work in 5a and it is not optional — without it there is no way to say `done 2`
against a search result, and therefore no way to clear the pile from chat or to
promote a note to a chore by position.

An earlier draft of this spec called that wiring. It is not.

**An unknown `!command` is not a capture.** It is a mistyped command, and
answering it with 👀 would silently swallow the correction. It gets a reply
naming what exists.

## A note becomes a chore

```
!find bins
 1. bin day is thursday now
 2. the recycling thing

!chore 1 every 2 weeks
```

`ParseEvery` and the chore upsert both exist. The note's text becomes the
chore's name, and **the note becomes `done`** — it did its job by turning into
something recurring.

**No `chore_id` column linking the two.** There is one case that would read it
("where did this chore come from") and no second, and the standing rule is two
concrete cases before an interface. If a reason appears later, the column is a
migration.

## The screen

One screen, no navigation. The pile, **newest first** — oldest-first is a
backlog you are behind on; newest-first is context you still remember writing.

Each note shows its text, when it landed, and four actions: **done · drop ·
keep · make it a chore**. The chore action takes an interval.

**Undo lives on the screen**, not on a history page. A row stays in place for a
moment after it is actioned so the undo has somewhere to be.

**Keyboard-first**: one key per action. Forty notes with a trackpad is forty
aimed clicks, and the whole point is to make clearing cheap enough to actually
happen.

**Letters are actions; movement never takes a letter.** An earlier draft of this
spec said `j`/`k` to move, which collides with `k` for keep — the two rules in
one sentence contradict each other, and the collision was found while designing
the screen rather than while writing this. It is resolved in favour of the
actions: `d` done, `k` keep, `x` drop, `c` make a chore, `u` undo, `/` search.
Movement is `space` and the arrow keys.

The collision is not really a key-binding problem. The screen shows one note at
a time, so there is nothing to move *between* — moving is skipping, and skipping
is not the same act as triaging. Rebinding keep to free up `k` would have kept
a navigation model the surface does not have.

**Search is on the same screen**, across every state, because "what did I say
about the boiler" is usually about a note you already filed.

**Postgres is on this screen's request path, and that is correct.** The spool
invariant exists because losing a capture loses a thought. Triage is not
capture: if the database is down the screen fails visibly and nothing is lost,
because the note is already durable. The spool is for the entrance, not the
exit.

**The screen is a transport, and lives like one.** The core must not learn what
HTML is, the same way it never learned what Campfire is. Rendering and routing
sit outside `internal/squirrel`; the core exposes the same kind of narrow
functions the Campfire transport already consumes.

## Authentication

The pile is every thought you have ever had at this bot. That is a materially
more sensitive surface than a webhook whose worst outcome is being told about
the bins.

**Authentik forward-auth at Traefik.** A middleware calls an Authentik outpost;
Squirrel receives an identity header and writes no authentication code at all.
The LAN-only `ipAllowList` from phase 4 stays on as a second layer.

Not app-level OIDC, which is how linkding and immich do it here. That would add
sessions, cookies and a redirect flow to a binary that has none, in the layer
this project keeps deliberately thin. Forward-auth is new infrastructure once,
then reusable by everything after it.

Not LAN-only alone. Every phone, TV and guest device on the wifi is not an
acceptable reader for this.

## Schema

```sql
-- 0008
alter table items add column state text not null default 'open';
alter table items add column state_at timestamptz;

create index items_state_received_idx on items (state, received_at desc);
```

Every existing row becomes `open`, which is true: nothing has ever been
triaged.

## Delivery

**5a — the chat half.** The lifecycle, the `!` commands, search, promotion to a
chore. Ships on its own and is useful on its own: search and `!notes` alone end
the write-only problem.

**5b — the screen.** The web UI and the forward-auth infrastructure.

One spec, because the two halves share a data model and splitting it across two
specs would leave the lifecycle argued for in one and used in the other. Two
plans, because 5a should not wait on an Authentik outpost.

## What must be tested

| Case | Why |
|---|---|
| `find my keys` is still a capture | The prefix rule is the whole defence against eating thoughts |
| `!find` with no argument does not list everything | An empty search reads as a mistake, not a request for all of it |
| An unknown `!command` replies rather than captures | Otherwise a typo is silently swallowed as a note |
| `done 2` against `!find` results resolves the right note | Numbered surfaces already collided once, in phase 4 |
| A triaged note still appears in that evening's message | The evening window is about `received_at`, not state |
| Every transition reverses, including twice in a row | A tap is a state assertion; a retry must be idempotent |
| A promoted note becomes `done` and the chore exists | Two writes, one intent — neither may land alone |
| Search matches across every state | `kept` exists precisely to be found later |
| Nothing anywhere emits a count of open notes | The rule this phase is most likely to break by accident |
| A capped list says there is more without saying how much more | The obvious way to write a cap leaks the total |
| The screen refuses to create an item | Read-and-triage is a permanent property, not a current state |

## Deferred

- Tags, folders, projects. The pile is flat until flat provably fails.
- Resurfacing old notes unprompted — explicitly rejected: it would become a
  second stream of messages competing with the nudge for the same attention.
- Full-text ranking. `!find` is substring matching until that is not enough.
- Editing a note's text. Captures are what you actually said.
- Liveness: last capture, unspooled count, last nudge, last presence ping.
- Task initiation — next-step breakdown, timers, body doubling.
- Monotonic totals, and noticing a chore repeated nudging has not shifted.
