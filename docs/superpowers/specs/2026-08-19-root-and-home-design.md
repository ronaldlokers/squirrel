# The screen moves to the root, and gains a front door

**Status:** design, approved 2026-08-19. Implementation plan to follow.

## What this changes

The screen lives at `/pile` and its second half at `/pile/chores`. That was
right when the pile *was* the screen. It has two halves now, and a phone opens
the installed app to whichever one happens to be the mount path.

After this:

- `/` is a home screen: two doors and a peek at the newest untriaged note.
- `/pile` is the deck, unchanged.
- `/chores` is the chores screen, moved up from `/pile/chores`.
- The configurable mount path goes away entirely.

## Why the mount path goes away

`WEB_PATH` exists because phase 5b did not want to assume where the screen
would live. Nothing has ever set it to anything but its default, and it costs
something everywhere: a `{{.Path}}` prefix on every URL in every template, a
`Service-Worker-Allowed` header so the worker can claim a scope one character
wider than the directory it was served from, `start_url` and `scope` computed
rather than stated, and an ingress that has to agree with all of it.

The screen is at the root. That is one fewer thing that can disagree with
itself, and the worker's scope becomes the thing it already wanted to be.

## The route table

| URL | What it is | Who may reach it from outside |
| --- | --- | --- |
| `/` | home | LAN or tailnet, then Authentik |
| `/pile` | the deck | LAN or tailnet, then Authentik |
| `/pile/act`, `/pile/chore` | the deck's writes | LAN or tailnet, then Authentik |
| `/chores` | what comes back | LAN or tailnet, then Authentik |
| `/chores/act` | a chore's writes | LAN or tailnet, then Authentik |
| `/pile/chores` | **301 to `/chores`** | LAN or tailnet, then Authentik |
| `/static/…` | stylesheet, script, fonts, mark, icons | LAN or tailnet, no identity |
| `/manifest.webmanifest` | the manifest | LAN or tailnet, no identity |
| `/sw.js` | the service worker | LAN or tailnet, no identity |
| `/hooks/home` | presence webhook | LAN or tailnet, its own token |
| `/transports/campfire` | Campfire's webhook | in-cluster; from outside, Authentik |
| `/healthz` | liveness and readiness | in-cluster; from outside, Authentik |

The assets, the manifest and the worker answer without an identity because a
browser fetches all three without cookies — the manifest from the page, the
icons and the worker from the browser process. Anything that can read a note
still requires one. This is the arrangement that took three releases to get
right in v0.9.1–v0.9.3 and it is not being revisited here, only re-pathed.

`/pile/chores` redirects rather than 404s: it is the URL the chores screen has
had since it existed, and a bookmark that dies quietly is worse than a
redirect nobody notices.

## At the edge

One rule replaces the enumeration:

```
/hooks/home            LAN + token, no identity      (unchanged)
/static/, /manifest…   LAN, no identity              (unchanged, re-pathed)
/outpost.goauthentik.io LAN, no identity             (unchanged)
/                       LAN + Authentik              (new, replaces /pile)
```

Traefik routes on the longest matching prefix, so the narrower rules keep
winning and none of them has to know about the broad one.

Two paths become reachable from the edge that were not before:
`/transports/campfire` and `/healthz`. Both now sit behind Authentik, so an
anonymous caller gets a login page rather than a webhook. Campfire itself
delivers in-cluster and never traverses the ingress, so nothing about delivery
changes. Nothing external monitors `/healthz` — gatus does not watch this
workload — and the kubelet's probes are in-cluster.

The alternative was enumerating every path at the edge. Rejected: every new
route would need a homelab change, and forgetting one produces a 404 that
looks like an application bug.

## The home screen

Three things, and deliberately no fourth:

1. **Two doors.** *The pile* and *the chores*, as equals. Neither is the
   primary action; they are the two halves of what Squirrel holds.
2. **A peek at the newest untriaged note** — its text and when it arrived,
   readable but not actionable. Tapping it goes to `/pile`, where that same
   note is the top card.
3. The shared lid: mark, wordmark, search. No cross-link on this screen, since
   both doors are on the page.

**The peek is read-only, and that is load-bearing.** If home could triage,
there would be two places that do, and the two views could disagree about what
a note is — the thing Principle 4 forbids. It also stops home growing into a
dashboard: a surface that shows state and does nothing is hard to turn into
one.

**No count, in any form.** Not "1 of 40", not "and more", not a dot beside a
door. The peek shows one note because the deck shows one note, not because one
is a number worth reporting.

**The empty pile is a normal state.** Home says so in the same words the deck
uses and keeps both doors exactly where they were. Nothing congratulates, and
nothing suggests filling it.

The visual design is Fable's comp at `.impeccable/comps/home-screen.html`,
which is normative for this screen the way the pile and chores comps are for
theirs.

## What the code stops carrying

- `Options.Path`, `Config.WebPath`, `WEB_PATH`, and every `{{.Path}}` in every
  template.
- `Service-Worker-Allowed` on the worker: a worker served from `/sw.js`
  already scopes to `/`.
`Here` stays, and gains a second job. The lid's cross-link keeps offering the
half you are not looking at, so moving between the pile and the chores never
goes via home; on home itself there is no cross-link, because both doors are on
the page. The mark and wordmark become a link to `/` on every screen except
home — the convention every website has, and the cheapest possible way back.

When the database is unreachable, home fails the way every other screen does:
503, the page that says Squirrel cannot reach its memory and that nothing has
been lost. The doors are part of that page's absence rather than shown over an
error — a door that leads to a 503 is worse than a page that says so once.

## Testing

- A route test that walks the whole table above, including the 301, so the
  shape is pinned rather than described.
- The home screen inherits the screen-wide tests: never a count, never a
  capture box, fails visibly when the database is unreachable.
- One test that home offers no triage control at all — no `act` field, no form
  posting to a write route. This is the property that keeps the two views from
  disagreeing, and it is invisible in a screenshot.
- A browser test that the worker's scope is `/` and that it controls the home
  page, replacing the current test of the same fact at `/pile`.
- The existing browser tests move with their screens.

## Rollout

The screen and the ingress must change together: a binary serving `/` behind
an ingress routing `/pile` is a 404, and the reverse is a 302 loop through
Authentik to a page that does not exist. Both land in one release, and the
homelab change merges after the image exists.

The installed app keeps working across the move: its `start_url` is `/pile`,
which still serves the deck. The manifest updates to `/` on its next fetch, so
the app opens at home after that. Nobody has to reinstall — which is the one
part of this that would be actively annoying to get wrong, since the last three
releases were spent teaching that app to have an icon at all.
