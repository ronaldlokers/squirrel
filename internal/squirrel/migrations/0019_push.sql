-- Where to reach you when you are not looking at the room.
--
-- Everything Squirrel says goes to a Campfire room you have to be looking at.
-- For a nudge that is right — it is a suggestion, and a suggestion that waits
-- is a suggestion doing its job. For "leave about 14:10" it is not good
-- enough: that message has one useful minute, and a chat notification you
-- notice at 14:40 is worse than none, because it teaches you not to trust the
-- next one.
--
-- One row per browser that asked. Deliberately not one per person: the same
-- person installs the screen on a phone and a laptop, and a warning that only
-- reached the machine you were not holding is the failure this exists to fix.
--
-- Nothing here is a preference or a setting. A subscription is a place to
-- send to; whether Squirrel *should* send is decided by the same rules that
-- decide whether it should say anything at all, and those live in code rather
-- than in a table of toggles nobody will ever revisit.
create table if not exists push_subscriptions (
    id         bigserial   primary key,
    person_id  bigint      not null references people (id) on delete cascade,
    -- The push service's own URL for this browser. Unique because a browser
    -- that re-subscribes hands back the same endpoint, and two rows would mean
    -- two copies of every message.
    endpoint   text        not null unique,
    -- The browser's public key and shared secret, base64url, exactly as the
    -- subscription hands them over. Stored as text rather than decoded: they
    -- are opaque to everything here except the encryption, and re-encoding
    -- them is a chance to be wrong.
    p256dh     text        not null,
    auth       text        not null,
    created_at timestamptz not null default now(),
    -- When the push service last refused this endpoint for good. A browser
    -- that has been uninstalled answers 404 or 410 forever, and a row that
    -- keeps being retried is a row that makes every send slower.
    gone_at    timestamptz
);

create index if not exists push_live on push_subscriptions (person_id)
    where gone_at is null;
