-- What it noticed about one thing, kept beside that thing.

-- The pulled strip has carried a noticed line since v0.65.0, but only the one
-- the picker had already chosen. Everything else in a rack is read without
-- anything ever having been said about it, and the things worth saying — this
-- is the third note about the same subject, the number you need is in another
-- one — are exactly the ones a person cannot see by looking at a list.

-- Keyed the way an offer is: a kind and the id within it, rather than a foreign
-- key per kind. Three tables' worth of rows can be noticed about and none of
-- them may gain a column for this.
create table if not exists noticed (
    id         bigserial   primary key,
    person_id  bigint      not null references people (id) on delete cascade,
    kind       text        not null,
    ref_id     bigint      not null,
    words      text        not null,
    made_at    timestamptz not null,
    -- When you said it was not useful. The row stays: what was refused is the
    -- signal that stops the next one being like it, and a deleted row teaches
    -- nothing.
    refused_at timestamptz
);

-- One line per thing. A second observation about the same strip replaces the
-- first rather than stacking, because two lines under one strip is a
-- conversation and this is deliberately not one.
create unique index if not exists noticed_one_per_thing
    on noticed (person_id, kind, ref_id);

-- The read: everything not refused, newest first.
create index if not exists noticed_person_made
    on noticed (person_id, made_at desc);
