-- "How is right now."
--
-- The owner asked for this and it is the sharpest thing in the product, so the
-- reasoning is here rather than in a commit message nobody will find.
--
-- A record of how someone felt is data *about the person*, and this product's
-- rule is that nothing accrues that can be destroyed. A fortnight of your own
-- bad days, rendered as a row of faces, is the counter this project refuses,
-- wearing a different hat: the same loss aversion, the same all-or-nothing
-- reading, the same abandonment when the run breaks.
--
-- The owner chose, deliberately, to keep the history and never show it. So:
--
--   * The rows accumulate here, because a nudge that knows you have been flat
--     for a while can be gentler, and that is the whole point of asking.
--
--   * No surface may render more than the latest one. Not a series, not a
--     count, not a trend, not an average, not "you have been low three days
--     running" — which is a sentence about the person, and this product does
--     not write those. That rule is enforced by the store's signature rather
--     than by this comment: LatestCheckin returns one row, and nothing else
--     reads this table.
--
-- Deliberately its own table rather than a column on people: a column would be
-- overwritten and the history is the thing that makes the asking useful. And
-- deliberately not the events table, which is what happened to a *chore* and
-- feeds the baseline that decides when one is next due.
create table if not exists checkins (
    id         bigserial primary key,
    person_id  bigint      not null references people (id) on delete cascade,
    -- One of the five drawn moods. Text rather than an enum: a sixth mood is a
    -- drawing plus a word, and should not also be a migration.
    mood       text        not null,
    -- Where it was said, so the two surfaces can be told apart later if that
    -- ever matters. Nothing reads it today.
    source     text        not null default 'chat',
    said_at    timestamptz not null default now()
);

-- The only query there is: the latest one for a person.
create index if not exists checkins_person_latest on checkins (person_id, said_at desc);
