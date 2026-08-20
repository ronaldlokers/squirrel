-- What kind of thing a row is, beside what state it is in.
--
-- A note is a thought you had. A task is a thing you decided to do, once. They
-- share a table because they are the same row at different moments — a task is
-- usually a note you made a decision about, and undoing that decision must
-- return the same row to the same place rather than copying it somewhere.
--
-- A kind rather than a fifth state, deliberately: a task that is done would
-- otherwise be indistinguishable from a note that is done, and the archive
-- could not exist. It is also the smaller change — search, undo, the
-- transitions and the capture path all carry on untouched, because none of
-- them cares what kind a row is.
--
-- Defaulting to 'note' makes every existing row what it already was, so there
-- is nothing to backfill and nothing to get wrong while backfilling it.
alter table items add column if not exists kind text not null default 'note'
    check (kind in ('note', 'task'));

-- The pile and the tasks screen both read (person, kind, state) and order by
-- arrival, which is what this covers.
create index if not exists items_person_kind_state
    on items (person_id, kind, state, received_at desc, id desc);
