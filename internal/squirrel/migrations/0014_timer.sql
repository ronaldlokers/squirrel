-- A timer, and only ever the current one.
--
-- This is a body double rather than a stopwatch: you say how long and what on,
-- Squirrel says go, and then says nothing at all until the end. The point is
-- the going, so nothing here is designed to be watched.
--
-- One row per person, replaced each time. There is deliberately no history of
-- timers started and abandoned — that record is a streak with a different
-- name, and abandoning one halfway is a normal ending here exactly as stopping
-- partway through the pile is. The unique constraint is what makes that
-- structural rather than a promise: there is nowhere to accumulate.
create table if not exists timers (
    person_id  bigint      primary key references people (id) on delete cascade,
    -- What you said you were doing. Free text: it is usually a chore's name,
    -- but "the kitchen" is a perfectly good answer and is not a chore.
    label      text        not null,
    started_at timestamptz not null default now(),
    ends_at    timestamptz not null,
    -- Set when the end has been announced, so a scheduler tick that runs twice
    -- does not say "time" twice. Null while it is still running.
    said_at    timestamptz
);
