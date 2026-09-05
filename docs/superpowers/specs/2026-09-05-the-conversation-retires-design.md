# The conversation retires into the board

**Date:** 5 September 2026
**Status:** drafted from the decision of 3 September; shape approved, phases not yet built

The decision was made on 3 September and written into the roadmap: the
conversation goes, what it does moves to the board, and chat ends up
Campfire-only. It was recorded as governing everything built after it, and then
nothing was written down about how. Three weeks and thirteen releases later the
capabilities that live only in the room are still there, and the roadmap has no
line item for any of them.

This is the missing half. It is not a new decision — it is the inventory the
decision implied, plus the one shape question that has to be answered before
anything moves.

## What is already on the board

Seven actions, and they are the ones the product is *for*: capture into any of
four bays, triage a strip, make a chore of a note, answer the offer, say how you
are, undo. Plus the readings and the marginalia line. The board is already the
whole of ordinary use.

## What still lives only in the conversation

Every one of these is reachable from the room and from nowhere else.

| | What it is | Where it lands |
| --- | --- | --- |
| `/buddy/ask`, `/buddy/say` | asking Buddy something about what is on screen | **the strip it is about** |
| `/buddy/do` | the six write tools that run and the four that ask first | **the strip**, as the answer to an ask |
| `/buddy/badly` | *that went badly* — the press that feeds the record | **the strip**, on Buddy's own line |
| `/steps` | breaking one task into steps, shown one at a time | **the strip**, opened |
| `/pile/split` | separating a brain dump into the things it was | **the notes bay**, on capture |
| `/now/stuck` | the ladder — four answers to *I can't start* | **the pulled strip** |
| `/timer` | starting, ending and re-entering a timer | **the pulled strip** |
| `/at/*` | an appointment's detail: notes on it, what to bring, detaching | **the agenda bay**, opened |
| `/find`, `/find/open` | search that answers as you type | **the lid**, where the box already is |
| `/pile/why`, `/pile/more` | why this was offered, and what else there is | **the strip** |
| `/pile/fix`, `/pile/reword` | correcting what was captured | **the strip**, opened |
| `/chores/name`, `/chores/often` | naming a chore and setting its rhythm | **the chores bay** (partly there already) |

Nothing in that list is a feature nobody uses. `/steps` and the ladder are the
two the product was extended for.

## The shape question, answered

**Buddy on the board answers through the thing you are looking at, not through a
box.**

The room is a free-text conversation. The board is strips, chips and stamps. The
cheap way to move Buddy is to drop a chat widget onto the board — and that is
the refused *general AI chat companion*, relocated rather than retired. The
refusal stands, so the widget is not available.

What replaces it: **a strip can be asked about, and answers on itself.** The ask
is a press on the strip, not a prompt; the reply is a line under that strip, in
the register marginalia already established; the write tools become the same
confirmation controls they already are, drawn on the strip they act on. Free
text goes in exactly one place — the capture boxes, which already exist and
which never invoke a model.

Three consequences worth stating because they are costs:

1. **You cannot ask Buddy a question that is not about something on the board.**
   That is a real loss and it is deliberate: the general question is the chat
   companion. If it turns out to be needed, Campfire is where it goes, because
   Campfire is allowed to be a conversation.
2. **The steps ladder needs somewhere to live that is not a turn.** A strip that
   opens into steps, one at a time, is a bigger drawing job than the rest of
   this put together.
3. **`coach_answers` keeps its shape.** The record is turns; it stays turns even
   when nothing renders them as a conversation. What "did not land" means does
   not change.

## Phases

Ordered so each one is shippable on its own and none of them leaves the product
worse than it found it.

**A. The strip can be asked about.** `/buddy/ask` and `/buddy/say` become a
press on a strip and a line under it. The room keeps working throughout. This is
the phase that proves the shape; if the answer reads badly under a strip, the
rest of the plan is wrong and this is where that is discovered cheaply.

**B. The ladder and the timer move to the pulled strip.** Both already act on
whatever the picker handed you, so neither needs a target it does not have.

**C. Search moves to the lid.** The box is already there; only the results have
nowhere to land.

**D. The strip's own corrections** — why, more, fix, reword — become the opened
strip's controls.

**E. Steps.** Last, because it is the largest and because by then the shape has
been proved four times.

**F. The room is deleted, and Campfire becomes the only conversation.** Not
before every phase above has shipped, and not before `PRODUCT.md`'s
one-directional parity rule has been re-read against what is left.

## Checked against the floor

`PRODUCT.md:259-266` names five things chat keeps forever, "because this is the
surface that works when the other one cannot be reached at all": capture,
answering a nudge, the four dispositions, setting one aside, and reading the
pile back.

**Nothing in this plan touches them.** The floor is *Campfire's*, and Campfire
is what the conversation retires into rather than what it retires from. The room
at `/r/{room}` is a second conversation inside the app, and it is the one that
goes; every command in that list keeps working in the room that is a room.

Phase D moves *fix* onto the strip and phase C moves search into the lid. Both
are the board growing a copy, not chat losing one — `!fix <n>` and `!find <text>`
stay exactly where the floor says they stay.

## What has to be true before A starts

- The `dev` screen can draw a strip with an answer under it, or none of this can
  be looked at before it ships.

## What this does not decide

Whether the room's *history* is migrated or simply stops being rendered. The
turns are the record and they are not deleted either way; what is open is
whether anything ever shows them again.
