-- Everything that has been said on the screen, in order.
--
-- The web surface stopped being a set of pages on 24 August 2026 and became one
-- conversation. This is that conversation, and it is kept indefinitely on the
-- same reasoning that keeps coach_answers and the check-in history: the record
-- is what the product is now, not a log of it.
--
-- words and shown are text. Neither is a foreign key, and that is the design
-- rather than an omission: a turn holding a chore id would re-read the chore at
-- render time and show today's name inside yesterday's sentence, which is
-- rewriting history. The duplication is what a record costs.
create table if not exists turns (
    id          bigserial   primary key,
    person_id   bigint      not null references people (id) on delete cascade,
    -- 'buddy' or 'you'. Text rather than an enum, for the same reason kind is
    -- text on coach_answers: a third speaker should be a phase, not a
    -- migration.
    who         text        not null,
    words       text        not null,
    -- What was drawn beneath the words — the cards, the chips, the picker — as
    -- it was drawn. Null for a turn that was only a sentence, and null is not
    -- an empty document: "there was no card" and "there was a card with no
    -- fields" are different facts about the turn.
    shown       jsonb,
    said_at     timestamptz not null default now()
);

-- The only two queries: the newest N for this person, and the N before a given
-- turn. Both walk this index backwards.
create index if not exists turns_person_said
    on turns (person_id, said_at desc, id desc);
