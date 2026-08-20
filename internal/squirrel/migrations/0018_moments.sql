-- A time the world imposed.
--
-- This product has refused deadlines from the beginning, and the refusal is
-- right: an invented due date accrues lateness, and lateness is the shape
-- Squirrel exists without. A chore has a rhythm; its clock starts when you do
-- it, and it is never late.
--
-- But a dentist appointment at 14:30 exists whether or not this app knows about
-- it. Refusing to hold it does not remove the lateness, it only means Squirrel
-- cannot help you leave. So the rule is sharpened rather than broken, and the
-- new wording is the one to hold everything here against:
--
--     Squirrel never invents a time you can be late for.
--     It may hold one the world did.
--
-- What that permits, and what it still forbids:
--
--   * Nothing here is ever marked late, and nothing accrues. A moment that has
--     passed is over — no overdue list, no count of missed ones, no history to
--     read back. The row stays only until it is done with.
--
--   * No recurrence. A thing that comes back on its own is a chore, and a
--     chore has a rhythm rather than a date.
--
--   * No list screen. A browsable set of your appointments is a calendar, and
--     a calendar is a thing you are behind on. A moment is only ever shown
--     inside the window where leaving matters.
--
-- The chain is the point rather than the time: 14:30 means start wrapping up,
-- get your things, leave, travel, arrive. Squirrel holds the whole chain and
-- says one thing at the moment it matters.
create table if not exists moments (
    id         bigserial   primary key,
    person_id  bigint      not null references people (id) on delete cascade,
    -- The note it came from, when it came from one. Null for a moment made
    -- outright — the same shape !task already allows.
    item_id    bigint      references items (id),
    label      text        not null,
    starts_at  timestamptz not null,

    -- How long it takes to get there. Null means nobody said, and the code
    -- assumes a quarter of an hour *and says so out loud* — a silent
    -- assumption is how you end up late while trusting the machine.
    travel_secs int,
    -- How long it takes to be ready to walk out. Null takes the default.
    ready_secs  int,

    -- What to take. One line of free text, because "keys, wallet" is the whole
    -- of what anyone needs written down, and a checklist of items is a second
    -- list to maintain at the exact moment nobody can maintain anything.
    bring      text,

    -- When the leave-by warning was sent, so a tick that runs twice does not
    -- say it twice. Null while it has not been said.
    said_at    timestamptz,
    -- When you said you were leaving, or when it stopped mattering. Set means
    -- the moment is done with and nothing will raise it again.
    done_at    timestamptz,
    created_at timestamptz not null default now()
);

-- The only query there is: this person's next moment that is not done with.
create index if not exists moments_person_next
    on moments (person_id, starts_at) where done_at is null;
