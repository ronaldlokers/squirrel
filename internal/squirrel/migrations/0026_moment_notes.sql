-- A note can point at a fixed point.
--
-- The pointer rather than new columns on `moments`, and the difference is the
-- product's deepest rule: a thought that lives on an appointment instead of in
-- the pile is a thought `!find` cannot reach. This way a note keeps everything
-- it already has — capture, photograph, editing, search, undo — and only gains
-- somewhere to be.
--
-- No new value in `items.state`, deliberately. The pointer is the disposition:
-- a note with somewhere to be is not waiting to be decided about, and one whose
-- appointment has passed is waiting again with no transition to write. That is
-- what makes this one column rather than an eighth state seven screens have to
-- learn about.
--
-- `on delete set null` rather than cascade. Deleting an appointment must never
-- delete the owner's words; the note returns to the pile, which is the same
-- thing that happens when the appointment is simply over.
alter table items add column moment_id bigint references moments(id) on delete set null;

-- Partial, because the overwhelming majority of notes point at nothing and an
-- index over all of them would be mostly nulls.
create index items_moment_id_idx on items (moment_id) where moment_id is not null;
