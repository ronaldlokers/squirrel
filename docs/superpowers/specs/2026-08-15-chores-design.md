# Squirrel — phase 2: chores, and the bot speaking first

Status: approved 2026-08-15.

Builds on [`2026-08-14-capture-path-design.md`](2026-08-14-capture-path-design.md)
and its Go delta [`2026-08-15-go-port-design.md`](2026-08-15-go-port-design.md).
Everything in those still binds — the principles, the Campfire contract, the
guard, the spool, the identity model. This adds behaviour.

## Why this, and why now

Phase 1 shipped and was immediately unusable, for a reason worth writing down:
**there was nothing to say to it.**

The original thesis named three bottlenecks: friction at capture, reminders that
stop landing, and no record of whether something already happened. Phase 1
addressed the first. The other two are both the bot talking first, and phase 1
built a bot that structurally cannot.

The evidence was in the intent set the whole time. *"Bare `done` resolves
against the most recent outstanding bot prompt."* Nothing in phase 1 can produce
an outstanding bot prompt. Four of the six intents — `complete`, `defer`,
`answer`, and half of `query` — assume a conversation the bot has started. We
built the half that requires the user to have initiative, for a tool whose
premise is that initiative is the scarce resource.

A capture-only bot is also write-only. A notebook can at least be re-read.

So phase 2 is: **the bot speaks first, and the store becomes visible.** Chores
carry it, because chores are the only part of the design where the bot has
something to say without the user having said anything first.

## Scope: chores only

Items keep sitting in `inbox` with no triggers. The general scheduler — arbitrary
times against arbitrary captures — is not in this phase. Chores have a far
simpler trigger model, relative to last completion rather than to a calendar, and
that is enough to get the whole loop working.

## The daily message

One message a day, 08:00 Europe/Amsterdam.

```
Due
 1. bin day — 2 days, usually 7
 2. water plants — 5 days, usually 4
 3. vacuum — 19 days, usually 14

Since yesterday
 · buy milk
 · ask about the flux migration
 · the thing with the boiler
```

Facts only. No streak, no count of times missed, no "still not done". *"19 days,
usually 14"* is the entire editorial position, and it does not change as the
number grows. Shame is not a feature and it is the fastest way to make someone
stop opening the thing.

If nothing is due **and** nothing was captured, nothing is sent. A daily "nothing
to report" is how you teach someone to skip the message.

The second half is the fix for the store being invisible: one query, and the
captures stop disappearing into a hole.

## Defining a chore, from chat

```
every 2 weeks: vacuum
every 2 weeks vacuum        ← the colon is optional
every week bin day
every 3 months change the filters
```

Upsert on the name. Saying it again with a different interval updates it — there
is no separate edit verb, and nothing to remember.

Units are `day`, `week`, `month` and their plurals. The count is optional
(`every week` means every 1 week). A month is 30 days; this is a nudge, not a
calendar.

The name is trimmed and matched case-insensitively for the upsert, and stored as
first written. So `every 3 weeks Vacuum` updates the chore created by
`every 2 weeks: vacuum` rather than creating a second one, which is the only
behaviour that makes upsert-by-name safe to rely on.

### The colon is optional on purpose

Requiring punctuation would disambiguate cheaply, and it is the wrong trade.
*"No command language to memorise"* is a stated principle, and a mandatory colon
is exactly that.

The cost is over-matching. `every 2 weeks I forget to call my mother` is a
plausible **note**, and it will become a chore called "I forget to call my
mother". That is handled the way everything else in this system is handled —
never by rejecting, always by making the correction cheap:

```
every 2 weeks: vacuum
→ vacuum, every 14 days. First nudge in 14 days.
  nvm if you meant that as a note.
```

`nvm` is already `drop` in the intent set. A wrong guess costs one word, and the
raw text was stored verbatim before any of this ran — as always, parsing is a
derived view and never a gate.

### Removing one

Chat-defined has to mean chat-removable, or chores accumulate with no exit. The
daily message is numbered, so:

- `2` or `done 2` — completed, resets the clock
- `stop 2` — deactivated, history kept
- `?` — the same list, on demand

Two verbs against one numbering.

## Tolerance is the re-nudge gap

A chore is **due** when `now ≥ baseline + interval`, where the baseline is the
last completion, or the chore's creation if it has never been completed. So a new
chore's first nudge is one full interval away, which is what the confirmation
message promises.

Once due, it **appears** when `now ≥ last_shown + tolerance`. `bin day` with a
tolerance of one day appears every morning; `vacuum` with seven appears weekly.

Tolerance controls frequency and never tone. An overdue chore is never described
as overdue, only as a number of days.

Ignoring a nudge does nothing at all. No acknowledgement, no counter, no "still
not done". It reappears per its tolerance and states the same plain fact.

## Data model

```sql
create table chores (
  id                bigint generated always as identity primary key,
  person_id         bigint      not null references people (id),
  name              text        not null,
  interval_seconds  bigint      not null,
  tolerance_seconds bigint      not null,
  active            boolean     not null default true,
  created_at        timestamptz not null default now(),
  updated_at        timestamptz not null default now(),
  unique (person_id, name)
);

create table events (
  id          bigint generated always as identity primary key,
  chore_id    bigint      references chores (id),
  item_id     bigint      references items (id),
  person_id   bigint      references people (id),
  -- 'ack' today. 'sensor' and 'inference' are why this table exists.
  source      text        not null,
  occurred_at timestamptz not null,
  payload     jsonb       not null default '{}',
  inserted_at timestamptz not null default now()
);

create index events_chore_occurred_idx on events (chore_id, occurred_at desc);

-- A bot message that offered numbered choices. `done 2` resolves against the
-- most recent one, which is what "never require an ID or a quote" means in
-- practice.
create table prompts (
  id              bigint generated always as identity primary key,
  person_id       bigint      not null references people (id),
  conversation_id text        not null,
  kind            text        not null,   -- 'digest' | 'query'
  sent_at         timestamptz not null,
  -- Set for digests, null for on-demand queries. The unique index is what makes
  -- the scheduler idempotent across restarts.
  sent_for_date   date
);

create unique index prompts_digest_per_day
  on prompts (person_id, sent_for_date)
  where sent_for_date is not null;

create table prompt_lines (
  id        bigint generated always as identity primary key,
  prompt_id bigint not null references prompts (id) on delete cascade,
  position  int    not null,
  chore_id  bigint not null references chores (id),
  unique (prompt_id, position)
);
```

### `last_done` is derived, not stored

The original sketch put `last_done_at` on the chore. It is computed instead:

```sql
select max(occurred_at) from events where chore_id = $1
```

This makes *"completion is just another event"* literally true rather than
aspirationally true. When a moisture sensor or the vacuum's own run history
writes an event later, the clock resets with **no additional code** — whereas a
stored column needs every future writer to remember to update it, and the one
that forgets produces a chore that is silently never due again.

`last_shown` is derived the same way, from `prompts` joined through
`prompt_lines`. The `chores` table therefore holds no mutable state except its
definition and its `active` flag.

The cost is two small joins per tick. At ten chores that is nothing, and it can
become a materialised column the day it isn't.

## The intent matcher

This is the first parsing in the system and the first place a capture could be
silently eaten. It is deliberately paranoid.

**A message matches an intent only if the entire trimmed message is one of these
forms.** Anything longer is a capture.

| Input | Intent |
|---|---|
| `.` prefix | capture, always, prefix stripped |
| `/` prefix | force command interpretation |
| `done`, `did it`, `✅` | complete, if exactly one line is outstanding |
| `done 2`, `2` | complete line 2 |
| `stop 2` | deactivate line 2 |
| `?` | list, on demand |
| `nvm`, `forget it` | undo the last chore definition |
| `every [n] <unit> [:] <name>` | define or update a chore |
| anything else | capture |

The example that governs the whole table is from the original kickoff:
`. done with the flux migration` is a note. So `done` matches only when it is the
whole message; `done with the flux migration` is a capture even without the dot.

**Outstanding** means: a line of the most recent prompt whose chore has had no
completion event since that prompt was sent. So a digest of three chores where
two have been done leaves one outstanding, and a bare `done` resolves to it
unambiguously.

`done` with several lines outstanding replies with the numbered list rather than
guessing. `done` with none does the same. `2` and `stop 2` address position 2 of
that same most-recent prompt, so one numbering serves every verb.

`nvm` undoes the most recent chore definition within ten minutes. It does **not**
drop a capture — that needs a `state` column on `items`, which belongs with the
clarification pass and is not in this phase.

Regex only. No LLM. The decision to write the parser against real captures rather
than imagined ones is unchanged for free-form text; chore definitions are exempt
because `every <n> <unit> <name>` is close to a closed grammar, which free-form
capture is not.

## Scheduling

In-process, ticking once a minute against `Europe/Amsterdam`.

Send when `now ≥ 08:00 local` **and** no digest row exists for today's local
date. The unique index does the work: a restart at 08:59 cannot produce a second
message, and a restart at 09:01 still sends the one it owed.

A day missed entirely — the process down from 07:00 Tuesday to 03:00 Wednesday —
is simply skipped. On Wednesday at 03:00 "today" is Wednesday and 08:00 has not
arrived, so nothing fires. **A stale digest is worse than a missing one**: a
message about yesterday's chores at three in the morning is noise, and the same
chores will appear in five hours anyway.

The original kickoff calls silent failure here the worst possible bug in the
system, so this gets its own tests rather than being assumed.

### Timezone data in a static binary

`gcr.io/distroless/static` carries no tzdata, and `time.LoadLocation` fails
without it — silently falling back to UTC would shift the digest by one or two
hours depending on the season, which is exactly the kind of failure nobody
notices for months.

The binary imports `_ "time/tzdata"` to embed the database (~450KB). The test
suite asserts that Amsterdam is two hours ahead of UTC in July and that the two
DST transition days are 23 and 25 hours long, and CI runs it inside the image —
the same guard `stringer` uses, for the same reason.

## Outbound

`Send()` was implemented, tested and left nil in phase 1. It is configured now:
`CAMPFIRE_BASE_URL` and `CAMPFIRE_BOT_KEY`, the latter a SOPS secret mirrored
into the `campfire` namespace, plus a NetworkPolicy egress rule squirrel →
campfire on port 80 and the matching ingress.

**This is the first time the pod holds a credential that can post as `@squirrel`
into any room in the account.** Phase 1's nicest property was that it held none;
that ends here, and it is the price of the bot speaking first. The key is
rotatable, scoped to nothing, and worth treating as the most sensitive thing in
the namespace.

## Testing

Beyond the usual, these carry the design:

| Test | Why it exists |
|---|---|
| A table of inputs that must **not** parse as chores or commands | The whole risk of this phase. Includes `. done with the flux migration`, `done with the flux migration`, `every day I think about leaving`, and every capture recorded so far |
| `nvm` within ten minutes removes the chore; after ten minutes does not | The correction has to be reliable or the over-match is unrecoverable |
| Two digests for one date insert one row | Idempotency, via the unique index rather than via application logic |
| Restart between 08:00 and the send | The failure the kickoff calls the worst possible one |
| A day fully missed sends nothing rather than sending late | A stale digest is worse than a missing one |
| July is UTC+2; the DST days are 23 and 25 hours | Catches tzdata missing from the image, which otherwise fails silently |
| An event with `source = 'sensor'` resets the clock with no chore code involved | Proves the derived-`last_done` claim rather than asserting it |
| A chore never completed is first due one interval after creation | The confirmation message promises this |
| Tolerance gaps the reappearance | Otherwise every overdue chore is a daily nag |

## Deferred

Unchanged from phase 1, minus what this phase takes: item triggers and the
general scheduler, `defer`, the nightly clarification pass, LLM parsing, Home
Assistant sensor evidence, context triggers, any web UI, and authentication
beyond the network boundary.

`events` is shaped for sensors and nothing writes them but `done`. That is the
seam phase 5 opens, and the derived `last_done` is what makes it a seam rather
than a rewrite.
