# The screen moves to the root, and gains a front door

**Status:** design, approved 2026-08-19. Implementation plan to follow.

## What this changes

The screen lives at `/pile` and its second half at `/pile/chores`. That was
right when the pile *was* the screen. It has two halves now, and a phone opens
the installed app to whichever one happens to be the mount path.

After this:

- `/` is a home screen: two doors, and nothing that depends on what the
  pile holds.
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

Two doors, and deliberately nothing else: *the pile* and *the chores*, as
equals, each carrying its own illustration. The shared lid keeps the mark, the
wordmark and the search field, and drops its cross-link — the doors are the
body of the page.

**There is no preview of the pile.** An earlier draft put the newest note here,
read-only and hedged about with guards. It was cut on sight: a home screen that
shows what is waiting greets you with what is waiting, however carefully it is
dressed. Nothing on this screen depends on what the pile holds, so a full pile
and an empty one render identically — which is what "stopping partway is a
normal ending" looks like when it is structural rather than a reassuring
sentence.

That also removes the question of whether home may triage: there is nothing on
it to triage. All writes stay on the two screens that own them, and the two
views cannot disagree about a note because only one of them ever shows one.

No count, in any form: not on a door, not beside it, not as a dot.

The visual design is Fable's comp at `.impeccable/comps/home-screen.html`,
which is normative for this screen the way the pile and chores comps are for
theirs. The door illustrations are the owner's own, and the chores one carries
a documented exception to two house rules — it shows grey and depicts a
completion. That exception is recorded in the comp's notes and belongs in
`DESIGN.md` beside the Door Art component rather than left to look like drift.

## What the code stops carrying

- `Options.Path`, `Config.WebPath`, `WEB_PATH`, and every `{{.Path}}` in every
  template.
- `Service-Worker-Allowed` on the worker: a worker served from `/sw.js`
  already scopes to `/`.

## What the lid does instead

`Here` stays, and gains a second job. The lid's cross-link keeps offering the
half you are not looking at, so moving between the pile and the chores never
goes via home; on home itself there is no cross-link, because both doors are on
the page. The mark and wordmark become a link to `/` on every screen except
home — the convention every website has, and the cheapest possible way back.

## When the database is down

Home reads nothing from the database, so there is nothing on it to fail. It
renders from the templates alone and the doors work whether Postgres is
reachable or not; the screen you walk into then says what is wrong, once, in
the words it already uses.

## Testing

- A route test that walks the whole table above, including the 301, so the
  shape is pinned rather than described.
- The home screen inherits the screen-wide tests: never a count, never a
  capture box, fails visibly when the database is unreachable.
- One test that home renders identically whether the pile is empty or full.
  That is the property that keeps it from growing a preview by accident, and it
  is the cheapest possible guard against the whole class of change.
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
