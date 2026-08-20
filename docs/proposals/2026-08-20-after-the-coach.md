# After the coach

*20 August 2026. Four things, chosen from the backlog once Buddy shipped.*

The coach answered for the first time at 16:02 today. Everything here was
already argued for before that; what changed is which of them matter most now
that something chooses on your behalf.

Sequenced deliberately: the one that improves every day first, the one that
protects against a rare disaster second, the two small ones third, and the one
that adds infrastructure last.

---

## 1. Three states for things you cannot act on

**waiting on someone · blocked on a thing · someday**

### Why it moved to the front

The argument for these was always "the picker keeps offering things you cannot
act on". That was an irritation when nothing chose for you. Now the picker
*and* Buddy each hand you exactly one thing, and an offer you cannot act on is
the most annoying failure the product has — it spends the one decision you were
given on a dead end.

There is currently no way to say **not me** or **not yet** without lying. Your
options are to do it, to turn it down for today, or to drop it. Turning down
something you are waiting on someone else for means being asked again tomorrow,
and the day after.

### Three, not one

They expire differently, and that is the whole reason they are not one "parked"
state:

| | What ends it |
| --- | --- |
| **waiting on someone** | they reply — an event outside you, that you will want to record when it happens |
| **blocked on a thing** | the thing arrives or gets fixed — also outside you, but not a person to chase |
| **someday** | nothing. It ends when you decide it does. |

A single state would make "chase the vet" and "learn to solder" the same kind
of thing, and they are not: one is work with a dependency and the other is a
wish. Collapsing them is how a someday list becomes a guilt list.

### They carry what you are waiting on

`!waiting 3 on the vet` stores *the vet* alongside the state. Without it you get
a list of stalled things and no idea what would unstall them, which is a worse
version of the pile.

New text you typed, not a rewrite of your words — the `!fix` rule is untouched.

### Why this is small

`itemsWhere` takes an explicit state on every list. The pile is
`state = 'open'`, the tasks are `state = 'open'`, the archive is
`state = 'done'`. **Three new states are excluded from every existing surface by
construction** — there is nowhere to forget a filter, and no query to audit.

What has to be built is the transitions, one screen to see them on, and the way
back.

### Where they are read

A screen, and one screen for all three rather than three. Grouped, each row
carrying what it is waiting on and one press to bring it back. Chat gets
`!waiting` with no argument as the same list.

**Not on home.** Home has three doors and an offer; a fourth door for things
you have explicitly set aside is the opposite of what setting them aside was
for.

### Reversal

Everything returns through the transition that already exists — `act=open` puts
any item back in the pile, and undo already works on it. Nothing new is needed
for the way back, which is the same property that made the split's original
note recoverable.

---

## 2. The capture gap

**The PWA is the front door and its capture has no spool.**

### What is actually wrong

Capture is sacred, and it is protected properly on exactly one of the two
surfaces:

```
chat  → fsynced spool → acknowledged → drain → Postgres
PWA   → Postgres → acknowledged
```

The chat path acknowledges after the words are on disk. The screen's path
acknowledges after the words are in Postgres, which means a healthy network and
an unhealthy database loses them. The screen fails loudly and keeps the words on
the page — which is honest, and is not the same as durable, because the page is
one reload from empty.

This was written down and accepted when the screen was secondary. It is wrong
now that it is the front door, and it was recorded as wrong at the time.

### Why it is second and not first

It needs Postgres to be down to matter, and Postgres has not been down. It is a
disaster-shaped risk rather than a daily one — but it is the product's first
principle, and the first principle should not have a footnote.

### The shape of the fix

The service worker already holds a capture when the network is gone, in
IndexedDB, and flushes it when the network returns. The gap is a live network
and a dead database: the request succeeds in reaching the server and then fails.

So the server should answer that case in a way the worker already understands,
and the worker should hold it exactly as it holds a network failure. One path,
one place where held captures live, and no second mechanism to keep in step
with the first.

**Open:** whether the spool should move server-side for both surfaces instead.
That is the tidier answer and a much larger change; the worker-side fix is
smaller and reuses something that already works.

---

## 3. Two small ones

### Mood, readable

The check-in has always been write-only by construction: the store exposes one
reading in and one reading out, and there is deliberately no function that
returns a series. That was to stop the product forming a picture of you from
your moods.

Principle 5 is open now, and this reverses the same rule from the other side:
**shown back on request, and never on its own.** Not on home, not in the
evening message, not to Buddy. You ask, you see it, and nothing else does.

### Resurfacing

A kept note may come back — **only** riding along with something already being
raised, never as its own stream. A shelf that periodically taps you on the
shoulder is a second inbox, which is the thing this product exists not to have.

---

## 4. Attachments, on a PVC

*Decided 20 August: a volume on the pod rather than adding object storage.*

Camera-first from the PWA, because the case is photographing a letter or a
serial plate, not uploading a file.

A Longhorn volume next to the pod is the whole of the infrastructure. Adding
MinIO or Garage would mean a new service to patch, back up and keep alive for
one feature, and this cluster does not otherwise need object storage.

**The cost, stated:** attachments become part of the pod's own lifecycle, and
the restore drill has to grow to cover the volume. That is a real obligation and
it is the price of not running a second service.

Last of the four because it is the only one that adds infrastructure, and
because everything above it improves a day that is happening now.

---

## Not now, and why

**`/v1/responses` for the deep turns.** Buddy currently decides without extended
reasoning, because tools and reasoning cannot both be asked for on
`/v1/chat/completions`. Moving endpoints would buy it back and spend the
portability that made `COACH_BASE_URL` worth having. **Two calls of evidence is
not enough to spend that on.** Revisit if the overwhelm turn reads shallow.

**Durations and asking windows.** Both need a month of ordinary use.
`timer_runs` began collecting on 20 August, so the clock has started. Late
September.
