# Running the tests

Unit tests need nothing:

    npm test

Integration tests need a real Postgres. Start one:

    docker run --rm -d --name squirrel-test-db -p 55432:5432 \
      -e POSTGRES_USER=squirrel -e POSTGRES_PASSWORD=squirrel \
      -e POSTGRES_DB=squirrel_test postgres:17-alpine

Then:

    TEST_DATABASE_URL=postgres://squirrel:squirrel@127.0.0.1:55432/squirrel_test \
      npm run test:integration

Stop it with `docker rm -f squirrel-test-db`.

The integration suite refuses to run without `TEST_DATABASE_URL` rather than
skipping, so a missing variable in CI is a failure rather than a green run over
nothing.
