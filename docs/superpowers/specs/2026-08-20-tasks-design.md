# Tasks: what you decided

**Status:** design, approved 2026-08-20. Implementation plan to follow.

## The sentence this adds

Squirrel holds two kinds of thing today, and each has one sentence:

- **the pile** — what you said
- **the chores** — what comes back

This adds a third:

- **the tasks** — what you decided

A note is a thought you had. A chore is a thing that returns on its own. A task
is a thing you decided to do, once, and it goes away when you have done it.

## Why this is not just a note marked done

It nearly is, and that is worth saying plainly rather than defending.

The difference is which list a thing sits in while it waits. A note in the pile
is untriaged — you have not decided anything about it yet, and the pile's whole
promise is that deciding is optional and stopping partway is normal. A task has
already been decided on. Keeping the two in one list would mean the pile
contained both the things you have not thought about and the things you have,
and triage would stop meaning anything.

So promoting a note to a task **removes it from the pile**. That is the point of
the feature, and it is also the honest test of whether it earned its place: if a
task did not leave the pile, this would be a label rather than a kind.

## The risk, named

A list of things to do is structurally what every product in this category
sells, and it is the place a counter most wants to appear. PRODUCT.md's
positioning is built on the opposite — nothing accrues that can be destroyed.

The tasks screen therefore refuses, explicitly:

- **No count.** Not of what is open, not of what was archived, not "3 left".
  A capped list may say *that* there is more, never how much.
- **No deadline.** A task is a thing to do, not a thing to be late for. Chores
  deliberately carry a rhythm rather than a deadline for exactly this reason,
  and a task with a date is a task you can fail.
- **Nothing red, no urgency, no overdue**, because none of those can exist
  without a deadline to be late against.
- **No priority and no ordering that implies one.** Newest first, like the
  pile: a task decided this morning is the one you still remember deciding.
- **No nudge.** Tasks are silent until you look. The nudge stays the chores'
  alone, so nothing new competes for the attention this product is careful
  with.

## The model: a kind, not a state

Items gain one column: `kind`, either `note` or `task`, orthogonal to the four
states they already have (`open`, `done`, `kept`, `dropped`).

| What | Query |
| --- | --- |
| the pile | `kind = note and state = open` |
| the tasks | `kind = task and state = open` |
| the archive | `kind = task and state = done` |
| the shelf | `kind = note and state = kept` |

A fifth *state* was considered and rejected: a task that is done would then be
indistinguishable from a note that is done, and the archive could not exist. A
kind is also the smaller change — search, undo, `!fix`, the transitions and the
capture path all work untouched, because none of them cares what kind a row is.

Promotion is `kind = task`. Demotion — "actually that was just a thought" — is
`kind = note`, and it is the same write backwards, like every other transition
here.

## Where a task comes from

**Promoted from a note.** `!task <n>` in chat, and a fourth action on the deck's
card beside *done*, *keep* and *drop*. The same gesture as `!chore <n>`, which
is the existing third verb; this is the fourth.

**Made directly.** `!task ring the vet` — words rather than a number — makes one
outright, the same number-or-name shape `!did` and `!snooze` already use. On the
screen, the tasks screen carries a field of its own, the slot's grammar rather
than the new-chore form's, because a task is a sentence rather than a
configuration.

## The tasks screen

`/tasks`, the third door.

Each task is a card in the result-card grammar — the same cream stock and the
same 3px outline, with no page tab, because a tab says what a note *ended up
as* and a task has no outcome until it has one.

Two actions, and only two:

- **DID IT** — archives it. The one thing a task is for.
- **NOT A TASK** — back to the pile as a note. Deciding was the mistake, and
  undoing a decision must not require finishing it.

Dropping is deliberately absent: a task you no longer want is a note you no
longer want, and it gets there by ceasing to be a task first. One way to say a
thing.

**The archive** is at the foot, as a link rather than a section — `/tasks/done`,
newest first, saying what you did and never how many. From there a task can come
back, because every transition here reverses.

**The empty state** is absence. "Nothing decided" and the mark; no
encouragement, no "add your first task", because an empty task list is a normal
state and not a failure to set up.

## The chat

| Command | What it does |
| --- | --- |
| `!tasks` | what is decided and not done, newest first |
| `!task <n>` | the note on line n becomes a task |
| `!task <words>` | a task, made outright |
| `done <n>` | archives the task on line n — the existing verb, unchanged |
| `!untask <n>` | back to the pile as a note |

`done <n>` already means "clear line n" and needs no new behaviour: archiving a
task is the same transition it performs on a note.

## The third door

DESIGN.md says the home screen has **only ever two** doors, and their equality
is the screen's one statement. This amends that rule to three, deliberately and
in writing, because the sentence the home screen tells is now three things
rather than two and a door is how each is named.

What is preserved: the doors remain equals — one grid, identical cells,
identical stock and depth, no ordering that implies importance. What changes is
that three cells of one row are narrower on a phone, so the door art steps down
and the names take the phone's Note-floor size rather than its step-up.

## The icon, and the second exception

The owner supplied a clipboard with two ticked rows, an unticked third, a grey
clip and a pen. It is used as supplied, and this records why that is a decision
rather than an oversight — the same treatment the chores door art got.

It breaks two documented rules:

- **It depicts a count and a completion.** Two of three ticked is a progress
  meter drawn as a picture, on the product whose hardest rule is never a count
  in any form. It also spends the `done` green as ornament.
- **It carries grey**, which The No Neutral Rule forbids.

DESIGN.md's Door Art entry, written when the chores art was accepted on the
same grounds, ends: *"If a third door is ever added, this is not the precedent
to follow."* A third door has been added and the precedent has been followed.
That sentence is now wrong and is replaced by what is actually true: **door art
is the owner's, and the guard rails govern anything drawn by anyone else.** The
cost is unchanged and is worth restating — this and the chores door are the only
places in the system where grey appears or a completion is depicted, and neither
reports state. Nothing on any screen counts.

The supplied file has a white background baked in, which cannot sit on the
purple. It is flood-filled from the corners rather than keyed on white, because
the clipboard's own paper is nearly white and keying would punch holes in it.
That is a change of container, not of drawing.

## What this does not do

- No sub-tasks, no projects, no tags. Those are the folder question, which
  PRODUCT.md leaves explicitly undecided, and a task list is exactly where they
  would arrive by accident.
- No "someday" or "waiting on". A task is decided; anything else is a note.
- No recurring tasks. That is a chore, and it already exists.

## Testing

- The pile excludes tasks, and the tasks screen excludes notes. One test each,
  because that separation is the feature.
- Promoting and demoting round-trip: a note made a task and made a note again
  is the same row in the same place.
- Never a count, on the tasks screen and on the archive, in the same shape the
  deck's own test uses.
- The archive holds only tasks that are done — never a done note.
- `done <n>` against a task line archives it, through the same transition a
  note takes.
- The three doors render identically to each other in every state, which is the
  equality the amended rule keeps.

## Rollout

One migration, one release. The column defaults to `note`, so every existing
row is what it already was and nothing needs backfilling.
