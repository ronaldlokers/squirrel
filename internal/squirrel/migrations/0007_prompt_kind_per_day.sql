-- The per-day index is what makes a dated prompt idempotent: a restart inside
-- the send window cannot produce a second message, because the second insert
-- is refused. Phase 4 sends two dated prompts a day — a nudge and the evening
-- message — and without `kind` in the index the second one silently loses.
--
-- That failure would present as "the evening message just stops appearing on
-- days something else happened", which is close to invisible.
drop index prompts_digest_per_day;

create unique index prompts_kind_per_day
  on prompts (person_id, kind, sent_for_date)
  where sent_for_date is not null;
