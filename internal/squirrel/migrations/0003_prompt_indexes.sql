-- Serves the baseline CTE in chores.go: a correlated subquery joins
-- prompt_lines to prompts per active chore to find each chore's last_shown
-- time. prompt_lines only has `unique (prompt_id, position)`, which does not
-- index the plain chore_id foreign key, so that subquery scanned the whole
-- table once per active chore.
create index prompt_lines_chore_id_idx on prompt_lines (chore_id);

-- Serves ChoreAtPosition in prompts.go: finding a person's most recent
-- prompt orders by sent_at desc. The existing partial index on
-- prompts (person_id, sent_for_date) does not serve that ordering.
create index prompts_person_sent_at_idx on prompts (person_id, sent_at desc);
