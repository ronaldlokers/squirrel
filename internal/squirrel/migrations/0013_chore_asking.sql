-- When a chore is worth asking about.
--
-- A chore has a rhythm rather than a deadline: its clock starts when you do
-- it, not on a date in a calendar. That is the whole model, and it is why
-- "first of the month" is not here and will not be — a calendar anchor turns a
-- rhythm into a deadline, and a deadline is a thing you can be late for.
--
-- What *is* here is the other half of the sentence people actually say. "Every
-- other tuesday" is a fortnightly rhythm plus a preference for which day to
-- raise it; "bins out, evenings" is a fortnightly rhythm plus a preference for
-- which part of the day. Neither changes when the chore is due. They change
-- when it is worth interrupting you about it, which until now was "whenever
-- the nudge happened to fire".
--
-- Missing the window does not make the chore late. It is still exactly as due
-- as it was; the asking simply waits for the next window, the same way a
-- snooze stops the asking without touching the baseline.

-- Which days it is worth asking on, as a bitmask: bit 0 is Sunday, matching
-- Go's time.Weekday and Postgres's extract(dow). Null means any day, which
-- every existing row is, truthfully.
alter table chores add column if not exists ask_days smallint;

-- Which part of the day. Deliberately a word rather than two timestamps: a
-- chore that wants the evening does not want 17:00, and a time picker with
-- minutes on it is a precision tax charged every time you make one.
-- Null means any time.
alter table chores add column if not exists ask_part text
    check (ask_part is null or ask_part in ('morning', 'afternoon', 'evening'));
