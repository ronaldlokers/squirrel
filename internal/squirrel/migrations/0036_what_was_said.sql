-- What Squirrel told you, kept.

-- A push has been fire-and-forget since it shipped: the payload went to the
-- push service and nothing on this side remembered that it had. So the app
-- could not answer the one question a bell in the top bar implies — what did
-- you tell me? — and a phone that was off, or a notification swiped away
-- without being read, lost it for good.

-- One row per push, written where the fan-out happens rather than per
-- subscription: two browsers on one account are two deliveries of one thing
-- said, and a list that showed it twice would be a list about plumbing.
create table if not exists said (
    id        bigserial   primary key,
    person_id bigint      not null references people (id) on delete cascade,
    title     text        not null,
    body      text        not null,
    -- Where pressing it went, kept as sent. A row here is a record of what was
    -- said at a moment, so it is not re-derived later from a board that has
    -- moved on.
    url       text        not null default '',
    said_at   timestamptz not null
);

-- The only read: this person's, newest first.
create index if not exists said_person_at on said (person_id, said_at desc);
