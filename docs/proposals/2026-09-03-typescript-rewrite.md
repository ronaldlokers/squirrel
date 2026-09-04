# What a TypeScript rewrite would cost

*3 September 2026. Asked for after a live-design session failed on the board,
with "I think we need to consider to rewrite the app in TypeScript if we can't
make this work — it keeps me from optimising the design."*

That sentence names the real requirement, and it is not TypeScript. It is: **the
design loop has to work.** This paper costs the rewrite honestly, then costs the
two things that would fix the design loop this week, so the comparison is
between remedies rather than between languages.

---

## What is actually in the repository

Measured today, not estimated.

| | Source | Tests |
| --- | ---: | ---: |
| `internal/squirrel` — store, scheduler, spool, push | 10,915 | 17,107 |
| `internal/web` — routes, templates, board, conversation | 9,992 | 17,501 |
| `internal/coach` — the model provider, budget, tools | 3,744 | 3,542 |
| `internal/boot` — wiring, facts, adapters | 1,543 | 2,155 |
| `internal/transport` — Campfire | 592 | 910 |
| `cmd/devscreen`, `cmd/squirrel` | 392 | 0 |
| **Total Go** | **27,178** | **41,215** |

- **1,822 test functions.** 597 of them need a real Postgres; 82 drive a real
  Chrome over CDP.
- **37 migrations**, applied in order at boot, in production twice today.
- **Four direct dependencies**: pgx, go-oidc, oauth2, testify. Everything else —
  HTTP, templating, ECDH, HKDF, AES-GCM, JPEG and PNG decoding, fsync — is the
  standard library.
- **5,189 lines of CSS, JS and templates.** This is the design.

That last line is the whole argument in one number. The design of this product
is 5,189 lines that a rewrite does not improve, sitting in front of 27,178 lines
that have nothing to do with design and would all have to be rewritten.

---

## The blocker is not the language

The live overlay failed on the board for two specific reasons, both verified:

1. **`html/template` strips HTML comments.** The tool's markers never reach the
   browser.
2. **The picked element is inside `{{range $bay := .Bays}}`.** One source block
   renders four times — notes, chores, tasks, agenda — so one picked element is
   four things on screen.

Reason 1 disappears in TypeScript. **Reason 2 does not.** A React or Svelte
board maps the same bays over the same component, and the same four instances
appear. The overlay's assumption — one picked element is one instance — is
broken by the *page*, not by the templating engine. A rewrite pays 27,000 lines
to fix the smaller half of a two-part problem.

---

## What a rewrite would actually cost

**Ports for free.** The three stylesheets, the authored SVGs, every string of
copy, and the Postgres schema. The 37 migrations are SQL; a new runner executes
the same files.

**Has to be rebuilt.** Everything else. The parts that are more than typing:

- **The store.** 10,915 lines over pgx with hand-written SQL, plus the spool
  that fsyncs before acknowledging a capture. Node has `fsync`, but the ordering
  guarantees the spool depends on have to be re-established and re-proved, and
  those are the guarantees that mean a thought typed on a train is not lost.
- **The scheduler.** Three clocks, one of which is the Postgres session and not
  the pod — the ambient-time discipline that took an incident to learn. It does
  not survive being retyped; it has to be rebuilt deliberately.
- **The coach.** Tool loops, budget ceilings, permit release, cache keys, and
  four separate prompt surfaces, each with refusals enforced in code as well as
  in the preamble.
- **Auth.** OIDC against Authentik, sessions, the group check.
- **The board and the conversation.** 9,992 lines of handlers whose behaviour is
  pinned by 17,501 lines of tests.

**The dominant cost is not the app.** It is that the test suite is **1.5× the
size of the source**, and this project's standing rule is that every test must
be proved to fail against the code it claims to cover — a compile error does not
count. 1,822 tests must be re-authored *and* mutation-proved. Six times in this
project's history a test passed against code that had been removed; that rule is
the scar tissue. A rewrite either re-earns all of it or ships a product whose
tests nobody has proved.

**What genuinely gets easier in Node.** Fair is fair:

- Web push stops being hand-rolled ECDH P-256 + HKDF + AES-GCM + a VAPID JWT and
  becomes `web-push`.
- Thumbnails stop being `image/jpeg` and become `sharp`, which is better at it.
- Vite gives real HMR, and **Storybook or Ladle gives a component gallery where
  each element exists exactly once** — which is the actual thing the design loop
  wants.

**What gets worse.**

- A 10MB static binary on distroless-nonroot becomes a Node runtime image.
- `CGO_ENABLED=0`, one binary, two architectures, no runtime dependencies — all
  of that is currently free and stops being free.
- Every dependency added is a dependency to keep. The current count is four.

---

## The same design gain, this week, for almost nothing

Both of these are available now and neither touches production behaviour.

### 1. A dev-only "one bay" board

The overlay's real blocker is four instances of the picked element. The dev
screen already serves the real templates from the working tree. Teaching the
board handler to draw a single bay **when development mode is on** — the mode
that already exists, `devDir != ""` — is a handful of lines that are inert in a
shipped binary. Live mode then works against `board.html` itself: one element,
one instance, real markup, real CSS, instant refresh.

This is the honest fix. It makes the tool work on the actual product.

### 2. The comp that is already here

`.impeccable/comps/strip-board.html` is a self-contained 835-line static board
with its own surface brief. Static HTML is the overlay's best case: nothing
repeats, nothing is templated, every element exists once. It is stale — written
before the blank strip grew its rhythms, before marginalia, before the bay bar —
and bringing it current is an afternoon. After that it is a design surface with
no server in the loop at all, and the port back is mechanical because the class
names are shared with `board.css`.

---

## Recommendation

**Do not rewrite.** The rewrite costs 27,178 lines of source and 41,215 lines of
proved tests to fix half of a two-part tooling problem, and leaves the design —
5,189 lines of CSS and markup — exactly where it is today.

Build the dev-only single-bay board, and refresh the comp. If, after both, the
design loop still fights back, the case to revisit is not "rewrite the app in
TypeScript" but "give the front end a component gallery", which is a much
smaller question and can be asked about a Go server too.

The one thing that would genuinely justify a rewrite is a decision that
Squirrel's future is a client application rather than a server-rendered one.
That is a product decision about what the thing is, and it should be made on
those grounds — not because an overlay could not find an element.
