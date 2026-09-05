-- The screen's own capture had an id to point at, InsertItemReturningID, but
-- nothing a client could repeat on a retry: a dropped response after the row
-- landed reads as a failure, and the retry that follows writes it twice.
--
-- Kept apart from external_id rather than reusing it: external_id is what a
-- transport says a message's own identity is, and this is a value the client
-- invents purely so a repeat of the same request is recognisable as a repeat.
-- Mixing the two would make a client-invented key collide with, or be
-- mistaken for, an id a transport actually issued.
--
-- Unique on its own, not paired with transport: the key is a UUID picked once
-- per capture attempt and never reused for a different one, so there is
-- nothing for the transport to disambiguate.
alter table items add column capture_key text;

create unique index items_capture_key_key
  on items (capture_key)
  where capture_key is not null;
