# Deploying the pile screen

The screen at `squirrel.ronaldlokers.nl` reads and triages the pile: one note at
a time, the four transitions, undo, and search across every state. It never
creates an item, and it never shows a count.

Squirrel's half of this is three environment variables. The other half —
Authentik and Traefik — is infrastructure, lives in
[ronaldlokers/homelab](https://github.com/ronaldlokers/homelab), and is written
down here because no test covers it.

## Configuration

| Variable | Default | What it does |
| --- | --- | --- |
| `WEB_IDENTITY` | *(empty)* | The one identity that may read the pile. **Empty leaves the screen unmounted** — the route does not exist, and `GET /pile` is an ordinary 404. |
| `WEB_IDENTITY_HEADER` | `X-Authentik-Username` | The header the forward-auth middleware fills. |
| `WEB_PATH` | `/pile` | Where the screen is mounted. Its sub-routes (`/pile/act`, `/pile/chore`, `/pile/static/`) hang off it. |

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

## What the screen will not do

- **It will not create an item.** There is no capture box and no route that
  writes one. Two capture surfaces means two places to look for a thought, which
  is the problem this bot exists to solve. This is permanent, not a limitation.
- **It will not show a count.** No badge, no total, no "N to review", no page
  count. A capped list may say *that* there is more; it will never say how much.

## Assets after a change

`internal/web/static/` is embedded in the binary and served with
`Cache-Control: public, max-age=31536000` and no fingerprint in the filename.
A rebuilt image therefore does not repaint a browser that already has the old
stylesheet.

One hard reload fixes it: **Ctrl-Shift-R** (Cmd-Shift-R on macOS), or open
DevTools and use *Empty cache and hard reload*. Fingerprinting the filenames
would mean a build step, and this binary has none.
