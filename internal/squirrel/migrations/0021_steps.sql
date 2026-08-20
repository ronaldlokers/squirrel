-- A thing broken into steps, handed back one at a time.
--
-- The table is here so that the pacing is the application's rather than the
-- reader's. A model may produce a sequence; a surface may never show one. What
-- makes that more than an intention is that the rows live here and the query
-- that reads them returns exactly one — there is deliberately no function in
-- the store that hands back the whole list, the same discipline the offers and
-- the check-ins already keep.
--
-- One sequence per person at a time, replaced wholesale, like the timer. Not
-- because a second one is unthinkable, but because two half-finished
-- breakdowns is a list of things you did not finish wearing a different hat —
-- and because the thing this helps with is starting, which is singular.
--
-- Steps are not notes. They are not in the pile, they are not searchable, they
-- do not survive being replaced, and nothing is ever promoted out of them.
-- What a model wrote is not a thought you had, and the two must not end up in
-- the same list.
--
-- item_id is null for a sequence about something that is not a row — a chore's
-- name, or whatever the picker was offering. The label is what it was about,
-- kept so a step can be shown next to the thing it belongs to without a join
-- that may find nothing.
create table if not exists steps (
    id        bigserial   primary key,
    person_id bigint      not null references people (id) on delete cascade,
    item_id   bigint      references items (id) on delete cascade,
    label     text        not null,
    -- First step first. Position rather than a linked list because the only
    -- questions ever asked are "which is next" and "there are no more".
    position  int         not null,
    body      text        not null,
    -- Set when it has been done. Marked rather than deleted, so pressing done
    -- on the last one leaves a finished sequence rather than an empty table
    -- that looks like nothing ever happened.
    done_at   timestamptz,
    made_at   timestamptz not null default now()
);

-- The only query: this person's next unfinished step.
create index if not exists steps_person_next
    on steps (person_id, position) where done_at is null;
