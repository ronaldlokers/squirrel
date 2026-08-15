<img src="assets/logo.png" alt="" width="128" align="right">

# squirrel

A Campfire-driven external memory bot.

Talk to it in a direct message. It stores what you say, verbatim, before it
tries to understand any of it — and answers with a 🐿️.

Design notes live in [`docs/superpowers/specs/`](docs/superpowers/specs/).
Manifests live in [ronaldlokers/homelab](https://github.com/ronaldlokers/homelab);
this repository owns the code and ships one image.

## Running it

    make build
    ./squirrel

Configuration is environment only; see the table in the design doc. The one
setting no code can check is `CAMPFIRE_CONVERSATION_ID`, which **must** name a
direct room — the webhook payload carries no room-type field, so Squirrel
cannot verify it.

Tests: see [`docs/testing.md`](docs/testing.md).

## History

Phase 1 was first implemented in TypeScript and merged in #1. That version is
in the history and its tests encode what review found about Campfire's
behaviour — the response-header rules in particular. It was replaced by this Go
implementation to shrink the dependency surface; the reasoning is in
[`docs/superpowers/specs/2026-08-15-go-port-design.md`](docs/superpowers/specs/2026-08-15-go-port-design.md).
