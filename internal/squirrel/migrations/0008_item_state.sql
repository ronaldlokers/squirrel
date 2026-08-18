-- items had no state at all: a note was raw text and a timestamp, permanently,
-- and the evening message showed it exactly once — the day it was written. This
-- is the exit the pile never had.
--
-- Four states, and `kept` is the load-bearing one. A serial number, a link, a
-- name someone mentioned: these are not tasks and will never be `done`. With
-- only done and dropped on offer, every reference note sits in triage forever
-- and the pile reappears inside the thing built to clear it.
--
-- Every existing row becomes 'open', which is true. Nothing has ever been
-- triaged, because until now there was nothing to triage it into.
alter table items add column state text not null default 'open';

-- Null until the first transition. received_at says when the thought arrived,
-- this says when it was dealt with, and those are different questions — the
-- evening message asks the first one and does not care about the second.
alter table items add column state_at timestamptz;

-- The vocabulary is closed here rather than in Go, because the Go constants
-- and a hand-written UPDATE are two different doors into the same table.
alter table items add constraint items_state_known
  check (state in ('open', 'done', 'dropped', 'kept'));

-- The pile is "open, newest first", which is exactly this index.
--
-- Search is deliberately left unindexed: it is a substring scan over one
-- person's rows, run by hand a few times a day. A trigram index would be real
-- work to maintain for a query that has never once been slow.
create index items_state_received_idx on items (state, received_at desc);
