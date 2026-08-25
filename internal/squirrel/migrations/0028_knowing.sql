-- What Squirrel has come to know about how you work.
--
-- Buddy read the last few things said and nothing else. He could not say that
-- you put the same thing off every Sunday, or that you finish anything with a
-- phone call in it and nothing with a form, because there was no memory
-- outside a rolling window that forgets by design.
--
-- The turns table made this possible on 24 August 2026: it is a complete
-- record of everything that has happened on the screen. This is what a weekly
-- read of that record concluded, in sentences.
--
-- Three things about the shape, each of them a rule rather than a convenience.
--
-- Sentences, not fields. A schema would decide in advance what is worth
-- knowing about somebody, and the useful observations are the ones nobody
-- thought to add a column for. Text is the only honest shape for this.
--
-- Replaced wholesale, never appended. What Squirrel knows about you is what it
-- concluded most recently, not a pile that grows — an accumulating profile is
-- a thing that gets less true and more confident at the same time.
--
-- Readable and deletable by the person it is about. This is the only table in
-- the product that holds an opinion about somebody rather than something they
-- said, and a product that quietly builds a picture of you that you cannot see
-- is not this product. `learned_at` exists so the screen can say when.
create table if not exists knowing (
    id          bigserial   primary key,
    person_id   bigint      not null references people (id) on delete cascade,
    -- One observation, in the model's words. Held to a length in code rather
    -- than here: the bound is a matter of what fits in a prompt and on a card,
    -- and both change more often than a schema should.
    said        text        not null,
    learned_at  timestamptz not null default now()
);

-- The only query: everything known about this person, oldest first, so the
-- order a pass wrote them in is the order they are read in.
create index if not exists knowing_person
    on knowing (person_id, id);
