-- How long a thing takes, measured.
--
-- Migration 0017 refused a history of timers, in writing, and the refusal was
-- right about what it refused: a record of every timer started and abandoned
-- is a record of what you do not finish, and this product does not keep those.
--
-- This is narrower, and the narrowing is the whole argument. **Only runs that
-- reached their end are written.** A timer stopped early writes nothing. A
-- timer replaced by starting another one writes nothing. A timer abandoned
-- because you closed the laptop writes nothing.
--
-- What that buys is a table with no failure rate in it. There is no query that
-- can be written against these rows that answers "how often do you not
-- finish", because the rows that would answer it are the ones that never
-- exist. What it can answer is "when you timed the bins, it was ten minutes",
-- which is a fact about the bins.
--
-- It is also the first thing here that *measures* rather than *remembers*, and
-- that is worth saying out loud. Everything else in this database is something
-- you said. This is something that happened, recorded so a model's guess about
-- duration can be replaced by an observation — decision 4, which allows a
-- guess for a first run and wants it overridden after a few real ones.
--
-- No person-facing surface reads this. The median goes to the coach's
-- typically() tool and nowhere else; nothing renders it, and there is
-- deliberately no store function that returns the runs themselves.
create table if not exists timer_runs (
    id        bigserial   primary key,
    person_id bigint      not null references people (id) on delete cascade,
    -- What you said you were doing, verbatim, as the timer carried it. Free
    -- text and usually a chore's name, but "the kitchen" is a perfectly good
    -- answer and is not a chore — which is why this is a label rather than a
    -- foreign key.
    label     text        not null,
    -- How long it ran for, in whole minutes. The length that was *asked for*
    -- and then seen through, which is the number worth having: a run that
    -- reached its end is one where ten minutes turned out to be enough.
    minutes   int         not null,
    ended_at  timestamptz not null default now()
);

-- The only query: this person's runs of one label.
create index if not exists timer_runs_person_label
    on timer_runs (person_id, label);
