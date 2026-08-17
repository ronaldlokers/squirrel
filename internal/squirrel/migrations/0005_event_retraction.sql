-- Undo must not delete the completion. The chore clock is
-- max(events.occurred_at), so deleting would restore the clock correctly and
-- destroy the record — and "completion is just an event" is exactly what lets a
-- sensor reset a chore with no new code. A retraction is a thing that visibly
-- happened rather than an absence.
alter table events add column retracted_at timestamptz;

-- Partial: the overwhelming majority of events are live, and every read filters
-- on this being null.
create index events_live_idx on events (chore_id, occurred_at desc)
  where retracted_at is null;
