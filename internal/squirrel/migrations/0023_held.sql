-- Three states for a thing you cannot act on.
--
-- The picker and the coach each hand over exactly one thing, so an offer you
-- cannot act on is not a small irritation — it spends the one decision you
-- were given. Until now the only ways to say so were to turn it down for
-- today, which asks again tomorrow, or to drop it, which is a lie about
-- something you still intend to do.
--
-- Three rather than one, because they end differently and that is the whole
-- argument:
--
--   * waiting — someone else replies. An event outside you, and one you will
--     want to record the moment it happens.
--   * blocked — a thing arrives or gets fixed. Also outside you, but there is
--     nobody to chase.
--   * someday — nothing ends it but you.
--
-- One "parked" state would make "chase the vet" and "learn to solder" the same
-- kind of thing. They are not: one is work with a dependency and the other is
-- a wish, and collapsing them is how a someday list becomes a guilt list.
--
-- Nothing else has to change to keep these out of the way. Every list in the
-- store names the state it wants — the pile is state = 'open', the tasks are
-- state = 'open', the archive is state = 'done' — so a new state is invisible
-- to every existing surface by construction. There is no filter anywhere that
-- has to remember these exist.
alter table items drop constraint if exists items_state_known;
alter table items add constraint items_state_known
  check (state in ('open', 'done', 'dropped', 'kept', 'waiting', 'blocked', 'someday'));

-- What you are waiting on, in your words.
--
-- Without it you get a list of stalled things and no idea what would unstall
-- them, which is a worse version of the pile it came out of. "the vet",
-- "the landlord", "the part to arrive".
--
-- New text you typed rather than a rewrite of the note, so the rule that keeps
-- a machine — and this column — away from raw_text is untouched. Null for
-- someday, which is not waiting on anything, and null for the times you did
-- not say.
alter table items add column if not exists held_because text;
