-- Who is signed in, and until when.
--
-- Squirrel's whole authentication was one header comparison behind an Authentik
-- forward-auth outpost until 25 August 2026. The application does OIDC itself
-- now, which means it has to remember who you are between requests.
--
-- The cookie carries 32 random bytes; this table stores only their SHA-256.
-- Not for secrecy — the token has 256 bits of entropy and is not guessable —
-- but so that read access to this table is not read access to the product. A
-- database dump is then a list of hashes rather than a set of live sessions.
--
-- sub is the OIDC subject, carried here rather than joined for. The capture
-- path writes it as a sender string, and the drain resolves a spooled
-- capture's owner from that — see the design's section 4 for why a person with
-- no screen identity spools notes belonging to nobody.
create table if not exists sessions (
    id           bigserial   primary key,
    person_id    bigint      not null references people (id) on delete cascade,
    token_sha256 bytea       not null unique,
    sub          text        not null,
    created_at   timestamptz not null default now(),
    seen_at      timestamptz not null default now(),
    expires_at   timestamptz not null
);

-- The only two queries: one session by its hash, and everything belonging to a
-- person when they are deleted or signed out everywhere.
create index if not exists sessions_person on sessions (person_id);
