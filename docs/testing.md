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

## What the screens look like

`TestTheScreensLookLikeThemselves` records the *computed* shape of about
seventy elements across ten screens — sizes, weights, colours, spacing — and
fails when any of them moves. It lives with the browser tests and runs with
them.

It exists because both critique passes over this product found visual drift by
eye: a rule redefined somewhere global, several screens quietly repainted,
nobody the wiser until a person went looking. The rest of the browser suite
covers behaviour, which is the hard part to fake, and covered nothing at all
about appearance.

**Why not screenshots.** Font rasterisation differs between a developer's
machine and the runner, so a committed PNG fails on the first CI run for a
reason that has nothing to do with the change — and a check that cries wolf
teaches you to re-run the job instead of reading it. Computed values are what
the cascade actually settled on, they do not depend on how a glyph is drawn,
and they diff as text: the failure names the screen, the element and the
property, and reads like a sentence.

    /kept  .rbtn  letter-spacing: 0.625px → 3.75px
    /kept  .rbtn  width: 145.266px → 195.266px

When a change to appearance is the point of the commit, regenerate the record
deliberately:

    APPEARANCE=rewrite go test -tags=browser -run TestTheScreensLookLike ./internal/web/

Then **read the diff before committing it.** A snapshot that rewrites itself on
failure records whatever happened, which is the opposite of a fence — which is
why nothing regenerates it automatically and why the environment variable is
spelled out rather than being a flag you might pass by habit.
