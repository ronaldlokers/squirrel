# Proper OIDC

**Date:** 25 August 2026
**Status:** approved for implementation

Squirrel's whole authentication is one line: Traefik calls an Authentik
outpost, Authentik decides, and `guard` compares one header to one configured
string. That was the right size while there was one person and one pile.

This replaces it with the application doing OIDC itself, so that a second
person is an Authentik action rather than a redeploy — and so demo accounts
become something you can hand out and take back.

**This phase is still one real user.** What it buys is that the next phase is a
step rather than a rewrite. Section 9 names exactly what still pins the product
to one person, so nobody has to go looking.

---

## 1. What replaces the header

`guard` keeps its name and its position and loses its body. It reads a session
cookie instead of `X-Authentik-Username`, and puts two things in the request's
context: the **person id**, which almost everything uses, and the **`sub`**,
which only capture needs — see section 4 for why it needs it.

Four new routes:

| Route | What it does |
| --- | --- |
| `GET /auth` | the door — see section 1a |
| `POST /auth/in` | sets `state` and the PKCE verifier, 302 to Authentik |
| `GET /auth/callback` | takes the code, opens a session, 302 to where you were going |
| `POST /auth/out` | deletes the session, 302 to `/auth?said=out` |

Starting a login is a POST because it writes — the same rule that makes opening
a door a POST rather than a link. It also means a prefetch or a crawler cannot
begin a login.

**A GET for a page redirects to the door; a POST does not.** An unauthenticated
POST, and any request carrying `X-Thread: fragment`, gets `401` with no body. A
redirect there would swallow the form and `thread.js` would paste the door into
the conversation as a turn.

The four `/auth` routes are the only ones outside the guard. Static assets stay
outside it as they are today.

### The refactor this forces

`Options.Owner` is a process-global `atomic.Int64` and `opts.person()` reads
it. That cannot survive two people.

`opts.person()` becomes `personOf(r)`, reading the value `guard` put in the
request context. It is a mechanical change across roughly forty call sites and
it is the largest single piece of this work. `Options.Owner` is deleted rather
than left in place, because a global that still compiles is a global something
will use.

---

## 1a. The door

Squirrel gets one screen of its own, and it is the first thing anybody ever
sees of this product.

It is built the way `/enough` is built, which DESIGN.md already describes as a
treatment rather than a page: *the mascot, a headline in the casual axis, a
screen that is an absence.* Nothing new is invented for it.

```
        ┌────────────────────────┐
        │                        │
        │        (buddy)         │
        │                        │
        │       Squirrel         │
        │                        │
        │   ┌────────────────┐   │
        │   │    LET ME IN   │   │
        │   └────────────────┘   │
        │                        │
        └────────────────────────┘
```

**One screen, four states.** Each is the same composition with a different
sentence under the mark, because they are the same moment arrived at four ways
— not four pages nobody designed.

| State | What it says | The button |
| --- | --- | --- |
| cold | nothing | LET ME IN |
| signed out | you are signed out | LET ME IN |
| refused | that account cannot use Squirrel | LET ME IN, to try another |
| Authentik unreachable | I cannot reach the door just now | TRY AGAIN |

Why a door rather than a straight redirect, since a redirect is one press
fewer:

- **This is a PWA.** A cold launch from the home-screen icon that immediately
  lurches to another domain is the awkward case, and worst on iOS.
- **Sign-out needs somewhere that does not bounce.** Deleting the session and
  landing on `/` would redirect to Authentik, which still has *its* session,
  and sign you straight back in. The door is where signing out lands.
- **The unhappy states need a page anyway.** Building them as states of one
  screen costs nothing; building them as an afterthought is three pages nobody
  drew.
- **It is the only screen a demo user sees** before deciding whether they trust
  this thing.

`?next=` carries where you were going, and is refused unless it is a path this
server serves — the same `backTolerant` rule the timer already uses, and for
the same reason: a value that arrives in a URL is a place a stranger can type.

## 2. The session

One migration, `0029_sessions`:

```sql
create table if not exists sessions (
    id           bigserial   primary key,
    person_id    bigint      not null references people (id) on delete cascade,
    token_sha256 bytea       not null unique,
    -- The login key, carried here so the request path resolves it without a
    -- second read. Capture writes it as a sender; see section 4.
    sub          text        not null,
    created_at   timestamptz not null default now(),
    seen_at      timestamptz not null default now(),
    expires_at   timestamptz not null
);
create index if not exists sessions_person on sessions (person_id);
```

The cookie carries 32 bytes from `crypto/rand`, base64url. The table stores
only their SHA-256, so a database dump is not a set of live sessions. Hashing
is not for secrecy — the token is high-entropy — it is so that read access to
the table is not read access to the product.

`HttpOnly; Secure; SameSite=Lax; Path=/`. Lax rather than Strict because the
callback arrives as a top-level navigation from Authentik, and Strict drops the
cookie on exactly that hop.

Thirty days, with `seen_at` refreshed on use. A session unused for thirty days
is gone; a session in daily use never expires out from under you.

### The cache, and what it costs

A session lookup is a database read on every request, which the header-based
design deliberately never had — `ResolvePerson` runs in the drain loop and the
comment says why: *"the request path not touching Postgres is what makes an
outage survivable."*

So: an in-process map of `token hash → (person id, expiry)`, sixty seconds.

- A Postgres wobble leaves an active session working, and capture still spools.
- Revoking a session takes up to a minute to bite.

That trade is right for what revocation is actually for here — *"I am done
playing with that demo account"* rather than an incident. It is stated so that
if revocation ever needs to be immediate, this is the line to delete.

The cache is bounded and evicts oldest-first, for the reason the door cache
needed a bottom on the same day: a map nothing evicts is a leak discovered by a
pod being OOM-killed months later.

---

## 3. Who gets in

The callback exchanges the code, verifies the ID token, and reads `sub`.

- **Known `sub`** → `ResolvePerson("oidc", sub)` finds the person → session.
- **Unknown `sub`** → create a person, then a session.

Creating a person is a departure from a written rule, and the rule is worth
quoting because it is being overturned rather than forgotten:

> An unknown identity is nil. It is never created: auto-vivifying a person on
> first sight would quietly re-admit anyone the guard had just turned away.

That rule was correct when `guard` was the only gate. Authentik's application
binding is the gate now, and it is a better one — it can be changed without a
deploy. `ResolvePerson` keeps its behaviour and its comment; the callback
calls a new, explicitly-named `PersonForLogin` which creates.

### The group is required

`WEB_REQUIRED_GROUP` must be set, and the `groups` claim must contain it, or
the login is refused.

Required rather than optional, which is a departure from this codebase's habit
of treating a missing value as a supported state. The reasoning: every other
optional thing degrades to *less product* — no coach, no push, no camera. This
one would degrade to *more access*. A missing group would mean every Authentik
account silently gets a pile.

Squirrel refuses to serve the web surface without it and says so at boot. The
failure is loud and the pod is otherwise healthy — chat and the scheduler still
run.

---

## 4. Keeping your pile

You are already a person, seeded with a Campfire identity and a `screen`
identity, and every note points at that row.

Your Authentik `sub` is added to the config as an `oidc` seed. `SeedOwner`
already reconciles seeds on every boot and is idempotent, so this needs no new
mechanism. Your first login resolves to the person you already are.

### The trap: capture resolves by sender, not by session

This is the part that is easy to miss and expensive to find.

A capture typed on the screen does not go straight to Postgres. It goes through
the spool with `SenderID` set to the configured identity string, and **the
drain resolves its owner** with `ResolvePerson(ScreenTransport, sender)`. That
is what `seedsFrom` exists for.

So a person with only an `oidc` identity would spool notes that resolve to
nobody, and land as rows belonging to no one.

Therefore: **a person always gets two identities, `oidc:<sub>` and
`screen:<sub>`, created together.** The capture path writes the `sub` from the
request context as the sender rather than the configured `opts.Identity`
string, and the drain resolves it exactly as it does today. The session row
carries the `sub` so this costs no extra read.

`opts.Identity` is deleted along with `Options.Owner`, for the same reason: a
process-wide identity that still compiles is one that something will use.

Your existing `screen:<WEB_IDENTITY>` seed stays, so nothing already in the
spool at deploy time is orphaned.

---

## 5. What a demo user costs you

`coach.Budget` carries one `CeilingMicros` for the process and applies it to
whoever asks. Two demo accounts would then be two monthly ceilings.

`CeilingMicros` becomes a lookup: the owner's ceiling, and a smaller one for
everyone else. The owner is the person whose handle matches `OWNER_HANDLE`,
resolved once at boot — no new column, no admin flag.

`COACH_BUDGET_GUEST_EUR` defaults to something small. A demo user can ask Buddy
a handful of things, see it work, and cannot spend a month's allowance.

Everything else about the budget is unchanged: it is still per person, still
recorded per answer, still enforced in the one place that can enforce it.

---

## 6. Proving the piles do not leak

Every query is person-scoped today. With one user, a missing `where person_id`
is invisible — and the first demo login is the worst possible time to find one.

A test seeds two people and gives both of them notes, tasks, chores,
appointments, kept notes, set-aside notes, turns, readings, steps and coach
answers. It then walks every read the web package can perform as the first
person and asserts that nothing belonging to the second ever appears — in a
rendered page, in a drawn turn, or in a store result.

It is written as a table of reads rather than a test per surface, so a new
surface that forgets to scope is a line somebody has to deliberately not add.

This is the one part of this work that is worth doing even if the rest were
abandoned.

---

## 7. When things fail

| What | What happens |
| --- | --- |
| Authentik unreachable at login | the door says so and offers TRY AGAIN; existing sessions keep working |
| `state` missing or wrong | refused, no session, logged |
| ID token fails verification | refused, no session, logged |
| `groups` lacks the required group | the door, refused, saying that account cannot use Squirrel |
| Postgres down at callback | login fails; an existing session still captures |
| Postgres down mid-session | the cache covers up to a minute; after that, refused |
| No `WEB_REQUIRED_GROUP` | the web surface refuses to mount and says so at boot |

A refusal never explains more than it must. "That account cannot use Squirrel"
is the whole of what a refused person is told — not which group they lack,
which is a fact about your Authentik rather than about them.

---

## 8. Testing

**A fake Authentik.** An `httptest` server serving discovery, JWKS and a token
endpoint, signing ID tokens with a generated RSA key. `go-oidc` then runs its
real verification path rather than being stubbed — which matters, because the
verification is the part being delegated to it.

Cases: a good login; an unknown `sub` that provisions; a wrong `state`; an
expired token; a token signed by the wrong key; a missing group; a login when
Postgres is down.

**Browser tests** for the shapes only markup can prove: a GET landing on the
door, the door's four states, the callback landing you on the conversation, a
POST getting 401 rather than HTML, `thread.js` not pasting the door into the
thread as a turn, and signing out landing somewhere that does not bounce
straight back in.

**The isolation sweep** from section 6.

Every one of these gets mutation-proved in the usual way, and the auth ones
matter more than most: a test that passes with the check deleted is worse than
no test on a login path.

---

## 9. What still pins this to one person

Stated so the next phase does not have to go looking.

1. **The scheduler takes one `PersonID`.** The evening message, nudges,
   leave-by warnings and the weekly read of the record all run for that one
   person. Multi-user means a scheduler per person, or one loop over people.
2. **Campfire is one bot in one conversation.** The drain already resolves a
   sender to a person, so inbound chat is closer to multi-user than the
   scheduler is; what is single is the room replies go back to.
3. **`OWNER_HANDLE` decides the budget ceiling**, which is a two-tier
   distinction rather than a real per-person setting.

Nothing in this phase makes any of the three harder. The person id is threaded
through the request path, every table is scoped, and the isolation sweep proves
it.

---

## 10. The Authentik side

Squirrel's OIDC client is one more entry in the homelab's
`authentik-blueprints` Secret, which already declares every other client. That
file says why, and the reason applies here too: letting Authentik generate the
client id and secret leaves the working values only in its database, so
recreating it breaks every client at once.

The application is bound to the group named by `WEB_REQUIRED_GROUP`. That
binding, not anything in Squirrel, is what decides who can get a pile.

The forward-auth outpost and its ingress come out in the same change, and that
is the only moment where getting the order wrong locks you out: the outpost
must not be removed before the OIDC client exists and Squirrel is serving the
door.

---

## 11. What is deliberately not here

- **No username and password form.** Squirrel could post credentials to
  Authentik with the OAuth password grant and never redirect at all. It would
  cost MFA and passkeys, which have nowhere to prompt in a form; it would make
  this binary a credential-handling application, which is a class of bug it
  currently cannot have; and it would bypass Authentik's own lockout, recovery
  and consent flows. The grant is deprecated in current guidance and removed in
  OAuth 2.1. Considered on 25 August 2026 and declined.

- **No Authentik branding.** Making the handover less jarring by styling
  Authentik to look like Squirrel is real and cheap — but Authentik matches a
  **brand by domain**, not by application, so one Authentik on one hostname
  means one brand for linkding, immich, tandoor and everything else behind it.
  Squirrel-only branding needs a second hostname, a second brand, another
  certificate, and an answer to whether `go-oidc`'s issuer check survives the
  same provider being served on two names. That is a great deal of machinery
  for a page seen once a month. Declined the same day; the door either side of
  the handover is what carries the identity.

- **No admin screen.** People come from Authentik; there is nothing to
  administer in Squirrel.
- **No sign-out-everywhere.** One session at a time is deleted. Killing all of
  a person's sessions is a `delete from sessions where person_id = …` and does
  not need a button until somebody wants one.
- **No refresh tokens.** Squirrel needs to know who you are at login and never
  acts on your behalf against Authentik afterwards. Storing a refresh token
  would be storing a credential for no reason.
- **No account deletion.** `on delete cascade` from `people` means the
  machinery exists; the decision about what deleting a person means is not this
  phase's.
