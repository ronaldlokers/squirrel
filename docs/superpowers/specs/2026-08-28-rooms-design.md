# Rooms

**Date:** 28 August 2026
**Status:** layout approved; data model and Buddy scoping unresolved

The thread replaced seven screens on 25 August, and everything became one
conversation. This is the argument for partitioning it again — not back into
screens, but into rooms that each keep their own conversation.

The owner's words for it were "a chat platform", and the first mockup took that
literally and failed: bubbles plus cards plus a dock, committing to neither. A
second attempt replaced the whole visual world with a railway departure board;
it was rejected for the right reason — the world was never the problem. What
follows keeps the shipping visual system unchanged and restructures only the
layout.

Four rounds of mockups sit behind it. What follows is only what survived.

---

## 1. It was never a chat

The forks answered in this order, and every one of them chose against chat:

- A room is **both** a place that keeps its own conversation **and** a scope
  that narrows Buddy. The narrowing is what makes a room worth having; the
  history is what makes it a place.
- **The room decides the filing.** Typing into the chores makes a chore. No
  model call classifies it, which is the saving.
- **Buddy's own room stores nothing.** It is where you talk; the other rooms
  are where things live.
- Four of the seven rooms are filing surfaces, two are shelves, and one is a
  conversation. See section 3 — the split is not even, and pretending it is
  would design a dock for rooms that have nothing to file.

A product where typing files a thing into the room you are standing in is a
cabinet, not a chat. What was wanted was **partitioning**; "chat platform" was
the nearest available word, borrowed from the reference product for rooms. The
distinction matters because it decides what to commit to: the rooms are the
feature, and the chat vocabulary is not owed anything.

## 2. The layout, and only the layout

**The visual system does not change.** The field and its radial, cream card
stock with the three-pixel outline and the `0 5px 0 0` sticker offset, Buddy
unbubbled in paper ink with his face in the gutter, your words in the cream
bubble, every button colour, the lid, the brim, the wordmark, the dock. All of
it ships today and all of it stays.

What changes:

**A rail that never leaves.** On a desktop the room list is furniture in a
left column, not a screen you navigate to. This is the single structural
difference between this design and the version that read as a page pretending
to be a chat: moving between rooms costs no screen and no gesture. On a phone
the rail is a screen, and the lid gains a way back.

**One thread becomes seven.** Buddy, the pile, the chores, the agenda, the
tasks, what you set aside, the things you kept.

**The rooms keep the product's own names.** *the chores*, not *#chores*.
Hash-names carry Slack's register and would bring its voice with them.

## 3. The button carries the consequence

The room decides the filing with no confirmation step — the owner chose the
symmetric rule over an asymmetric one, on the grounds that a press on every
capture is a worse tax than an occasional misfile.

That puts the whole weight of "what will typing do here" on one control, and a
grey placeholder cannot carry it: it is invisible by the third day. So **the
submit button names the consequence** rather than the act.

| room | placeholder | button | where the words go |
| --- | --- | --- | --- |
| the pile | what is it | Put it in the pile | `/capture` — the spool, then the drain |
| the chores | what comes back? | Make a chore | the words, then the how-often picker |
| the tasks | what did you decide? | Make a task | `/tasks/new` |
| the agenda | what is happening? | Put it in the agenda | the words, then the day picker |
| the things you kept | what is it | Put it in the pile | `/capture` |
| what you set aside | what is it | Put it in the pile | `/capture` |
| Buddy | what's going on? | Tell it | `/capture`, and Buddy answers |

You read what is about to happen at the moment you commit to it. That is the
confirmation, without a press.

Three of these are already two-step and stay two-step. A chore needs a rhythm
and an appointment needs a day, and both pickers exist — the room's dock
replaces the *first* step of `/chores/ask` and `/at/new`, which is a box asking
for exactly the words you have already typed.

**The two shelves file into the pile, and say so.** Nothing is *kept* or *set
aside* by being typed — both states are reached by deciding about a note that
already exists. A shelf with no dock would be the one screen where the thumb
has nowhere to go, and a shelf whose dock lies about its destination is worse
than either. So the room still decides; a shelf's decision is the pile.

**Buddy's room files into the pile too**, and this is not an exception to
"Buddy's room stores nothing". No *turn* of Buddy's is kept as a turn anywhere
else; what a capture there produces is a pile note, exactly as the single dock
produces one today. That is the behaviour shipping now, unchanged.

## 4. The rail carries counts

Each room shows what is waiting, as the doors do today. A room with nothing
waiting has no pill.

**This was decided against the evidence, deliberately.** A mockup of a bad week
— five rooms carrying numbers, thirty-four things waiting — reads as a
scoreboard, because five pills in a column invite a total the eye computes
whether or not the product prints one. That is the shape PRODUCT.md named when
it retired the no-count rule, and this is that shape at seven rooms instead of
four.

The owner kept them anyway, and the decision is cheap to reverse: PRODUCT.md
already records the trigger ("if the doors start reading as a scoreboard") and
the cost (one call, since the numbers are computed at render time and stored
nowhere). Reversing gives up nothing else, because the last line of each room —
*the letter from the council* — already says a room has something and what kind
of something.

**Nothing here counts what was not done, and no number survives being dealt
with.** Those two halves of the retired rule are not reopened.

## 5. What the rail does not inherit

Worth recording, because it is the reason rooms are safe: **inside a room the
one-thing-at-a-time rule survives a bad week untouched.** The pile holds twelve
and hands over one. The chores hold four and show two, with *the rest* as a
chip. That capping already ships and rooms do not change it.

Only the rail totals. Whatever the rail ends up showing, no room's own screen
ever shows a queue.

---

## Unresolved

**The data model.** A room is most naturally a column on `turns`, which is
`(id, person_id, who, words, shown, said_at)` today. That is a one-column
migration plus a widened index — `turns_person_said` is `(person_id, said_at
desc, id desc)` and both of the two queries it serves become per-room. Room is
text, for the reason `who` is text: a new room should be a phase, not a
migration. Not yet decided: whether the existing thread backfills into Buddy's
room or the pile's.

**Buddy's scope, and its cost.** Per-room Buddy means a narrower toolset and
bounded facts, which pulls cost down, and a separate rolling window per room,
which pulls it up. `Conversations` is keyed by person and would become
`(person, room)`. The net is unknown and should be **measured before it is
chosen**, as the dock's toolset was in v0.43.0 — that precedent is in the
roadmap with the number written down.

**Multi-person.** One person now; the design should not assume it forever. The
`turns` schema already anticipates a third speaker, and rooms should not close
that door. No presence, no real-time, no read state until somebody else is
actually in a room.

**What this costs to build.** DESIGN.md is 2,296 lines describing one screen.
The appearance snapshot pins the current look and the CI design gate requires
DESIGN.md to move when it changes. Both fire on the first commit, correctly.

## Not in scope

Search across rooms. Threading or replies. Notifications per room. Any of the
chat-platform affordances the vocabulary implies but the product has not asked
for.

---

## Provenance

The layout study is at `.impeccable/mocks/`, rendered in Squirrel's own tokens
with the shipping `logo.png` and `buddy.png`. The rejected departure-board
world is under `.impeccable/mocks/decision/` and `comp-*.png`; it is kept as
evidence of what was considered, not as a live option.
