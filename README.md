<img src="assets/logo.png" alt="" width="128" align="right">

# squirrel

A Campfire-driven external memory bot.

Talk to it in a direct message. It stores what you say, verbatim, before it
tries to understand any of it — and answers with a 🐿️.

Design notes live in [`docs/superpowers/specs/`](docs/superpowers/specs/).
Manifests live in [ronaldlokers/homelab](https://github.com/ronaldlokers/homelab);
this repository owns the code and ships one image.

## Running it

    npm ci
    npm run build
    node dist/src/index.js

Configuration is environment only; see the table in the design doc. The one
setting no code can check is `CAMPFIRE_CONVERSATION_ID`, which **must** name a
direct room — the webhook payload carries no room-type field, so Squirrel
cannot verify it.

Tests: see [`docs/testing.md`](docs/testing.md).
