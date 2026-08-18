-- A numbered line used to be a chore, always. The pile and search results are
-- numbered too, and `done 2` has to resolve against whichever kind of line
-- position 2 turned out to be.
--
-- Both columns nullable with a check that exactly one is set, rather than a
-- second table for note lines. The numbering, the uniqueness on
-- (prompt_id, position), and the cascade from prompts are all properties of a
-- *line*; a second table would have to duplicate every one of them, and
-- "which table holds position 2 of the most recent prompt" is a question no
-- caller should ever have to ask.
alter table prompt_lines alter column chore_id drop not null;

alter table prompt_lines add column item_id bigint references items (id);

-- Exactly one target, enforced here rather than by every caller remembering.
-- `<>` on two booleans is XOR: true when exactly one side is null.
alter table prompt_lines add constraint prompt_lines_one_target
  check ((chore_id is null) <> (item_id is null));
