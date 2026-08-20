# Deploying the pile screen

The screen at `squirrel.ronaldlokers.nl` reads and triages the pile: one note at
a time, the four transitions, undo, and search across every state. It never
creates an item, and it never shows a count.

Squirrel's half of this is two environment variables. The other half —
Authentik and Traefik — is infrastructure, lives in
[ronaldlokers/homelab](https://github.com/ronaldlokers/homelab), and is written
down here because no test covers it.

## The route table

| URL | What it is | Who may reach it from outside |
| --- | --- | --- |
| `/` | home: three doors, the slot and the check-in | LAN or tailnet, then Authentik |
| `/capture` | the slot's write | LAN or tailnet, then Authentik |
| `/mood` | the check-in's write | LAN or tailnet, then Authentik |
| `/now/act` | the offer's answers: did it, ten minutes, not now | LAN or tailnet, then Authentik |
| `/now/stuck` | I can't start, and its four answers | LAN or tailnet, then Authentik |
| `/push/subscribe` | where to reach this browser — **only mounted when a VAPID pair is configured** | LAN or tailnet, then Authentik |
| `/pile` | the deck | LAN or tailnet, then Authentik |
| `/pile/act`, `/pile/chore`, `/pile/fix` | the deck's writes | LAN or tailnet, then Authentik |
| `/kept` | the shelf: notes you kept | LAN or tailnet, then Authentik |
| `/tasks` | what you decided | LAN or tailnet, then Authentik |
| `/tasks/done` | what you have done | LAN or tailnet, then Authentik |
| `/tasks/act`, `/tasks/new` | a task's writes | LAN or tailnet, then Authentik |
| `/chores` | what comes back | LAN or tailnet, then Authentik |
| `/chores/act`, `/chores/new` | a chore's writes | LAN or tailnet, then Authentik |
| `/timer` | starting and stopping the body double | LAN or tailnet, then Authentik |
| `/pile/chores` | **301 to `/chores`** | LAN or tailnet, then Authentik |
| `/static/…` | stylesheet, script, fonts, mark, icons, door art | LAN or tailnet, no identity |
| `/manifest.webmanifest` | the manifest | LAN or tailnet, no identity |
| `/sw.js` | the service worker | LAN or tailnet, no identity |
| `/hooks/home` | presence webhook | LAN or tailnet, its own token |
| `/transports/campfire` | Campfire's webhook | in-cluster; from outside, Authentik |
| `/healthz` | liveness and readiness | in-cluster; from outside, Authentik |

The assets, the manifest and the worker answer without an identity because a
browser fetches all three without cookies — the manifest from the page, the
icons and the worker from the browser process. Anything that can read a note
still requires one.

`/` is registered as Go's `GET /{$}`, which matches that path and nothing under
it. A bare `/` would be the catch-all, and every typo would arrive looking like
a working page.

`/pile/chores` redirects rather than 404s: it is the URL the chores screen had
for its whole life, and a bookmark that dies quietly is worse than a redirect
nobody notices.

There is no configurable mount path. `WEB_PATH` existed through v0.9.x, was
never set to anything but its default, and cost a prefix on every URL in every
template plus a header to widen the worker's scope by one character.

## Configuration

| Variable | Default | What it does |
| --- | --- | --- |
| `WEB_IDENTITY` | *(empty)* | The one identity that may read the pile. **Empty leaves the screen unmounted** — the routes do not exist, and `GET /` is an ordinary 404. |
| `WEB_IDENTITY_HEADER` | `X-Authentik-Username` | The header the forward-auth middleware fills. |
| `WEB_URL` | *(empty)* | Where the screen is reachable from outside, so chat can link to it. **Empty means chat says nothing about the screen** — a link built from a guess is a link that 404s, and a bot that confidently sends you nowhere is worse than one that stays quiet. |
| `VAPID_PUBLIC_KEY` | *(empty)* | The application server key the browser subscribes with. **Empty leaves `/push/subscribe` unmounted** and the screen never offers — a subscribe button with no key behind it fails silently, which is worse than one that was never drawn. |
| `VAPID_PRIVATE_KEY` | *(empty)* | The raw 32-byte P-256 scalar, base64url. **From Proton Pass, never from this repository.** |
| `PUSH_CONTACT` | *(empty)* | A `mailto:` the push service can complain to. Part of RFC 8292 rather than a courtesy: services reject a token without one. |

All three must be set for pushing to happen at all. None of them being set is a
supported state rather than a degraded one — the leave-by warning still reaches
the Campfire room, which is the channel that always works.

### Minting the VAPID pair

Any Web Push key generator produces a P-256 pair; the two values Squirrel wants
are the uncompressed public point and the raw private scalar, both base64url
without padding. Put the private half in the Proton Pass **Dotfiles** vault and
reference it the way `PRESENCE_SECRET` already is. Nothing about the pair may
land in this repository, and rotating it only costs the subscriptions, which
every browser re-creates on its next visit.

The comparison against `WEB_IDENTITY` is exact — no trimming, no case folding.
Two identities that differ by a space are two identities.

An unset `WEB_IDENTITY` logs `no web identity configured; the pile screen is not
mounted` at boot. A screen that is missing looks exactly like one that is
working until you go looking for it, so the log line is the only way a
mis-wired secret announces itself.

Immediately after boot the routes are live but the owner is not yet known —
`connectAndDrain` learns it when Postgres first answers. Requests in that window
get a 503 that says Squirrel cannot reach its memory, which is what is
happening.

## Authentik and Traefik

Squirrel writes **no authentication code**. It holds no session, sets no cookie,
runs no redirect flow, and has no OIDC library. Traefik calls an Authentik
outpost, Authentik decides who this is, and Squirrel compares one header to one
configured value. App-level OIDC would add sessions, cookies and a callback
route to a binary that has none of that anywhere else, for one user.

The middleware, in Traefik's dynamic configuration:

```yaml
http:
  middlewares:
    authentik:
      forwardAuth:
        address: http://authentik-outpost.authentik:9000/outpost.goauthentik.io/auth/traefik
        trustForwardHeader: true
        authResponseHeaders:
          - X-Authentik-Username
          - X-Authentik-Groups
          - X-Authentik-Email
```

`X-Authentik-Username` is the one Squirrel reads; the others are listed because
the outpost sets them and stripping them here would be a change with no reason
behind it.

Chain it **after** the phase 4 `ipAllowList`, so LAN-only stays the outer layer
and Authentik is the inner one:

```yaml
    # on the router for squirrel.ronaldlokers.nl
    middlewares:
      - lan-only@file      # the phase 4 ipAllowList — first
      - authentik@file     # then the identity
```

Order matters in one direction only: the allow list must run first, so a request
from outside the LAN is refused before it can reach an authentication flow at
all.

Because the identity arrives in a plain header, the only thing keeping it
truthful is that nothing but Traefik can reach the pod. That is the ipAllowList's
job, and it is why the two middlewares are a chain rather than a choice.

## Cross-site writes

Header-based auth means a form on someone else's site posting to `/pile/act`
would travel with the Authentik session cookie like any other request. The two
write routes therefore check `Origin` (falling back to `Referer`) against the
request's own `Host`, and refuse anything that does not match or that says
nothing at all. No token, no cookie, no secret — a browser will not let a page
lie about its own origin.

This requires the proxy to pass the original `Host` through. Traefik does by
default; if a middleware is ever added that rewrites it, every write on this
screen turns into a 403 and the log line is
`refused a cross-site write`.

## Home

`/` is two doors — *the pile* and *the chores* — and nothing else. It reads
nothing: the handler takes no store, so a full pile and an empty one render the
same bytes, and the page answers even when Postgres does not.

That is the mechanism rather than a policy. A home screen that shows what is
waiting greets you with what is waiting, however carefully it is dressed, and a
handler with no way to ask cannot start showing it by accident. It also settles
whether home may triage: there is nothing on it to triage, so the screen and the
chat can never disagree about a note.

The lid keeps the mark, the wordmark and the search field, and drops its
cross-link — both doors are already on the page. Everywhere else the mark is a
link back here.

## Chores

`/chores` is the other half of what Squirrel holds. A chore used to be
invisible: it appeared only when it nudged you, which is the one moment you are
least able to decide you never want it again.

Each one says what it is, how often it comes back, and *roughly* when it was
last done — `today`, `this week`, `a while back` — with `DID IT`, `HOW OFTEN`
and `STOP ASKING`.

Roughly, and never a day count. An exact number attached to something undone
goes up while you are not looking, which is the accumulating shape this product
exists without, and it is one short step from "3 days late". The buckets stop at
"a while back": there is no bucket for a long time, because that sentence is
about the person rather than the chore.

It never says how many chores there are, how many are due, or how late anything
is.

`STOP ASKING` is the screen's half of `!retire`: `active` goes false, the
history stays, and defining the chore again brings the same row back. Changing
how often is an upsert by name, the same write the chat command makes, so the
two surfaces cannot drift into meaning different things by "every 2 weeks".

## The keyboard

Everything below is progressive enhancement — every one of these has a control
or a URL that works without it.

| Key | What it does |
| --- | --- |
| `d` `k` `x` | done, keep, drop |
| `c` then `1`-`4` | make a chore, then how often; `ESC` withdraws the question |
| `space`, `→`, `↓` | skip: move past this note to the next-oldest one |
| `←`, `↑` | back, which is the browser's own history |
| `/` | the search field |
| `ESC` in search | clear it |

**Skipping has a control, not just a key.** `LATER →` sits in the card's
titlebar and is a plain link, so it works on a phone, which has no space bar,
and with scripting off, which has no key handler. The key presses that link
rather than knowing where it points, so there is one answer to where skipping
goes.

**Skipping does nothing to a note.** It puts `?after=<id>` in the address bar
and the deck reads from there, so a skipped note is untouched — still open,
still first the next time the pile is opened from the top. Reloading `/pile`
is how you get back to the top, and running out of notes below the cursor is
its own page rather than an empty pile: what you skipped is still there.

**Search answers as you type**, by fetching the same URL the form submits to
and swapping in what comes back. It is one renderer and one code path, so with
JavaScript off the identical page arrives by pressing Enter. The address bar
tracks the query with `replaceState`, so leaving a search goes back where you
came from rather than through every letter you typed.

## Installing it

The screen is a web manifest and a service worker away from being a thing on
the home screen, so it is one. `Add to Home Screen` on a phone gives it its own
icon and no browser chrome; nothing else about it changes.

**The worker caches assets and never the pile.** A cached pile would show notes
that have already been triaged — the two views disagreeing with each other, in
the one place you could not tell. So the stylesheet, the script, the fonts and
the mark come from the cache, and everything else goes to the network. With no
network it answers with a page that says so and points out that nothing has
been lost, since capture was never the screen's job.

**The worker is served from `/sw.js`, not from `/static/sw.js`.** A worker's
scope is the directory it came from, so one served out of `/static/` could only
ever answer for the assets — the one thing it does not need to intercept. From
the root it scopes to `/` and controls every screen, and it takes no
`Service-Worker-Allowed` header to be allowed to: the header exists to widen a
scope, and this one is already as wide as it goes. `pile.js` registers it with
no `scope` option for the same reason — naming one is how a previous version
ended up claiming whichever page happened to register it.

**It is not behind squirrel's own identity check**, and at the edge it sits with
the assets rather than with the screens. A browser registering a worker does
send the session cookie, so an authenticated visit would fetch it either way;
the file itself contains no notes — only which files to keep and what to say
when the network is gone.

**The installed app survived the move to the root.** Its `start_url` was `/pile`
before v0.10.0, which still serves the deck; the manifest names `/` from its
next fetch onward, so the app opens at home after that. Nobody had to reinstall.

## Reading it without seeing it

The screen is keyboard-first, and the parts that change without a navigation
say so out loud: a live region outside the stage announces a search result set,
an action taken, the interval question being asked or withdrawn, and a skip. It
lives outside the stage because live search replaces everything inside it, and
a region that is itself replaced announces nothing.

The key badges (`D`, `K`, `X`) are `aria-hidden` — a poster on the wall, not
part of a button's name — and so is the stack of cards behind the top one,
which already says "there is more underneath" in words on the deck.

`prefers-reduced-motion` is honoured in both places it lives: the stylesheet
shortens the animations, and the script shortens the pause before a card
leaves. Someone who asked for less motion should not have to sit through a card
sliding away before the write happens.

## What the screen will not do

- **It will not create an item.** There is no capture box and no route that
  writes one. Two capture surfaces means two places to look for a thought, which
  is the problem this bot exists to solve. This is permanent, not a limitation.
- **It will not show a count.** No badge, no total, no "N to review", no page
  count. A capped list may say *that* there is more; it will never say how much.

## Assets after a change

`internal/web/static/` is embedded in the binary and served with
`Cache-Control: public, max-age=31536000`. That is safe because every asset URL
a template writes carries `?v=<stamp>`, where the stamp is a hash of the
embedded files themselves — change a file and the URL changes with it, so a
browser fetches the new one without being asked to.

Nothing to do after a change, and no build step: the stamp is computed from the
bytes at startup.

It is worth knowing why this exists. v0.7.0 shipped without it and arrived
broken: HTML is served `no-store`, so browsers rendered the new markup against
the stylesheet and script they already had — a link with no styling and a
button whose handler did not exist yet. The failure is silent and looks like a
bug in the feature rather than in its delivery.

The fonts are the one gap: they are named from inside `pile.css` rather than
from a template, so their URLs carry no stamp. Replacing a font means renaming
the file, which is why the stamp hashes names as well as contents.
