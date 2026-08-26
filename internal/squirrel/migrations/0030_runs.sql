-- Where you got to, when something interrupted you.
--
-- Losing your place is the failure this product is built around, and until
-- 26 August 2026 it kept no memory of a run in progress at all: you triaged
-- three notes, the phone rang, and forty minutes later the conversation opened
-- as if nothing had been happening.
--
-- One row per person, not one per run. A second run replaces the first because
-- you are not doing two — and a table that accumulated runs would be a history
-- of your afternoons, which is the one thing this product refuses to keep.
--
-- No count is stored. How many are left is read from the pile when the run is
-- offered, so a note dropped from Campfire in between cannot make this row lie.
create table if not exists runs (
    person_id  bigint      primary key references people (id) on delete cascade,
    -- Which door. `pile` is the only one that loops today; the column exists so
    -- that the tasks and the chores can join without a migration.
    place      text        not null,
    started_at timestamptz not null,
    seen_at    timestamptz not null
);
