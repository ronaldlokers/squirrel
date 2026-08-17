create table chores (
  id                bigint generated always as identity primary key,
  person_id         bigint      not null references people (id),
  name              text        not null,
  interval_seconds  bigint      not null,
  tolerance_seconds bigint      not null,
  active            boolean     not null default true,
  created_at        timestamptz not null default now(),
  updated_at        timestamptz not null default now()
);

-- Upsert is by name, case-insensitively, so "Vacuum" updates "vacuum" rather
-- than creating a second chore. Without this the upsert is not safe to rely on.
create unique index chores_person_name_key on chores (person_id, lower(name));

create table events (
  id          bigint generated always as identity primary key,
  chore_id    bigint      references chores (id),
  item_id     bigint      references items (id),
  person_id   bigint      references people (id),
  -- 'ack' today. 'sensor' and 'inference' are why this table exists.
  source      text        not null,
  occurred_at timestamptz not null,
  payload     jsonb       not null default '{}',
  inserted_at timestamptz not null default now()
);

create index events_chore_occurred_idx on events (chore_id, occurred_at desc);

create table prompts (
  id              bigint generated always as identity primary key,
  person_id       bigint      not null references people (id),
  conversation_id text        not null,
  kind            text        not null,
  sent_at         timestamptz not null,
  sent_for_date   date
);

-- Partial, because on-demand queries have no date. This index is the whole of
-- the scheduler's idempotency: a restart cannot produce a second digest.
create unique index prompts_digest_per_day
  on prompts (person_id, sent_for_date)
  where sent_for_date is not null;

create table prompt_lines (
  id        bigint generated always as identity primary key,
  prompt_id bigint not null references prompts (id) on delete cascade,
  position  int    not null,
  chore_id  bigint not null references chores (id),
  unique (prompt_id, position)
);
