-- Chores that come back on a day, not after an interval.
--
-- A chore was `every N seconds` from 0002 until 26 August 2026, and almost
-- nothing real is: the bins are alternating Thursdays, rent is the first of the
-- month, a boiler service is annual and drifts. An interval measured from the
-- last completion also *slides* — do the bins a day late once and every
-- subsequent reminder is a day late too, which is exactly the failure the
-- feature exists to prevent.
--
-- Null in both columns is the old behaviour, and that is deliberate: every
-- chore that exists today keeps its interval and nothing about it changes.
--
-- interval_seconds is still written for a weekday chore — seven days times
-- every_weeks — and it is not dead. The tolerance gate, the asking window and
-- everything that renders "how often" read it, so a chore that answers those
-- questions in the old vocabulary needs no change anywhere else. What the two
-- columns below add is only *when it is due*.
alter table chores add column if not exists on_weekday  smallint;
alter table chores add column if not exists every_weeks smallint;

-- 0 is Sunday, matching Postgres's extract(dow) and Go's time.Weekday, which
-- agree with each other and are the two things this value is compared against.
alter table chores drop constraint if exists chores_on_weekday_range;
alter table chores add constraint chores_on_weekday_range
  check (on_weekday is null or (on_weekday >= 0 and on_weekday <= 6));

-- Weekly or alternating. Three-weekly exists in nobody's house, and a column
-- that permits it is a column somebody sets to 7 by accident.
alter table chores drop constraint if exists chores_every_weeks_range;
alter table chores add constraint chores_every_weeks_range
  check (every_weeks is null or (every_weeks >= 1 and every_weeks <= 2));

-- Both or neither. A weekday with no period, or a period with no weekday, is a
-- half-configured rhythm that would fall back to the interval silently.
alter table chores drop constraint if exists chores_weekday_pair;
alter table chores add constraint chores_weekday_pair
  check ((on_weekday is null) = (every_weeks is null));
