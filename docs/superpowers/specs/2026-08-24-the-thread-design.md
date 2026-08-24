# Buddy is the app

**Status:** design, 24 August 2026. Decided in conversation with the owner over
the course of the same day, starting from *"how do I add an appointment from
web??"* and ending at a single-page conversation with the four doors pinned
above it.

## What is being asked for

Buddy is a hidden chat box behind an acorn. The owner wants Buddy to be the main
character: the app opens as a conversation, Buddy offers the four doors, and
what a door holds arrives as a message rather than as a page. Buddy asks for
what it needs the way a customer-service chat does — one question at a time,
with buttons — and a typed sentence still skips all of it.

The route to that direction is recorded because two of the turns were wrong:
a link to Buddy was rejected (*"that still does not make buddy the main
character"*), and a persistent sidebar was rejected in favour of the whole app
becoming the conversation. A third correction matters more than either —
*"I think you think too much about chat. It needs to be smarter than that."*
The guided flow is deterministic. It is not the model.

## What the owner decided

All on 24 August 2026, in answer to four objections raised against the mockup:

| | decision |
| --- | --- |
| the doors scrolled away | **they are always visible** |
| Buddy's line carried two counts | **counts are acceptable** |
| a transcript accrues history | **history is never rewritten; conversations work that way** |
| a day needs picking | **the date/time picker is in** |
| intervals were four fixed chips | **1, 2, 3, 4 × days/weeks/months** |
| the transcript's lifetime | **forever, in Postgres** |
| the thirteen existing screens | **absorbed into the thread; the standalone pages are deleted** |

## The two rules this retires

**Principle 2 — *nothing accrues that can be destroyed*.** Retired by the owner
on 24 August 2026, deliberately, having been shown the rule and told what it
was holding up. It banned every counter, streak and percentage on every
surface, and `PRODUCT.md`'s Positioning named it as the one thing a competitor
could not copy without abandoning its own metrics.

Counts are now permitted everywhere: in what Buddy says, on the doors, in the
thread. Principle 5's opening on 20 August had already permitted them in
*speech*; this permits them on *surfaces*, which is the half that was left.

What the rule protected is real and is now unprotected, and that is worth
writing down in the form it will actually arrive in: a number beside a door,
with an implied target of zero, that grows while nobody is looking and that a
bad week makes worse. The counter-argument the owner is acting on is that not
knowing how much is waiting is its own weight. If the doors ever start reading
as a scoreboard, this decision is what to reverse, and it reverses cleanly —
the numbers are computed at render time and stored nowhere.

**Progressive enhancement.** `DESIGN.md` requires that every upgrade `pile.js`
makes works with scripting off — the chore picker's `<details>`, the coach
sheet over a real `/buddy` route. Retired the same day: the thread requires
JavaScript, and without it the page is the lid, the rail and nothing else.

What is *not* retired is the single rendering path. Handlers return HTML from
the same Go templates whether the browser asked for a page or a fragment.
There is no JSON API and no client-side templating, so there is no second
description of what a chore card looks like that can drift from the first.

## The shape

**One URL. One page. JavaScript required.**

A press posts, the handler does the write, and the response is the HTML for the
turns that changed. The client appends it to the thread and scrolls the live
edge into view. No full-page paint, no scroll jump, no flash — which is the
whole reason for the JavaScript.

Persistence is what makes a reload cheap rather than what makes the app work:
the thread comes back because it is in Postgres, not because anything was held
in a session.

One new module under `internal/web/static/`, and `pile.js`'s existing upgrades
fold into it as the screens they upgraded are deleted.

### The live edge

**Only the newest Buddy turn carries controls.** Older turns render as what was
said and what was shown, with the buttons gone.

This is what *never change history* means in practice. A chore card from this
morning cannot re-render live, because a live card rewrites what was said; and
it cannot keep its buttons, because pressing DID IT on a card from a
conversation three days old acts on a state nobody is looking at. Scrollback is
a record. The bottom of the page is the app.

### The doors

A sticky rail directly under the lid, above the thread, at every width. Four
equal pills; the current one takes orange fill.

- Above 620px: art, name and number side by side in a pill.
- At 620px and below: art over name, number in the corner, still four across.
  At 390px the frame gives each door about 81px, which holds a 24px drawing,
  an 11px label and two digits.

**What each door counts** — *what is waiting for you*, so an empty door reads
as finished rather than as absent:

| door | number |
| --- | --- |
| the pile | notes not yet decided about |
| the tasks | tasks not done |
| the chores | chores due right now |
| the agenda | fixed points still ahead today |

Computed at render time, stored nowhere, and never held in the `turns` table —
a number frozen into scrollback would be a count that lies, which is worse than
a count that accrues. Zero renders as no number rather than as `0`.

This replaces `DESIGN.md:760`'s fourth item and `DESIGN.md:748`'s fixed order
for home, both of which describe a screen that stops existing. The equality of
the doors survives verbatim: one grid, four cells, same stock, same depth,
identical in every state, and nothing about them depends on what the pile holds.

### The dock

The slot, pinned to the bottom, on every view of the app because there is only
one view. `anything at all` — unchanged placeholder, unchanged behaviour,
unchanged rule that the slot is a slot and not a command line.

## Data

One table, one migration.

```sql
create table if not exists turns (
    id        bigserial   primary key,
    person_id bigint      not null references people (id) on delete cascade,
    -- 'buddy' or 'you'. Two values, as text, for the same reason kind is text
    -- on coach_answers: a third speaker should not also be a migration.
    who       text        not null,
    -- What was said, as it was said. Never a reference, never a join.
    words     text        not null,
    -- What was shown beneath the words: the cards, the chips, the picker, as
    -- rendered. Null for a turn that was only a sentence.
    shown     jsonb,
    said_at   timestamptz not null default now()
);

create index if not exists turns_person_said
    on turns (person_id, said_at desc, id desc);
```

**`words` and `shown` are text, not pointers.** A turn that stored a chore id
and re-read it would show today's chore inside yesterday's sentence, which is
the definition of rewriting history. The cost is duplication — the same chore's
name in the chores table and in forty turns — and it is accepted, because the
turn is a record of an utterance rather than a view of a row.

**Kept indefinitely**, on the same reasoning that keeps `coach_answers` and the
check-in history. The page does not render all of it: the thread renders the
most recent turns and pages backwards the way the pile already does, with a
control that says *there is more* and never how much more.

## Buddy has two voices

This is the part the owner corrected, and it is the centre of the design.

**The deterministic voice** answers a door, a button, and a picker. It is a Go
function that returns a sentence and a set of cards. It costs nothing, cannot
be unavailable, cannot be rate-limited, and says the same thing every time. It
handles: greeting, the doors, every list, every guided flow, and every
confirmation.

**The model** is reached only when a typed sentence is neither a capture nor
anything `ParseIntent` recognises, or when the owner asks Buddy something. This
is `opts.Ask` exactly as it exists today, with `coach_answers` recording every
call and the budget unchanged.

A person cannot tell which one is speaking, and does not need to. What matters
is that the app is fully usable with no key configured — which is already true
of the four chips today, and must stay true of every guided flow here.

## Asking

Every question Buddy asks renders as one card of choices under the bubble.

### How often — number × unit

Two chip rows, `every [1 2 3 4 6]` and `of these [days weeks months]`, with the
composed phrase written back underneath: *every 2 weeks*.

**This needs no core work.** `ParseEveryAsking` in `internal/squirrel/intent.go`
already accepts any count against any unit, and `unitDurations` already holds
day, week, fortnight, month, quarter and year. The picker composes the same
phrase the existing chips post — `every 2 weeks` — so the chore path underneath
is untouched, and the fast lane and the picker cannot disagree because they
produce the same string.

There is no `…` chip and no keypad. Six numbers cover what anyone reaches for,
and *every 9 weeks* is a sentence you type. The unit row shows three; year is
available through the sentence at no cost.

### Which day — the picker

`today / tomorrow / a day`, then a month grid, then time chips with
`another time` as the escape, then the composed line: *Thursday 27 August,
14:30*.

Seven columns at 390px gives day cells 43px wide against the 44px floor. Height
is 44. The alternatives — cropping to weekdays, or scrolling the month
sideways — are both worse than the missing pixel, and the pixel is accepted
deliberately rather than discovered later.

**The picker produces a sentence too.** `at 14:30 dentist` is what gets posted,
so `ParseMoment` remains the single place a time is understood. This also means
the picker inherits the ambient-timezone defect recorded in issue #148 rather
than adding a second one.

### The fast lane

`at 14:30 dentist, 20 minutes away` in the dock skips every picker. The pickers
exist for when you do not want to phrase it — *"a quick way to offload my
brain, but also a more structured way of inputting data when I have time"*.

## Headings, and what absorbing the screens costs

`DESIGN.md:632` — The One Title Rule — exists because ten of thirteen templates
had no `<h1>`, so heading navigation did not work in a product built for
someone who may well navigate by headings. Collapsing thirteen screens into one
page would undo that.

**So a turn that opens a place carries the place's name as an `<h2>`.** Walking
the headings walks the conversation, which is a better map of this app than a
list of pages was. The thread itself has no `<h1>`, on home's own exemption:
nobody arrives at the place they started wondering where they are.

The live edge is an `aria-live="polite"` region, so a turn that appears after a
press is announced.

## Staging

"Absorbed" is the largest change in this project's life: 22 templates and about
40 routes. It lands in four shippable phases, each of which leaves the app
working.

1. **The thread exists.** `/` becomes the conversation: pinned rail, greeting,
   dock, the `turns` table, paging backwards. The doors still link to the
   existing screens. Nothing is deleted. Shippable, and the riskiest piece —
   persistence and PRG — is proven before anything is torn out.
2. **The chores and the tasks are absorbed**, with the number × unit picker
   arriving alongside them because it is a chore control. `/chores`, `/tasks`,
   `/tasks/done` and their templates are deleted.
3. **The pile and the agenda are absorbed**, with the day/time picker. `/pile`,
   `/at` and the deck are deleted. `/at/{id}` stops being a page and becomes a
   redirect that opens the thread with that fixed point at the live edge — it
   must not 404 while a notification sent yesterday is still on a lock screen,
   and phase 4 is what finally retires the URL.
4. **The notification's destination.** `sw.js` currently opens `/at/{id}`. It
   opens `/` with the fixed point as the live edge. This is last because push
   is the one path with no way to test a regression except on a phone, and
   because #147 — no way back once permission is refused — should be fixed in
   the same pass.

`/buddy` and the acorn disappear in phase 1: the sheet is the app now.

## Chat parity

`PRODUCT.md`'s rule 4 — two views, one pile — needs restating rather than
changing, because the screen has become a chat and there are now two chats.

- **The room is still the room.** Campfire is where Squirrel speaks to you when
  you are not looking at it. The thread is where you speak to Squirrel.
- **The thread does not mirror the room, and the room does not mirror the
  thread.** Two transcripts of the same conversation that can diverge is worse
  than two surfaces that agree about the pile, which is what the rule actually
  asks for.
- **`!` commands are unchanged**, and every guided flow composes a sentence the
  same parser reads, so there is nothing the screen can express that chat
  cannot.

## Testing

Beyond ordinary coverage, and each of these because the equivalent claim has
been asserted-but-not-proven in this project before:

- **A turn survives a reload**, against a real database. The whole justification
  for persisting is that history is real; a test that only checks the row was
  written proves nothing about the page.
- **An old turn has no buttons.** Assert on the second-newest Buddy turn, after
  a third turn is added — a test that checks the newest turn is interactive
  passes with the live-edge rule deleted.
- **A card in scrollback does not change when its row changes.** Write a turn
  showing a chore, rename the chore, re-render, assert the old turn still says
  the old name. This fails if `shown` ever becomes a join.
- **A fragment and a full page render the same card.** Request the chores turn
  both ways and assert the markup matches. This is the only thing holding the
  single rendering path in place, and it fails the moment somebody adds a
  client-side template.
- **`every 3 months` composed by the picker and typed as a sentence produce the
  same interval**, asserted on the duration, not on the string.
- **A door with nothing waiting shows no number, not a zero.** Both directions,
  because the empty case is the one nobody renders by hand.
- **Four doors render at 390px without the label wrapping**, in the appearance
  snapshot.
- **Every element clears 4.5:1** by the existing contrast walk, which stops
  walking thirteen screens and starts walking one page in several states.

Every test written for this must be proved by mutation: revert the behaviour,
watch the assertion fail, and report the failure text. A compile error is not
proof.

## What this deliberately does not do

- **No model in the guided flow.** A flow that needs a key to work is a flow
  that breaks when the budget runs out.
- **No streaming, no typing indicator, no delay before Buddy answers.** Buddy is
  a program answering a form post. Pretending otherwise is theatre and costs
  latency.
- **No editing a turn, no deleting a turn.** History is not rewritten. A mistake
  is answered by the next turn, which is how conversations work.
- **No second transcript in Campfire.**
- **No recurrence in the day picker.** That is a chore, and the record is
  explicit.

## Records to amend

- `PRODUCT.md` — **Principle 2 struck through and dated**, the way the
  no-list-screen rule was, with what it protected and what reverses it; the
  Positioning paragraph that rests on it rewritten rather than quietly left
  standing; rule 4 restated for two chats.
- `DESIGN.md` — the progressive-enhancement requirement struck through and
  dated, and the door art guard rail that forbids depicting a count relaxed to
  cover only the drawing, since the pill now carries a number beside it.
- `DESIGN.md` — home's fixed order and the four-doors item replaced by the rail
  and the thread; The One Title Rule amended for `<h2>` per place; the live edge
  added as a named rule.
- `docs/roadmap.md` — moved to Shipped as each phase lands.
