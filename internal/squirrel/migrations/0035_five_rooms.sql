-- Seven rooms became five.

-- The rail had been carrying two different kinds of thing under one shape.
-- Five of the seven were places you keep something; two — the things you kept
-- and what you set aside — were states a note is in, promoted to doors on
-- 28 August 2026 because there was nowhere else to put them. A state with a
-- door on the rail reads as a fourth pile to stay on top of, which is the
-- weight this product exists to remove. Both are chips inside the notes now.

-- The collapse is what forces a migration: three keys become one, so a
-- conversation held in any of them has to be told where it lives. Nothing is
-- deleted and nothing is merged out of order — the three shared one person's
-- timeline already, and said_at still orders them.
update turns set room = 'notes' where room in ('pile', 'held', 'kept');

-- And the rename, which rides along because the write is already happening.
-- 'buddy' named the speaker rather than the place; 'everything' names what the
-- room holds, which is the only one of the five that holds all of it.
update turns set room = 'everything' where room = 'buddy';

-- The default follows the column's meaning, not its history: a turn written
-- with no room named belongs where the conversation lived before rooms
-- existed, and that room is called 'everything' now.
alter table turns alter column room set default 'everything';
