# A fixed point you can put things on

**Status:** design, 24 August 2026. Decided in conversation with the owner on
the same day, immediately after push was proven working end to end.

## What is being asked for

Tapping a leave-by notification lands on the front door. The owner wants it to
land on the appointment itself, showing what to take and anything else that
belongs to it — and wants to be able to look at that days ahead, not only in the
window where leaving matters.

## What is already true, and was nearly missed

**The card already says what to take.** `pickMoment` folds `Bring` into the
offer's `Because`, so the screen reads *"leave about 14:10 · take keys, wallet"*.
An early reading of this work concluded the screen dropped it — a grep for
`Bring` in `internal/web` comes back empty, because the value arrives from the
core already inside the sentence. Recorded because the same mistake was made
twice in one day: an absence in one package is not an absence in the product.

So this design is not a fix. It is a feature, and it costs two recorded
decisions.

## The two rules this overturns

**`PRODUCT.md:281` — no list screen.**

> there is no list screen, because a browsable set of your appointments is a
> calendar and a calendar is a thing you are behind on. A moment is shown only
> inside the window where leaving matters, and afterwards it is simply over.

The owner overturned this deliberately on 24 August, having been shown the rule
and asked directly. What the rule protected against is real and is kept by the
guard rails below: the list holds **only what is still coming**. Nothing past,
nothing done, no count, and no way to be behind on it — because a thing you have
not reached yet is not a thing you are late for.

**`DESIGN.md:1398` — a notification goes to the front door.**

> a link to something that has since been done is worse than a page that says
> what is true now

The reasoning survives the change rather than being discarded. That rule feared
landing on something already finished; a fixed point inside its leave-by window
is the one thing in the product that cannot be stale, because the notification
and the window are the same fact.

## The shape: notes point at the appointment

Three shapes were considered.

| | where attached things live | why not |
| --- | --- | --- |
| new columns on `moments` | on the moment | a second place a thought can live, and `!find` cannot reach it |
| a moment becomes an item | in the pile | correct, and far too large — touches every screen that reads `kind` |
| **notes point at a moment** | **in the pile** | **chosen** |

A note keeps everything it already has — capture, photograph, editing, search,
undo — and gains a pointer. The appointment gains a view over the notes pointing
at it. Nothing about a note changes except where it can be seen.

This is the closest thing in the product to a folder, and folders are on the
refused list. The difference that makes it acceptable: **a folder is where you
file something instead of the pile.** This is a pointer, and the note is still a
row in `items` that `!find` reaches, that carries its own photograph, and that
returns to the pile the moment the appointment is over.

## Data

One column and one migration.

```sql
alter table items add column moment_id bigint references moments(id) on delete set null;
create index items_moment_id_idx on items (moment_id) where moment_id is not null;
```

`on delete set null` rather than cascade: deleting an appointment must never
delete the owner's words. The note returns to the pile instead, which is the
same thing that happens when the appointment is over.

**No new state value.** The pointer is the disposition. `items.state` is
untouched, and the pile's query gains a clause about `moment_id` — stated in
full under *What happens when it is over*, because it is wider than "is null"
and the two must not be written down twice. A note that has somewhere to be is
not waiting to be decided about; a note whose appointment has passed is waiting
again, automatically, with no transition to write.

That is the whole reason this shape was chosen over an eighth state: there is
nothing to migrate, nothing for seven screens that read `state` to learn, and
the reversal is `moment_id = null` rather than a remembered previous value.

## Screens

### `/at` — what is coming

Reached from **a fourth door on home**, and this is the part that changes an
existing screen rather than adding one.

- Upcoming fixed points, soonest first. `starts_at > now and done_at is null`.
- Each row: the label, when it starts, when to leave. The Door's grammar at one
  line high, the way a chore in search results already borrows it.
- **Never a count**, never a total, and no "late" — nothing here has been missed,
  because everything here is still ahead.
- Empty is the empty-state treatment, and says nothing encouraging about making
  plans.

### `/at/{id}` — one fixed point

- Title: the label, under The One Title Rule.
- When it starts, when to leave, and what to take — the last as **its own line**
  rather than a clause after a middot, which is the presentation half of what
  was asked for.
- The notes pointing at it, in the Result role, each with its photograph if it
  has one, and each with a way back to the pile.
- **The slot**, exactly as home has it. Anything typed becomes an ordinary note
  in the pile that points at this moment. The camera comes with it, so
  photographing a letter onto an appointment works on the first day.
- `LEAVING` **only inside the window.** Outside it there is nothing to press,
  because the appointment is not yet something you can act on, and a button that
  closes a thing three days early is a button that will be pressed by accident.

### Home's fourth door

`DESIGN.md` states the doors are *"three, side by side at every width"* and that
their equality is the screen's one statement. This adds a fourth, and the width
is the cost: at 390px the frame is 364px, so three cells with a 9px gap measure
about 115px and four measure about **84px** — where *"what you decided"* already
wraps to two lines at 115px.

**So the phone breaks the row rather than the cells: two by two below 620px**,
holding the cells near 178px, with four equals side by side above it. That
trades *"side by side at every width"* for rearranging instead of cramping, and
it is a deliberate amendment rather than a drift.

The door's name and art are the owner's. **`what is coming` is the working
name** so the work is not blocked on it; it is provisional and expected to
change. The name may never carry a count, which is the one thing a door about
appointments is most likely to reach for.

The art must obey the Door Art guard rails, and the one that matters here:
**never depict a count, a progress state, a tick or a completion** — which for a
door about appointments means no calendar page with a date ringed, because that
is lateness drawn as a picture.

## The notification

`sw.js` navigates to `/` today. It navigates to `/at/{id}` instead, which means
the payload's `url` field stops being dead weight — it is written, tagged, and
currently read by nothing.

Both carve-outs in the existing click handler are kept: a window already at the
destination is left alone rather than reloaded, and `navigate()` is only
attempted on a client this worker controls, falling back to a raise.

## What happens when it is over

**A note attached to a fixed point that is over goes back to the pile.** It was
set aside for an occasion, the occasion happened, and it is a thought again that
you have not decided about.

Implemented as a read rule rather than a write: the pile shows notes where
`moment_id is null` **or the moment it points at is done or past**. Nothing runs
on a schedule, nothing needs a migration to catch up, and a moment deleted
outright leaves its notes in the pile by the same rule.

Considered and rejected: archiving them with the appointment. A photograph of a
letter outlives the appointment it was taken for often enough that quietly
filing it away would lose things — and losing things is the one failure this
product exists to prevent.

## Chat

The screen may do what chat cannot; chat may not do what the screen cannot, and
both must read the same. So:

- **No chat command to attach.** Pointing a note at an appointment is a gesture
  on a surface where both are in front of you, and the parity rule permits the
  screen to have gestures.
- **`!notes` and the pile agree**, because they run the same query. A note
  pointing at a live appointment is in neither.
- **`!find` still reaches it**, and a search result for an attached note says
  which appointment it is on — the way a result already says what state it is in.
- `!at` and `!bring` are unchanged.

## Testing

Beyond the ordinary unit and integration coverage:

- **The pile excludes attached notes, and stops excluding them when the moment
  is past.** Both directions, against a real database, because the second is a
  read rule with no write to observe.
- **`!find` still returns an attached note.** The whole justification for this
  shape over new columns is that nothing becomes unfindable; a test is what makes
  that true rather than intended.
- **A deleted appointment leaves its notes in the pile**, exercising
  `on delete set null` rather than trusting the constraint.
- **`LEAVING` is absent outside the window**, present inside it.
- **The four doors render as two rows of two below 620px** and four across above,
  in the appearance snapshot.
- **Every element on both new screens clears 4.5:1**, by the contrast walk that
  already runs on eleven screens — these make thirteen.

## What this deliberately does not do

- **No recurrence.** That is a chore, and the record is explicit.
- **No past view.** Over is over; the list holds only what is coming.
- **No second attachment surface.** A photograph still belongs to a note. An
  appointment has photographs only because notes do.
- **No calendar import, no sync, no deadlines on tasks.** All on the refused
  list and all untouched by this.
- **No count anywhere**, including the door's sub-line and the list's title.

## Records to amend

Both before merging, and the design gate will require the second anyway:

- `PRODUCT.md` — the no-list-screen rule, amended with the guard rails that
  replace it and the date it was overturned.
- `DESIGN.md` — the notification's destination, the fourth door, and the two-by-
  two phone layout.
- `docs/roadmap.md` — moved out of Open into Shipped when it lands.
