create table people (
  id         bigint generated always as identity primary key,
  handle     text        not null unique,
  created_at timestamptz not null default now()
);

create table identities (
  id          bigint generated always as identity primary key,
  person_id   bigint      not null references people (id),
  transport   text        not null,
  external_id text        not null,
  created_at  timestamptz not null default now(),
  unique (transport, external_id)
);

create table items (
  id              bigint generated always as identity primary key,
  transport       text        not null,
  external_id     text,
  conversation_id text,
  sender_id       text,
  person_id       bigint      references people (id),
  raw_text        text        not null,
  payload         jsonb       not null,
  received_at     timestamptz not null,
  inserted_at     timestamptz not null default now()
);

-- Partial, because the fail-open path has no external id to be unique on and
-- several such rows must coexist. The drain's ON CONFLICT must carry a
-- matching predicate or Postgres rejects the statement.
create unique index items_transport_external_id_key
  on items (transport, external_id)
  where external_id is not null;

create index items_received_at_idx on items (received_at);
