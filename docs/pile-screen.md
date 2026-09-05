# Deploying the board

The app at `squirrel.ronaldlokers.nl` is one board: every bay — notes, chores,
tasks, the agenda — as strips on one page, plus the conversation at
`/r/everything`, the page about you at `/me`, and Buddy, reached from a chip
rather than a screen of his own. There is no card-by-card pile screen any
more; that deck, and the separate `/kept`, `/tasks`, `/moods` and `/held`
pages beside it, retired between 20 August and 31 August 2026. Their
addresses still work — they redirect rather than 404 — and so do `/coach` and
`/buddy`, which is what this file used to be about.

Squirrel's half of reaching the board is the configuration below and, since
25 August 2026, its own OIDC client rather than a header Traefik set. The
other half — Authentik and Traefik — is infrastructure, lives in
[ronaldlokers/homelab](https://github.com/ronaldlokers/homelab), and is
written down here because no test covers it.

## The route table

| URL | What it is | Who may reach it from outside |
| --- | --- | --- |
| `/`, `/board` | the board: every bay, drawn fresh on every load | LAN or tailnet, then signed in |
| `/r/everything` | Buddy's room: the conversation, reading the whole record | LAN or tailnet, then signed in |
| `/r/{room}` | any other room by name | LAN or tailnet, then signed in |
| `/capture`, `/open` | the dock's write, and the door, in whichever room | LAN or tailnet, then signed in |
| `/board/act`, `/board/undo`, `/board/new`, `/board/now`, `/board/capture`, `/board/chore`, `/board/mood`, `/board/notuseful` | the board's writes | LAN or tailnet, then signed in |
| `/me`, `/me/face` | the page about you, and your photo | LAN or tailnet, then signed in |
| `/me/forget` | forgetting what Squirrel has worked out | LAN or tailnet, then signed in |
| `/photo/{id}`, `/photo/{id}/thumb` | a photograph on a note, by the note's id — **only mounted when a volume is configured** | LAN or tailnet, then signed in |
| `/mood` | the check-in's write | LAN or tailnet, then signed in |
| `/now/act` | the offer's answers: did it, ten minutes, not now | LAN or tailnet, then signed in |
| `/now/stuck` | I can't start, and its four answers | LAN or tailnet, then signed in |
| `/push/subscribe`, `/push/forget` | where to reach this browser, and forgetting it — **only mounted when a VAPID pair is configured** | LAN or tailnet, then signed in |
| `/pile/act`, `/pile/later`, `/pile/undo`, `/pile/often`, `/pile/reword`, `/pile/why`, `/pile/chore`, `/pile/fix`, `/pile/split`, `/pile/more`, `/pile/ask` | a note's writes, and the questions asked about one | LAN or tailnet, then signed in |
| `/place/fresh` | starting fresh rather than carrying on a run Buddy offered back | LAN or tailnet, then signed in |
| `/find`, `/find/open`, `/find/ask` | search, and turning a hit into a card | LAN or tailnet, then signed in |
| `/buddy/ask`, `/buddy/say`, `/buddy/badly`, `/buddy/do` | talking to Buddy, as turns in the conversation | LAN or tailnet, then signed in |
| `/chores/ask`, `/chores/name`, `/chores/act`, `/chores/often`, `/chores/new` | a chore's writes | LAN or tailnet, then signed in |
| `/at/ask`, `/at/new`, `/at/make`, `/at/open`, `/at/{id}`, `/at/{id}/note`, `/at/{id}/detach` | a fixed point: making one, reading one, attaching or detaching a note | LAN or tailnet, then signed in |
| `/tasks/ask`, `/tasks/act`, `/tasks/new` | a task's writes | LAN or tailnet, then signed in |
| `/notes/shelf` | the two shelves, held and kept, as a press inside the notes | LAN or tailnet, then signed in |
| `/held/act` | setting something aside, and picking it back up | LAN or tailnet, then signed in |
| `/steps` | a step finished, or a breakdown thrown away | LAN or tailnet, then signed in |
| `/timer` | starting and stopping the body double | LAN or tailnet, then signed in |
| `/coach`, `/buddy`, `/r/buddy`, `/r/pile`, `/r/held`, `/r/kept`, `/moods`, `/knowing`, `/pile/chores` | **301, to wherever what they held lives now** | LAN or tailnet, then signed in |
| `/auth` | the way in — see `internal/web/templates/gate.html` | outside the guard: there is no session yet |
| `/auth/in` | sets `state` and a PKCE verifier, 303 to Authentik | outside the guard: there is no session yet |
| `/auth/callback` | takes the code, opens a session, 303 to where you were going | outside the guard: there is no session yet |
| `/auth/out` | ends the session, 303 to `/auth?said=out` | outside the guard: there is no session yet |
| `/static/…` | stylesheet, script, fonts, mark, icons, door art | LAN or tailnet, no identity |
| `/manifest.webmanifest` | the manifest | LAN or tailnet, no identity |
| `/sw.js` | the service worker | LAN or tailnet, no identity |
| `/hooks/home` | presence webhook | in-cluster; homelab decides what reaches it from outside |
| `/transports/campfire` | Campfire's webhook | in-cluster; homelab decides what reaches it from outside |
| `/healthz` | liveness and readiness | in-cluster; homelab decides what reaches it from outside |

The assets, the manifest and the worker answer without an identity because a
browser fetches all three without cookies — the manifest from the page, the
icons and the worker from the browser process. Anything that can read a note
still requires one.

`/` is registered as Go's `GET /{$}`, which matches that path and nothing
under it. A bare `/` would be the catch-all, and every typo would arrive
looking like a working page.

The redirect row is four product generations' worth of bookmark: the deck,
the four object rooms, and `/coach` before it was `/buddy`. A bookmark that
dies quietly is worse than a redirect nobody notices, so none of these 404.

There is no configurable mount path. `WEB_PATH` existed through v0.9.x, was
never set to anything but its default, and cost a prefix on every URL in every
template plus a header to widen the worker's scope by one character.

## Configuration

| Variable | Default | What it does |
| --- | --- | --- |
| `WEB_IDENTITY` | *(empty)* | **No longer an identity anybody authenticates with.** It is seeded as a `screen` identity so captures already sitting in the spool at deploy time still resolve to their person. **Empty leaves the board unmounted** — the routes do not exist, and `GET /` is an ordinary 404. |
| `WEB_REQUIRED_GROUP` | *(empty)* | The Authentik group an account must be in. **Empty refuses the mount**, and it is the only setting here that is dangerous to default: every other missing value costs a feature, this one would cost the pile. |
| `WEB_OIDC_ISSUER` | *(empty)* | Authentik's issuer URL for this application. |
| `WEB_OIDC_CLIENT_ID` | *(empty)* | The OIDC client id. |
| `WEB_OIDC_CLIENT_SECRET` | *(empty)* | **A credential: it comes from a Kubernetes Secret, never from this repository.** |
| `WEB_OIDC_REDIRECT_URL` | *(empty)* | Where Authentik sends you back — `https://<host>/auth/callback`. |
| `WEB_OIDC_SUB` | *(empty)* | The owner's OIDC subject, seeded so the first login lands on the person who already owns the pile rather than making a second one beside it. |
| `COACH_BUDGET_GUEST_EUR` | `1` | The monthly ceiling for anybody who is not the owner. A demo account can try Buddy without spending a month's allowance, and two of them are not two allowances. |
| `WEB_URL` | *(empty)* | Where the board is reachable from outside, so chat can link to it. **Empty means chat says nothing about the board** — a link built from a guess is a link that 404s, and a bot that confidently sends you nowhere is worse than one that stays quiet. |
| `VAPID_PUBLIC_KEY` | *(empty)* | The application server key the browser subscribes with. **Empty leaves `/push/subscribe` unmounted** and the board never offers — a subscribe button with no key behind it fails silently, which is worse than one that was never drawn. |
| `VAPID_PRIVATE_KEY` | *(empty)* | The raw 32-byte P-256 scalar, base64url. **A credential: it comes from a Kubernetes Secret, never from this repository.** |
| `PUSH_CONTACT` | *(empty)* | A `mailto:` the push service can complain to. Part of RFC 8292 rather than a courtesy: services reject a token without one. |

The four `WEB_OIDC_*` values are all-or-nothing: a partially configured way in
is a boot that half-works, and the half that works is the half that lets people
in. With `WEB_IDENTITY` set and no way in configured, **boot fails** rather than
mounting a board nobody can sign into — a deploy that looked healthy and locked
you out of your own pile is the failure this exists to prevent.

All three VAPID variables must be set for pushing to happen at all, and the
board checks all three before it offers to subscribe: a public key on its own
would draw a button, spend a permission prompt, and store a subscription
nothing can send to. None of them being set is a supported state rather than a
degraded one — the leave-by warning still reaches the Campfire room, which is
the channel that always works.

### Where the pair lives

The public key and the contact are not credentials and are plain values on the
Deployment. The private scalar is a credential and follows `PRESENCE_SECRET`
exactly: a SOPS-encrypted Secret in
[ronaldlokers/homelab](https://github.com/ronaldlokers/homelab), decrypted by
Flux with the cluster's age key, mounted through an `optional: true`
`secretKeyRef` so a rollout before the Secret exists costs pushing and nothing
else.

Not Proton Pass. The vault is where *dotfiles* secrets come from, and nothing
in the cluster can read it at deploy time — a copy may live there for the day
the age key is lost, but the Secret in homelab is the source of truth.

Rotating the pair costs only the subscriptions, which every browser re-creates
on its next visit. See `docs/runbooks/squirrel.md` in homelab for the
procedure.

An unset `WEB_IDENTITY` logs `no web identity configured; the pile screen is
not mounted` at boot — the log line still says "pile screen" because that is
the literal string in `internal/boot/boot.go`, not because the screen it names
still exists. A board that is missing looks exactly like one that is working
until you go looking for it, so the log line is the only way a mis-wired
secret announces itself.

## Authentik

Until 25 August 2026 Squirrel wrote **no authentication code**: Traefik called
an Authentik forward-auth outpost, and Squirrel compared `X-Authentik-Username`
to `WEB_IDENTITY`. That was the right size while there was one person and one
pile. The outpost could only ever say "somebody Authentik likes" — never *which*
somebody in a way Squirrel could act on — so a second person meant a redeploy.

The application does OIDC itself now. Four routes, and they are the only ones
outside the guard besides the manifest, the worker and the static files:

| Route | What it does |
| --- | --- |
| `GET /auth` | the way in |
| `POST /auth/in` | sets `state` and a PKCE verifier, 303 to Authentik |
| `GET /auth/callback` | takes the code, opens a session, 303 to where you were going |
| `POST /auth/out` | ends the session, 303 to `/auth?said=out` |

Starting a login is a POST because it writes, which also means a prefetch or a
crawler cannot begin one.

**The group is checked twice**: bound on the application in Authentik, and
checked again on the ID token's `groups` claim. Not because one gate is
insufficient, but because a misconfigured binding would otherwise hand out piles
silently. An absent claim is refused rather than treated as unrestricted — this
is the one place in the product where a missing value would mean *more* access.

**The session** is 32 bytes from `crypto/rand` in an `HttpOnly; Secure;
SameSite=Lax; Path=/` cookie. The table stores only their SHA-256, so a database
dump is a list of hashes rather than a set of live sessions. `Lax` and not
`Strict` because the callback arrives as a top-level navigation from Authentik,
and `Strict` drops the cookie on exactly that hop.

A session is remembered in-process for a minute, which is what is left of "the
request path does not touch Postgres". The cost, stated so it can be found
again: **signing out elsewhere takes up to a minute to bite here.**

### The middleware comes off

The forward-auth middleware must not be removed before the OIDC client exists
and Squirrel is serving `/auth`, or the deploy locks you out of your own pile.
The order is: add the client and the configuration with the outpost still in
front, verify signing in from behind it, then remove the middleware and verify
again from a browser with no Authentik session at all. That sequence has
already run once, in homelab, for the cutover on 25 August 2026; it is
recorded here for the next time the client or the group binding has to move.

The `ipAllowList` is unaffected and stays the outer layer.

## Cross-site writes

A form on someone else's site posting to `/board/act` would travel with the
session cookie like any other same-site-lax navigation. Every write route
therefore checks `Origin` (falling back to `Referer`) against the request's own
`Host`, and refuses anything that does not match or that says nothing at all.
`/auth/in` and `/auth/out` carry the same check: a cross-site POST to the first
is a login started by somebody else's page, and to the second is being signed
out of your own notes from a page you were only reading.

This requires the proxy to pass the original `Host` through. Traefik does by
default; if a middleware is ever added that rewrites it, every write on this
board turns into a 403 and the log line is
`refused a cross-site write`.

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

**The worker caches assets and never the board.** A cached board would show
strips that have already been triaged — the two views disagreeing with each
other, in the one place you could not tell. So the stylesheet, the script, the
fonts and the mark come from the cache, and everything else goes to the
network. With no network it answers with a page that says so and points out
that nothing has been lost, since capture was never the worker's job. See
`internal/web/static/sw.js`.
