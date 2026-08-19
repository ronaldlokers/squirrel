-- "Not today."
--
-- A chore had two answers: do it, or let it keep asking. The second is what
-- makes a nudge into a nag, and the nag arrives at the moment you are least
-- able to decide anything about it — so the honest third answer is to say when
-- to ask again.
--
-- Deliberately a timestamp on the chore rather than a state or an event:
--
--   * Not a state, because a snoozed chore is not a different kind of chore.
--     It is the same chore with a date before which nobody wants to hear from
--     it, and it becomes ordinary again by the clock rather than by anything
--     anyone has to remember to do.
--
--   * Not an event, because events are things that happened to a chore and
--     feed the baseline that decides when it is next due. A snooze must not
--     move that baseline: putting the bins out is what resets the bin clock,
--     and "ask me tomorrow" is not putting the bins out. It only silences the
--     asking.
--
-- Null means "never snoozed", which every existing row is, truthfully.
alter table chores add column if not exists snoozed_until timestamptz;

-- The nudge path filters on it, and a chore is only ever snoozed one at a
-- time, so this stays small and is only read on the due query.
create index if not exists chores_snoozed_until on chores (snoozed_until)
    where snoozed_until is not null;
