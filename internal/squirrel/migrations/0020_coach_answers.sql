-- Every time a model spoke, and what it cost.
--
-- Two jobs, and the second is why the first is kept in full:
--
--   * The budget. Spend for a calendar month is a sum over this table, so
--     there is no counter to drift out of step with what actually happened.
--     Priced at write time rather than derived on read, so that changing the
--     price table later cannot silently rewrite what last month cost.
--
--   * Looking back at what it said. This product's coach is allowed to
--     evaluate, compare, and mention counts and streaks — Principle 5 was
--     opened deliberately on 20 August 2026 — which means it is allowed to say
--     something that lands badly on a bad day. A record you can go back and
--     read is the only way to tell "it was tactless" from "I remember it as
--     tactless", and the only way to judge whether changing the model in
--     configuration actually helped.
--
-- Kept indefinitely, by decision, on the same reasoning that keeps the
-- check-in history: the rows accumulate because they are what makes the thing
-- improvable, and nothing renders them as a series. Unlike the check-ins there
-- is no rule against reading these back — that is the whole point of them.
--
-- Rejected replies are stored too, with used = false. The guard throwing an
-- answer away does not make it free, and a budget that counted only the good
-- ones would be wrong in the direction that costs money.
create table if not exists coach_answers (
    id          bigserial   primary key,
    person_id   bigint      not null references people (id) on delete cascade,
    -- Which call site asked: 'chat', 'sheet', 'overwhelm', 'smaller', 'split',
    -- 'decide', 'interrupt'. Text rather than an enum — a new call site is a
    -- phase, and should not also be a migration.
    kind        text        not null,
    model       text        not null,
    prompt      text        not null,
    reply       text        not null,
    in_tokens   int         not null default 0,
    out_tokens  int         not null default 0,
    -- Micro-euros: millionths of a euro. A routine answer costs well under a
    -- tenth of a cent, so cents would round almost every row to zero and a
    -- month of them would sum to nothing.
    cost_micros bigint      not null default 0,
    -- Whether the reply reached a human. False for a guard rejection or a
    -- caller that had already fallen back.
    used        boolean     not null default false,
    said_at     timestamptz not null default now()
);

-- The only query the budget makes: this person's spend since the start of the
-- month.
create index if not exists coach_answers_person_month
    on coach_answers (person_id, said_at desc);
