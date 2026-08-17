# Phase 3 — Actions, and a receipt that tells the truth

Builds on `2026-08-14-capture-path-design.md`, `2026-08-15-go-port-design.md`
and `2026-08-15-chores-design.md`. Those three still bind; this one adds a
second inbound event type and changes what the receipt means.

## The problem

Phase 2 made the bot speak first. Using it for a day surfaced two things.

**Correcting the bot costs recall.** The parser deliberately over-matches —
`every 2 weeks I have a headache` becomes a chore — and that is only tolerable
because the definition is announced and `nvm` undoes it. But `nvm` is a word you
have to remember, inside a ten-minute window you cannot see. The correction is
cheaper than the mistake only if you know the word.

**The receipt says "received", not "done".** A 🐿️ appears the moment the message
is on disk. Everything after that — the drain, the parse, the chore actually
being created — is invisible. The one question the bot cannot currently answer
is the one that matters: *did it do the thing?*

Campfire actions fix both, and they need a forked Campfire.

## Scope

In:

- Squirrel understands `type: "action"` webhooks.
- Prompts carry buttons; the newest prompt is the only live one.
- Tapping completes a chore; tapping again retracts it.
- The definition confirmation gains a correction button.
- The receipt becomes two-stage: 👀 stored, then ✅ handled alongside it.
- The forked Campfire built, published and deployed.

Out, deliberately:

- **Calendar chores.** `every tuesday and friday` is a different trigger model.
  A chore fires relative to when it was last done; a calendar chore fires
  regardless. Adding one would mean "overdue is a gradient since you last did
  it" stops being true of every chore. Ruled out, not deferred.
- **`!note` / `!chore` explicit commands**, and liberal name-first phrasing
  (`water the plants every 3 days`). Both are real and both are phase 4. They
  need no fork, which is exactly why they are not competing for room here.

## The fork

Campfire today runs `ghcr.io/basecamp/once-campfire`, tag `main`, pinned by
digest. Phase 3 depends on `feat/bot-message-actions` in
`ronaldlokers/once-campfire`, which adds four things upstream does not have:

| Capability | Needed for |
|---|---|
| Interactive actions on bot messages | Every button in this document |
| `PATCH` a bot's own message | Disabling the previous prompt's buttons |

The branch also lets bots delete their own boosts. This design does not use it —
see *The receipt, in two stages* — but it is why the branch can shrink later
without breaking anything here.

**The standing cost is rebasing.** Upstream ships security fixes — there are
four `security/*` branches in the remote today — and every one now lands through
a fork. That is the price of this phase and it does not go away when the phase
ends. If the branch is upstreamed the cost disappears, and until then the
payload shapes below are a moving target.

Rollback is the reason to keep this cheap: reverting the image digest in
homelab returns Campfire to upstream. Squirrel keeps working — actions stop
arriving, buttons stop rendering, and every typed path is untouched. **Phase 3
must degrade to phase 2 behaviour on an upstream Campfire**, not break.

## Two inbound event types

Message webhooks gain `type: "message"`. Today's Squirrel ignores the field, so
the fork can land before any Squirrel change.

Action webhooks are new:

```json
{
  "type": "action",
  "room": { "id": 1, "name": "Lobby" },
  "user": { "id": 42, "name": "Ada" },
  "message": { "id": 123 },
  "action": { "value": "done:2", "selected": true }
}
```

Three differences matter, all of them load-bearing:

**No `room.path`.** The message payload carries it; the action payload does not.
`fireBoost` builds the boost URL from that path today. Anything Squirrel sends in
response to a tap must use the configured `CAMPFIRE_BASE_URL` and
`CAMPFIRE_BOT_KEY` instead.

**The response body is ignored.** Campfire answers the click with `202` and
delivers the webhook from a background job. Phase 1's central trick — the reply
travelling inside the webhook's own HTTP response — does not exist here. Every
answer to a tap goes out through `Send`.

**No event id and no timestamp.** A retried background job is indistinguishable
from a second tap. See *Idempotency* below; the answer is to make the operation
idempotent rather than to dedupe the delivery.

An action is input. **It is spooled and acknowledged before anything else
happens**, exactly like a message, and applied after the drain. The request path
still never touches Postgres.

Routing is on `type`, with an absent `type` meaning `message` so that an
upstream Campfire keeps working.

## What the room looks like

The digest, `selection_mode: "multiple"`:

```
Due
 1. bin day — 2 days, usually 7
 2. water plants — 5 days, usually 4

[✓ bin day]  [✓ water plants]

Since yesterday
 · buy milk
 · the thing with the boiler
```

Tapping renders the button selected, and Campfire restores that state per user
on reload. Tapping again retracts the completion. **Neither posts a reply** — the
boost is the receipt. A message per tap would make the room unreadable, which is
the same argument that turned the 🐿️ into a boost in phase 2.

Numbering stays. `done 2` and a bare `2` keep working, and they are the route
that works from a notification without opening the app. Buttons are a second
path to the same intent, which is why `prompt_lines` survives unchanged.

The definition confirmation:

```
Got it: water the plants — every 3 days
[📝 make it a note]
```

There is no `[✓ right]`. Doing nothing already means right, and a confirm button
is a decision you would have to make every time you spoke to it.

`[📝 make it a note]` deactivates the chore — the same effect `nvm` has, without
the recall or the invisible window.

There is no `[✎ change interval]` either, though the first draft of this design
had one. It would have posted a follow-up asking for an interval, and the answer
would have had to bind back to the chore being edited — the only piece of state
that has to survive between two messages anywhere in this system. You already
change an interval by saying it again: `every 4 days: water the plants` upserts
by name. A button that adds conversational state to replicate something one
sentence already does is not worth its cost.

## Two kinds of surface

Prompts split in two, and conflating them is what the first draft of this
document got wrong.

**Numbered surfaces** — the digest (`kind = 'digest'`) and the query list
(`kind = 'query'`). Their lines are numbered, `done 2` means something against
them, and each line's button carries a position.

**Standalone surfaces** — the definition confirmation (`kind = 'define'`, one
line, one chore). Its button refers only to itself. It is never numbered and
`done 1` never resolves against it.

Only numbered surfaces participate in anything positional. Concretely,
**`ChoreAtPosition` must resolve against the person's most recent prompt whose
kind is numbered**, not the most recent prompt of any kind. Phase 2 already
carried a note that a query prompt sent after a digest retargets a stale number;
without this change, defining a chore would do the same thing, so that
`every 3 days: vacuum` immediately followed by `done 1` would resolve against
the confirmation rather than the morning's digest.

## Only the newest numbered prompt is live

When a new numbered prompt is sent, the previous numbered one is `PATCH`ed with
its actions set to `disabled: true`. In the fork, `disabled` also rejects forged
callbacks, so this is enforced rather than cosmetic.

This is what keeps undo bounded without any date arithmetic: there is nothing
old to un-tap. It also agrees with the rule above, so the button surface and the
typed surface share one definition of "current" instead of drifting apart.

A `?` query list is a numbered surface: it disables the previous one. Otherwise
a digest and a query list are both live and disagree about what `2` means.

**Definition confirmations are never disabled**, and need no bound. Their only
button deactivates a chore, which is idempotent and sane whenever it is tapped —
tapping `[📝 make it a note]` on last week's confirmation deactivates a chore you
apparently no longer want, which is exactly what it says.

If the `PATCH` fails, log it and carry on. The old buttons stay live, which
degrades to two live surfaces — bad, but strictly better than failing to send
the new digest. Never let disabling the past block speaking in the present.

**The twelve-action cap.** A message holds at most twelve actions. With more
than twelve due chores, the first twelve get buttons and every chore keeps its
number. The digest says nothing about the cut-off: a line about what the bot
could not render is noise, and the numbers already work.

## Idempotency, without a dedup key

The action payload has nothing unique in it, so there is no key that
distinguishes a retry from a genuine second tap. Rather than invent one, **treat
the action as a state assertion**:

- `selected: true` — ensure a live completion exists for this chore from this
  prompt. If one already does, do nothing.
- `selected: false` — retract this chore's most recent live completion. If there
  is none, do nothing.

Applying either twice reaches the same state as applying it once, so a retried
delivery is harmless and a lost one is recoverable by tapping again. The spool
entry still gets a unique external id (the derived key plus the receive
timestamp) so that two genuine taps are not collapsed into one by
`InsertItem`'s conflict clause.

## Retraction

Completion is an event, and undo must not delete it. The chore clock is
`max(events.occurred_at)`, so a deletion would restore the clock correctly and
destroy the record — and "completion is just an event" is precisely what lets a
sensor reset a chore with no new code.

`events` gains `retracted_at timestamptz`. The baseline subquery ignores
retracted rows. A retraction is then a thing that visibly happened rather than
an absence, and the sensor seam is untouched.

Retract the chore's most recent live completion for that person. That is
unambiguous because only the newest prompt has live buttons.

## Resolving a tap to a chore

The button's `value` carries a **position**, not a chore id: `done:2`.

A chore id in the value would be client-echoed, and phase 2 left a standing
finding that `RecordCompletion` never verifies the chore belongs to the person —
harmless only because every chore id reaches it through `ChoreAtPosition`, which
is scoped by `person_id`. Putting an id in a button value would make that the
only check standing between one person's tap and another's chore.

So a tap resolves entirely server-side: the prompt is found by its Campfire
message id **scoped by person**, then the position resolves through
`prompt_lines`. The client supplies a small integer and nothing else.

The confirmation's button resolves the same way — `undefine` against a prompt of
kind `define`, which has exactly one line. No id crosses the wire there either.

`prompts` gains `external_message_id text` — needed regardless, because
disabling the previous prompt requires knowing which Campfire message it was.
It is parsed from the `Location` header of the create response.

**Independently, close the ownership gap.** `RecordCompletion` gains a check
that the chore belongs to the person. It is one predicate, it was only ever
deferred because it was unreachable, and this phase is the moment "unreachable"
starts depending on a decision rather than on structure.

## The receipt, in two stages

👀 when the capture is on disk. ✅ when it has been handled.

This is not decoration: it makes the durability boundary visible. 👀 means the
spool write and its fsync completed — the thought survives a crash. ✅ means the
drain reached Postgres and the applier ran — the chore exists, the completion
landed, the note is queryable. The gap between them is exactly the window phase 1
was built around, and until now nothing in the room could see it.

**Both stay.** The ✅ is added; the 👀 is not removed. An earlier draft deleted
it, which needed the boost id from the create response and therefore somewhere
to persist it — and the boost is fired from the transport, which never touches
Postgres, after the capture has already been spooled and fsynced. There is no
honest place for that id short of the transport holding state in memory and
stranding it on restart.

Keeping both is also the better artefact: 👀✅ is a visible trail of two stages
that genuinely happened, rather than a single state that overwrites its own
history. It costs nothing and removes a migration, a fork dependency and a
failure mode.

Stage two is fired by the applier, which already runs after the drain and
already knows whether handling succeeded. It boosts the message named by the
item's external id, using the configured base URL and bot key — not `room.path`,
which the applier does not have.

Everything here is fail-open, unchanged from phase 1: a boost that cannot be
created never affects whether the capture was stored, and never changes the HTTP
response. A missing receipt is cosmetic; a missing capture is the failure the
system exists to prevent.

## Schema

Two migrations, additive:

```sql
-- 0005
alter table events add column retracted_at timestamptz;

-- 0006
alter table prompts add column external_message_id text;
```

The baseline subquery in `chores.go` gains `and e.retracted_at is null`.

## What must be tested

| Case | Why |
|---|---|
| An action webhook is spooled and acknowledged before Postgres is touched | The invariant the whole project rests on, now on a second entry point |
| A tap completes; the same tap delivered twice completes once | Retries are indistinguishable from taps, so the operation carries the guarantee |
| Un-tap retracts, and the chore is due again | The undo is the thing that makes over-matching survivable |
| A retracted event does not move the clock | The baseline query is where retraction actually takes effect |
| Sending a numbered prompt disables the previous numbered one's actions | The bound on undo is this and nothing else |
| A definition confirmation does not disable the digest, and `done 1` after defining a chore still resolves against the digest | Two surfaces, one of them positional; conflating them was this design's first bug |
| A failed disable still sends the new prompt | Never let the past block the present |
| A tap resolves through a person-scoped prompt lookup | One person's tap must not reach another's chore |
| More than twelve due chores | The cap is silent by design; the numbers must still cover everything |
| An action payload with no `room.path` still answers | The boost path differs between the two inbound types |
| Everything above with an upstream Campfire image | Phase 3 must degrade to phase 2, not break |

## Deployment

1. Build and publish `feat/bot-message-actions` as an image.
2. Repoint `apps/production/campfire` at it, pinned by digest as today.
3. Ship Squirrel with action support; it is inert until buttons exist.
4. Runbook: how to roll Campfire back to upstream, and what stops working when
   you do.

The order matters. Campfire first, because Squirrel sending actions to an
upstream Campfire is the one combination that fails loudly rather than
degrading.
