# Phase 4 — Make the nudge land

Builds on the phase 1, 2 and 3 specs. Those still bind. This one changes **when**
Squirrel speaks and **how much** it says, and it retires the daily digest as a
single message.

## Why

The kickoff named three bottlenecks: friction at capture, reminders that stop
landing once you habituate, and no record of whether something already happened.
Phases 1 to 3 addressed the first and third. The second is untouched, and phase
2 arguably made it worse by building the shape that habituates fastest.

Two findings from the ADHD literature drive this phase.

**Interventions have to act at the point of performance** — the place and time
where someone is failing to use what they already know. Barkley is explicit that
support delivered away from that moment is unlikely to succeed. A message at a
fixed hour is not that moment; it arrives at the wrong time for most chores, and
a single daily notification at a fixed time becomes invisible through
habituation within about a week.

**Externalising information is the weaker half.** Barkley again: externalised
cues "will prove only partially successful. Even then it will prove only
temporarily so. Internal sources of motivation must be augmented with more
powerful external forms as well." Squirrel externalises information and offers
no motivational counterpart at all.

There is a third, quieter finding. Self-regulation draws on a depletable pool,
and every decision spends it. A digest listing six due chores asks for six
decisions before anything gets done — decision load charged to exactly the
resource that is short.

## What changes

- The daily digest splits. Chores become a **nudge**; captures keep a
  once-daily message of their own.
- A nudge names **one chore**, never a list.
- Nudges fire from three triggers, at most **once per local day**.
- Completing a chore earns a **varied reaction** — small, immediate,
  non-cumulative.
- The evening message gains **what you did**, alongside what you captured.

## What does not change

Everything phase 1 to 3 guarantees. Capture is still the default and a command
still the exception; raw text is still stored before any parsing; **Postgres is
still never on the request path**; overdue is still a gradient, not a cliff.

And the rule that outlasts all of them: **no streaks, no counts of times missed,
no scolding.** This phase adds reinforcement, which is new, and the section on
it explains why that is not the same thing.

## Triggers

Three, in priority of whichever fires first:

| Trigger | Signal | Why it is a point of performance |
|---|---|---|
| You message Squirrel | Any inbound capture | You are awake, holding your phone, already in the conversation. The best moment available, and today it is wasted — Squirrel only ever answers |
| You arrive home | Home Assistant webhook | The closest thing a household chore has to a place |
| 19:00 local | Clock | A floor, not a schedule |

**At most one chore nudge per local day**, whichever trigger gets there first.
Plus the evening capture message, which runs regardless. Two messages a day at
the ceiling, and one of them is your own words coming back.

A second nudge is a list delivered in instalments, which is the thing this phase
exists to stop. Wanting more is what `?` is for: on demand is different, because
you asked.

**The once-a-day guarantee already exists.** Phase 2 built digest idempotency on
a partial unique index over `prompts (person_id, sent_for_date)` so a restart
inside the send window could not post twice. A nudge is a prompt of kind `nudge`
carrying today's date; if two triggers race, the second is refused by the index
and simply does not send. No new mechanism, no in-memory state to get wrong.

**One thing has to change for that to work.** The index does not include `kind`,
so a `nudge` and the evening capture message on the same date would collide and
the second would be silently swallowed as a duplicate. It becomes
`(person_id, kind, sent_for_date)`. Without this, the evening message stops
appearing on any day a nudge already fired — a bug that would present as "it
just goes quiet sometimes."

## Choosing the chore

**Weighted random among what is due, biased toward how overdue it is.**

Concretely: weight each due chore by its overdue ratio — how far past its own
interval it has gone, which `DueChores` already computes and orders by — and draw
one. A chore three intervals late is three times as likely to surface as one just
past due, and never certain.

Not simply the most overdue, for two reasons. The most overdue chore is the one
you have been avoiding longest, which usually means it is the aversive or vague
or large one — so that rule leads with the thing you least want to see, every
day, until it is done. And more sharply: the most overdue chore is by definition
the one nudging has already failed to shift. A fourth week of naming it is not
the intervention.

Weighting keeps the urgent thing usually surfacing while leaving the nudge
unpredictable, which is the same mechanism that fights habituation.

**Deliberately out of scope:** noticing that a chore has been nudged repeatedly
and ignored, and doing something else about it. That is the right idea and it
needs a notion of "nudged and not done" that does not exist yet. It is also easy
to build something that reads as an accusation. Later, carefully, or not at all.

## What a nudge says

One chore, in the existing voice, with the button phase 3 built:

```
bin day — 19 days, usually 14
[✅ done]
```

Riding back on a message you sent, it acknowledges the piggyback:

```
Got it. While you're here — bin day, 19 days, usually 14.
[✅ done]
```

Framing varies by trigger. Not for personality — varied phrasing is the same
anti-habituation mechanism as varied timing, and it costs nothing.

## The evening message

Runs at 19:00 local **every day**, whether or not a nudge already fired:

```
Today
 · bin day
 · water the plants

Since yesterday
 · buy milk
 · the thing with the boiler
```

**When no nudge has fired yet that day, the chore joins this message rather than
arriving as a second one.** 19:00 is both the nudge's fallback and the capture
message's slot, so on a quiet day the two would otherwise land together as two
notifications a second apart — which is worse than either. One message:

```
bin day — 19 days, usually 14
[✅ done]

Since yesterday
 · buy milk
```

The nudge still counts against the one-a-day budget and still records a prompt of
kind `nudge`, so the accounting is unchanged; only the delivery is shared.

**When you completed nothing, the section is absent.** Never a zero, never "no
chores completed today." An empty list is a scoreboard reading nil; an absent
section says nothing about you. This is the difference between the two, and it
is the whole reason "what you did" is safe to add.

If both sections would be empty, nothing is sent — unchanged from phase 2.

The capture half stays because it is currently the only way notes are ever seen.
Retrieval and note lifecycle belong to a later phase; until then, removing this
would make notes less visible than they are today.

## Reinforcement, without punishment

What makes a streak punish is not the reward. It is **accumulation that can be
lost**. Loss aversion makes losing hurt about twice as much as the equivalent
gain pleases; an all-or-nothing counter makes one miss read as total failure;
and the abstinence violation effect turns that into abandonment. The reset is a
statement about you.

So the rule for this project is not "no gamification." It is **nothing that
accrues and can be destroyed.**

**On completion, Squirrel boosts its own nudge message with a reaction that
varies** — a small fixed set, drawn at random, all of them positive or neutral.
Immediate, small, never negative, and unpredictable. Nothing accumulates, so there is nothing to lose. Intermittent
reinforcement is also the strongest schedule there is, and it is the same
mechanism that defeats habituation, so one change serves both.

It also fills a real gap: tapping a button currently gives you nothing but the
checkbox filling in.

**The ✅ receipt is not part of this and must not be varied.** 👀 and ✅ are
information — the thought is on disk, the thought reached Postgres. Randomising
them would turn an honest signal about durability into decoration, on the one
surface that reports whether the system actually worked.

**Out of scope, deliberately:** monotonic totals ("the 12th time since March").
The idea is sound and it cannot punish, since it only ever grows. But it needs a
longer history to mean anything, and adding two reinforcements at once makes it
impossible to tell which one worked.

**Never:** comparison against other people or against your own past. Gamification
research finds benefits to autonomy and relatedness and almost none to perceived
competence — comparison is where it turns corrosive.

One caveat for the record. The overjustification effect says extrinsic reward can
undermine intrinsic motivation — but only for activities that were already
intrinsically rewarding. Chores are not. This is the case Barkley calls a
motivational prosthesis, and the risk is low here. It would be high if this were
ever pointed at something you already enjoy.

## The presence webhook

A new inbound route, separate from the Campfire webhook. Home Assistant POSTs on
arrival; the body carries nothing that matters.

**It does not go through the spool, and that is a deliberate exception to the
central invariant.** Everything inbound is otherwise written to disk and fsynced
before being acknowledged, because losing it means losing a thought. A presence
ping is not a thought — losing one costs a nudge, and the evening fallback
catches the same day. Spooling it would also give it an `items` row, putting
"you came home" in the capture list. Phase 3 spent a fix removing button taps
from that list; adding arrivals would be the same mistake wearing a different
hat.

So: acknowledge immediately, signal in memory, done. The spool exists to never
lose a thought. This is not one.

**Authentication.** The Campfire webhook has none — its NetworkPolicy is the
entire story, documented as a deliberate and uncomfortable choice, tolerable
because Campfire shares the namespace and the rule is tight. Home Assistant is in
a different namespace, so this is a cross-namespace policy, and it gets a shared
secret as well, held in SOPS like the bot key. The blast radius if it leaked is
small — someone could make the bot nudge you — but small is not a reason for
nothing when the something is one header.

**A few minutes' delay after arrival**, rather than firing as you come through
the door — you have a coat on. And an in-memory debounce over roughly the same
window, because phones flap between wifi and cellular and Home Assistant will
fire on each flap.

**The failure mode worth naming:** if the automation breaks or Home Assistant is
down, the trigger stops and everything still works, because 19:00 catches it.
Good degradation, bad observability — you would never notice it had died. The
answer is to surface "last presence ping" wherever liveness eventually lives, not
to try to detect it here.

## Schema

```sql
-- 0007
drop index prompts_digest_per_day;
create unique index prompts_kind_per_day
  on prompts (person_id, kind, sent_for_date)
  where sent_for_date is not null;
```

The evening capture message and a nudge are both prompts with a date; without
`kind` in the index they collide.

## What must be tested

| Case | Why |
|---|---|
| Two triggers on one day produce one nudge | The budget is the design; the index enforces it |
| A nudge and the evening message coexist on one date | The index change is exactly this, and the old index silently swallowed one |
| The evening message runs on a day a nudge already fired | It is a floor for captures, not an alternative to them |
| On a day with no earlier trigger, 19:00 sends one message, not two | Two notifications a second apart is worse than either alone |
| No completions means no "Today" section, not an empty one | The difference between an absent line and a scoreboard reading nil |
| Selection is weighted, not deterministic | Otherwise it is "most overdue" with extra steps |
| A presence ping creates no `items` row | It is not a capture, and the capture list is load-bearing |
| A presence ping with a bad secret changes nothing | The only authentication this endpoint has |
| Losing every presence ping still yields a nudge that day | The fallback is what makes the exception to the spool safe |
| The ✅ receipt never varies | It is information, not reinforcement |
| Nothing due and nothing captured sends nothing | Unchanged from phase 2, and still the rule |

## Deferred

- Note retrieval, search, and lifecycle — the pile problem. A web UI belongs
  here, as a triage surface only, never for capture or cues.
- Monotonic totals.
- Noticing a chore that repeated nudging has not shifted.
- Liveness: last capture, unspooled count, last digest, last presence ping.
- Task initiation — next-step breakdown, timers, body doubling.
