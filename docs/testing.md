# Running the tests

Unit tests need nothing:

    make test

Integration tests need a real Postgres. Start one:

    docker run --rm -d --name squirrel-test-db -p 55432:5432 \
      -e POSTGRES_USER=squirrel -e POSTGRES_PASSWORD=squirrel \
      -e POSTGRES_DB=squirrel_test postgres:17-alpine

Then:

    TEST_DATABASE_URL=postgres://squirrel:squirrel@127.0.0.1:55432/squirrel_test \
      make test-integration

Stop it with `docker rm -f squirrel-test-db`.

Integration tests are behind the `integration` build tag, so `make test` never
needs a database. They fail rather than skip when `TEST_DATABASE_URL` is unset,
so a missing variable in CI is a failure and not a green run over nothing.

Browser tests drive the real `pile.js` in a real browser:

    make test-browser

They need Chrome or Chromium on the path (or `BROWSER` pointing at one), and
nothing else — no database. They are behind the `browser` build tag so
`make test` keeps needing nothing at all, and CI runs them on every push.

They talk to the browser over the DevTools protocol, using a small websocket
client in `internal/web/cdp_test.go` rather than a library. `--dump-dom` was
tried first and cannot do this: it serialises the page before deferred scripts
have run, and with a service worker registered its virtual clock never expires.
Both failures look exactly like "the script did nothing", which is the one
answer a test must never give wrongly.
