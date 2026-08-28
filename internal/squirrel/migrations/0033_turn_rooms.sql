-- Which room it was said in.
--
-- The screen became one conversation on 24 August 2026 and seven rooms on
-- 28 August; see docs/superpowers/specs/2026-08-28-rooms-design.md. A room is
-- both a place that keeps its own conversation and a scope that narrows what
-- Buddy can reach, and this column is the first half.
--
-- Text rather than an enum, for the same reason `who` is text: a new room
-- should be a phase, not a migration.
--
-- The default is what backfills the record. Everything said before rooms
-- existed was one conversation, and the room a conversation lives in is
-- Buddy's — not the pile's: the record holds his openings, his offers and his
-- answers as well as your notes, and filing all of that into the pile would
-- say the pile is where talking happens.
alter table turns add column if not exists room text not null default 'buddy';

-- Both queries are per-room now. Room leads because it is the equality and
-- said_at is the range; the other way round the index cannot skip a room.
drop index if exists turns_person_said;
create index if not exists turns_person_room_said
    on turns (person_id, room, said_at desc, id desc);
