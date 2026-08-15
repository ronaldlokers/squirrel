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
