-- "That landed badly", against the reply it is about.
--
-- Principle 5 was opened on 20 August 2026 so the coach could be useful at the
-- only thing a coach is for: it may evaluate, compare, and mention counts. The
-- cost was stated at the time and is real — it can now say something that
-- lands badly on a bad day — and `coach_answers` has kept every exchange since
-- for exactly that reason, so that "it was tactless" can be told apart from "I
-- remember it as tactless".
--
-- Kept, and never read. This column is the missing half.
--
-- A flag rather than a comment box, because the moment this exists to serve is
-- the moment you have least to spend on it: a bad reply on a bad night, and
-- one press to say so. What it is *for* is not a feelings log — it is the
-- input to the next prompt, so the coach is shown what does not land here
-- rather than told again in an instruction nobody can check.
--
-- Nullable rather than a default of false. Null is "never said", false is
-- nothing at all, and only the first is true of every row written before this
-- migration. Nothing counts these and nothing renders them as a number: rule 2
-- holds here as everywhere, and a tally of one's own bad nights is exactly the
-- accruing thing it forbids.
alter table coach_answers
    add column if not exists landed_badly_at timestamptz;

-- The one query this supports: the most recent few, newest first, so a prompt
-- can carry them. Partial, because the rows that matter are a tiny fraction of
-- the table and there is no question anybody asks of the rest.
create index if not exists coach_answers_landed_badly
    on coach_answers (person_id, landed_badly_at desc)
    where landed_badly_at is not null;
