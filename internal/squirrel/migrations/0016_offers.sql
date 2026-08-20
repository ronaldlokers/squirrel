-- What was offered, and what you said back.
--
-- Squirrel has never chosen anything. Every surface is organised by what kind
-- of thing a row is — a note, a task, a chore — and none of them by whether it
-- deserves your attention now. The picker is the thing that chooses, and this
-- table is the only state it needs: what you have already turned down today,
-- so that turning something down means something for longer than one page
-- load.
--
-- Answers only, never showings. Two reasons, and the first is the sharper one:
--
--   * A row written every time the offer is *rendered* would be a write on a
--     GET, on the one screen that is opened idly and repeatedly. The picker
--     would then be a thing that accumulates rows for looking, which is close
--     to a log of how often you opened the app while unable to start.
--
--   * There is nothing to learn from a showing that the answer does not
--     already say better. "Offered and refused" is a fact; "offered" alone is
--     a fact about the renderer.
--
-- So: no row exists until you press something. The picker reads today's
-- refusals and skips them, and that is the whole of its memory.
--
-- Deliberately unreadable as a series, the same discipline the checkins table
-- keeps and for the same reason: a fortnight of "you turned this down again"
-- is a sentence about the person, and this product does not write those. The
-- store exposes the refusals of one window as a set to skip, and nothing that
-- returns a history.
create table if not exists offers (
    id          bigserial   primary key,
    person_id   bigint      not null references people (id) on delete cascade,
    -- Which of the picker's rules produced it: 'chore', 'task', 'again',
    -- 'moment'. Text rather than an enum, because a seventh rule is a
    -- function and should not also be a migration.
    kind        text        not null,
    -- The chore or item it named. Null for an offer that names no row of its
    -- own — a timer already running is a thing you are doing, not a thing
    -- that was picked out of a table.
    ref_id      bigint,
    -- 'later' is the only answer that suppresses. 'did' and 'started' are
    -- recorded because they cost nothing and because "what did the picker
    -- actually cause" is the one question worth being able to ask of it later.
    answer      text        not null check (answer in ('later', 'did', 'started')),
    answered_at timestamptz not null default now()
);

-- The only query there is: this person's answers since a moment.
create index if not exists offers_person_recent on offers (person_id, answered_at desc);
